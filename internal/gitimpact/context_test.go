package gitimpact

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewAnalysisContext_WithDefaults(t *testing.T) {
	t.Parallel()

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}

	configPath := writeTestConfig(t, t.TempDir())
	ctx, err := NewAnalysisContext("2026-01-01", 0, "", configPath)
	if err != nil {
		t.Fatalf("NewAnalysisContext returned error: %v", err)
	}
	if ctx.WorkingDirectory != filepath.Clean(cwd) {
		t.Fatalf("expected working directory %q, got %q", filepath.Clean(cwd), ctx.WorkingDirectory)
	}
	if ctx.ConfigPath != filepath.Clean(configPath) {
		t.Fatalf("expected config path %q, got %q", filepath.Clean(configPath), ctx.ConfigPath)
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

	configPath := writeTestConfig(t, t.TempDir())
	_, err := NewAnalysisContext("01-01-2026", 0, "", configPath)
	if err == nil {
		t.Fatalf("expected invalid --since error")
	}
}

func TestNewAnalysisContext_RejectsPRAndFeatureTogether(t *testing.T) {
	t.Parallel()

	configPath := writeTestConfig(t, t.TempDir())
	_, err := NewAnalysisContext("", 42, "onboarding-v2", configPath)
	if err == nil {
		t.Fatalf("expected mutually exclusive --pr and --feature error")
	}
}

func TestNewAnalysisContext_RejectsMissingConfig(t *testing.T) {
	t.Parallel()

	_, err := NewAnalysisContext("", 0, "", filepath.Join(t.TempDir(), "missing.yaml"))
	if err == nil {
		t.Fatalf("expected config load error")
	}
}

func TestBuildInitialPrompt_IncludesScopeAndSources(t *testing.T) {
	t.Parallel()

	parsed, err := NewAnalysisContext("2026-01-01", 142, "", writeTestConfig(t, t.TempDir()))
	if err != nil {
		t.Fatalf("NewAnalysisContext returned error: %v", err)
	}

	cfg := &Config{
		OneQuery: OneQueryConfig{
			Org: "impactable",
			Sources: OneQuerySources{
				GitHub:    "github-main",
				Analytics: "amplitude-prod",
			},
		},
	}
	prompt := BuildInitialPrompt(parsed, cfg)

	expected := []string{
		"scope_since: 2026-01-01",
		"scope_pr: PR #142 only",
		"github_source_key: github-main",
		"analytics_source_key: amplitude-prod",
	}
	for _, fragment := range expected {
		if !strings.Contains(prompt, fragment) {
			t.Fatalf("expected prompt to contain %q, got:\n%s", fragment, prompt)
		}
	}
}
