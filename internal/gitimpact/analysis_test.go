package gitimpact

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

func TestAnalyzeSinglePR_ComputesImpactScore(t *testing.T) {
	t.Parallel()

	cfg := defaultConfig()
	cfg.Velen.Sources.GitHub = "github-main"
	cfg.Velen.Sources.Warehouse = "prod-warehouse"
	cfg.Velen.Sources.Analytics = "amplitude-prod"
	cfg.Analysis.BeforeWindowDays = 7
	cfg.Analysis.AfterWindowDays = 7
	cfg.Analysis.CooldownHours = 24
	cfg.Analysis.MinConfidence = 0.6

	result, stages, err := analyzeSinglePR(context.Background(), fakeVelenClient{
		queryFunc: func(sourceKey string, queryFile string) ([]byte, error) {
			sql, readErr := os.ReadFile(queryFile)
			if readErr != nil {
				return nil, readErr
			}
			text := string(sql)
			switch sourceKey {
			case "github-main":
				if strings.Contains(text, "FROM pull_requests") {
					return []byte(`{"rows":[{"pr_number":142,"title":"Checkout Redesign","author":"kim","merged_at":"2026-02-15T00:00:00Z"}]}`), nil
				}
			case "prod-warehouse":
				if strings.Contains(text, "FROM deployments") {
					return []byte(`{"rows":[{"deployed_at":"2026-02-15T03:00:00Z"}]}`), nil
				}
			case "amplitude-prod":
				if strings.Contains(text, "phase: before") {
					return []byte(`{"rows":[{"metric_value":0.10,"sample_size":2000}]}`), nil
				}
				if strings.Contains(text, "phase: after") {
					return []byte(`{"rows":[{"metric_value":0.12,"sample_size":2100}]}`), nil
				}
			}
			return nil, assertErr("unexpected query")
		},
	}, cfg, 142)
	if err != nil {
		t.Fatalf("analyzeSinglePR returned error: %v", err)
	}
	if stages["collector"] != "completed" || stages["linker"] != "completed" || stages["scorer"] != "completed" {
		t.Fatalf("unexpected stages payload: %#v", stages)
	}
	if result.PR.Number != 142 {
		t.Fatalf("unexpected pr number: %d", result.PR.Number)
	}
	if result.Metric.Before != 0.10 || result.Metric.After != 0.12 {
		t.Fatalf("unexpected metric values: %#v", result.Metric)
	}
	if result.Score.Score <= 0 {
		t.Fatalf("expected positive impact score, got %#v", result.Score)
	}
}

func TestLinkDeployment_FallbackWhenNoRows(t *testing.T) {
	t.Parallel()

	mergedAt := time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC)
	link, deployedAt, err := linkDeployment(context.Background(), fakeVelenClient{
		queryFunc: func(sourceKey string, queryFile string) ([]byte, error) {
			return []byte(`{"rows":[]}`), nil
		},
	}, "prod-warehouse", 142, mergedAt, 24)
	if err != nil {
		t.Fatalf("linkDeployment returned error: %v", err)
	}
	if link.Source != "fallback.merged_at_plus_cooldown" {
		t.Fatalf("expected fallback link source, got %q", link.Source)
	}
	expected := mergedAt.Add(24 * time.Hour)
	if !deployedAt.Equal(expected) {
		t.Fatalf("expected fallback deployedAt %s, got %s", expected, deployedAt)
	}
}

func TestAnalyzeSinglePR_ErrorsOnInvalidMergedAt(t *testing.T) {
	t.Parallel()

	cfg := defaultConfig()
	cfg.Velen.Sources.GitHub = "github-main"
	cfg.Velen.Sources.Warehouse = "prod-warehouse"
	cfg.Velen.Sources.Analytics = "amplitude-prod"

	_, _, err := analyzeSinglePR(context.Background(), fakeVelenClient{
		queryFunc: func(sourceKey string, queryFile string) ([]byte, error) {
			switch sourceKey {
			case "github-main":
				return []byte(`{"rows":[{"pr_number":142,"title":"Checkout Redesign","author":"kim","merged_at":"not-a-time"}]}`), nil
			default:
				return []byte(`{"rows":[{"metric_value":0.10,"sample_size":100}]}`), nil
			}
		},
	}, cfg, 142)
	if err == nil {
		t.Fatalf("expected analyzeSinglePR error")
	}
	if !strings.Contains(err.Error(), "invalid merged_at") {
		t.Fatalf("unexpected error: %v", err)
	}
}
