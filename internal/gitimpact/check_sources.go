package gitimpact

import (
	"context"
	"fmt"
	"strings"
)

var newVelenClient = func(_ *Config) VelenClient {
	return NewCLIClient()
}

func CheckSources(ctx context.Context, cfg *Config) (*SourceCheckResult, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	return CheckSourcesWithClient(ctx, cfg, newVelenClient(cfg))
}

func CheckSourcesWithClient(ctx context.Context, cfg *Config, client VelenClient) (*SourceCheckResult, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	if client == nil {
		return nil, fmt.Errorf("velen client is required")
	}

	result := &SourceCheckResult{}

	if err := client.AuthWhoAmI(ctx); err != nil {
		result.Error = fmt.Sprintf("velen auth whoami failed: %v", err)
		return result, err
	}

	orgName, err := client.OrgCurrent(ctx)
	if err != nil {
		result.Error = fmt.Sprintf("velen org current failed: %v", err)
		return result, err
	}
	result.OrgName = orgName

	sources, err := client.SourceList(ctx)
	if err != nil {
		result.Error = fmt.Sprintf("velen source list failed: %v", err)
		return result, err
	}

	preferredGitHubKey := strings.TrimSpace(cfg.Velen.Sources.GitHub)
	preferredAnalyticsKey := strings.TrimSpace(cfg.Velen.Sources.Analytics)

	result.GitHubSource = pickSource(sources, preferredGitHubKey, isGitHubProvider)
	result.AnalyticsSource = pickSource(sources, preferredAnalyticsKey, isAnalyticsProvider)

	var issues []string

	if result.GitHubSource == nil {
		issues = append(issues, "github source not found")
	} else if !result.GitHubSource.SupportsQuery() {
		issues = append(issues, "github source does not support QUERY")
	} else {
		result.GitHubOK = true
	}

	if result.AnalyticsSource == nil {
		issues = append(issues, "analytics source not found")
	} else if !result.AnalyticsSource.SupportsQuery() {
		issues = append(issues, "analytics source does not support QUERY")
	} else {
		result.AnalyticsOK = true
	}

	if len(issues) > 0 {
		result.Error = strings.Join(issues, "; ")
		return result, fmt.Errorf("%s", result.Error)
	}

	return result, nil
}

func pickSource(sources []Source, preferredKey string, matchFn func(string) bool) *Source {
	if preferredKey != "" {
		for i := range sources {
			source := sources[i]
			if !strings.EqualFold(strings.TrimSpace(source.Key), preferredKey) {
				continue
			}
			if matchFn(source.ProviderType) {
				selected := source
				return &selected
			}
		}
	}
	for i := range sources {
		source := sources[i]
		if matchFn(source.ProviderType) {
			selected := source
			return &selected
		}
	}
	return nil
}

func isGitHubProvider(providerType string) bool {
	return strings.Contains(strings.ToLower(strings.TrimSpace(providerType)), "github")
}

func isAnalyticsProvider(providerType string) bool {
	provider := strings.ToLower(strings.TrimSpace(providerType))
	for _, fragment := range []string{"analytics", "amplitude", "mixpanel", "segment"} {
		if strings.Contains(provider, fragment) {
			return true
		}
	}
	return false
}
