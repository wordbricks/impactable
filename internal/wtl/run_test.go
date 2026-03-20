package wtl

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestRun_JSONComplete(t *testing.T) {
	restore := newRunner
	newRunner = func(cfg runConfig) (turnRunner, error) {
		return &fakeRunner{
			turns: []fakeTurn{
				{result: turnResult{Status: "completed", Response: "done " + completionMarker}},
			},
		}, nil
	}
	defer func() {
		newRunner = restore
	}()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"run"}, t.TempDir(), strings.NewReader("ship it"), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}

	response := map[string]any{}
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse json response: %v (%q)", err, stdout.String())
	}
	if response["command"] != "run" {
		t.Fatalf("expected command run, got %#v", response["command"])
	}
	if response["status"] != string(statusCompleted) {
		t.Fatalf("expected completed status, got %#v", response["status"])
	}
}

func TestRun_TextOutput(t *testing.T) {
	restore := newRunner
	newRunner = func(cfg runConfig) (turnRunner, error) {
		return &fakeRunner{
			turns: []fakeTurn{
				{
					result: turnResult{Status: "completed", Response: "hello\n" + completionMarker},
					deltas: []string{"hello\n", completionMarker + "\n"},
				},
			},
		}, nil
	}
	defer func() {
		newRunner = restore
	}()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"run", "--output", "text"}, t.TempDir(), strings.NewReader("ship it"), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	text := stdout.String()
	if !strings.Contains(text, "[turn 1] running...") {
		t.Fatalf("expected turn start text, got %q", text)
	}
	if !strings.Contains(text, "Done: your request was completed successfully.") {
		t.Fatalf("expected completion message, got %q", text)
	}
}

func TestRun_ParseErrorStructuredInNDJSONMode(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"run", "--output", "ndjson", "--max-iter", "nope"}, t.TempDir(), strings.NewReader(""), &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr in ndjson mode, got %q", stderr.String())
	}
	line := strings.TrimSpace(stdout.String())
	response := map[string]any{}
	if err := json.Unmarshal([]byte(line), &response); err != nil {
		t.Fatalf("failed to parse ndjson response: %v (%q)", err, line)
	}
	if response["status"] != string(statusFailed) {
		t.Fatalf("expected failed status, got %#v", response["status"])
	}
}
