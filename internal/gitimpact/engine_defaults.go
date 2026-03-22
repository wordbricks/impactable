package gitimpact

import (
	"context"
	"fmt"
)

// ReportHandler finalizes the analysis run.
type ReportHandler struct{}

// Handle completes the run and emits a summary output string.
func (h *ReportHandler) Handle(_ context.Context, runCtx *RunContext) (*TurnResult, error) {
	if runCtx == nil {
		return nil, fmt.Errorf("run context is required")
	}

	return &TurnResult{
		Directive: DirectiveComplete,
		Output:    buildReportOutput(runCtx),
	}, nil
}

// DefaultHandlers returns the standard phase handler set for git-impact analysis.
func DefaultHandlers() map[Phase]PhaseHandler {
	return map[Phase]PhaseHandler{
		PhaseSourceCheck: &SourceCheckHandler{},
		PhaseCollect:     &CollectHandler{},
		PhaseLink:        &LinkHandler{},
		PhaseScore:       &ScoreHandler{},
		PhaseReport:      &ReportHandler{},
	}
}

// NewDefaultEngine builds an engine with all default phase handlers configured.
func NewDefaultEngine(client *VelenClient, observer Observer, waitHandler WaitHandler) *Engine {
	_ = client
	return &Engine{
		Handlers:    DefaultHandlers(),
		Observer:    observer,
		WaitHandler: waitHandler,
		MaxRetries:  defaultMaxRetries,
	}
}

func buildReportOutput(runCtx *RunContext) string {
	prCount := 0
	deploymentCount := 0
	impactCount := 0

	if runCtx.CollectedData != nil {
		prCount = len(runCtx.CollectedData.PRs)
	}
	if runCtx.LinkedData != nil {
		deploymentCount = len(runCtx.LinkedData.Deployments)
	}
	if runCtx.ScoredData != nil {
		impactCount = len(runCtx.ScoredData.PRImpacts)
	}

	return fmt.Sprintf(
		"analysis complete: %d PRs, %d deployments, %d scored impacts",
		prCount,
		deploymentCount,
		impactCount,
	)
}
