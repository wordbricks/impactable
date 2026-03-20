package wtl

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type observer interface {
	Observe(runEvent)
}

type turnRunner interface {
	Start(context.Context, runConfig) (string, error)
	RunTurn(context.Context, string, string, func(string)) (turnResult, error)
	Close() error
}

type threadCompactor interface {
	Compact(context.Context, string) error
}

type engine struct {
	runner    turnRunner
	policy    policy
	observers []observer
	maxIter   int
	maxRetry  int
	runID     string
	phase     string
}

func (e *engine) run(ctx context.Context, cfg runConfig, prompt string) (runSummary, error) {
	summary := runSummary{
		RunID:         e.runID,
		MaxIterations: e.maxIter,
		MaxRetry:      e.maxRetry,
		LastPhase:     e.phase,
	}
	e.emit(newEvent(eventRunStarted))

	threadID, err := e.runner.Start(ctx, cfg)
	if err != nil {
		summary.Status = statusFailed
		summary.LastError = err.Error()
		summary.CompletedAtUTC = time.Now().UTC().Format(time.RFC3339)
		e.emit(runFailureEvent(eventRunFailed, e.runID, "", 0, err))
		return summary, err
	}
	summary.ThreadID = threadID

	retries := 0
	for iteration := 1; iteration <= e.maxIter; iteration++ {
		if err := ctx.Err(); err != nil {
			summary.Status = statusInterrupted
			summary.Iterations = iteration - 1
			summary.CompletedAtUTC = time.Now().UTC().Format(time.RFC3339)
			e.emit(runInterruptedEvent(e.runID, threadID, iteration-1))
			return summary, err
		}

		started := newEvent(eventTurnStarted)
		started.RunID = e.runID
		started.ThreadID = threadID
		started.Phase = e.phase
		started.Iteration = iteration
		e.emit(started)

		result, turnErr := e.runner.RunTurn(ctx, threadID, prompt, func(delta string) {
			deltaEvent := newEvent(eventTurnDelta)
			deltaEvent.RunID = e.runID
			deltaEvent.ThreadID = threadID
			deltaEvent.Phase = e.phase
			deltaEvent.Iteration = iteration
			deltaEvent.Text = delta
			e.emit(deltaEvent)
		})

		if errors.Is(turnErr, context.Canceled) || result.Status == "interrupted" {
			summary.Status = statusInterrupted
			summary.Iterations = iteration
			summary.FinalResponse = result.Response
			summary.CompletedAtUTC = time.Now().UTC().Format(time.RFC3339)
			e.emit(turnFinishedEvent(e.runID, threadID, e.phase, iteration, statusInterrupted, directiveUnsupported, result.Response, turnErr))
			e.emit(runInterruptedEvent(e.runID, threadID, iteration))
			return summary, nil
		}

		outcome := turnOutcome{
			Response: result.Response,
			Err:      turnErr,
			Status:   result.Status,
		}
		next := e.policy.Next(outcome)
		summary.LastDirective = next
		if turnErr != nil {
			summary.LastError = turnErr.Error()
		} else if result.ErrorMessage != "" {
			summary.LastError = result.ErrorMessage
		}
		e.emit(turnFinishedEvent(e.runID, threadID, e.phase, iteration, statusFromOutcome(outcome), next, result.Response, turnErr))

		switch next {
		case directiveContinue:
			retries = 0
			summary.Iterations = iteration
			summary.FinalResponse = result.Response
		case directiveComplete:
			summary.Status = statusCompleted
			summary.Iterations = iteration
			summary.FinalResponse = result.Response
			summary.CompletedAtUTC = time.Now().UTC().Format(time.RFC3339)
			e.emit(runTerminalEvent(eventRunCompleted, e.runID, threadID, iteration, statusCompleted, "", result.Response))
			return summary, nil
		case directiveRetry:
			if retries >= e.maxRetry {
				summary.Status = statusExhausted
				summary.Iterations = iteration
				summary.FinalResponse = result.Response
				summary.ExhaustedBy = "max_retry"
				summary.CompletedAtUTC = time.Now().UTC().Format(time.RFC3339)
				e.emit(runTerminalEvent(eventRunExhausted, e.runID, threadID, iteration, statusExhausted, "max_retry", result.Response))
				return summary, nil
			}
			retries++
		case directiveCompact:
			compactor, ok := e.runner.(threadCompactor)
			if !ok {
				err := fmt.Errorf("compact directive not supported by runner")
				summary.Status = statusFailed
				summary.Iterations = iteration
				summary.LastError = err.Error()
				summary.CompletedAtUTC = time.Now().UTC().Format(time.RFC3339)
				e.emit(runFailureEvent(eventRunFailed, e.runID, threadID, iteration, err))
				return summary, err
			}
			if err := compactor.Compact(ctx, threadID); err != nil {
				summary.Status = statusFailed
				summary.Iterations = iteration
				summary.LastError = err.Error()
				summary.CompletedAtUTC = time.Now().UTC().Format(time.RFC3339)
				e.emit(runFailureEvent(eventRunFailed, e.runID, threadID, iteration, err))
				return summary, err
			}
		case directiveAdvance:
			e.phase = "advanced"
			changed := newEvent(eventPhaseChanged)
			changed.RunID = e.runID
			changed.ThreadID = threadID
			changed.Phase = e.phase
			changed.Iteration = iteration
			changed.Directive = directiveAdvance
			e.emit(changed)
		case directiveWait:
			waiting := newEvent(eventWaitEntered)
			waiting.RunID = e.runID
			waiting.ThreadID = threadID
			waiting.Phase = e.phase
			waiting.Iteration = iteration
			waiting.Status = statusFailed
			waiting.Directive = directiveWait
			waiting.Reason = "wait_not_supported"
			e.emit(waiting)
			err := fmt.Errorf("wait directive is out of scope for the WTL CLI")
			summary.Status = statusFailed
			summary.Iterations = iteration
			summary.LastError = err.Error()
			summary.CompletedAtUTC = time.Now().UTC().Format(time.RFC3339)
			e.emit(runFailureEvent(eventRunFailed, e.runID, threadID, iteration, err))
			return summary, err
		default:
			err := fmt.Errorf("unsupported directive %q", next)
			summary.Status = statusFailed
			summary.Iterations = iteration
			summary.LastError = err.Error()
			summary.CompletedAtUTC = time.Now().UTC().Format(time.RFC3339)
			e.emit(runFailureEvent(eventRunFailed, e.runID, threadID, iteration, err))
			return summary, err
		}
	}

	summary.Status = statusExhausted
	summary.Iterations = e.maxIter
	summary.ExhaustedBy = "max_iter"
	summary.CompletedAtUTC = time.Now().UTC().Format(time.RFC3339)
	e.emit(runTerminalEvent(eventRunExhausted, e.runID, summary.ThreadID, e.maxIter, statusExhausted, "max_iter", summary.FinalResponse))
	return summary, nil
}

func (e *engine) emit(event runEvent) {
	if event.RunID == "" {
		event.RunID = e.runID
	}
	if event.Phase == "" {
		event.Phase = e.phase
	}
	for _, observer := range e.observers {
		observer.Observe(event)
	}
}

func turnFinishedEvent(runID string, threadID string, phase string, iteration int, status runStatus, next directive, response string, err error) runEvent {
	event := newEvent(eventTurnFinished)
	event.RunID = runID
	event.ThreadID = threadID
	event.Phase = phase
	event.Iteration = iteration
	event.Status = status
	event.Directive = next
	event.Response = response
	if err != nil {
		event.Error = err.Error()
	}
	return event
}

func runTerminalEvent(kind eventType, runID string, threadID string, iteration int, status runStatus, reason string, response string) runEvent {
	event := newEvent(kind)
	event.RunID = runID
	event.ThreadID = threadID
	event.Iteration = iteration
	event.Status = status
	event.Reason = reason
	event.Response = response
	return event
}

func runFailureEvent(kind eventType, runID string, threadID string, iteration int, err error) runEvent {
	event := newEvent(kind)
	event.RunID = runID
	event.ThreadID = threadID
	event.Iteration = iteration
	event.Status = statusFailed
	if err != nil {
		event.Error = err.Error()
	}
	return event
}

func runInterruptedEvent(runID string, threadID string, iteration int) runEvent {
	event := newEvent(eventRunInterrupted)
	event.RunID = runID
	event.ThreadID = threadID
	event.Iteration = iteration
	event.Status = statusInterrupted
	return event
}

func statusFromOutcome(outcome turnOutcome) runStatus {
	switch {
	case outcome.Err != nil || outcome.Status == "failed":
		return statusFailed
	case outcome.Status == "interrupted":
		return statusInterrupted
	default:
		return statusCompleted
	}
}
