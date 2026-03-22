package gitimpact

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestCollectHandlerHandle_Success(t *testing.T) {
	t.Parallel()

	since := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	expectedPRSQL := "SELECT number, title, author, merged_at, head_branch, labels FROM pull_requests WHERE merged_at > '2026-01-02' ORDER BY merged_at DESC LIMIT 100"
	expectedTagSQL := "SELECT name, created_at FROM tags ORDER BY created_at DESC LIMIT 50"
	expectedReleaseSQL := "SELECT name, tag_name, published_at FROM releases ORDER BY published_at DESC LIMIT 20"

	seenSQL := make([]string, 0, 3)
	handler := &CollectHandler{
		Query: func(_ *VelenClient, sourceKey string, sql string) (*QueryResult, error) {
			if sourceKey != "github-main" {
				return nil, errors.New("unexpected source key")
			}
			seenSQL = append(seenSQL, sql)

			switch sql {
			case expectedPRSQL:
				return &QueryResult{
					Rows: [][]interface{}{
						{float64(101), "Improve onboarding", "alice", "2026-01-03T12:30:00Z", "feature/onboarding-v2", []interface{}{"feature/onboarding-v2", "ux"}},
					},
				}, nil
			case expectedTagSQL:
				return &QueryResult{
					Rows: [][]interface{}{
						{"v1.2.3", "2026-01-04T08:00:00Z"},
					},
				}, nil
			case expectedReleaseSQL:
				return &QueryResult{
					Rows: [][]interface{}{
						{"January Release", "v1.2.3", "2026-01-05T09:15:00Z"},
					},
				}, nil
			default:
				return nil, errors.New("unexpected sql")
			}
		},
	}

	runCtx := &RunContext{
		VelenClient: &VelenClient{},
		Config: &Config{
			Velen: VelenConfig{
				Sources: VelenSources{GitHub: "github-main"},
			},
		},
		AnalysisCtx: &AnalysisContext{Since: &since},
	}

	result, err := handler.Handle(context.Background(), runCtx)
	if err != nil {
		t.Fatalf("handle returned error: %v", err)
	}
	if result == nil || result.Directive != DirectiveAdvancePhase {
		t.Fatalf("expected advance directive, got %+v", result)
	}
	if runCtx.CollectedData == nil {
		t.Fatal("expected collected data to be populated")
	}
	if len(runCtx.CollectedData.PRs) != 1 {
		t.Fatalf("expected 1 PR, got %d", len(runCtx.CollectedData.PRs))
	}
	if runCtx.CollectedData.PRs[0].Number != 101 {
		t.Fatalf("unexpected PR number: %d", runCtx.CollectedData.PRs[0].Number)
	}
	if !reflect.DeepEqual(runCtx.CollectedData.PRs[0].Labels, []string{"feature/onboarding-v2", "ux"}) {
		t.Fatalf("unexpected PR labels: %#v", runCtx.CollectedData.PRs[0].Labels)
	}
	if !reflect.DeepEqual(runCtx.CollectedData.Tags, []string{"v1.2.3|2026-01-04T08:00:00Z"}) {
		t.Fatalf("unexpected tags: %#v", runCtx.CollectedData.Tags)
	}
	if len(runCtx.CollectedData.Releases) != 1 || runCtx.CollectedData.Releases[0].TagName != "v1.2.3" {
		t.Fatalf("unexpected releases: %#v", runCtx.CollectedData.Releases)
	}
	if !reflect.DeepEqual(seenSQL, []string{expectedPRSQL, expectedTagSQL, expectedReleaseSQL}) {
		t.Fatalf("unexpected query sequence: %#v", seenSQL)
	}
}

func TestCollectHandlerHandle_RequiresGitHubSourceKey(t *testing.T) {
	t.Parallel()

	handler := &CollectHandler{}
	_, err := handler.Handle(context.Background(), &RunContext{
		VelenClient: &VelenClient{},
		Config:      &Config{},
	})
	if err == nil {
		t.Fatal("expected error when github source key is missing")
	}
	if !strings.Contains(err.Error(), "github source key is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCollectHandlerHandle_PropagatesQueryError(t *testing.T) {
	t.Parallel()

	handler := &CollectHandler{
		Query: func(*VelenClient, string, string) (*QueryResult, error) {
			return nil, errors.New("query failed")
		},
	}
	_, err := handler.Handle(context.Background(), &RunContext{
		VelenClient: &VelenClient{},
		Config: &Config{
			Velen: VelenConfig{
				Sources: VelenSources{GitHub: "github-main"},
			},
		},
	})
	if err == nil {
		t.Fatal("expected query error")
	}
	if !strings.Contains(err.Error(), "collect prs") {
		t.Fatalf("expected collect prs wrapper, got %v", err)
	}
}

func TestCollectHandlerHandle_InvalidPRRowsReturnError(t *testing.T) {
	t.Parallel()

	prSQL := "SELECT number, title, author, merged_at, head_branch, labels FROM pull_requests WHERE merged_at > '1970-01-01' ORDER BY merged_at DESC LIMIT 100"

	handler := &CollectHandler{
		Query: func(_ *VelenClient, _ string, sql string) (*QueryResult, error) {
			if sql == prSQL {
				return &QueryResult{
					Rows: [][]interface{}{
						{float64(1), "missing columns"},
					},
				}, nil
			}
			return &QueryResult{}, nil
		},
	}

	_, err := handler.Handle(context.Background(), &RunContext{
		VelenClient: &VelenClient{},
		Config: &Config{
			Velen: VelenConfig{
				Sources: VelenSources{GitHub: "github-main"},
			},
		},
	})
	if err == nil {
		t.Fatal("expected parsing error")
	}
	if !strings.Contains(err.Error(), "collect prs: row 0 has 2 columns, expected 6") {
		t.Fatalf("unexpected error: %v", err)
	}
}
