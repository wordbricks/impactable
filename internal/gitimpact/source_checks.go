package gitimpact

import (
	"context"
	"fmt"
	"strings"
)

type sourceCheckSummary struct {
	Required int  `json:"required"`
	OK       int  `json:"ok"`
	Missing  int  `json:"missing"`
	Failed   int  `json:"failed"`
	Ready    bool `json:"ready"`
}

func checkRequiredSources(ctx context.Context, cfg Config, requiredRoles []string, client VelenClient) ([]sourceCheckContract, sourceCheckSummary, map[string]any, error) {
	identity, err := client.WhoAmI(ctx)
	if err != nil {
		return nil, sourceCheckSummary{}, nil, fmt.Errorf("velen auth whoami failed: %w", err)
	}
	currentOrg, err := client.CurrentOrg(ctx)
	if err != nil {
		return nil, sourceCheckSummary{}, nil, fmt.Errorf("velen org current failed: %w", err)
	}
	sources, err := client.ListSources(ctx)
	if err != nil {
		return nil, sourceCheckSummary{}, nil, fmt.Errorf("velen source list failed: %w", err)
	}

	sourcesByKey := map[string]VelenSource{}
	for _, source := range sources {
		if strings.TrimSpace(source.Key) == "" {
			continue
		}
		sourcesByKey[source.Key] = source
	}

	checks := make([]sourceCheckContract, 0, len(requiredRoles))
	summary := sourceCheckSummary{
		Required: len(requiredRoles),
	}
	for _, role := range requiredRoles {
		sourceKey := sourceKeyFromRole(cfg, role)
		check := sourceCheckContract{
			Role:      role,
			SourceKey: sourceKey,
		}
		if strings.TrimSpace(sourceKey) == "" {
			check.Status = "failed"
			check.Message = "no source mapping configured for role"
			summary.Failed++
			checks = append(checks, check)
			continue
		}

		source, exists := sourcesByKey[sourceKey]
		if !exists {
			// Fallback to detail lookup in case source list omits results.
			detail, showErr := client.ShowSource(ctx, sourceKey)
			if showErr != nil {
				check.Status = "missing"
				check.Message = "source not found in velen source list"
				summary.Missing++
				checks = append(checks, check)
				continue
			}
			source = detail
		}
		check.Provider = source.Provider
		check.QuerySupported = source.SupportsQuery

		if !source.SupportsQuery {
			check.Status = "failed"
			check.Message = "source does not support QUERY"
			summary.Failed++
			checks = append(checks, check)
			continue
		}
		check.Status = "ok"
		summary.OK++
		checks = append(checks, check)
	}

	orgMatch := strings.EqualFold(strings.TrimSpace(cfg.Velen.Org), strings.TrimSpace(currentOrg))
	if !orgMatch {
		summary.Failed++
	}
	summary.Ready = summary.Missing == 0 && summary.Failed == 0

	contextPayload := map[string]any{
		"identity": map[string]any{
			"handle": identity.Handle,
		},
		"org": map[string]any{
			"expected": cfg.Velen.Org,
			"current":  currentOrg,
			"match":    orgMatch,
		},
		"discovered_sources": len(sources),
	}
	return checks, summary, contextPayload, nil
}

func sourceKeyFromRole(cfg Config, role string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "github":
		return cfg.Velen.Sources.GitHub
	case "warehouse":
		return cfg.Velen.Sources.Warehouse
	case "analytics":
		return cfg.Velen.Sources.Analytics
	default:
		return ""
	}
}
