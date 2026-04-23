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

func TestThreadStartParamsAllowProductSpecificServiceName(t *testing.T) {
	t.Parallel()

	params := threadStartParams(runConfig{
		CWD:            "/tmp/project",
		Model:          "gpt-test",
		ServiceName:    "git-impact",
		ApprovalPolicy: "on-request",
		Sandbox:        "read-only",
	})

	if got := params["serviceName"]; got != "git-impact" {
		t.Fatalf("expected git-impact serviceName, got %#v", got)
	}
	if got := params["approvalPolicy"]; got != "on-request" {
		t.Fatalf("expected approvalPolicy override, got %#v", got)
	}
	if got := params["sandbox"]; got != "read-only" {
		t.Fatalf("expected sandbox override, got %#v", got)
	}
}

func TestTurnStartParamsCanEnableNetworkAccess(t *testing.T) {
	t.Parallel()

	params := turnStartParams("thr-test", "run onequery", runConfig{
		Sandbox:       "workspace-write",
		NetworkAccess: true,
	})

	policy, ok := params["sandboxPolicy"].(map[string]any)
	if !ok {
		t.Fatalf("expected sandboxPolicy map, got %#v", params["sandboxPolicy"])
	}
	if got := policy["type"]; got != "workspaceWrite" {
		t.Fatalf("expected workspaceWrite sandbox policy, got %#v", got)
	}
	if got := policy["networkAccess"]; got != true {
		t.Fatalf("expected networkAccess true, got %#v", got)
	}
}

func TestClientInfoParamsAllowProductSpecificIdentity(t *testing.T) {
	t.Parallel()

	params := clientInfoParams(runConfig{
		ClientName:    "git-impact",
		ClientTitle:   "Git Impact Analyzer",
		ClientVersion: "0.2.0",
	})

	if got := params["name"]; got != "git-impact" {
		t.Fatalf("expected git-impact client name, got %#v", got)
	}
	if got := params["title"]; got != "Git Impact Analyzer" {
		t.Fatalf("expected client title override, got %#v", got)
	}
	if got := params["version"]; got != "0.2.0" {
		t.Fatalf("expected client version override, got %#v", got)
	}
}
