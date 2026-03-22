package gitimpact

import (
	"context"
	"testing"
)

func TestDefaultHandlers_RegistersAllPhases(t *testing.T) {
	t.Parallel()

	handlers := DefaultHandlers(&VelenClient{})

	required := []Phase{
		PhaseSourceCheck,
		PhaseCollect,
		PhaseLink,
		PhaseScore,
		PhaseReport,
	}
	for _, phase := range required {
		if handlers[phase] == nil {
			t.Fatalf("expected handler for phase %q", phase)
		}
	}
}

func TestDefaultHandlers_AdvanceStubsReturnAdvanceDirective(t *testing.T) {
	t.Parallel()

	handlers := DefaultHandlers(&VelenClient{})
	for _, phase := range []Phase{PhaseLink, PhaseScore, PhaseReport} {
		result, err := handlers[phase].Handle(context.Background(), &RunContext{})
		if err != nil {
			t.Fatalf("phase %q returned error: %v", phase, err)
		}
		if result == nil || result.Directive != DirectiveAdvancePhase {
			t.Fatalf("phase %q expected advance directive, got %+v", phase, result)
		}
	}
}

