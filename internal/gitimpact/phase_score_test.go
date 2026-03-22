package gitimpact

import (
	"context"
	"math"
	"strings"
	"testing"
	"time"
)

func TestCalculateScore(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		before float64
		after  float64
		want   float64
	}{
		{name: "no change", before: 100, after: 100, want: 0},
		{name: "half change", before: 100, after: 150, want: 5},
		{name: "full change", before: 10, after: 0, want: 10},
		{name: "zero baseline", before: 0, after: 0.4, want: 4},
		{name: "cap at ten", before: 1, after: 5, want: 10},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := calculateScore(tc.before, tc.after)
			if math.Abs(got-tc.want) > 0.0001 {
				t.Fatalf("calculateScore(%v, %v) = %v, want %v", tc.before, tc.after, got, tc.want)
			}
		})
	}
}

func TestAssessConfidence(t *testing.T) {
	t.Parallel()

	base := time.Date(2026, 2, 20, 12, 0, 0, 0, time.UTC)

	high := assessConfidence(base, []Deployment{{PRNumber: 1, DeployedAt: base}})
	if high != "high" {
		t.Fatalf("expected high confidence, got %q", high)
	}

	medium := assessConfidence(base, []Deployment{
		{PRNumber: 1, DeployedAt: base},
		{PRNumber: 2, DeployedAt: base.Add(24 * time.Hour)},
		{PRNumber: 3, DeployedAt: base.Add(-2 * 24 * time.Hour)},
		{PRNumber: 4, DeployedAt: base.Add(10 * 24 * time.Hour)},
	})
	if medium != "medium" {
		t.Fatalf("expected medium confidence, got %q", medium)
	}

	low := assessConfidence(base, []Deployment{
		{PRNumber: 1, DeployedAt: base},
		{PRNumber: 2, DeployedAt: base.Add(2 * time.Hour)},
		{PRNumber: 3, DeployedAt: base.Add(-3 * time.Hour)},
		{PRNumber: 4, DeployedAt: base.Add(48 * time.Hour)},
	})
	if low != "low" {
		t.Fatalf("expected low confidence, got %q", low)
	}
}

func TestBuildContributorStats(t *testing.T) {
	t.Parallel()

	impacts := []PRImpact{
		{PRNumber: 10, Score: 8.0},
		{PRNumber: 11, Score: 4.0},
		{PRNumber: 12, Score: 9.0},
	}
	prs := []PR{
		{Number: 10, Author: "alice"},
		{Number: 11, Author: "alice"},
		{Number: 12, Author: "bob"},
	}

	stats := buildContributorStats(impacts, prs)
	if len(stats) != 2 {
		t.Fatalf("expected 2 contributor stats rows, got %d", len(stats))
	}

	byAuthor := map[string]ContributorStats{}
	for _, stat := range stats {
		byAuthor[stat.Author] = stat
	}

	alice, ok := byAuthor["alice"]
	if !ok {
		t.Fatal("missing alice contributor stats")
	}
	if alice.PRCount != 2 {
		t.Fatalf("alice PR count = %d, want 2", alice.PRCount)
	}
	if math.Abs(alice.AverageScore-6.0) > 0.0001 {
		t.Fatalf("alice average score = %v, want 6", alice.AverageScore)
	}
	if alice.TopPRNumber != 10 {
		t.Fatalf("alice top PR = %d, want 10", alice.TopPRNumber)
	}

	bob, ok := byAuthor["bob"]
	if !ok {
		t.Fatal("missing bob contributor stats")
	}
	if bob.PRCount != 1 {
		t.Fatalf("bob PR count = %d, want 1", bob.PRCount)
	}
	if math.Abs(bob.AverageScore-9.0) > 0.0001 {
		t.Fatalf("bob average score = %v, want 9", bob.AverageScore)
	}
	if bob.TopPRNumber != 12 {
		t.Fatalf("bob top PR = %d, want 12", bob.TopPRNumber)
	}
}

func TestScoreHandlerHandle_EmptyAnalyticsSchemaGracefulDegradation(t *testing.T) {
	t.Parallel()

	schemaCalls := 0
	metricCalls := 0
	handler := &ScoreHandler{
		Query: func(_ *VelenClient, sourceKey string, sql string) (*QueryResult, error) {
			if sourceKey != "analytics-main" {
				t.Fatalf("unexpected source key %q", sourceKey)
			}
			if sql == analyticsSchemaSQL {
				schemaCalls++
				return &QueryResult{Rows: [][]interface{}{}}, nil
			}
			metricCalls++
			return &QueryResult{}, nil
		},
	}

	deployedAt := time.Date(2026, 2, 20, 10, 0, 0, 0, time.UTC)
	runCtx := &RunContext{
		VelenClient: &VelenClient{},
		Config: &Config{
			Velen: VelenConfig{
				Sources: VelenSources{Analytics: "analytics-main"},
			},
		},
		CollectedData: &CollectedData{
			PRs: []PR{{Number: 101, Author: "alice"}},
		},
		LinkedData: &LinkedData{
			Deployments: []Deployment{{PRNumber: 101, DeployedAt: deployedAt}},
		},
	}

	result, err := handler.Handle(context.Background(), runCtx)
	if err != nil {
		t.Fatalf("handle returned error: %v", err)
	}
	if result == nil || result.Directive != DirectiveAdvancePhase {
		t.Fatalf("expected advance directive, got %+v", result)
	}
	if schemaCalls != 1 {
		t.Fatalf("expected one schema query, got %d", schemaCalls)
	}
	if metricCalls != 0 {
		t.Fatalf("expected no metric queries for empty schema, got %d", metricCalls)
	}

	if runCtx.ScoredData == nil {
		t.Fatal("expected scored data to be populated")
	}
	if len(runCtx.ScoredData.PRImpacts) != 1 {
		t.Fatalf("expected one PR impact, got %d", len(runCtx.ScoredData.PRImpacts))
	}
	impact := runCtx.ScoredData.PRImpacts[0]
	if impact.PRNumber != 101 {
		t.Fatalf("unexpected PR number %d", impact.PRNumber)
	}
	if impact.Score != 0 {
		t.Fatalf("expected neutral score 0, got %v", impact.Score)
	}
	if impact.Confidence != "high" {
		t.Fatalf("expected high confidence for isolated deployment, got %q", impact.Confidence)
	}
	if !strings.Contains(impact.Reasoning, "No analytics metric discovered") {
		t.Fatalf("unexpected reasoning: %q", impact.Reasoning)
	}

	if len(runCtx.ScoredData.ContributorStats) != 1 {
		t.Fatalf("expected one contributor stat, got %d", len(runCtx.ScoredData.ContributorStats))
	}
	stat := runCtx.ScoredData.ContributorStats[0]
	if stat.Author != "alice" {
		t.Fatalf("unexpected contributor author %q", stat.Author)
	}
	if stat.PRCount != 1 || stat.TopPRNumber != 101 || stat.AverageScore != 0 {
		t.Fatalf("unexpected contributor stats: %#v", stat)
	}
}
