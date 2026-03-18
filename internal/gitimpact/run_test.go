package gitimpact

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestRun_ParseErrorStructuredInJSONMode(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	configPath := writeTestConfig(t, repoRoot, testConfigOptions{})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"check-sources", "--config", configPath, "--pr", "1", "--output", "json"}, repoRoot, strings.NewReader(""), &stdout, &stderr)
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
	if response["command"] != commandCheckSources {
		t.Fatalf("expected command check-sources, got %#v", response["command"])
	}
	if response["status"] != "failed" {
		t.Fatalf("expected failed status, got %#v", response["status"])
	}
}

func TestRun_ParseErrorStructuredInNDJSONMode(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	configPath := writeTestConfig(t, repoRoot, testConfigOptions{})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"report-scaffold", "--config", configPath, "--mode", "pdf", "--output", "ndjson"}, repoRoot, strings.NewReader(""), &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr in ndjson mode, got %q", stderr.String())
	}

	line := strings.TrimSpace(stdout.String())
	if line == "" {
		t.Fatalf("expected ndjson payload")
	}
	response := map[string]any{}
	if err := json.Unmarshal([]byte(line), &response); err != nil {
		t.Fatalf("failed to parse ndjson payload: %v (%q)", err, line)
	}
	if response["command"] != commandReportScaffold {
		t.Fatalf("expected command report-scaffold, got %#v", response["command"])
	}
	if response["status"] != "failed" {
		t.Fatalf("expected failed status, got %#v", response["status"])
	}
}

func TestRun_RuntimeErrorStructuredInJSONMode(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"analyze", "--output", "json"}, repoRoot, strings.NewReader(""), &stdout, &stderr)
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
	if response["command"] != commandAnalyze {
		t.Fatalf("expected command analyze, got %#v", response["command"])
	}
	if response["status"] != "failed" {
		t.Fatalf("expected failed status, got %#v", response["status"])
	}
}
