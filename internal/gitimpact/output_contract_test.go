package gitimpact

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestRunAnalyzeJSONEnvelope(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	configPath := writeTestConfig(t, repoRoot, testConfigOptions{})

	withVelenClientFactory(func() VelenClient {
		return fakeVelenClient{
			queryFunc: func(sourceKey string, queryFile string) ([]byte, error) {
				sqlBody, err := os.ReadFile(queryFile)
				if err != nil {
					return nil, err
				}
				sql := string(sqlBody)
				switch sourceKey {
				case "github-main":
					if strings.Contains(sql, "FROM pull_requests") {
						return []byte(`{"rows":[{"pr_number":142,"title":"Checkout Redesign","author":"kim","merged_at":"2026-02-15T00:00:00Z"}]}`), nil
					}
				case "prod-warehouse":
					if strings.Contains(sql, "FROM deployments") {
						return []byte(`{"rows":[{"deployed_at":"2026-02-15T03:00:00Z"}]}`), nil
					}
				case "amplitude-prod":
					if strings.Contains(sql, "phase: before") {
						return []byte(`{"rows":[{"metric_value":0.10,"sample_size":2000}]}`), nil
					}
					if strings.Contains(sql, "phase: after") {
						return []byte(`{"rows":[{"metric_value":0.12,"sample_size":2100}]}`), nil
					}
				}
				return nil, assertErr("unexpected query input")
			},
		}
	}, func() {
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		code := Run([]string{"analyze", "--config", configPath, "--pr", "142", "--output", "json"}, repoRoot, strings.NewReader(""), &stdout, &stderr)
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
		singlePR, _ := result["single_pr"].(map[string]any)
		if singlePR == nil {
			t.Fatalf("expected single_pr result payload")
		}
		score, _ := singlePR["impact_score"].(map[string]any)
		if score["score"] == nil {
			t.Fatalf("expected impact score")
		}
	})
}

func TestRunCheckSourcesJSONEnvelope(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	configPath := writeTestConfig(t, repoRoot, testConfigOptions{})

	withVelenClientFactory(func() VelenClient {
		return fakeVelenClient{
			identity:   VelenIdentity{Handle: "ci-user"},
			currentOrg: "impactable",
			sources: []VelenSource{
				{Key: "github-main", Provider: "github", SupportsQuery: true},
				{Key: "prod-warehouse", Provider: "bigquery", SupportsQuery: true},
				{Key: "amplitude-prod", Provider: "amplitude", SupportsQuery: true},
			},
			showByKey: map[string]VelenSource{
				"github-main":    {Key: "github-main", Provider: "github", SupportsQuery: true},
				"prod-warehouse": {Key: "prod-warehouse", Provider: "bigquery", SupportsQuery: true},
				"amplitude-prod": {Key: "amplitude-prod", Provider: "amplitude", SupportsQuery: true},
			},
		}
	}, func() {
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		code := Run([]string{"check-sources", "--config", configPath, "--require", "github,warehouse", "--output", "json"}, repoRoot, strings.NewReader(""), &stdout, &stderr)
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
		result, _ := response["result"].(map[string]any)
		if result == nil {
			t.Fatalf("expected result envelope")
		}
		sources, _ := result["sources"].([]any)
		if len(sources) != 2 {
			t.Fatalf("expected two source contracts, got %d", len(sources))
		}
		summary, _ := result["summary"].(map[string]any)
		if summary["ready"] != true {
			t.Fatalf("expected ready=true summary, got %#v", summary["ready"])
		}
	})
}

func TestRunReportScaffoldJSONEnvelope(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	configPath := writeTestConfig(t, repoRoot, testConfigOptions{})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"report-scaffold", "--config", configPath, "--mode", "markdown", "--mode", "html", "--output", "json"}, repoRoot, strings.NewReader(""), &stdout, &stderr)
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
	result, _ := response["result"].(map[string]any)
	if result == nil {
		t.Fatalf("expected result envelope")
	}
	reports, _ := result["reports"].([]any)
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
	result, _ := response["result"].(map[string]any)
	if result == nil {
		t.Fatalf("expected result envelope")
	}
	items, _ := result["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("expected one schema item, got %d", len(items))
	}
}
