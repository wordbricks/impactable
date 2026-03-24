package wtl

import (
	"context"
	"errors"
	"testing"
)

type fakeRunner struct {
	threadID    string
	startIDs    []string
	startIndex  int
	turns       []fakeTurn
	index       int
	compactRuns int
}

type fakeTurn struct {
	result turnResult
	err    error
	deltas []string
}

func (f *fakeRunner) Start(context.Context, runConfig) (string, error) {
	if len(f.startIDs) > 0 {
		id := f.startIDs[f.startIndex%len(f.startIDs)]
		f.startIndex++
		return id, nil
	}
	if f.threadID == "" {
		f.threadID = "thr_test"
	}
	return f.threadID, nil
}

func (f *fakeRunner) RunTurn(_ context.Context, _ string, _ string, onDelta func(string)) (turnResult, error) {
	if f.index >= len(f.turns) {
		return turnResult{Status: "completed"}, nil
	}
	current := f.turns[f.index]
	f.index++
	for _, delta := range current.deltas {
		if onDelta != nil {
			onDelta(delta)
		}
	}
	return current.result, current.err
}

func (f *fakeRunner) Close() error {
	return nil
}

func (f *fakeRunner) Compact(context.Context, string) error {
	f.compactRuns++
	return nil
}

type collectingObserver struct {
	events []runEvent
}

func (o *collectingObserver) Observe(event runEvent) {
	o.events = append(o.events, event)
}

type waitPolicy struct{}

func (waitPolicy) Initialize(prompt string) policyState {
	return policyState{
		Runnable:  true,
		Directive: directiveContinue,
		Plan: executionPlan{
			Phase:      "active",
			Prompt:     prompt,
			ThreadMode: threadModeReuse,
		},
	}
}

func (waitPolicy) AfterTurn(state policyState, outcome turnOutcome) policyDecision {
	if outcome.Response == "need approval" {
		next := state
		next.Runnable = false
		next.Waiting = true
		next.Directive = directiveWait
		next.WaitReason = "approval_required"
		return policyDecision{Directive: directiveWait, State: next}
	}
	next := state
	next.Terminal = true
	next.Runnable = false
	next.Waiting = false
	next.Directive = directiveComplete
	return policyDecision{Directive: directiveComplete, State: next}
}

func (waitPolicy) OnWaitResolved(state policyState, input string) policyDecision {
	next := state
	next.Waiting = false
	next.WaitReason = ""
	next.Runnable = true
	next.Directive = directiveContinue
	next.Plan.Prompt = "approval: " + input
	return policyDecision{Directive: directiveContinue, State: next}
}

type phasePolicy struct{}

func (phasePolicy) Initialize(prompt string) policyState {
	return policyState{
		Runnable:  true,
		Directive: directiveContinue,
		Plan: executionPlan{
			Phase:      "planning",
			Prompt:     prompt,
			ThreadMode: threadModeNew,
		},
	}
}

func (phasePolicy) AfterTurn(state policyState, outcome turnOutcome) policyDecision {
	next := state
	if state.Plan.Phase == "planning" {
		next.Plan = executionPlan{
			Phase:      "review",
			Prompt:     "review",
			ThreadMode: threadModeNew,
		}
		next.Directive = directiveAdvance
		return policyDecision{Directive: directiveAdvance, State: next}
	}
	next.Terminal = true
	next.Runnable = false
	next.Directive = directiveComplete
	next.Plan.Prompt = outcome.Response
	return policyDecision{Directive: directiveComplete, State: next}
}

func (phasePolicy) OnWaitResolved(state policyState, _ string) policyDecision {
	return policyDecision{Directive: directiveContinue, State: state}
}

func TestEngineCompletesWhenMarkerAppears(t *testing.T) {
	t.Parallel()

	collector := &collectingObserver{}
	eng := &engine{
		runner: &fakeRunner{
			turns: []fakeTurn{
				{result: turnResult{TurnID: "turn-1", Status: "completed", Response: "still working"}},
				{result: turnResult{TurnID: "turn-2", Status: "completed", Response: "done " + completionMarker}},
			},
		},
		policy:    simpleLoopPolicy{},
		observers: []observer{collector},
		maxIter:   5,
		maxRetry:  3,
		runID:     "run-1",
	}

	summary, err := eng.run(context.Background(), runConfig{}, "ship it")
	if err != nil {
		t.Fatalf("engine returned error: %v", err)
	}
	if summary.Status != statusCompleted {
		t.Fatalf("expected completed status, got %q", summary.Status)
	}
	if summary.Iterations != 2 {
		t.Fatalf("expected 2 iterations, got %d", summary.Iterations)
	}
	if collector.events[0].Event != eventRunStarted {
		t.Fatalf("expected first event run.started, got %q", collector.events[0].Event)
	}
	if collector.events[len(collector.events)-1].Event != eventRunCompleted {
		t.Fatalf("expected terminal run.completed event, got %q", collector.events[len(collector.events)-1].Event)
	}
}

func TestEngineExhaustsOnRetryLimit(t *testing.T) {
	t.Parallel()

	eng := &engine{
		runner: &fakeRunner{
			turns: []fakeTurn{
				{result: turnResult{TurnID: "turn-1", Status: "failed", Response: "boom"}, err: errors.New("boom")},
				{result: turnResult{TurnID: "turn-2", Status: "failed", Response: "boom"}, err: errors.New("boom")},
				{result: turnResult{TurnID: "turn-3", Status: "failed", Response: "boom"}, err: errors.New("boom")},
			},
		},
		policy:   simpleLoopPolicy{},
		maxIter:  5,
		maxRetry: 2,
		runID:    "run-2",
	}

	summary, err := eng.run(context.Background(), runConfig{}, "fix it")
	if err != nil {
		t.Fatalf("engine returned error: %v", err)
	}
	if summary.Status != statusExhausted {
		t.Fatalf("expected exhausted status, got %q", summary.Status)
	}
	if summary.ExhaustedBy != "max_retry" {
		t.Fatalf("expected max_retry exhaustion, got %q", summary.ExhaustedBy)
	}
}

func TestEngineResolvesWaitAndResumes(t *testing.T) {
	t.Parallel()

	collector := &collectingObserver{}
	eng := &engine{
		runner: &fakeRunner{
			turns: []fakeTurn{
				{result: turnResult{TurnID: "turn-1", Status: "completed", Response: "need approval"}},
				{result: turnResult{TurnID: "turn-2", Status: "completed", Response: "done " + completionMarker}},
			},
		},
		policy:    waitPolicy{},
		observers: []observer{collector},
		maxIter:   5,
		maxRetry:  2,
		runID:     "run-wait",
		waitHandler: func(context.Context, policyState) (string, error) {
			return "approved", nil
		},
	}

	summary, err := eng.run(context.Background(), runConfig{}, "ship it")
	if err != nil {
		t.Fatalf("engine returned error: %v", err)
	}
	if summary.Status != statusCompleted {
		t.Fatalf("expected completed status, got %q", summary.Status)
	}

	sawWaitEntered := false
	sawWaitResolved := false
	for _, event := range collector.events {
		if event.Event == eventWaitEntered && event.Reason == "approval_required" {
			sawWaitEntered = true
		}
		if event.Event == eventWaitResolved && event.Text == "approved" {
			sawWaitResolved = true
		}
	}
	if !sawWaitEntered || !sawWaitResolved {
		t.Fatalf("expected wait enter and resolve events, got %#v", collector.events)
	}
}

func TestEngineStartsNewThreadWhenPhaseRequestsIt(t *testing.T) {
	t.Parallel()

	collector := &collectingObserver{}
	eng := &engine{
		runner: &fakeRunner{
			startIDs: []string{"thr-plan", "thr-review"},
			turns: []fakeTurn{
				{result: turnResult{TurnID: "turn-1", Status: "completed", Response: "planned"}},
				{result: turnResult{TurnID: "turn-2", Status: "completed", Response: "reviewed"}},
			},
		},
		policy:    phasePolicy{},
		observers: []observer{collector},
		maxIter:   5,
		maxRetry:  2,
		runID:     "run-phase",
	}

	summary, err := eng.run(context.Background(), runConfig{}, "ship it")
	if err != nil {
		t.Fatalf("engine returned error: %v", err)
	}
	if summary.Status != statusCompleted {
		t.Fatalf("expected completed status, got %q", summary.Status)
	}

	threadStarted := 0
	sawPhaseChange := false
	for _, event := range collector.events {
		if event.Event == eventThreadStarted {
			threadStarted++
		}
		if event.Event == eventPhaseChanged && event.Phase == "review" {
			sawPhaseChange = true
		}
	}
	if threadStarted != 2 {
		t.Fatalf("expected 2 thread.started events, got %d", threadStarted)
	}
	if !sawPhaseChange {
		t.Fatalf("expected phase.changed event, got %#v", collector.events)
	}
}

func TestEngineObserverContract(t *testing.T) {
	t.Parallel()

	collector := &collectingObserver{}
	eng := &engine{
		runner: &fakeRunner{
			turns: []fakeTurn{
				{result: turnResult{TurnID: "turn-1", Status: "completed", Response: "keep going"}},
				{result: turnResult{TurnID: "turn-2", Status: "completed", Response: "done " + completionMarker}},
			},
		},
		policy:    simpleLoopPolicy{},
		observers: []observer{collector},
		maxIter:   5,
		maxRetry:  2,
		runID:     "run-contract",
	}

	if _, err := eng.run(context.Background(), runConfig{}, "ship it"); err != nil {
		t.Fatalf("engine returned error: %v", err)
	}

	if len(collector.events) == 0 || collector.events[0].Event != eventRunStarted {
		t.Fatalf("expected run.started to be first event, got %#v", collector.events)
	}

	started := 0
	finished := 0
	terminalSeen := false
	completedSeen := false
	exhaustedSeen := false
	for _, event := range collector.events {
		if terminalSeen {
			switch event.Event {
			case eventRunCompleted, eventRunExhausted, eventRunInterrupted, eventRunFailed:
			default:
				t.Fatalf("saw non-terminal event %q after terminal event", event.Event)
			}
		}
		switch event.Event {
		case eventTurnStarted:
			started++
		case eventTurnFinished:
			finished++
		case eventRunCompleted:
			terminalSeen = true
			completedSeen = true
		case eventRunExhausted:
			terminalSeen = true
			exhaustedSeen = true
		case eventRunInterrupted, eventRunFailed:
			terminalSeen = true
		}
	}
	if finished > started {
		t.Fatalf("turn.finished count exceeded turn.started count: %d > %d", finished, started)
	}
	if completedSeen && exhaustedSeen {
		t.Fatalf("completed and exhausted must be mutually exclusive")
	}
}
