package gitimpact

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestResultsModelTabCyclesViews(t *testing.T) {
	t.Parallel()

	model := NewResultsModel(sampleAnalysisResultForResultsModel(), nil)
	modelPtr := &model

	updated, _ := modelPtr.Update(tea.KeyMsg{Type: tea.KeyTab})
	state := updated.(*ResultsModel)
	if state.activeView != resultsViewFeatures {
		t.Fatalf("active view after first tab = %q, want %q", state.activeView, resultsViewFeatures)
	}

	updated, _ = state.Update(tea.KeyMsg{Type: tea.KeyTab})
	state = updated.(*ResultsModel)
	if state.activeView != resultsViewContributors {
		t.Fatalf("active view after second tab = %q, want %q", state.activeView, resultsViewContributors)
	}

	updated, _ = state.Update(tea.KeyMsg{Type: tea.KeyTab})
	state = updated.(*ResultsModel)
	if state.activeView != resultsViewPRs {
		t.Fatalf("active view after third tab = %q, want %q", state.activeView, resultsViewPRs)
	}
}

func TestResultsModelUpDownMovesCursor(t *testing.T) {
	t.Parallel()

	model := NewResultsModel(sampleAnalysisResultForResultsModel(), nil)
	modelPtr := &model

	updated, _ := modelPtr.Update(tea.KeyMsg{Type: tea.KeyTab})
	state := updated.(*ResultsModel)

	updated, _ = state.Update(tea.KeyMsg{Type: tea.KeyDown})
	state = updated.(*ResultsModel)
	if state.cursor != 1 {
		t.Fatalf("cursor after down = %d, want 1", state.cursor)
	}

	updated, _ = state.Update(tea.KeyMsg{Type: tea.KeyUp})
	state = updated.(*ResultsModel)
	if state.cursor != 0 {
		t.Fatalf("cursor after up = %d, want 0", state.cursor)
	}
}

func TestResultsModelPRDetailEnterEsc(t *testing.T) {
	t.Parallel()

	model := NewResultsModel(sampleAnalysisResultForResultsModel(), nil)
	modelPtr := &model

	updated, _ := modelPtr.Update(tea.KeyMsg{Type: tea.KeyEnter})
	state := updated.(*ResultsModel)
	if state.selectedPR == nil {
		t.Fatal("selectedPR is nil after Enter")
	}
	if !strings.Contains(state.View(), "PR #142") {
		t.Fatalf("detail view missing PR header: %q", state.View())
	}

	updated, _ = state.Update(tea.KeyMsg{Type: tea.KeyEsc})
	state = updated.(*ResultsModel)
	if state.selectedPR != nil {
		t.Fatal("selectedPR expected nil after Esc")
	}
}

func TestResultsModelSaveFlow(t *testing.T) {
	t.Parallel()

	called := ""
	model := NewResultsModel(sampleAnalysisResultForResultsModel(), func(format string) (string, error) {
		called = format
		return "/tmp/report." + format, nil
	})
	modelPtr := &model

	updated, _ := modelPtr.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	state := updated.(*ResultsModel)
	if !state.savePrompt {
		t.Fatal("savePrompt = false after pressing s")
	}

	updated, cmd := state.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	state = updated.(*ResultsModel)
	if cmd == nil {
		t.Fatal("expected non-nil save cmd after selecting m")
	}

	msg := cmd()
	updated, _ = state.Update(msg)
	state = updated.(*ResultsModel)
	if called != "md" {
		t.Fatalf("save format = %q, want md", called)
	}
	if state.savePrompt {
		t.Fatal("savePrompt = true after save completion")
	}
	if !strings.Contains(state.saveMessage, "Saved MD report") {
		t.Fatalf("unexpected save message: %q", state.saveMessage)
	}
}

func TestResultsModelContributorViewRendersLeaderboard(t *testing.T) {
	t.Parallel()

	model := NewResultsModel(sampleAnalysisResultForResultsModel(), nil)
	modelPtr := &model

	updated, _ := modelPtr.Update(tea.KeyMsg{Type: tea.KeyTab})
	state := updated.(*ResultsModel)
	updated, _ = state.Update(tea.KeyMsg{Type: tea.KeyTab})
	state = updated.(*ResultsModel)

	view := state.View()
	if !strings.Contains(view, "Contributor Leaderboard") {
		t.Fatalf("contributor view header missing: %q", view)
	}
	if !strings.Contains(view, "Avg Impact") {
		t.Fatalf("contributor table column missing: %q", view)
	}
}

func TestResultsModelDetailScrollUsesViewport(t *testing.T) {
	t.Parallel()

	result := sampleAnalysisResultForResultsModel()
	result.PRImpacts[0].Reasoning = strings.Repeat("signal line\n", 24)
	model := NewResultsModel(result, nil)
	modelPtr := &model

	updated, _ := modelPtr.Update(tea.WindowSizeMsg{Width: 80, Height: 12})
	state := updated.(*ResultsModel)
	updated, _ = state.Update(tea.KeyMsg{Type: tea.KeyEnter})
	state = updated.(*ResultsModel)
	initial := state.viewport.YOffset

	updated, _ = state.Update(tea.KeyMsg{Type: tea.KeyDown})
	state = updated.(*ResultsModel)
	if state.viewport.YOffset <= initial {
		t.Fatalf("expected viewport offset to increase, before=%d after=%d", initial, state.viewport.YOffset)
	}
}

func TestRenderPRDetailContentWrapsReasoningAndDropsQuotes(t *testing.T) {
	t.Parallel()

	result := sampleAnalysisResultForResultsModel()
	result.PRImpacts[0].Reasoning = "Metric conversion_rate was chosen because it is the closest shipped outcome metric for this deployment and other metrics were either sparse or downstream only. The before window ran from 2026-02-08 to 2026-02-15 and the after window ran from 2026-02-15 to 2026-02-22. Conversion moved from 0.10 to 0.13, which is a meaningful relative lift for this flow. Confidence stays high because there were no overlapping deployments in the same window."
	result.PRImpacts[0].PrimaryMetric = "conversion_rate"
	result.PRImpacts[0].BeforeValue = 0.10
	result.PRImpacts[0].AfterValue = 0.13
	result.PRImpacts[0].DeltaValue = 0.03
	result.PRImpacts[0].BeforeWindowStart = time.Date(2026, 2, 8, 0, 0, 0, 0, time.UTC)
	result.PRImpacts[0].BeforeWindowEnd = time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC)
	result.PRImpacts[0].AfterWindowStart = time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC)
	result.PRImpacts[0].AfterWindowEnd = time.Date(2026, 2, 22, 0, 0, 0, 0, time.UTC)

	model := NewResultsModel(result, nil)
	modelPtr := &model

	updated, _ := modelPtr.Update(tea.WindowSizeMsg{Width: 60, Height: 18})
	state := updated.(*ResultsModel)
	updated, _ = state.Update(tea.KeyMsg{Type: tea.KeyEnter})
	state = updated.(*ResultsModel)

	content := state.renderPRDetailContent()
	if strings.Contains(content, "\"Metric conversion_rate") {
		t.Fatalf("reasoning should not be wrapped in quotes: %q", content)
	}
	reasoningSection := strings.SplitN(content, "Agent reasoning:\n", 2)
	if len(reasoningSection) != 2 {
		t.Fatalf("missing reasoning section: %q", content)
	}
	if strings.Count(reasoningSection[1], "\n") < 2 {
		t.Fatalf("expected wrapped multi-line reasoning, got %q", reasoningSection[1])
	}
	if !strings.Contains(content, "Windows: before 2026-02-08 -> 2026-02-15, after 2026-02-15 -> 2026-02-22") {
		t.Fatalf("expected structured window details, got %q", content)
	}
	if !strings.Contains(content, "Values: conversion_rate 0.1000 -> 0.1300 (delta +0.0300)") {
		t.Fatalf("expected structured metric details, got %q", content)
	}
}

func sampleAnalysisResultForResultsModel() *AnalysisResult {
	merged := time.Date(2026, 2, 15, 14, 30, 0, 0, time.UTC)
	return &AnalysisResult{
		GeneratedAt: merged,
		PRs: []PR{
			{Number: 142, Title: "Payment Page Redesign", Author: "kim", MergedAt: merged},
			{Number: 140, Title: "API Caching Improvement", Author: "lee", MergedAt: merged.Add(-24 * time.Hour)},
		},
		Deployments: []Deployment{
			{PRNumber: 142, Marker: "v2.4.1", Source: "release", DeployedAt: merged},
			{PRNumber: 140, Marker: "v2.4.0", Source: "release", DeployedAt: merged.Add(-24 * time.Hour)},
		},
		PRImpacts: []PRImpact{
			{PRNumber: 142, Score: 8.2, Confidence: "high", PrimaryMetric: "conversion_rate", BeforeValue: 0.10, AfterValue: 0.13, DeltaValue: 0.03, Reasoning: "Metric conversion_rate moved from 0.10 to 0.13 (delta +0.03)"},
			{PRNumber: 140, Score: 6.5, Confidence: "medium", PrimaryMetric: "cache_hit_rate", BeforeValue: 0.70, AfterValue: 0.78, DeltaValue: 0.08, Reasoning: "Metric cache_hit_rate moved from 0.70 to 0.78 (delta +0.08)"},
		},
		FeatureGroups: []FeatureGroup{
			{Name: "feature/onboarding-v2", PRNumbers: []int{140, 142}},
			{Name: "feature/payments", PRNumbers: []int{142}},
		},
		Contributors: []ContributorStats{
			{Author: "kim", PRCount: 1, AverageScore: 8.2, TopPRNumber: 142},
			{Author: "lee", PRCount: 1, AverageScore: 6.5, TopPRNumber: 140},
		},
		Output: fmt.Sprintf("analysis complete: %d PRs", 2),
	}
}
