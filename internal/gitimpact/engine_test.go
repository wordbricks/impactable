package gitimpact

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type phaseHandlerFunc func(ctx context.Context, runCtx *RunContext) (*TurnResult, error)

func (fn phaseHandlerFunc) Handle(ctx context.Context, runCtx *RunContext) (*TurnResult, error) {
	return fn(ctx, runCtx)
}

type recordingObserver struct {
	turns         []Phase
	iterations    []int
	advances      [][2]Phase
	waitMessages  []string
	waitResponses []string
	completed     *AnalysisResult
	exhaustedErr  error
}

func (o *recordingObserver) OnTurnStarted(phase Phase, iteration int) {
	o.turns = append(o.turns, phase)
	o.iterations = append(o.iterations, iteration)
}

func (o *recordingObserver) OnPhaseAdvanced(from, to Phase) {
	o.advances = append(o.advances, [2]Phase{from, to})
}

func (o *recordingObserver) OnWaitEntered(message string) {
	o.waitMessages = append(o.waitMessages, message)
}

func (o *recordingObserver) OnWaitResolved(response string) {
	o.waitResponses = append(o.waitResponses, response)
}

func (o *recordingObserver) OnRunCompleted(result *AnalysisResult) {
	o.completed = result
}

func (o *recordingObserver) OnRunExhausted(err error) {
	o.exhaustedErr = err
}

func TestEngineRun_PhaseProgression(t *testing.T) {
	t.Parallel()

	observer := &recordingObserver{}
	engine := &Engine{
		Observer: observer,
		Handlers: map[Phase]PhaseHandler{
			PhaseSourceCheck: phaseHandlerFunc(func(context.Context, *RunContext) (*TurnResult, error) {
				return &TurnResult{Directive: DirectiveAdvancePhase}, nil
			}),
			PhaseCollect: phaseHandlerFunc(func(context.Context, *RunContext) (*TurnResult, error) {
				return &TurnResult{Directive: DirectiveAdvancePhase}, nil
			}),
			PhaseLink: phaseHandlerFunc(func(context.Context, *RunContext) (*TurnResult, error) {
				return &TurnResult{Directive: DirectiveAdvancePhase}, nil
			}),
			PhaseScore: phaseHandlerFunc(func(context.Context, *RunContext) (*TurnResult, error) {
				return &TurnResult{Directive: DirectiveAdvancePhase}, nil
			}),
			PhaseReport: phaseHandlerFunc(func(context.Context, *RunContext) (*TurnResult, error) {
				return &TurnResult{Directive: DirectiveComplete, Output: "analysis complete"}, nil
			}),
		},
	}

	runCtx := &RunContext{}
	result, err := engine.Run(context.Background(), runCtx)
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Output != "analysis complete" {
		t.Fatalf("expected complete output, got %q", result.Output)
	}
	if result.Phase != PhaseReport {
		t.Fatalf("expected final phase %q, got %q", PhaseReport, result.Phase)
	}
	if result.Iteration != 5 {
		t.Fatalf("expected 5 iterations, got %d", result.Iteration)
	}
	if len(observer.turns) != 5 {
		t.Fatalf("expected 5 turn-start events, got %d", len(observer.turns))
	}
	if len(observer.advances) != 4 {
		t.Fatalf("expected 4 phase-advanced events, got %d", len(observer.advances))
	}
	if observer.exhaustedErr != nil {
		t.Fatalf("did not expect exhausted event, got %v", observer.exhaustedErr)
	}
	if observer.completed == nil {
		t.Fatal("expected run-completed event")
	}
}

func TestEngineRun_RetryExhaustion(t *testing.T) {
	t.Parallel()

	observer := &recordingObserver{}
	collectCalls := 0
	engine := &Engine{
		Observer:   observer,
		MaxRetries: 3,
		Handlers: map[Phase]PhaseHandler{
			PhaseSourceCheck: phaseHandlerFunc(func(context.Context, *RunContext) (*TurnResult, error) {
				return &TurnResult{Directive: DirectiveAdvancePhase}, nil
			}),
			PhaseCollect: phaseHandlerFunc(func(context.Context, *RunContext) (*TurnResult, error) {
				collectCalls++
				return &TurnResult{Directive: DirectiveRetry, Error: errors.New("transient failure")}, nil
			}),
		},
	}

	_, err := engine.Run(context.Background(), &RunContext{})
	if err == nil {
		t.Fatal("expected retry exhaustion error")
	}
	if !strings.Contains(err.Error(), "exceeded max retries") {
		t.Fatalf("expected max retries error, got %v", err)
	}
	if collectCalls != 4 {
		t.Fatalf("expected 4 collect attempts (3 retries + 1 exhausted attempt), got %d", collectCalls)
	}
	if observer.exhaustedErr == nil {
		t.Fatal("expected exhausted observer event")
	}
}

func TestEngineRun_WaitHandling(t *testing.T) {
	t.Parallel()

	observer := &recordingObserver{}
	sourceCheckCalls := 0
	waitMessages := make([]string, 0, 1)
	engine := &Engine{
		Observer: observer,
		WaitHandler: func(message string) (string, error) {
			waitMessages = append(waitMessages, message)
			return "proceed", nil
		},
		Handlers: map[Phase]PhaseHandler{
			PhaseSourceCheck: phaseHandlerFunc(func(context.Context, *RunContext) (*TurnResult, error) {
				sourceCheckCalls++
				if sourceCheckCalls == 1 {
					return &TurnResult{Directive: DirectiveWait, WaitMessage: "confirm source mapping"}, nil
				}
				return &TurnResult{Directive: DirectiveAdvancePhase}, nil
			}),
			PhaseCollect: phaseHandlerFunc(func(context.Context, *RunContext) (*TurnResult, error) {
				return &TurnResult{Directive: DirectiveAdvancePhase}, nil
			}),
			PhaseLink: phaseHandlerFunc(func(context.Context, *RunContext) (*TurnResult, error) {
				return &TurnResult{Directive: DirectiveAdvancePhase}, nil
			}),
			PhaseScore: phaseHandlerFunc(func(context.Context, *RunContext) (*TurnResult, error) {
				return &TurnResult{Directive: DirectiveAdvancePhase}, nil
			}),
			PhaseReport: phaseHandlerFunc(func(context.Context, *RunContext) (*TurnResult, error) {
				return &TurnResult{Directive: DirectiveComplete, Output: "done"}, nil
			}),
		},
	}

	runCtx := &RunContext{}
	result, err := engine.Run(context.Background(), runCtx)
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}
	if result.Output != "done" {
		t.Fatalf("expected done output, got %q", result.Output)
	}
	if len(waitMessages) != 1 || waitMessages[0] != "confirm source mapping" {
		t.Fatalf("unexpected wait handler messages: %#v", waitMessages)
	}
	if len(observer.waitMessages) != 1 || observer.waitMessages[0] != "confirm source mapping" {
		t.Fatalf("unexpected wait-entered events: %#v", observer.waitMessages)
	}
	if len(observer.waitResponses) != 1 || observer.waitResponses[0] != "proceed" {
		t.Fatalf("unexpected wait-resolved events: %#v", observer.waitResponses)
	}
	if runCtx.AnalysisCtx == nil || runCtx.AnalysisCtx.LastWaitResponse != "proceed" {
		t.Fatalf("expected wait response stored in analysis context, got %#v", runCtx.AnalysisCtx)
	}
}
