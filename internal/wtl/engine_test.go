package wtl

import (
	"context"
	"errors"
	"testing"
)

type fakeRunner struct {
	threadID string
	turns    []fakeTurn
	index    int
}

type fakeTurn struct {
	result turnResult
	err    error
	deltas []string
}

func (f *fakeRunner) Start(context.Context, runConfig) (string, error) {
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

type collectingObserver struct {
	events []runEvent
}

func (o *collectingObserver) Observe(event runEvent) {
	o.events = append(o.events, event)
}

func TestEngineCompletesWhenMarkerAppears(t *testing.T) {
	t.Parallel()

	collector := &collectingObserver{}
	eng := &engine{
		runner: &fakeRunner{
			turns: []fakeTurn{
				{result: turnResult{Status: "completed", Response: "still working"}},
				{result: turnResult{Status: "completed", Response: "done " + completionMarker}},
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
				{result: turnResult{Status: "failed", Response: "boom"}, err: errors.New("boom")},
				{result: turnResult{Status: "failed", Response: "boom"}, err: errors.New("boom")},
				{result: turnResult{Status: "failed", Response: "boom"}, err: errors.New("boom")},
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
