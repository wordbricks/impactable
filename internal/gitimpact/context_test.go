package gitimpact

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewAnalysisContextLoadsConfigAndOptionalFields(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "impact-analyzer.yaml")
	configBody := `
velen:
  org: my-org
  sources:
    github: gh-main
    analytics: amp-prod
analysis:
  before_window_days: 7
  after_window_days: 7
  cooldown_hours: 24
`
	if err := os.WriteFile(configPath, []byte(strings.TrimSpace(configBody)+"\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	ctx, err := NewAnalysisContext("2026-01-01", 142, "onboarding-v2", configPath)
	if err != nil {
		t.Fatalf("NewAnalysisContext returned error: %v", err)
	}
	if ctx == nil {
		t.Fatalf("expected context")
	}
	if ctx.Since != "2026-01-01" {
		t.Fatalf("unexpected since value: %q", ctx.Since)
	}
	if ctx.PRNumber == nil || *ctx.PRNumber != 142 {
		t.Fatalf("unexpected pr value: %#v", ctx.PRNumber)
	}
	if ctx.FeatureName != "onboarding-v2" {
		t.Fatalf("unexpected feature value: %q", ctx.FeatureName)
	}
	if ctx.Config == nil {
		t.Fatalf("expected config to be loaded")
	}
	if ctx.Config.Velen.Sources.GitHub != "gh-main" {
		t.Fatalf("unexpected github source: %q", ctx.Config.Velen.Sources.GitHub)
	}
}

func TestBuildInitialPromptIncludesStructuredInputs(t *testing.T) {
	pr := 142
	ctx := &AnalysisContext{
		Since:       "2026-01-01",
		PRNumber:    &pr,
		FeatureName: "onboarding-v2",
		ConfigPath:  "impact-analyzer.yaml",
		Config: &Config{
			Velen: VelenConfig{
				Org: "my-org",
				Sources: VelenSourcesConfig{
					GitHub:    "gh-main",
					Analytics: "amp-prod",
				},
			},
		},
	}

	prompt := BuildInitialPrompt(ctx)
	for _, expected := range []string{
		"- since: 2026-01-01",
		"- pr_number: 142",
		"- feature_name: onboarding-v2",
		"- github_source_key: gh-main",
		"- analytics_source_key: amp-prod",
	} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("prompt missing %q\n%s", expected, prompt)
		}
	}
}
