package wtl

import (
	"context"
	"errors"
	"fmt"
	"strings"
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

type waitResolver func(context.Context, policyState) (string, error)

type engine struct {
	runner      turnRunner
	policy      policy
	observers   []observer
	maxIter     int
	maxRetry    int
	runID       string
	phase       string
	waitHandler waitResolver
}

func (e *engine) run(ctx context.Context, cfg runConfig, prompt string) (runSummary, error) {
	state := e.policy.Initialize(prompt)
	e.phase = state.Plan.Phase

	summary := runSummary{
		RunID:         e.runID,
		MaxIterations: e.maxIter,
		MaxRetry:      e.maxRetry,
		LastPhase:     e.phase,
	}
	e.emit(newEvent(eventRunStarted))

	if state.Terminal {
		if state.Directive != directiveComplete {
			err := fmt.Errorf("policy initialized in terminal state without complete directive")
			summary.Status = statusFailed
			summary.LastError = err.Error()
			summary.CompletedAtUTC = time.Now().UTC().Format(time.RFC3339)
			e.emit(runFailureEvent(eventRunFailed, e.runID, "", "", 0, err))
			return summary, err
		}
		summary.Status = statusCompleted
		summary.LastDirective = directiveComplete
		summary.CompletedAtUTC = time.Now().UTC().Format(time.RFC3339)
		e.emit(runTerminalEvent(eventRunCompleted, e.runID, "", "", 0, statusCompleted, "", ""))
		return summary, nil
	}

	activeThreadID := ""
	activeThreadPhase := ""
	retries := 0
	iterations := 0

	for iterations < e.maxIter {
		if err := ctx.Err(); err != nil {
			summary.Status = statusInterrupted
			summary.Iterations = iterations
			summary.CompletedAtUTC = time.Now().UTC().Format(time.RFC3339)
			e.emit(runInterruptedEvent(e.runID, activeThreadID, "", iterations))
			return summary, err
		}

		if state.Waiting {
			nextState, err := e.resolveWait(ctx, state, activeThreadID, iterations, &summary)
			if err != nil {
				return summary, err
			}
			state = nextState
			summary.LastPhase = state.Plan.Phase
			e.transitionPhase(activeThreadID, iterations, state.Plan.Phase, state.Directive)
			continue
		}
		if !state.Runnable {
			err := fmt.Errorf("policy is not runnable and not waiting")
			summary.Status = statusFailed
			summary.Iterations = iterations
			summary.LastError = err.Error()
			summary.CompletedAtUTC = time.Now().UTC().Format(time.RFC3339)
			e.emit(runFailureEvent(eventRunFailed, e.runID, activeThreadID, "", iterations, err))
			return summary, err
		}

		iterations++
		threadID, threadPhase, err := e.ensureThread(ctx, cfg, state.Plan, activeThreadID, activeThreadPhase, iterations)
		if err != nil {
			summary.Status = statusFailed
			summary.Iterations = iterations - 1
			summary.LastError = err.Error()
			summary.CompletedAtUTC = time.Now().UTC().Format(time.RFC3339)
			e.emit(runFailureEvent(eventRunFailed, e.runID, activeThreadID, "", iterations-1, err))
			return summary, err
		}
		activeThreadID = threadID
		activeThreadPhase = threadPhase
		summary.ThreadID = threadID

		started := newEvent(eventTurnStarted)
		started.RunID = e.runID
		started.ThreadID = threadID
		started.Phase = state.Plan.Phase
		started.Iteration = iterations
		e.emit(started)

		turnPhase := state.Plan.Phase
		result, turnErr := e.runner.RunTurn(ctx, threadID, state.Plan.Prompt, func(delta string) {
			deltaEvent := newEvent(eventTurnDelta)
			deltaEvent.RunID = e.runID
			deltaEvent.ThreadID = threadID
			deltaEvent.Phase = turnPhase
			deltaEvent.Iteration = iterations
			deltaEvent.Text = delta
			e.emit(deltaEvent)
		})

		if errors.Is(turnErr, context.Canceled) || result.Status == "interrupted" {
			summary.Status = statusInterrupted
			summary.Iterations = iterations
			summary.FinalResponse = result.Response
			summary.CompletedAtUTC = time.Now().UTC().Format(time.RFC3339)
			e.emit(turnFinishedEvent(e.runID, threadID, result.TurnID, turnPhase, iterations, statusInterrupted, directiveUnsupported, result.Response, turnErr))
			e.emit(runInterruptedEvent(e.runID, threadID, result.TurnID, iterations))
			return summary, nil
		}

		outcome := turnOutcome{
			TurnID:   result.TurnID,
			Response: result.Response,
			Err:      turnErr,
			Status:   result.Status,
		}
		decision := e.policy.AfterTurn(state, outcome)
		state = decision.State
		summary.LastDirective = decision.Directive
		summary.LastPhase = state.Plan.Phase
		if turnErr != nil {
			summary.LastError = turnErr.Error()
		} else if result.ErrorMessage != "" {
			summary.LastError = result.ErrorMessage
		}
		e.emit(turnFinishedEvent(e.runID, threadID, result.TurnID, turnPhase, iterations, statusFromOutcome(outcome), decision.Directive, result.Response, turnErr))

		switch decision.Directive {
		case directiveContinue:
			retries = 0
			summary.Iterations = iterations
			summary.FinalResponse = result.Response
			e.transitionPhase(threadID, iterations, state.Plan.Phase, decision.Directive)
		case directiveComplete:
			summary.Status = statusCompleted
			summary.Iterations = iterations
			summary.FinalResponse = result.Response
			summary.CompletedAtUTC = time.Now().UTC().Format(time.RFC3339)
			e.transitionPhase(threadID, iterations, state.Plan.Phase, decision.Directive)
			e.emit(runTerminalEvent(eventRunCompleted, e.runID, threadID, result.TurnID, iterations, statusCompleted, "", result.Response))
			return summary, nil
		case directiveRetry:
			if retries >= e.maxRetry {
				summary.Status = statusExhausted
				summary.Iterations = iterations
				summary.FinalResponse = result.Response
				summary.ExhaustedBy = "max_retry"
				summary.CompletedAtUTC = time.Now().UTC().Format(time.RFC3339)
				e.emit(runTerminalEvent(eventRunExhausted, e.runID, threadID, result.TurnID, iterations, statusExhausted, "max_retry", result.Response))
				return summary, nil
			}
			retries++
		case directiveCompact:
			compactor, ok := e.runner.(threadCompactor)
			if !ok {
				err := fmt.Errorf("compact directive not supported by runner")
				summary.Status = statusFailed
				summary.Iterations = iterations
				summary.LastError = err.Error()
				summary.CompletedAtUTC = time.Now().UTC().Format(time.RFC3339)
				e.emit(runFailureEvent(eventRunFailed, e.runID, threadID, result.TurnID, iterations, err))
				return summary, err
			}
			if err := compactor.Compact(ctx, threadID); err != nil {
				summary.Status = statusFailed
				summary.Iterations = iterations
				summary.LastError = err.Error()
				summary.CompletedAtUTC = time.Now().UTC().Format(time.RFC3339)
				e.emit(runFailureEvent(eventRunFailed, e.runID, threadID, result.TurnID, iterations, err))
				return summary, err
			}
		case directiveAdvance:
			retries = 0
			summary.Iterations = iterations
			summary.FinalResponse = result.Response
			e.transitionPhase(threadID, iterations, state.Plan.Phase, decision.Directive)
		case directiveWait:
			summary.Iterations = iterations
			summary.FinalResponse = result.Response
			nextState, err := e.resolveWait(ctx, state, threadID, iterations, &summary)
			if err != nil {
				return summary, err
			}
			retries = 0
			state = nextState
			summary.LastPhase = state.Plan.Phase
			e.transitionPhase(threadID, iterations, state.Plan.Phase, state.Directive)
		default:
			err := fmt.Errorf("unsupported directive %q", decision.Directive)
			summary.Status = statusFailed
			summary.Iterations = iterations
			summary.LastError = err.Error()
			summary.CompletedAtUTC = time.Now().UTC().Format(time.RFC3339)
			e.emit(runFailureEvent(eventRunFailed, e.runID, threadID, result.TurnID, iterations, err))
			return summary, err
		}
	}

	summary.Status = statusExhausted
	summary.Iterations = e.maxIter
	summary.ExhaustedBy = "max_iter"
	summary.CompletedAtUTC = time.Now().UTC().Format(time.RFC3339)
	e.emit(runTerminalEvent(eventRunExhausted, e.runID, summary.ThreadID, "", e.maxIter, statusExhausted, "max_iter", summary.FinalResponse))
	return summary, nil
}

func (e *engine) ensureThread(ctx context.Context, cfg runConfig, plan executionPlan, activeThreadID string, activePhase string, iteration int) (string, string, error) {
	mode := plan.ThreadMode
	if mode == "" {
		mode = threadModeReuse
	}
	phase := plan.Phase
	if strings.TrimSpace(activeThreadID) == "" || (mode == threadModeNew && activePhase != phase) {
		threadID, err := e.runner.Start(ctx, cfg)
		if err != nil {
			return "", activePhase, err
		}
		started := newEvent(eventThreadStarted)
		started.RunID = e.runID
		started.ThreadID = threadID
		started.Phase = phase
		started.Iteration = iteration
		e.emit(started)
		return threadID, phase, nil
	}

	reused := newEvent(eventThreadReused)
	reused.RunID = e.runID
	reused.ThreadID = activeThreadID
	reused.Phase = phase
	reused.Iteration = iteration
	e.emit(reused)
	return activeThreadID, phase, nil
}

func (e *engine) resolveWait(ctx context.Context, state policyState, threadID string, iteration int, summary *runSummary) (policyState, error) {
	waitingState := state
	waitingState.Waiting = true
	waitingState.Runnable = false
	waitingState.Directive = directiveWait
	if strings.TrimSpace(waitingState.WaitReason) == "" {
		waitingState.WaitReason = "external_input_required"
	}

	waiting := newEvent(eventWaitEntered)
	waiting.RunID = e.runID
	waiting.ThreadID = threadID
	waiting.Phase = waitingState.Plan.Phase
	waiting.Iteration = iteration
	waiting.Directive = directiveWait
	waiting.Reason = waitingState.WaitReason
	e.emit(waiting)

	if e.waitHandler == nil {
		err := fmt.Errorf("wait directive received without a wait handler")
		summary.Status = statusFailed
		summary.Iterations = iteration
		summary.LastError = err.Error()
		summary.CompletedAtUTC = time.Now().UTC().Format(time.RFC3339)
		e.emit(runFailureEvent(eventRunFailed, e.runID, threadID, "", iteration, err))
		return state, err
	}

	input, err := e.waitHandler(ctx, waitingState)
	if err != nil {
		summary.Status = statusFailed
		summary.Iterations = iteration
		summary.LastError = err.Error()
		summary.CompletedAtUTC = time.Now().UTC().Format(time.RFC3339)
		e.emit(runFailureEvent(eventRunFailed, e.runID, threadID, "", iteration, err))
		return state, err
	}

	resolved := newEvent(eventWaitResolved)
	resolved.RunID = e.runID
	resolved.ThreadID = threadID
	resolved.Phase = waitingState.Plan.Phase
	resolved.Iteration = iteration
	resolved.Directive = directiveContinue
	resolved.Text = input
	e.emit(resolved)

	decision := e.policy.OnWaitResolved(waitingState, input)
	return decision.State, nil
}

func (e *engine) transitionPhase(threadID string, iteration int, nextPhase string, next directive) {
	if nextPhase == e.phase {
		return
	}
	e.phase = nextPhase
	changed := newEvent(eventPhaseChanged)
	changed.RunID = e.runID
	changed.ThreadID = threadID
	changed.Phase = nextPhase
	changed.Iteration = iteration
	changed.Directive = next
	e.emit(changed)
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

func turnFinishedEvent(runID string, threadID string, turnID string, phase string, iteration int, status runStatus, next directive, response string, err error) runEvent {
	event := newEvent(eventTurnFinished)
	event.RunID = runID
	event.ThreadID = threadID
	event.TurnID = turnID
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

func runTerminalEvent(kind eventType, runID string, threadID string, turnID string, iteration int, status runStatus, reason string, response string) runEvent {
	event := newEvent(kind)
	event.RunID = runID
	event.ThreadID = threadID
	event.TurnID = turnID
	event.Iteration = iteration
	event.Status = status
	event.Reason = reason
	event.Response = response
	return event
}

func runFailureEvent(kind eventType, runID string, threadID string, turnID string, iteration int, err error) runEvent {
	event := newEvent(kind)
	event.RunID = runID
	event.ThreadID = threadID
	event.TurnID = turnID
	event.Iteration = iteration
	event.Status = statusFailed
	if err != nil {
		event.Error = err.Error()
	}
	return event
}

func runInterruptedEvent(runID string, threadID string, turnID string, iteration int) runEvent {
	event := newEvent(eventRunInterrupted)
	event.RunID = runID
	event.ThreadID = threadID
	event.TurnID = turnID
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
