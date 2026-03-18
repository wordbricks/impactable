package gitimpact

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfig_AppliesAnalysisDefaults(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	configPath := writeTestConfig(t, repoRoot, testConfigOptions{
		omitAnalysis: true,
	})

	cfg, resolved, err := loadConfig(repoRoot, configPath)
	if err != nil {
		t.Fatalf("loadConfig returned error: %v", err)
	}
	if resolved != configPath {
		t.Fatalf("expected resolved path %q, got %q", configPath, resolved)
	}
	if cfg.Analysis.BeforeWindowDays != 7 {
		t.Fatalf("expected default before_window_days=7, got %d", cfg.Analysis.BeforeWindowDays)
	}
	if cfg.Analysis.AfterWindowDays != 7 {
		t.Fatalf("expected default after_window_days=7, got %d", cfg.Analysis.AfterWindowDays)
	}
	if cfg.Analysis.CooldownHours != 24 {
		t.Fatalf("expected default cooldown_hours=24, got %d", cfg.Analysis.CooldownHours)
	}
	if cfg.Analysis.MinConfidence != 0.6 {
		t.Fatalf("expected default min_confidence=0.6, got %v", cfg.Analysis.MinConfidence)
	}
}

func TestLoadConfig_RejectsMissingSourceMapping(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	configPath := writeTestConfig(t, repoRoot, testConfigOptions{
		omitAnalyticsSource: true,
	})

	_, _, err := loadConfig(repoRoot, configPath)
	if err == nil {
		t.Fatalf("expected loadConfig error for missing analytics source")
	}
	if !strings.Contains(err.Error(), "velen.sources.analytics is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadConfig_RejectsOutOfRangeMinConfidence(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	configPath := writeTestConfig(t, repoRoot, testConfigOptions{
		minConfidence: "1.2",
	})

	_, _, err := loadConfig(repoRoot, configPath)
	if err == nil {
		t.Fatalf("expected loadConfig error for out-of-range min_confidence")
	}
	if !strings.Contains(err.Error(), "analysis.min_confidence must be between 0 and 1") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadConfig_ResolvesRelativePath(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	configPath := writeTestConfig(t, repoRoot, testConfigOptions{})
	relative := filepath.Base(configPath)

	cfg, resolved, err := loadConfig(repoRoot, relative)
	if err != nil {
		t.Fatalf("loadConfig returned error: %v", err)
	}
	if resolved != configPath {
		t.Fatalf("expected resolved path %q, got %q", configPath, resolved)
	}
	if cfg.Velen.Org != "impactable" {
		t.Fatalf("unexpected org: %q", cfg.Velen.Org)
	}
}

func TestLoadConfig_UsesDefaultConfigPathWhenPathEmpty(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	configPath := writeTestConfig(t, repoRoot, testConfigOptions{})

	cfg, resolved, err := loadConfig(repoRoot, "  ")
	if err != nil {
		t.Fatalf("loadConfig returned error: %v", err)
	}
	if resolved != configPath {
		t.Fatalf("expected resolved path %q, got %q", configPath, resolved)
	}
	if cfg.Velen.Sources.GitHub != "github-main" {
		t.Fatalf("unexpected github source: %q", cfg.Velen.Sources.GitHub)
	}
}

func TestLoadConfig_AppliesDefaultsForUnsetAnalysisFields(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	configPath := writeTestConfig(t, repoRoot, testConfigOptions{
		omitAfterWindowDays: true,
		omitCooldownHours:   true,
		omitMinConfidence:   true,
	})

	cfg, _, err := loadConfig(repoRoot, configPath)
	if err != nil {
		t.Fatalf("loadConfig returned error: %v", err)
	}
	if cfg.Analysis.BeforeWindowDays != 14 {
		t.Fatalf("expected configured before_window_days=14, got %d", cfg.Analysis.BeforeWindowDays)
	}
	if cfg.Analysis.AfterWindowDays != 7 {
		t.Fatalf("expected default after_window_days=7, got %d", cfg.Analysis.AfterWindowDays)
	}
	if cfg.Analysis.CooldownHours != 24 {
		t.Fatalf("expected default cooldown_hours=24, got %d", cfg.Analysis.CooldownHours)
	}
	if cfg.Analysis.MinConfidence != 0.6 {
		t.Fatalf("expected default min_confidence=0.6, got %v", cfg.Analysis.MinConfidence)
	}
}

func TestLoadConfig_RejectsNaNMinConfidence(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	configPath := writeTestConfig(t, repoRoot, testConfigOptions{
		minConfidence: "nan",
	})

	_, _, err := loadConfig(repoRoot, configPath)
	if err == nil {
		t.Fatalf("expected loadConfig error for NaN min_confidence")
	}
	if !strings.Contains(err.Error(), "analysis.min_confidence must be between 0 and 1") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadConfig_RejectsNonIntegerBeforeWindow(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	configPath := writeTestConfig(t, repoRoot, testConfigOptions{
		beforeWindowDays: "seven",
	})

	_, _, err := loadConfig(repoRoot, configPath)
	if err == nil {
		t.Fatalf("expected loadConfig error for non-integer before_window_days")
	}
	if !strings.Contains(err.Error(), "analysis.before_window_days must be an integer") {
		t.Fatalf("unexpected error: %v", err)
	}
}

type testConfigOptions struct {
	omitAnalysis        bool
	omitAnalyticsSource bool
	omitAfterWindowDays bool
	omitCooldownHours   bool
	omitMinConfidence   bool
	beforeWindowDays    string
	afterWindowDays     string
	cooldownHours       string
	minConfidence       string
}

func writeTestConfig(t *testing.T, dir string, opts testConfigOptions) string {
	t.Helper()

	beforeWindowDays := opts.beforeWindowDays
	if beforeWindowDays == "" {
		beforeWindowDays = "14"
	}
	afterWindowDays := opts.afterWindowDays
	if afterWindowDays == "" {
		afterWindowDays = "10"
	}
	cooldownHours := opts.cooldownHours
	if cooldownHours == "" {
		cooldownHours = "12"
	}
	minConfidence := opts.minConfidence
	if minConfidence == "" {
		minConfidence = "0.8"
	}

	lines := []string{
		"velen:",
		"  org: impactable",
		"  sources:",
		"    github: github-main",
		"    warehouse: prod-warehouse",
	}
	if !opts.omitAnalyticsSource {
		lines = append(lines, "    analytics: amplitude-prod")
	}
	if !opts.omitAnalysis {
		lines = append(lines, "analysis:")
		lines = append(lines, "  before_window_days: "+beforeWindowDays)
		if !opts.omitAfterWindowDays {
			lines = append(lines, "  after_window_days: "+afterWindowDays)
		}
		if !opts.omitCooldownHours {
			lines = append(lines, "  cooldown_hours: "+cooldownHours)
		}
		if !opts.omitMinConfidence {
			lines = append(lines, "  min_confidence: "+minConfidence)
		}
	}

	configPath := filepath.Join(dir, defaultConfigPath)
	if err := os.WriteFile(configPath, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
	return configPath
}
