package gitimpact

import (
	"context"
	"fmt"
	"strings"
)

type checkSourcesFn func(ctx context.Context, client *VelenClient, cfg *Config) (*SourceCheckResult, error)

// SourceCheckHandler verifies required Velen sources before collection starts.
type SourceCheckHandler struct {
	CheckSources checkSourcesFn
}

// Handle validates source readiness and can pause for user confirmation when requirements are not met.
func (h *SourceCheckHandler) Handle(ctx context.Context, runCtx *RunContext) (*TurnResult, error) {
	if runCtx == nil {
		return nil, fmt.Errorf("run context is required")
	}

	waitResponse := ""
	if runCtx.AnalysisCtx != nil {
		waitResponse = strings.ToLower(strings.TrimSpace(runCtx.AnalysisCtx.LastWaitResponse))
	}
	if waitResponse != "" {
		switch waitResponse {
		case "y":
			return &TurnResult{Directive: DirectiveAdvancePhase}, nil
		case "n":
			return nil, fmt.Errorf("source check aborted by user")
		default:
			return nil, fmt.Errorf("invalid wait response %q: expected y or n", waitResponse)
		}
	}

	checker := h.CheckSources
	if checker == nil {
		checker = CheckSources
	}

	result, err := checker(ctx, runCtx.VelenClient, runCtx.Config)
	if err != nil {
		return nil, err
	}
	if result != nil && result.GitHubOK && result.AnalyticsOK {
		return &TurnResult{Directive: DirectiveAdvancePhase}, nil
	}

	return &TurnResult{
		Directive:   DirectiveWait,
		WaitMessage: sourceCheckWaitMessage(result),
	}, nil
}

func sourceCheckWaitMessage(result *SourceCheckResult) string {
	if result == nil {
		return "Required sources could not be verified. Continue anyway? (y/n)"
	}

	if len(result.Errors) == 0 {
		return "Required sources are not QUERY-capable. Continue anyway? (y/n)"
	}

	return fmt.Sprintf(
		"Required Velen sources are not ready: %s. Continue anyway? (y/n)",
		strings.Join(result.Errors, "; "),
	)
}
