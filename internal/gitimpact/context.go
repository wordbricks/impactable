package gitimpact

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

func LoadConfig(configPath string) (*Config, string, error) {
	resolvedPath := strings.TrimSpace(configPath)
	if resolvedPath == "" {
		resolvedPath = DefaultConfigPath
	}
	resolvedPath = filepath.Clean(resolvedPath)

	loader := viper.New()
	loader.SetConfigFile(resolvedPath)
	if err := loader.ReadInConfig(); err != nil {
		return nil, "", fmt.Errorf("read config %q: %w", resolvedPath, err)
	}

	cfg := &Config{}
	if err := loader.Unmarshal(cfg); err != nil {
		return nil, "", fmt.Errorf("decode config %q: %w", resolvedPath, err)
	}

	return cfg, resolvedPath, nil
}

func NewAnalysisContext(since string, prNum int, feature string, configPath string) (*AnalysisContext, error) {
	cfg, resolvedPath, err := LoadConfig(configPath)
	if err != nil {
		return nil, err
	}

	analysisCtx := &AnalysisContext{
		Since:       strings.TrimSpace(since),
		FeatureName: strings.TrimSpace(feature),
		ConfigPath:  resolvedPath,
		Config:      cfg,
	}
	if prNum > 0 {
		analysisCtx.PRNumber = &prNum
	}

	return analysisCtx, nil
}

func BuildInitialPrompt(ctx *AnalysisContext) string {
	if ctx == nil {
		return "You are the Git Impact Analyzer agent. No analysis context was provided."
	}

	since := optionalString(ctx.Since)
	prNumber := "not provided"
	if ctx.PRNumber != nil {
		prNumber = fmt.Sprintf("%d", *ctx.PRNumber)
	}
	feature := optionalString(ctx.FeatureName)
	githubSource := "not configured"
	analyticsSource := "not configured"
	velenOrg := "not configured"

	if ctx.Config != nil {
		velenOrg = optionalString(ctx.Config.Velen.Org)
		if strings.TrimSpace(ctx.Config.Velen.Sources.GitHub) != "" {
			githubSource = strings.TrimSpace(ctx.Config.Velen.Sources.GitHub)
		}
		if strings.TrimSpace(ctx.Config.Velen.Sources.Analytics) != "" {
			analyticsSource = strings.TrimSpace(ctx.Config.Velen.Sources.Analytics)
		}
	}

	prompt := []string{
		"You are the Git Impact Analyzer agent for a monorepo analysis run.",
		"",
		"Use this structured run context:",
		fmt.Sprintf("- since: %s", since),
		fmt.Sprintf("- pr_number: %s", prNumber),
		fmt.Sprintf("- feature_name: %s", feature),
		fmt.Sprintf("- config_path: %s", optionalString(ctx.ConfigPath)),
		fmt.Sprintf("- velen_org: %s", velenOrg),
		fmt.Sprintf("- github_source_key: %s", githubSource),
		fmt.Sprintf("- analytics_source_key: %s", analyticsSource),
		"",
		"Execution rules:",
		"- Use Velen CLI for all external data access.",
		"- Start by checking source availability and QUERY capability.",
		"- Use read-only SELECT queries with date ranges and LIMIT.",
		"- If source discovery is ambiguous or required sources are missing, issue wait and ask the user.",
	}

	return strings.Join(prompt, "\n")
}

func optionalString(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "not provided"
	}
	return trimmed
}
