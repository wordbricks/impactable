package gitimpact

import "context"

// DefaultHandlers returns the baseline phase-handler registry for git-impact runs.
func DefaultHandlers(client *VelenClient) map[Phase]PhaseHandler {
	_ = client
	return map[Phase]PhaseHandler{
		PhaseSourceCheck: &SourceCheckHandler{},
		PhaseCollect:     &CollectHandler{},
		PhaseLink:        advancePhaseHandler{},
		PhaseScore:       advancePhaseHandler{},
		PhaseReport:      advancePhaseHandler{},
	}
}

type advancePhaseHandler struct{}

func (advancePhaseHandler) Handle(context.Context, *RunContext) (*TurnResult, error) {
	return &TurnResult{Directive: DirectiveAdvancePhase}, nil
}
