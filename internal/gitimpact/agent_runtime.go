package gitimpact

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"impactable/internal/wtl"
)

const (
	defaultAgentModel     = "gpt-5.4"
	agentServiceName      = "git-impact"
	agentCommandEnv       = "GIT_IMPACT_CODEX_COMMAND"
	agentModelEnv         = "GIT_IMPACT_MODEL"
	agentFallbackModelEnv = "WTL_MODEL"
	agentApprovalPolicy   = "never"
	agentSandbox          = "workspace-write"
	agentClientTitle      = "Git Impact Analyzer"
	agentClientVersion    = "0.1.0"
)

type agentTurnRunner interface {
	StartThread(context.Context) (string, error)
	RunTurn(context.Context, string, string, func(string)) (AgentTurnResult, error)
	Close() error
}

// AgentTurnResult is the app-server turn result consumed by the git-impact
// phase agent.
type AgentTurnResult struct {
	TurnID       string
	Status       string
	Response     string
	ErrorMessage string
}

// CodexAgentConfig configures the Codex app-server backed git-impact agent.
type CodexAgentConfig struct {
	CWD   string
	Model string
}

// CodexAgentRuntime runs each git-impact phase as one Codex app-server turn on
// a single thread.
type CodexAgentRuntime struct {
	runner   agentTurnRunner
	threadID string
	model    string
	cwd      string
}

// NewCodexAgentRuntime starts a Codex app-server backed agent runtime.
func NewCodexAgentRuntime(cfg CodexAgentConfig) (*CodexAgentRuntime, error) {
	client, err := wtl.NewAppServerClient(wtl.AppServerConfig{
		CWD:            strings.TrimSpace(cfg.CWD),
		Model:          ResolveAgentModel(cfg.Model),
		ServiceName:    agentServiceName,
		ClientName:     agentServiceName,
		ClientTitle:    agentClientTitle,
		ClientVersion:  agentClientVersion,
		ApprovalPolicy: agentApprovalPolicy,
		Sandbox:        agentSandbox,
		NetworkAccess:  true,
		CommandEnv:     agentCommandEnv,
	})
	if err != nil {
		return nil, err
	}
	return NewCodexAgentRuntimeWithRunner(cfg, &wtlClientAdapter{client: client}), nil
}

func NewCodexAgentRuntimeWithRunner(cfg CodexAgentConfig, runner agentTurnRunner) *CodexAgentRuntime {
	return &CodexAgentRuntime{
		runner: runner,
		model:  ResolveAgentModel(cfg.Model),
		cwd:    strings.TrimSpace(cfg.CWD),
	}
}

// ResolveAgentModel returns the configured model with environment overrides.
func ResolveAgentModel(model string) string {
	if value := strings.TrimSpace(os.Getenv(agentModelEnv)); value != "" {
		return value
	}
	if value := strings.TrimSpace(os.Getenv(agentFallbackModelEnv)); value != "" {
		return value
	}
	if value := strings.TrimSpace(model); value != "" {
		return value
	}
	return defaultAgentModel
}

// Close terminates the underlying app-server process.
func (r *CodexAgentRuntime) Close() error {
	if r == nil || r.runner == nil {
		return nil
	}
	return r.runner.Close()
}

func (r *CodexAgentRuntime) runPhase(ctx context.Context, runCtx *RunContext, phase Phase) (*TurnResult, error) {
	if r == nil || r.runner == nil {
		return nil, fmt.Errorf("codex agent runtime is not configured")
	}
	if strings.TrimSpace(r.threadID) == "" {
		threadID, err := r.runner.StartThread(ctx)
		if err != nil {
			return nil, err
		}
		r.threadID = threadID
	}

	prompt, err := BuildAgentPhasePrompt(runCtx, phase)
	if err != nil {
		return nil, err
	}
	result, err := r.runner.RunTurn(ctx, r.threadID, prompt, nil)
	if err != nil {
		return &TurnResult{Directive: DirectiveRetry, Error: err, Output: result.Response}, nil
	}
	if strings.EqualFold(strings.TrimSpace(result.Status), "failed") {
		message := strings.TrimSpace(result.ErrorMessage)
		if message == "" {
			message = "codex app-server turn failed"
		}
		return &TurnResult{Directive: DirectiveRetry, Error: fmt.Errorf("%s", message), Output: result.Response}, nil
	}

	payload, err := ParseAgentPhasePayload(result.Response)
	if err != nil {
		return &TurnResult{Directive: DirectiveRetry, Error: err, Output: result.Response}, nil
	}
	applyAgentPhasePayload(runCtx, payload)

	directive := payload.Directive
	if directive == "" {
		directive = DirectiveContinue
	}
	return &TurnResult{
		Directive:   directive,
		WaitMessage: payload.WaitMessage,
		Output:      payload.Output,
		Error:       payload.err(),
	}, nil
}

type wtlClientAdapter struct {
	client *wtl.AppServerClient
}

func (a *wtlClientAdapter) StartThread(ctx context.Context) (string, error) {
	return a.client.StartThread(ctx)
}

func (a *wtlClientAdapter) RunTurn(ctx context.Context, threadID string, prompt string, onDelta func(string)) (AgentTurnResult, error) {
	result, err := a.client.RunTurn(ctx, threadID, prompt, onDelta)
	return AgentTurnResult{
		TurnID:       result.TurnID,
		Status:       result.Status,
		Response:     result.Response,
		ErrorMessage: result.ErrorMessage,
	}, err
}

func (a *wtlClientAdapter) Close() error {
	return a.client.Close()
}

type AgentPhaseHandler struct {
	Phase Phase
	Agent *CodexAgentRuntime
}

func (h *AgentPhaseHandler) Handle(ctx context.Context, runCtx *RunContext) (*TurnResult, error) {
	if h == nil || h.Agent == nil {
		return nil, fmt.Errorf("agent phase handler is not configured")
	}
	return h.Agent.runPhase(ctx, runCtx, h.Phase)
}

// AgentHandlers returns a phase-handler set where every phase is executed by
// the same Codex app-server thread.
func AgentHandlers(agent *CodexAgentRuntime) map[Phase]PhaseHandler {
	return map[Phase]PhaseHandler{
		PhaseSourceCheck: &AgentPhaseHandler{Phase: PhaseSourceCheck, Agent: agent},
		PhaseCollect:     &AgentPhaseHandler{Phase: PhaseCollect, Agent: agent},
		PhaseLink:        &AgentPhaseHandler{Phase: PhaseLink, Agent: agent},
		PhaseScore:       &AgentPhaseHandler{Phase: PhaseScore, Agent: agent},
		PhaseReport:      &AgentPhaseHandler{Phase: PhaseReport, Agent: agent},
	}
}

// NewAgentEngine builds a git-impact engine that delegates each phase turn to
// Codex app-server.
func NewAgentEngine(agent *CodexAgentRuntime, observer Observer, waitHandler WaitHandler) *Engine {
	return &Engine{
		Handlers:    AgentHandlers(agent),
		Observer:    observer,
		WaitHandler: waitHandler,
		MaxRetries:  defaultMaxRetries,
	}
}

func applyAgentPhasePayload(runCtx *RunContext, payload AgentPhasePayload) {
	if runCtx == nil {
		return
	}
	if payload.CollectedData != nil {
		runCtx.CollectedData = payload.CollectedData
	}
	if payload.LinkedData != nil {
		runCtx.LinkedData = payload.LinkedData
	}
	if payload.ScoredData != nil {
		runCtx.ScoredData = payload.ScoredData
	}
	if payload.AnalysisResult != nil {
		if len(payload.AnalysisResult.PRs) > 0 || len(payload.AnalysisResult.Deployments) > 0 {
			runCtx.CollectedData = &CollectedData{PRs: payload.AnalysisResult.PRs}
			runCtx.LinkedData = &LinkedData{
				Deployments:    payload.AnalysisResult.Deployments,
				FeatureGroups:  payload.AnalysisResult.FeatureGroups,
				AmbiguousItems: nil,
			}
		}
		if len(payload.AnalysisResult.PRImpacts) > 0 || len(payload.AnalysisResult.Contributors) > 0 {
			runCtx.ScoredData = &ScoredData{
				PRImpacts:        payload.AnalysisResult.PRImpacts,
				ContributorStats: payload.AnalysisResult.Contributors,
			}
		}
	}
}

type agentPromptSnapshot struct {
	GeneratedAtUTC string           `json:"generated_at_utc"`
	Phase          Phase            `json:"phase"`
	Config         *Config          `json:"config,omitempty"`
	Analysis       *AnalysisContext `json:"analysis_context,omitempty"`
	CollectedData  *CollectedData   `json:"collected_data,omitempty"`
	LinkedData     *LinkedData      `json:"linked_data,omitempty"`
	ScoredData     *ScoredData      `json:"scored_data,omitempty"`
}

// BuildAgentPhasePrompt constructs the app-server prompt for a single
// git-impact phase turn.
func BuildAgentPhasePrompt(runCtx *RunContext, phase Phase) (string, error) {
	if runCtx == nil {
		return "", fmt.Errorf("run context is required")
	}
	snapshot := agentPromptSnapshot{
		GeneratedAtUTC: time.Now().UTC().Format(time.RFC3339),
		Phase:          phase,
		Config:         runCtx.Config,
		Analysis:       runCtx.AnalysisCtx,
		CollectedData:  runCtx.CollectedData,
		LinkedData:     runCtx.LinkedData,
		ScoredData:     runCtx.ScoredData,
	}
	state, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(fmt.Sprintf(`You are the Git Impact Analyzer WTL Agent.

You are running inside Codex app-server. This is one WTL turn for phase %q.
Use only OneQuery CLI for external data access. Do not use direct credentials or write operations.
All OneQuery SQL must be SELECT-only and bounded by date filters and LIMITs where practical.

Configured flow:
- Source check: run "onequery --org <org> auth whoami", "onequery --org <org> org current", "onequery --org <org> source list", and "source show" for required sources.
- Collect: query GitHub source for PRs, commits, branches, tags, releases, labels, and changed files for the requested scope.
- Link: infer deployment markers from releases, tags, then PR merge time; wait when mapping is ambiguous.
- Score: explore analytics schema, choose relevant metrics, query before/after deployment windows, and explain confidence.
- Report: produce final analysis data and complete the run.

Current state JSON:
%s

Phase instructions:
%s

Return exactly one JSON object. No markdown. No prose outside JSON.
Schema:
{
  "directive": "advance_phase|wait|retry|continue|complete",
  "wait_message": "only when directive is wait",
  "output": "short human-readable phase summary",
  "collected_data": {
    "PRs": [{"Number": 0, "Title": "", "Author": "", "MergedAt": "RFC3339 timestamp", "Branch": "", "Labels": [], "ChangedFile": []}],
    "Tags": [{"Name": "", "Sha": "", "CreatedAt": "RFC3339 timestamp or empty"}],
    "Releases": [{"Name": "", "TagName": "", "PublishedAt": "RFC3339 timestamp"}]
  },
  "linked_data": {"Deployments": [], "FeatureGroups": [], "AmbiguousItems": []},
  "scored_data": {"PRImpacts": [], "ContributorStats": []},
  "analysis_result": {"Output": "", "PRs": [], "Deployments": [], "FeatureGroups": [], "Contributors": [], "PRImpacts": []},
  "error": ""
}
Only include data fields that are relevant to this phase.
When returning collected_data, use exactly the PRs, Tags, and Releases top-level keys shown above; put repository, scope, labels, and commit summaries in output text, not in collected_data.`, phase, string(state), phaseInstructions(phase))), nil
}

func phaseInstructions(phase Phase) string {
	switch phase {
	case PhaseSourceCheck:
		return "Verify org and required GitHub/Analytics OneQuery sources. If a required source is missing, return directive wait with a specific question. Otherwise return advance_phase."
	case PhaseCollect:
		return "Collect GitHub PR, tag, and release data for the requested scope. Return collected_data using the exact typed schema: Tags must be objects with Name, optional Sha, and optional CreatedAt; never return Tags as strings or as arbitrary metadata objects. Return advance_phase when enough data is present."
	case PhaseLink:
		return "Infer deployments and feature groups from collected_data. If a deployment or feature mapping is ambiguous, return wait with the concrete mapping question. Otherwise return linked_data and advance_phase."
	case PhaseScore:
		return "Explore analytics schema through OneQuery, choose relevant metrics, query bounded before/after windows, calculate PR impact and contributor stats. For every PRImpact, write a detailed multi-line Reasoning string instead of a one-line summary. The reasoning must explain: (1) why the primary metric was chosen over other available metrics, (2) the deployment marker and before/after analysis windows used, with concrete dates, (3) the before/after metric values and delta, (4) why that movement implies the assigned impact score, and (5) what reduces or increases confidence, including overlapping deployments or weak attribution. Start the first line with `Metric <name> ...` so downstream views can still identify the primary metric. Use explicit numbers and dates. Return scored_data and advance_phase."
	case PhaseReport:
		return "Assemble the final report data from current state. Return complete with output and optionally analysis_result."
	default:
		return "Complete the current phase according to the Git Impact Analyzer SPEC."
	}
}
