package gitimpact

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestRunAnalyzeJSONEnvelope(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"analyze", "--pr", "142", "--output", "json"}, t.TempDir(), strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q", code, stderr.String())
	}

	response := map[string]any{}
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode output: %v (%q)", err, stdout.String())
	}
	if response["command"] != commandAnalyze {
		t.Fatalf("unexpected command: %#v", response["command"])
	}
	if response["status"] != "ok" {
		t.Fatalf("unexpected status: %#v", response["status"])
	}
	result, _ := response["result"].(map[string]any)
	if result["analysis_path"] != "single_pr" {
		t.Fatalf("unexpected analysis_path: %#v", result["analysis_path"])
	}
}

func TestRunCheckSourcesJSONEnvelope(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"check-sources", "--require", "github,warehouse", "--output", "json"}, t.TempDir(), strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q", code, stderr.String())
	}

	response := map[string]any{}
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode output: %v (%q)", err, stdout.String())
	}
	if response["command"] != commandCheckSources {
		t.Fatalf("unexpected command: %#v", response["command"])
	}
	sources, _ := response["sources"].([]any)
	if len(sources) != 2 {
		t.Fatalf("expected two source contracts, got %d", len(sources))
	}
}

func TestRunReportScaffoldJSONEnvelope(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"report-scaffold", "--mode", "markdown", "--mode", "html", "--output", "json"}, t.TempDir(), strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q", code, stderr.String())
	}

	response := map[string]any{}
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode output: %v (%q)", err, stdout.String())
	}
	if response["command"] != commandReportScaffold {
		t.Fatalf("unexpected command: %#v", response["command"])
	}
	reports, _ := response["reports"].([]any)
	if len(reports) != 2 {
		t.Fatalf("expected 2 report entries, got %d", len(reports))
	}
	first, _ := reports[0].(map[string]any)
	if first["mode"] != "markdown" {
		t.Fatalf("expected first mode markdown, got %#v", first["mode"])
	}
}

func TestRunSchemaJSONEnvelope(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	if err := runSchema(t.TempDir(), schemaRequest{Output: "json", TargetCommand: commandAnalyze}, &stdout); err != nil {
		t.Fatalf("runSchema returned error: %v", err)
	}

	response := map[string]any{}
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode schema output: %v (%q)", err, stdout.String())
	}
	if response["command"] != commandSchema {
		t.Fatalf("unexpected command: %#v", response["command"])
	}
	items, _ := response["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("expected one schema item, got %d", len(items))
	}
}
