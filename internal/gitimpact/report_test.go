package gitimpact

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSaveMarkdownWritesExpectedSections(t *testing.T) {
	t.Parallel()

	result := sampleAnalysisResultForReports()
	path := filepath.Join(t.TempDir(), "impact.md")

	if err := SaveMarkdown(result, path); err != nil {
		t.Fatalf("SaveMarkdown returned error: %v", err)
	}

	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read markdown file: %v", err)
	}
	content := string(body)
	if !strings.Contains(content, "# Git Impact Report") {
		t.Fatalf("markdown report missing title: %q", content)
	}
	if !strings.Contains(content, "## PR Impact Results") {
		t.Fatalf("markdown report missing PR section: %q", content)
	}
	if !strings.Contains(content, "Payment Page Redesign") {
		t.Fatalf("markdown report missing PR title: %q", content)
	}
	if !strings.Contains(content, "## Feature Impact") {
		t.Fatalf("markdown report missing feature section: %q", content)
	}
	if !strings.Contains(content, "## Contributor Leaderboard") {
		t.Fatalf("markdown report missing contributor section: %q", content)
	}
}

func TestSaveHTMLWritesExpectedSections(t *testing.T) {
	t.Parallel()

	result := sampleAnalysisResultForReports()
	path := filepath.Join(t.TempDir(), "impact.html")

	if err := SaveHTML(result, path); err != nil {
		t.Fatalf("SaveHTML returned error: %v", err)
	}

	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read HTML file: %v", err)
	}
	content := string(body)
	if !strings.Contains(content, "<!DOCTYPE html>") {
		t.Fatalf("html report missing doctype: %q", content)
	}
	if !strings.Contains(content, "<h2>PR Impact Results</h2>") {
		t.Fatalf("html report missing PR section: %q", content)
	}
	if !strings.Contains(content, "Contributor Leaderboard") {
		t.Fatalf("html report missing contributor section: %q", content)
	}
	if !strings.Contains(content, "Payment Page Redesign") {
		t.Fatalf("html report missing PR title: %q", content)
	}
}

func TestSaveReportRejectsNilResult(t *testing.T) {
	t.Parallel()

	mdErr := SaveMarkdown(nil, filepath.Join(t.TempDir(), "impact.md"))
	if mdErr == nil {
		t.Fatal("expected SaveMarkdown error for nil result")
	}

	htmlErr := SaveHTML(nil, filepath.Join(t.TempDir(), "impact.html"))
	if htmlErr == nil {
		t.Fatal("expected SaveHTML error for nil result")
	}
}

func sampleAnalysisResultForReports() *AnalysisResult {
	merged := time.Date(2026, 2, 15, 14, 30, 0, 0, time.UTC)
	return &AnalysisResult{
		GeneratedAt: merged,
		PRs: []PR{
			{Number: 142, Title: "Payment Page Redesign", Author: "kim", MergedAt: merged},
			{Number: 140, Title: "API Caching Improvement", Author: "lee", MergedAt: merged.Add(-24 * time.Hour)},
		},
		Deployments: []Deployment{
			{PRNumber: 142, Source: "release", Marker: "v2.4.1", DeployedAt: merged},
		},
		PRImpacts: []PRImpact{
			{PRNumber: 142, Score: 8.2, Confidence: "high", Reasoning: "Metric conversion_rate moved from 0.10 to 0.13 (delta +0.03)"},
			{PRNumber: 140, Score: 6.5, Confidence: "medium", Reasoning: "Metric cache_hit_rate moved from 0.70 to 0.78 (delta +0.08)"},
		},
		FeatureGroups: []FeatureGroup{
			{Name: "feature/onboarding-v2", PRNumbers: []int{140, 142}},
		},
		Contributors: []ContributorStats{
			{Author: "kim", PRCount: 1, AverageScore: 8.2, TopPRNumber: 142},
			{Author: "lee", PRCount: 1, AverageScore: 6.5, TopPRNumber: 140},
		},
	}
}
