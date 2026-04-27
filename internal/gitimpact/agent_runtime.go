package gitimpact

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
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
	agentPhaseTimeoutEnv  = "GIT_IMPACT_PHASE_TIMEOUT"
	agentApprovalPolicy   = "never"
	agentSandbox          = "workspace-write"
	agentClientTitle      = "Git Impact Analyzer"
	agentClientVersion    = "0.1.0"
	defaultPhaseTimeout   = 30 * time.Minute
	agentTraceDir         = ".git-impact"
	agentTraceFile        = "agent-events.jsonl"
)

type agentTurnRunner interface {
	StartThread(context.Context) (string, error)
	RunTurn(context.Context, string, string, func(string), func(AgentRuntimeEvent)) (AgentTurnResult, error)
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

// AgentRuntimeEvent is a compact trace event from the app-server turn.
type AgentRuntimeEvent struct {
	At      time.Time `json:"at"`
	Phase   Phase     `json:"phase"`
	Method  string    `json:"method"`
	Summary string    `json:"summary,omitempty"`
}

// CodexAgentConfig configures the Codex app-server backed git-impact agent.
type CodexAgentConfig struct {
	CWD          string
	Model        string
	PhaseTimeout time.Duration
}

// CodexAgentRuntime runs each git-impact phase as one Codex app-server turn on
// a single thread.
type CodexAgentRuntime struct {
	runner   agentTurnRunner
	threadID string
	model    string
	cwd      string
	timeout  time.Duration
	trace    []AgentRuntimeEvent
	output   string
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
		runner:  runner,
		model:   ResolveAgentModel(cfg.Model),
		cwd:     strings.TrimSpace(cfg.CWD),
		timeout: resolvePhaseTimeout(cfg.PhaseTimeout),
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

func resolvePhaseTimeout(timeout time.Duration) time.Duration {
	if value := strings.TrimSpace(os.Getenv(agentPhaseTimeoutEnv)); value != "" {
		if parsed, err := time.ParseDuration(value); err == nil && parsed > 0 {
			return parsed
		}
	}
	if timeout > 0 {
		return timeout
	}
	return defaultPhaseTimeout
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

	phaseCtx := ctx
	cancel := func() {}
	if r.timeout > 0 {
		phaseCtx, cancel = context.WithTimeout(ctx, r.timeout)
	}
	defer cancel()

	var partial strings.Builder
	r.output = ""
	result, err := r.runner.RunTurn(phaseCtx, r.threadID, prompt, func(delta string) {
		partial.WriteString(delta)
		r.output = partial.String()
	}, func(event AgentRuntimeEvent) {
		event.Phase = phase
		r.recordEvent(runCtx, event)
	})
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			output := strings.TrimSpace(result.Response)
			if output == "" {
				output = strings.TrimSpace(partial.String())
			}
			return nil, r.phaseTimeoutError(phase, output)
		}
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

func (r *CodexAgentRuntime) recordEvent(runCtx *RunContext, event AgentRuntimeEvent) {
	if r == nil {
		return
	}
	if event.At.IsZero() {
		event.At = time.Now().UTC()
	}
	event.Method = strings.TrimSpace(event.Method)
	event.Summary = strings.TrimSpace(event.Summary)
	r.trace = append(r.trace, event)
	if len(r.trace) > 40 {
		r.trace = r.trace[len(r.trace)-40:]
	}
	appendAgentTrace(runCtx, event)
}

func (r *CodexAgentRuntime) phaseTimeoutError(phase Phase, partialOutput string) error {
	lastEvent := "none"
	if r != nil {
		for idx := len(r.trace) - 1; idx >= 0; idx-- {
			last := r.trace[idx]
			if last.Phase != phase {
				continue
			}
			lastEvent = fmt.Sprintf("%s %s", last.Method, last.Summary)
			break
		}
	}
	output := truncateAgentText(partialOutput, 800)
	if output == "" && r != nil {
		output = truncateAgentText(r.output, 800)
	}
	if output == "" {
		return fmt.Errorf("phase %q timed out after %s waiting for codex app-server turn completion; last_event=%s; trace_file=%s", phase, r.timeout, lastEvent, filepath.Join(agentTraceDir, agentTraceFile))
	}
	return fmt.Errorf("phase %q timed out after %s waiting for codex app-server turn completion; last_event=%s; partial_output=%q; trace_file=%s", phase, r.timeout, lastEvent, output, filepath.Join(agentTraceDir, agentTraceFile))
}

func appendAgentTrace(runCtx *RunContext, event AgentRuntimeEvent) {
	if runCtx == nil || runCtx.AnalysisCtx == nil {
		return
	}
	baseDir := strings.TrimSpace(runCtx.AnalysisCtx.WorkingDirectory)
	if baseDir == "" {
		return
	}
	dir := filepath.Join(baseDir, agentTraceDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}
	path := filepath.Join(dir, agentTraceFile)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return
	}
	defer func() {
		_ = file.Close()
	}()
	_ = json.NewEncoder(file).Encode(event)
}

func truncateAgentText(value string, limit int) string {
	trimmed := strings.TrimSpace(value)
	if limit <= 0 || len(trimmed) <= limit {
		return trimmed
	}
	return strings.TrimSpace(trimmed[:limit]) + "..."
}

type wtlClientAdapter struct {
	client *wtl.AppServerClient
}

func (a *wtlClientAdapter) StartThread(ctx context.Context) (string, error) {
	return a.client.StartThread(ctx)
}

func (a *wtlClientAdapter) RunTurn(ctx context.Context, threadID string, prompt string, onDelta func(string), onEvent func(AgentRuntimeEvent)) (AgentTurnResult, error) {
	result, err := a.client.RunTurnWithEvents(ctx, threadID, prompt, onDelta, func(event wtl.AppServerEvent) {
		if onEvent == nil {
			return
		}
		onEvent(AgentRuntimeEvent{
			At:      time.Now().UTC(),
			Method:  event.Method,
			Summary: event.Summary,
		})
	})
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

// AgentHandlers returns the phase-handler set for the agent-backed runtime.
// Source checks stay local because they are deterministic CLI probes; later
// phases use the Codex app-server thread for analysis and judgment.
func AgentHandlers(agent *CodexAgentRuntime) map[Phase]PhaseHandler {
	return map[Phase]PhaseHandler{
		PhaseSourceCheck: &SourceCheckHandler{},
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
Use "onequery query exec" for SQL-queryable sources and "onequery api --source <source_key>" for connected API sources, including sources whose source metadata has queryable=false. All OneQuery SQL must be SELECT-only and bounded by date filters and LIMITs where practical. API calls must use read-only HTTP methods unless a phase explicitly requires otherwise.

Configured flow:
- Source check: run "onequery --org <org> auth whoami", "onequery --org <org> org current", "onequery --org <org> source list", and "source show" for required sources.
- Collect: use OneQuery query or API commands against the GitHub source for PRs, commits, branches, tags, releases, labels, and changed files for the requested scope.
- Link: infer deployment markers from releases, tags, then PR merge time; wait when mapping is ambiguous.
- Score: explore the analytics source through OneQuery query or API commands, choose relevant metrics, fetch bounded before/after deployment windows, and explain confidence.
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
		return "Verify org and required GitHub/Analytics OneQuery sources. A source with queryable=false can still be usable through onequery api, so do not reject a source only because it is not SQL-queryable. If a required source is missing, return directive wait with a specific question. Otherwise return advance_phase."
	case PhaseCollect:
		return "Collect GitHub PR, tag, and release data for the requested scope. Prefer onequery api for GitHub connected API sources, for example `onequery --org <org> api --source <github_source> <owner>/<repo>/pulls -f params[state]=closed -f params[per_page]=100`; follow with API calls for commits, tags, releases, labels, and changed files as needed. If Config.OneQuery.GitHubRepository is set, analyze exactly that GitHub repository full name and do not infer the repository from the current worktree or local git remote. Return collected_data using the exact typed schema: Tags must be objects with Name, optional Sha, and optional CreatedAt; never return Tags as strings or as arbitrary metadata objects. Return advance_phase when enough data is present."
	case PhaseLink:
		return "Infer deployments and feature groups from collected_data. If a deployment or feature mapping is ambiguous, return wait with the concrete mapping question. Otherwise return linked_data and advance_phase."
	case PhaseScore:
		return "Explore analytics through OneQuery. Use onequery query exec when the source is SQL-queryable, or onequery api for connected API sources such as Amplitude, for example `onequery --org <org> api --source <analytics_source> /2/events/segmentation -f 'params[e]=[...]' -f params[start]=YYYY-MM-DD -f params[end]=YYYY-MM-DD`. Choose relevant metrics, fetch bounded before/after windows, calculate PR impact and contributor stats. For every PRImpact, populate structured fields as available: PrimaryMetric, BeforeValue, AfterValue, DeltaValue, BeforeWindowStart, BeforeWindowEnd, AfterWindowStart, and AfterWindowEnd. Use RFC3339 timestamps for window boundaries. Also write a detailed multi-line Reasoning string instead of a one-line summary. The reasoning must explain: (1) why the primary metric was chosen over other available metrics, (2) the deployment marker and before/after analysis windows used, with concrete dates, (3) the before/after metric values and delta, (4) why that movement implies the assigned impact score, and (5) what reduces or increases confidence, including overlapping deployments or weak attribution. Use explicit numbers and dates. Return scored_data and advance_phase."
	case PhaseReport:
		return "Assemble the final report data from current state. Return complete with output and optionally analysis_result."
	default:
		return "Complete the current phase according to the Git Impact Analyzer SPEC."
	}
}
