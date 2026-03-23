package gitimpact

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const sinceDateLayout = "2006-01-02"

// NewAnalysisContext converts parsed CLI arguments into a runtime context.
func NewAnalysisContext(since string, prNum int, feature string, configPath string) (*AnalysisContext, error) {
	workingDirectory, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("resolve working directory: %w", err)
	}

	resolvedConfigPath, err := resolveConfigPath(workingDirectory, configPath)
	if err != nil {
		return nil, err
	}

	// Load once here to ensure CLI context creation fails fast on config issues.
	if _, err := LoadConfig(resolvedConfigPath); err != nil {
		return nil, err
	}

	parsedSince, err := parseSince(since)
	if err != nil {
		return nil, err
	}

	trimmedFeature := strings.TrimSpace(feature)
	if prNum < 0 {
		return nil, fmt.Errorf("--pr must be zero or a positive integer")
	}
	if prNum > 0 && trimmedFeature != "" {
		return nil, fmt.Errorf("--pr and --feature cannot be set together")
	}

	return &AnalysisContext{
		WorkingDirectory: filepath.Clean(workingDirectory),
		ConfigPath:       resolvedConfigPath,
		Since:            parsedSince,
		PRNumber:         prNum,
		Feature:          trimmedFeature,
	}, nil
}

// BuildInitialPrompt constructs the first system prompt for the WTL analysis agent.
func BuildInitialPrompt(ctx *AnalysisContext, cfg *Config) string {
	if ctx == nil {
		ctx = &AnalysisContext{}
	}
	if cfg == nil {
		cfg = &Config{}
	}

	since := "not set"
	if ctx.Since != nil {
		since = ctx.Since.Format(sinceDateLayout)
	}

	prScope := "all PRs"
	if ctx.PRNumber > 0 {
		prScope = fmt.Sprintf("PR #%d only", ctx.PRNumber)
	}

	featureScope := "not set"
	if strings.TrimSpace(ctx.Feature) != "" {
		featureScope = strings.TrimSpace(ctx.Feature)
	}

	githubSource := strings.TrimSpace(cfg.Velen.Sources.GitHub)
	if githubSource == "" {
		githubSource = "not configured"
	}
	analyticsSource := strings.TrimSpace(cfg.Velen.Sources.Analytics)
	if analyticsSource == "" {
		analyticsSource = "not configured"
	}

	org := strings.TrimSpace(cfg.Velen.Org)
	if org == "" {
		org = "not configured"
	}

	return strings.TrimSpace(fmt.Sprintf(`
You are the WTL agent for git-impact. Analyze repository changes and estimate product impact.

Task context:
- working_directory: %s
- config_path: %s
- scope_since: %s
- scope_pr: %s
- scope_feature: %s

Configured Velen context:
- org: %s
- github_source_key: %s
- analytics_source_key: %s

Required startup flow:
1) Run source checks first (whoami, current org, source list).
2) Confirm GitHub and Analytics sources are available.
3) If required inputs are missing or ambiguous, pause and ask the user before continuing.
`, strings.TrimSpace(ctx.WorkingDirectory), strings.TrimSpace(ctx.ConfigPath), since, prScope, featureScope, org, githubSource, analyticsSource))
}

func resolveConfigPath(cwd string, configPath string) (string, error) {
	trimmed := strings.TrimSpace(configPath)
	if trimmed == "" {
		trimmed = DefaultConfigFile
	}
	if filepath.IsAbs(trimmed) {
		return filepath.Clean(trimmed), nil
	}
	if strings.TrimSpace(cwd) == "" {
		return "", fmt.Errorf("working directory is required to resolve relative --config")
	}
	return filepath.Clean(filepath.Join(cwd, trimmed)), nil
}

func parseSince(value string) (*time.Time, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, nil
	}
	parsed, err := time.Parse(sinceDateLayout, trimmed)
	if err != nil {
		return nil, fmt.Errorf("invalid --since value %q (expected YYYY-MM-DD)", trimmed)
	}
	return &parsed, nil
}
