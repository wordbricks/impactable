package gitimpact

import (
	"context"
	"errors"
	"fmt"
)

const defaultMaxRetries = 3

// Phase identifies the current stage of the git-impact analysis run.
type Phase string

const (
	PhaseSourceCheck Phase = "source_check"
	PhaseCollect     Phase = "collect"
	PhaseLink        Phase = "link"
	PhaseScore       Phase = "score"
	PhaseReport      Phase = "report"
)

// Directive instructs the engine what to do after a turn.
type Directive string

const (
	DirectiveAdvancePhase Directive = "advance_phase"
	DirectiveContinue     Directive = "continue"
	DirectiveRetry        Directive = "retry"
	DirectiveWait         Directive = "wait"
	DirectiveComplete     Directive = "complete"
)

// TurnResult is the phase handler output consumed by the engine loop.
type TurnResult struct {
	Directive   Directive
	WaitMessage string
	Output      string
	Error       error
}

// PhaseHandler executes one phase turn.
type PhaseHandler interface {
	Handle(ctx context.Context, runCtx *RunContext) (*TurnResult, error)
}

// Config holds analysis configuration.
type Config struct{}

// AnalysisContext holds request context for one run.
type AnalysisContext struct {
	LastWaitResponse string
}

// VelenClient encapsulates Velen interactions.
type VelenClient struct{}

// PR is a pull request record.
type PR struct{}

// Release is a release record.
type Release struct{}

// Deployment is an inferred deployment record.
type Deployment struct{}

// FeatureGroup is a grouped feature result.
type FeatureGroup struct{}

// AmbiguousDeployment captures unresolved deployment mappings.
type AmbiguousDeployment struct{}

// PRImpact stores scored impact per PR.
type PRImpact struct{}

// ContributorStats stores contributor rollups.
type ContributorStats struct{}

// RunContext is mutable state shared across phase turns.
type RunContext struct {
	Config        *Config
	AnalysisCtx   *AnalysisContext
	VelenClient   *VelenClient
	Phase         Phase
	Iteration     int
	CollectedData *CollectedData
	LinkedData    *LinkedData
	ScoredData    *ScoredData
}

// CollectedData stores source collection outputs.
type CollectedData struct {
	PRs       []PR
	Tags      []string
	Releases  []Release
	RawOutput string
}

// LinkedData stores deployment-linking outputs.
type LinkedData struct {
	Deployments    []Deployment
	FeatureGroups  []FeatureGroup
	AmbiguousItems []AmbiguousDeployment
}

// ScoredData stores scoring outputs.
type ScoredData struct {
	PRImpacts        []PRImpact
	ContributorStats []ContributorStats
}

// AnalysisResult is the terminal run payload.
type AnalysisResult struct {
	Output        string
	Phase         Phase
	Iteration     int
	CollectedData *CollectedData
	LinkedData    *LinkedData
	ScoredData    *ScoredData
}

// Engine executes ordered git-impact phases using phased-delivery directives.
type Engine struct {
	Handlers    map[Phase]PhaseHandler
	Observer    Observer
	WaitHandler WaitHandler
	MaxRetries  int
}

var phaseOrder = []Phase{
	PhaseSourceCheck,
	PhaseCollect,
	PhaseLink,
	PhaseScore,
	PhaseReport,
}

// Run executes the phased-delivery policy for one analysis run.
func (e *Engine) Run(ctx context.Context, runCtx *RunContext) (*AnalysisResult, error) {
	if runCtx == nil {
		err := errors.New("run context is required")
		e.notifyRunExhausted(err)
		return nil, err
	}

	phaseIndex, err := resolveStartPhase(runCtx.Phase)
	if err != nil {
		e.notifyRunExhausted(err)
		return nil, err
	}

	maxRetries := e.MaxRetries
	if maxRetries <= 0 {
		maxRetries = defaultMaxRetries
	}

	retries := 0
	for {
		if err := ctx.Err(); err != nil {
			e.notifyRunExhausted(err)
			return nil, err
		}

		phase := phaseOrder[phaseIndex]
		runCtx.Phase = phase
		runCtx.Iteration++
		e.notifyTurnStarted(phase, runCtx.Iteration)

		handler, ok := e.Handlers[phase]
		if !ok || handler == nil {
			err := fmt.Errorf("no handler registered for phase %q", phase)
			e.notifyRunExhausted(err)
			return nil, err
		}

		turnResult, err := handler.Handle(ctx, runCtx)
		if err != nil {
			e.notifyRunExhausted(err)
			return nil, err
		}
		if turnResult == nil {
			err := fmt.Errorf("phase %q returned nil result", phase)
			e.notifyRunExhausted(err)
			return nil, err
		}

		switch turnResult.Directive {
		case DirectiveAdvancePhase:
			retries = 0
			nextIndex := phaseIndex + 1
			if nextIndex >= len(phaseOrder) {
				result := newAnalysisResult(runCtx, turnResult.Output)
				e.notifyRunCompleted(result)
				return result, nil
			}
			from := phaseOrder[phaseIndex]
			to := phaseOrder[nextIndex]
			phaseIndex = nextIndex
			runCtx.Phase = to
			e.notifyPhaseAdvanced(from, to)
		case DirectiveComplete:
			result := newAnalysisResult(runCtx, turnResult.Output)
			e.notifyRunCompleted(result)
			return result, nil
		case DirectiveRetry:
			if retries >= maxRetries {
				err := fmt.Errorf("phase %q exceeded max retries (%d)", phase, maxRetries)
				if turnResult.Error != nil {
					err = fmt.Errorf("%w: %v", err, turnResult.Error)
				}
				e.notifyRunExhausted(err)
				return nil, err
			}
			retries++
		case DirectiveWait:
			if e.WaitHandler == nil {
				err := errors.New("wait directive received but wait handler is not configured")
				e.notifyRunExhausted(err)
				return nil, err
			}
			e.notifyWaitEntered(turnResult.WaitMessage)
			response, err := e.WaitHandler(turnResult.WaitMessage)
			if err != nil {
				e.notifyRunExhausted(err)
				return nil, err
			}
			if runCtx.AnalysisCtx == nil {
				runCtx.AnalysisCtx = &AnalysisContext{}
			}
			runCtx.AnalysisCtx.LastWaitResponse = response
			e.notifyWaitResolved(response)
			retries = 0
		case DirectiveContinue:
			retries = 0
		default:
			err := fmt.Errorf("unsupported directive %q", turnResult.Directive)
			e.notifyRunExhausted(err)
			return nil, err
		}
	}
}

func resolveStartPhase(start Phase) (int, error) {
	if start == "" {
		return 0, nil
	}
	for i, phase := range phaseOrder {
		if phase == start {
			return i, nil
		}
	}
	return 0, fmt.Errorf("unsupported start phase %q", start)
}

func newAnalysisResult(runCtx *RunContext, output string) *AnalysisResult {
	return &AnalysisResult{
		Output:        output,
		Phase:         runCtx.Phase,
		Iteration:     runCtx.Iteration,
		CollectedData: runCtx.CollectedData,
		LinkedData:    runCtx.LinkedData,
		ScoredData:    runCtx.ScoredData,
	}
}

func (e *Engine) notifyTurnStarted(phase Phase, iteration int) {
	if e.Observer == nil {
		return
	}
	e.Observer.OnTurnStarted(phase, iteration)
}

func (e *Engine) notifyPhaseAdvanced(from, to Phase) {
	if e.Observer == nil {
		return
	}
	e.Observer.OnPhaseAdvanced(from, to)
}

func (e *Engine) notifyWaitEntered(message string) {
	if e.Observer == nil {
		return
	}
	e.Observer.OnWaitEntered(message)
}

func (e *Engine) notifyWaitResolved(response string) {
	if e.Observer == nil {
		return
	}
	e.Observer.OnWaitResolved(response)
}

func (e *Engine) notifyRunCompleted(result *AnalysisResult) {
	if e.Observer == nil {
		return
	}
	e.Observer.OnRunCompleted(result)
}

func (e *Engine) notifyRunExhausted(err error) {
	if e.Observer == nil {
		return
	}
	e.Observer.OnRunExhausted(err)
}
