package gitimpact

import "testing"

func TestDefaultHandlersIncludesAllPhases(t *testing.T) {
	t.Parallel()

	handlers := DefaultHandlers()
	for _, phase := range phaseOrder {
		handler, ok := handlers[phase]
		if !ok {
			t.Fatalf("missing default handler for phase %q", phase)
		}
		if handler == nil {
			t.Fatalf("default handler for phase %q is nil", phase)
		}
	}
}

func TestNewDefaultEngineWiresDependencies(t *testing.T) {
	t.Parallel()

	waitFn := func(string) (string, error) { return "y", nil }
	engine := NewDefaultEngine(NewVelenClient(0), nil, waitFn)
	if engine == nil {
		t.Fatal("expected non-nil engine")
	}
	if engine.MaxRetries != defaultMaxRetries {
		t.Fatalf("max retries = %d, want %d", engine.MaxRetries, defaultMaxRetries)
	}
	if len(engine.Handlers) != len(phaseOrder) {
		t.Fatalf("handler count = %d, want %d", len(engine.Handlers), len(phaseOrder))
	}
	if engine.WaitHandler == nil {
		t.Fatal("wait handler is nil")
	}
}
