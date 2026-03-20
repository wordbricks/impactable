package wtl

import "testing"

func TestParseCLI_RunFlags(t *testing.T) {
	t.Parallel()

	req, err := parseCLI([]string{"run", "--max-iter", "4", "--max-retry", "2", "--output", "ndjson", "--model", "gpt-test"})
	if err != nil {
		t.Fatalf("parseCLI returned error: %v", err)
	}
	if req.MaxIter != 4 {
		t.Fatalf("expected max-iter 4, got %d", req.MaxIter)
	}
	if req.MaxRetry != 2 {
		t.Fatalf("expected max-retry 2, got %d", req.MaxRetry)
	}
	if req.Output != "ndjson" {
		t.Fatalf("expected ndjson output, got %q", req.Output)
	}
	if req.Model != "gpt-test" {
		t.Fatalf("expected model override, got %q", req.Model)
	}
}

func TestParseCLI_RejectsUnknownCommand(t *testing.T) {
	t.Parallel()

	if _, err := parseCLI([]string{"ship"}); err == nil {
		t.Fatalf("expected unknown command error")
	}
}

func TestParseCLI_RejectsUnknownOption(t *testing.T) {
	t.Parallel()

	if _, err := parseCLI([]string{"run", "--bogus"}); err == nil {
		t.Fatalf("expected unknown option error")
	}
}

func TestParseCLI_RejectsInvalidOutput(t *testing.T) {
	t.Parallel()

	if _, err := parseCLI([]string{"run", "--output", "yaml"}); err == nil {
		t.Fatalf("expected invalid output error")
	}
}
