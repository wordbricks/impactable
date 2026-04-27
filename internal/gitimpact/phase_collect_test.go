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
		Query: func(_ *OneQueryClient, sourceKey string, sql string) (*QueryResult, error) {
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
		OneQueryClient: &OneQueryClient{},
		Config: &Config{
			OneQuery: OneQueryConfig{
				Sources: OneQuerySources{GitHub: "github-main"},
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
	if !reflect.DeepEqual(runCtx.CollectedData.Tags, []Tag{newTag("v1.2.3", time.Date(2026, 1, 4, 8, 0, 0, 0, time.UTC))}) {
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
		OneQueryClient: &OneQueryClient{},
		Config:         &Config{},
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
		Query: func(*OneQueryClient, string, string) (*QueryResult, error) {
			return nil, errors.New("query failed")
		},
	}
	_, err := handler.Handle(context.Background(), &RunContext{
		OneQueryClient: &OneQueryClient{},
		Config: &Config{
			OneQuery: OneQueryConfig{
				Sources: OneQuerySources{GitHub: "github-main"},
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

func TestCollectHandlerHandle_FallsBackToAPIForNonQueryableSource(t *testing.T) {
	t.Parallel()

	since := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	apiCalls := []string{}
	handler := &CollectHandler{
		Query: func(*OneQueryClient, string, string) (*QueryResult, error) {
			return nil, &OneQueryError{Code: "SOURCE_NOT_QUERYABLE", Message: "not queryable"}
		},
		API: func(_ *OneQueryClient, sourceKey string, target string, fields []string, jq string) ([]byte, error) {
			if sourceKey != "github-main" {
				t.Fatalf("unexpected source key: %q", sourceKey)
			}
			apiCalls = append(apiCalls, target)
			switch target {
			case "wordbricks/wordbricks/pulls":
				if !strings.Contains(jq, "2026-04-01T00:00:00Z") {
					t.Fatalf("jq did not include since filter: %s", jq)
				}
				return []byte(`[{"Number":6054,"Title":"Feature","Author":"alice","MergedAt":"2026-04-13T12:41:07Z","Branch":"feature/x","Labels":["impact"]}]`), nil
			case "wordbricks/wordbricks/pulls/6054/files":
				return []byte(`["apps/web/page.tsx"]`), nil
			case "wordbricks/wordbricks/tags":
				return []byte(`[{"Name":"v1.0.0","Sha":"abc123"}]`), nil
			case "wordbricks/wordbricks/releases":
				return []byte(`[{"Name":"Release","TagName":"v1.0.0","PublishedAt":"2026-04-14T00:00:00Z"}]`), nil
			default:
				t.Fatalf("unexpected api target: %q fields=%#v jq=%s", target, fields, jq)
				return nil, nil
			}
		},
	}

	runCtx := &RunContext{
		OneQueryClient: &OneQueryClient{},
		Config: &Config{
			OneQuery: OneQueryConfig{
				GitHubRepository: "wordbricks/wordbricks",
				Sources:          OneQuerySources{GitHub: "github-main"},
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
	if len(runCtx.CollectedData.PRs) != 1 {
		t.Fatalf("expected 1 PR, got %#v", runCtx.CollectedData.PRs)
	}
	if !reflect.DeepEqual(runCtx.CollectedData.PRs[0].ChangedFile, []string{"apps/web/page.tsx"}) {
		t.Fatalf("unexpected changed files: %#v", runCtx.CollectedData.PRs[0].ChangedFile)
	}
	if len(runCtx.CollectedData.Tags) != 1 || runCtx.CollectedData.Tags[0].Sha != "abc123" {
		t.Fatalf("unexpected tags: %#v", runCtx.CollectedData.Tags)
	}
	if len(runCtx.CollectedData.Releases) != 1 || runCtx.CollectedData.Releases[0].TagName != "v1.0.0" {
		t.Fatalf("unexpected releases: %#v", runCtx.CollectedData.Releases)
	}
	wantCalls := []string{
		"wordbricks/wordbricks/pulls",
		"wordbricks/wordbricks/pulls/6054/files",
		"wordbricks/wordbricks/tags",
		"wordbricks/wordbricks/releases",
	}
	if !reflect.DeepEqual(apiCalls, wantCalls) {
		t.Fatalf("unexpected api calls: %#v", apiCalls)
	}
}

func TestCollectHandlerHandle_InvalidPRRowsReturnError(t *testing.T) {
	t.Parallel()

	prSQL := "SELECT number, title, author, merged_at, head_branch, labels FROM pull_requests WHERE merged_at > '1970-01-01' ORDER BY merged_at DESC LIMIT 100"

	handler := &CollectHandler{
		Query: func(_ *OneQueryClient, _ string, sql string) (*QueryResult, error) {
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
		OneQueryClient: &OneQueryClient{},
		Config: &Config{
			OneQuery: OneQueryConfig{
				Sources: OneQuerySources{GitHub: "github-main"},
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
