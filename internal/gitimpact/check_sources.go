package gitimpact

import (
	"context"
	"fmt"
	"strings"
)

// SourceCheckResult summarizes source connectivity status for required providers.
type SourceCheckResult struct {
	GitHubSource    *Source  `json:"github_source,omitempty"`
	AnalyticsSource *Source  `json:"analytics_source,omitempty"`
	GitHubOK        bool     `json:"github_ok"`
	AnalyticsOK     bool     `json:"analytics_ok"`
	OrgName         string   `json:"org_name"`
	Errors          []string `json:"errors,omitempty"`
}

// CheckSources validates auth/org state and discovers required sources.
func CheckSources(ctx context.Context, client *VelenClient, cfg *Config) (*SourceCheckResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if client == nil {
		return nil, fmt.Errorf("velen client is nil")
	}

	whoAmI, err := client.WhoAmI()
	if err != nil {
		return nil, fmt.Errorf("velen auth whoami: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	org, err := client.CurrentOrg()
	if err != nil {
		return nil, fmt.Errorf("velen org current: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	sources, err := client.ListSources()
	if err != nil {
		return nil, fmt.Errorf("velen source list: %w", err)
	}

	result := &SourceCheckResult{
		OrgName: strings.TrimSpace(org.Name),
	}
	if result.OrgName == "" {
		result.OrgName = strings.TrimSpace(org.Slug)
	}
	if result.OrgName == "" && whoAmI != nil {
		result.OrgName = strings.TrimSpace(whoAmI.Org)
	}

	for idx := range sources {
		source := &sources[idx]
		providerType := strings.ToLower(strings.TrimSpace(source.ProviderType))
		if result.GitHubSource == nil && strings.Contains(providerType, "github") {
			result.GitHubSource = source
			continue
		}
		if result.AnalyticsSource == nil && containsAny(providerType, "analytics", "amplitude", "mixpanel", "segment") {
			result.AnalyticsSource = source
		}
	}

	// Fallback to configured source keys when provider type metadata is absent or non-standard.
	if result.GitHubSource == nil && cfg != nil {
		result.GitHubSource = sourceByKey(sources, cfg.Velen.Sources.GitHub)
	}
	if result.AnalyticsSource == nil && cfg != nil {
		result.AnalyticsSource = sourceByKey(sources, cfg.Velen.Sources.Analytics)
	}

	if result.GitHubSource == nil {
		result.Errors = append(result.Errors, "github source not found")
	} else {
		result.GitHubOK = result.GitHubSource.SupportsQuery()
		if !result.GitHubOK {
			result.Errors = append(result.Errors, fmt.Sprintf("github source %q does not support QUERY", result.GitHubSource.Key))
		}
	}

	if result.AnalyticsSource == nil {
		result.Errors = append(result.Errors, "analytics source not found")
	} else {
		result.AnalyticsOK = result.AnalyticsSource.SupportsQuery()
		if !result.AnalyticsOK {
			result.Errors = append(result.Errors, fmt.Sprintf("analytics source %q does not support QUERY", result.AnalyticsSource.Key))
		}
	}

	return result, nil
}

func containsAny(value string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(value, needle) {
			return true
		}
	}
	return false
}

func sourceByKey(sources []Source, key string) *Source {
	trimmed := strings.TrimSpace(key)
	if trimmed == "" {
		return nil
	}
	for idx := range sources {
		source := &sources[idx]
		if strings.EqualFold(strings.TrimSpace(source.Key), trimmed) {
			return source
		}
	}
	return nil
}
