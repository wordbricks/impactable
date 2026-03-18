package ralphloop

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestRun_ParseErrorStructuredInJSONMode(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"ls", "--max-iterations", "2", "--output", "json"}, t.TempDir(), strings.NewReader(""), &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr in json mode, got %q", stderr.String())
	}

	response := map[string]any{}
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse json response: %v (%q)", err, stdout.String())
	}
	if response["command"] != "ls" {
		t.Fatalf("expected command ls, got %#v", response["command"])
	}
	if response["status"] != "failed" {
		t.Fatalf("expected failed status, got %#v", response["status"])
	}
}

func TestRun_ParseErrorStructuredInNDJSONMode(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"ls", "--max-iterations", "2", "--output", "ndjson"}, t.TempDir(), strings.NewReader(""), &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr in ndjson mode, got %q", stderr.String())
	}

	line := strings.TrimSpace(stdout.String())
	if line == "" {
		t.Fatalf("expected ndjson line output")
	}
	response := map[string]any{}
	if err := json.Unmarshal([]byte(line), &response); err != nil {
		t.Fatalf("failed to parse ndjson line: %v (%q)", err, line)
	}
	if response["command"] != "ls" {
		t.Fatalf("expected command ls, got %#v", response["command"])
	}
	if response["status"] != "failed" {
		t.Fatalf("expected failed status, got %#v", response["status"])
	}
}

func TestRun_RuntimeErrorStructuredInJSONMode(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"ls", "--output", "json"}, t.TempDir(), strings.NewReader(""), &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr in json mode, got %q", stderr.String())
	}

	response := map[string]any{}
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse json response: %v (%q)", err, stdout.String())
	}
	if response["command"] != "ls" {
		t.Fatalf("expected command ls, got %#v", response["command"])
	}
	if response["status"] != "failed" {
		t.Fatalf("expected failed status, got %#v", response["status"])
	}
}
