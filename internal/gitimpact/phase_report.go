package gitimpact

import (
	"context"
	"fmt"
)

// ReportHandler completes the run with a concise summary from accumulated state.
type ReportHandler struct{}

func (h *ReportHandler) Handle(_ context.Context, runCtx *RunContext) (*TurnResult, error) {
	if runCtx == nil {
		return nil, fmt.Errorf("run context is required")
	}
	prCount := 0
	if runCtx.CollectedData != nil {
		prCount = len(runCtx.CollectedData.PRs)
	}
	impactCount := 0
	if runCtx.ScoredData != nil {
		impactCount = len(runCtx.ScoredData.PRImpacts)
	}
	return &TurnResult{
		Directive: DirectiveComplete,
		Output:    fmt.Sprintf("Completed Git impact analysis for %d PRs with %d scored impacts.", prCount, impactCount),
	}, nil
}
