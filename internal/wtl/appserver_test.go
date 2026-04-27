package wtl

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

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

func TestSummarizeNotificationCapturesUsefulDebugDetails(t *testing.T) {
	t.Parallel()

	summary := summarizeNotification(notification{
		Method: "item/completed",
		Params: map[string]any{
			"item": map[string]any{
				"type":    "toolCall",
				"command": "onequery api --source wordbricks-github wordbricks/wordbricks/pulls",
				"status":  "completed",
			},
		},
	})

	for _, expected := range []string{"type=toolCall", "status=completed", "onequery api"} {
		if !strings.Contains(summary, expected) {
			t.Fatalf("summary missing %q: %q", expected, summary)
		}
	}
}

func TestReadLoopHandlesLargeJSONRPCNotification(t *testing.T) {
	t.Parallel()

	largeText := strings.Repeat("x", 2*1024*1024)
	line, err := json.Marshal(rpcEnvelope{
		Method: "item/commandExecution/outputDelta",
		Params: mustRawMessage(t, map[string]any{
			"delta": largeText,
		}),
	})
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}

	runner := &appServerRunner{
		notifications: make(chan notification, 1),
		readErr:       make(chan error, 1),
		pending:       map[int64]chan rpcEnvelope{},
	}
	go runner.readLoop(strings.NewReader(string(line)+"\n"), false)

	select {
	case note := <-runner.notifications:
		if note.Method != "item/commandExecution/outputDelta" {
			t.Fatalf("unexpected method: %q", note.Method)
		}
		if got, _ := note.Params["delta"].(string); got != largeText {
			t.Fatalf("large delta was not preserved: got %d bytes", len(got))
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for large notification")
	}

	select {
	case err := <-runner.readErr:
		if err != nil {
			t.Fatalf("readLoop returned error for large notification: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for readLoop completion")
	}
}

func mustRawMessage(t *testing.T, value any) json.RawMessage {
	t.Helper()

	body, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal raw message: %v", err)
	}
	return body
}
