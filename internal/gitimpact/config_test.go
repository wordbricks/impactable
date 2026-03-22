package gitimpact

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_AppliesDefaults(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "impact-analyzer.yaml")
	content := `velen:
  org: my-company
  sources:
    github: github-main
    analytics: amplitude-prod
feature_grouping:
  strategies:
    - label_prefix
    - branch_prefix
  custom_mappings_file: feature-map.yaml
`
	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}

	if cfg.Velen.Org != "my-company" {
		t.Fatalf("expected org my-company, got %q", cfg.Velen.Org)
	}
	if cfg.Velen.Sources.GitHub != "github-main" {
		t.Fatalf("expected github source github-main, got %q", cfg.Velen.Sources.GitHub)
	}
	if cfg.Velen.Sources.Analytics != "amplitude-prod" {
		t.Fatalf("expected analytics source amplitude-prod, got %q", cfg.Velen.Sources.Analytics)
	}
	if cfg.Analysis.BeforeWindowDays != DefaultBeforeWindowDays {
		t.Fatalf("expected before window default %d, got %d", DefaultBeforeWindowDays, cfg.Analysis.BeforeWindowDays)
	}
	if cfg.Analysis.AfterWindowDays != DefaultAfterWindowDays {
		t.Fatalf("expected after window default %d, got %d", DefaultAfterWindowDays, cfg.Analysis.AfterWindowDays)
	}
	if cfg.Analysis.CooldownHours != DefaultCooldownHours {
		t.Fatalf("expected cooldown default %d, got %d", DefaultCooldownHours, cfg.Analysis.CooldownHours)
	}
}

func TestLoadConfig_UsesExplicitAnalysisValues(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "impact-analyzer.yaml")
	content := `velen:
  org: my-company
  sources:
    github: github-main
    analytics: amplitude-prod
analysis:
  before_window_days: 10
  after_window_days: 5
  cooldown_hours: 12
feature_grouping:
  strategies:
    - label_prefix
  custom_mappings_file: custom-feature-map.yaml
`
	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}

	if cfg.Analysis.BeforeWindowDays != 10 {
		t.Fatalf("expected before window 10, got %d", cfg.Analysis.BeforeWindowDays)
	}
	if cfg.Analysis.AfterWindowDays != 5 {
		t.Fatalf("expected after window 5, got %d", cfg.Analysis.AfterWindowDays)
	}
	if cfg.Analysis.CooldownHours != 12 {
		t.Fatalf("expected cooldown 12, got %d", cfg.Analysis.CooldownHours)
	}
	if cfg.FeatureGrouping.CustomMappingsFile != "custom-feature-map.yaml" {
		t.Fatalf("expected custom mapping file custom-feature-map.yaml, got %q", cfg.FeatureGrouping.CustomMappingsFile)
	}
}
