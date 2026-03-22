package gitimpact

import (
	"errors"
	"testing"

	"github.com/charmbracelet/bubbles/spinner"
)

func TestAnalysisModelUpdateTurnStarted(t *testing.T) {
	t.Parallel()

	model := NewAnalysisModel(DefaultAnalysisPhases())
	modelPtr := &model

	updated, cmd := modelPtr.Update(TurnStartedMsg{Phase: PhaseCollect, Iteration: 2})
	if cmd != nil {
		t.Fatalf("expected nil cmd for TurnStartedMsg, got %#v", cmd)
	}
	got, ok := updated.(*AnalysisModel)
	if !ok {
		t.Fatalf("expected *AnalysisModel, got %T", updated)
	}

	if got.currentPhase != PhaseCollect {
		t.Fatalf("current phase = %q, want %q", got.currentPhase, PhaseCollect)
	}
	if got.iteration != 2 {
		t.Fatalf("iteration = %d, want 2", got.iteration)
	}
	if got.isWaiting {
		t.Fatal("isWaiting = true, want false")
	}

	assertPhaseStatus(t, got, PhaseCollect, phaseStatusRunning)
	assertPhaseStatus(t, got, PhaseScore, phaseStatusWaiting)
}

func TestAnalysisModelUpdatePhaseAdvanced(t *testing.T) {
	t.Parallel()

	model := NewAnalysisModel(DefaultAnalysisPhases())
	modelPtr := &model
	modelPtr.setRunning(PhaseCollect)

	updated, cmd := modelPtr.Update(PhaseAdvancedMsg{From: PhaseCollect, To: PhaseLink})
	if cmd != nil {
		t.Fatalf("expected nil cmd for PhaseAdvancedMsg, got %#v", cmd)
	}
	got := updated.(*AnalysisModel)

	assertPhaseStatus(t, got, PhaseCollect, phaseStatusDone)
	assertPhaseStatus(t, got, PhaseLink, phaseStatusRunning)
}

func TestAnalysisModelUpdateWaitState(t *testing.T) {
	t.Parallel()

	model := NewAnalysisModel(DefaultAnalysisPhases())
	modelPtr := &model

	updated, _ := modelPtr.Update(WaitEnteredMsg{Message: "Need confirmation"})
	got := updated.(*AnalysisModel)
	if !got.isWaiting {
		t.Fatal("isWaiting = false, want true")
	}
	if got.waitMessage != "Need confirmation" {
		t.Fatalf("waitMessage = %q, want %q", got.waitMessage, "Need confirmation")
	}

	updated, _ = got.Update(WaitResolvedMsg{Response: "y"})
	got = updated.(*AnalysisModel)
	if got.isWaiting {
		t.Fatal("isWaiting = true, want false")
	}
	if got.waitMessage != "" {
		t.Fatalf("waitMessage = %q, want empty", got.waitMessage)
	}
}

func TestAnalysisModelUpdateRunCompleted(t *testing.T) {
	t.Parallel()

	model := NewAnalysisModel(DefaultAnalysisPhases())
	modelPtr := &model
	result := &AnalysisResult{Output: "done"}

	updated, cmd := modelPtr.Update(RunCompletedMsg{Result: result})
	if cmd == nil {
		t.Fatal("expected tea quit cmd for RunCompletedMsg, got nil")
	}
	got := updated.(*AnalysisModel)

	if !got.done {
		t.Fatal("done = false, want true")
	}
	if got.result != result {
		t.Fatalf("result pointer mismatch: got %p want %p", got.result, result)
	}
	for _, phase := range got.phases {
		if phase.Status != phaseStatusDone {
			t.Fatalf("phase %q status = %q, want %q", phase.Phase, phase.Status, phaseStatusDone)
		}
	}
}

func TestAnalysisModelUpdateRunExhausted(t *testing.T) {
	t.Parallel()

	model := NewAnalysisModel(DefaultAnalysisPhases())
	modelPtr := &model
	expectedErr := errors.New("run exhausted")

	updated, cmd := modelPtr.Update(RunExhaustedMsg{Err: expectedErr})
	if cmd == nil {
		t.Fatal("expected tea quit cmd for RunExhaustedMsg, got nil")
	}
	got := updated.(*AnalysisModel)
	if !errors.Is(got.err, expectedErr) {
		t.Fatalf("err = %v, want %v", got.err, expectedErr)
	}
}

func TestAnalysisModelUpdateSpinnerTick(t *testing.T) {
	t.Parallel()

	model := NewAnalysisModel(DefaultAnalysisPhases())
	modelPtr := &model

	updated, cmd := modelPtr.Update(spinner.TickMsg{})
	if cmd == nil {
		t.Fatal("expected non-nil cmd for spinner.TickMsg, got nil")
	}
	if _, ok := updated.(*AnalysisModel); !ok {
		t.Fatalf("expected *AnalysisModel, got %T", updated)
	}
}

func assertPhaseStatus(t *testing.T, model *AnalysisModel, phase Phase, wantStatus string) {
	t.Helper()

	for _, item := range model.phases {
		if item.Phase != phase {
			continue
		}
		if item.Status != wantStatus {
			t.Fatalf("phase %q status = %q, want %q", phase, item.Status, wantStatus)
		}
		return
	}
	t.Fatalf("phase %q not found", phase)
}
