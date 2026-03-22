package gitimpact

import "testing"

func TestNewAnalysisContext_WithDefaults(t *testing.T) {
	t.Parallel()

	ctx, err := NewAnalysisContext("/repo", CLIArgs{Since: "2026-01-01"})
	if err != nil {
		t.Fatalf("NewAnalysisContext returned error: %v", err)
	}
	if ctx.WorkingDirectory != "/repo" {
		t.Fatalf("expected working directory /repo, got %q", ctx.WorkingDirectory)
	}
	if ctx.ConfigPath != "/repo/impact-analyzer.yaml" {
		t.Fatalf("expected default config path /repo/impact-analyzer.yaml, got %q", ctx.ConfigPath)
	}
	if ctx.Since == nil {
		t.Fatalf("expected since date to be set")
	}
	if got := ctx.Since.Format(sinceDateLayout); got != "2026-01-01" {
		t.Fatalf("expected since date 2026-01-01, got %s", got)
	}
	if ctx.PRNumber != 0 {
		t.Fatalf("expected default PR number 0, got %d", ctx.PRNumber)
	}
	if ctx.Feature != "" {
		t.Fatalf("expected empty feature, got %q", ctx.Feature)
	}
}

func TestNewAnalysisContext_RejectsInvalidSince(t *testing.T) {
	t.Parallel()

	_, err := NewAnalysisContext("/repo", CLIArgs{Since: "01-01-2026"})
	if err == nil {
		t.Fatalf("expected invalid --since error")
	}
}

func TestNewAnalysisContext_RejectsPRAndFeatureTogether(t *testing.T) {
	t.Parallel()

	_, err := NewAnalysisContext("/repo", CLIArgs{PR: 42, Feature: "onboarding-v2"})
	if err == nil {
		t.Fatalf("expected mutually exclusive --pr and --feature error")
	}
}
