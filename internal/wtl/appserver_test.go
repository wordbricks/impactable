package wtl

import "testing"

func TestThreadStartParamsUseWireSandboxValue(t *testing.T) {
	t.Parallel()

	params := threadStartParams(runConfig{
		CWD:   "/tmp/project",
		Model: "gpt-test",
	})

	if got := params["sandbox"]; got != "workspace-write" {
		t.Fatalf("expected workspace-write sandbox, got %#v", got)
	}
	if got := params["approvalPolicy"]; got != "never" {
		t.Fatalf("expected approvalPolicy never, got %#v", got)
	}
}
