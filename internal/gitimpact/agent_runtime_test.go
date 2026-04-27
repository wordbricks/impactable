package gitimpact

import (
	"context"
	"strings"
	"testing"
	"time"
)

type scriptedAgentRunner struct {
	threadStarts int
	prompts      []string
	responses    []AgentTurnResult
	closeCalls   int
}

func (r *scriptedAgentRunner) StartThread(context.Context) (string, error) {
	r.threadStarts++
	return "thr-agent", nil
}

func (r *scriptedAgentRunner) RunTurn(_ context.Context, _ string, prompt string, onDelta func(string), onEvent func(AgentRuntimeEvent)) (AgentTurnResult, error) {
	r.prompts = append(r.prompts, prompt)
	index := len(r.prompts) - 1
	if onEvent != nil {
		onEvent(AgentRuntimeEvent{Method: "item/completed", Summary: "type=toolCall command=onequery api"})
	}
	if onDelta != nil {
		onDelta("partial agent output")
	}
	if index >= len(r.responses) {
		return AgentTurnResult{Status: "completed", Response: `{"directive":"advance_phase"}`}, nil
	}
	return r.responses[index], nil
}

func (r *scriptedAgentRunner) Close() error {
	r.closeCalls++
	return nil
}

func TestParseAgentPhasePayloadExtractsFencedJSON(t *testing.T) {
	t.Parallel()

	response := "result:\n```json\n{\"directive\":\"advance_phase\",\"output\":\"ok\"}\n```"
	payload, err := ParseAgentPhasePayload(response)
	if err != nil {
		t.Fatalf("parse payload: %v", err)
	}
	if payload.Directive != DirectiveAdvancePhase {
		t.Fatalf("directive = %q, want %q", payload.Directive, DirectiveAdvancePhase)
	}
	if payload.Output != "ok" {
		t.Fatalf("output = %q, want ok", payload.Output)
	}
}

func TestAgentPhaseHandlerRunsCollectTurnAndAppliesPayload(t *testing.T) {
	t.Parallel()

	runner := &scriptedAgentRunner{
		responses: []AgentTurnResult{{
			Status: "completed",
			Response: `{
				"directive":"advance_phase",
				"output":"collected 1 PR",
				"collected_data":{
					"PRs":[{"Number":142,"Title":"Payment Page","Author":"kim","Branch":"feature/payments"}],
					"Tags":[{"Name":"v1.0.0","Sha":"abc123","CreatedAt":"2026-02-15T00:00:00Z"}],
					"Releases":[{"Name":"v1.0.0","TagName":"v1.0.0","PublishedAt":"2026-02-15T00:00:00Z"}]
				}
			}`,
		}},
	}
	agent := NewCodexAgentRuntimeWithRunner(CodexAgentConfig{CWD: "/repo"}, runner)
	handler := &AgentPhaseHandler{Phase: PhaseCollect, Agent: agent}
	runCtx := &RunContext{
		Config: &Config{
			OneQuery: OneQueryConfig{
				Org: "jay",
				Sources: OneQuerySources{
					GitHub:    "wordbricks-github",
					Analytics: "getgpt-prod",
				},
			},
		},
		AnalysisCtx: &AnalysisContext{WorkingDirectory: "/repo"},
	}

	result, err := handler.Handle(context.Background(), runCtx)
	if err != nil {
		t.Fatalf("handle phase: %v", err)
	}
	if result.Directive != DirectiveAdvancePhase {
		t.Fatalf("directive = %q, want %q", result.Directive, DirectiveAdvancePhase)
	}
	if runCtx.CollectedData == nil || len(runCtx.CollectedData.PRs) != 1 {
		t.Fatalf("expected collected PR payload, got %#v", runCtx.CollectedData)
	}
	if got := runCtx.CollectedData.PRs[0].Number; got != 142 {
		t.Fatalf("PR number = %d, want 142", got)
	}
	if got := runCtx.CollectedData.Tags; len(got) != 1 || got[0].Name != "v1.0.0" || got[0].Sha != "abc123" || !got[0].CreatedAt.Equal(time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected tags: %#v", got)
	}
	if runner.threadStarts != 1 {
		t.Fatalf("thread starts = %d, want 1", runner.threadStarts)
	}
	if len(runner.prompts) != 1 || !strings.Contains(runner.prompts[0], "phase \"collect\"") {
		t.Fatalf("expected collect phase prompt, got %#v", runner.prompts)
	}
}

func TestAgentPhaseHandlerTimesOutWithTraceAndPartialOutput(t *testing.T) {
	t.Parallel()

	runner := &timeoutAgentRunner{}
	agent := NewCodexAgentRuntimeWithRunner(CodexAgentConfig{
		CWD:          "/repo",
		PhaseTimeout: time.Millisecond,
	}, runner)
	handler := &AgentPhaseHandler{Phase: PhaseCollect, Agent: agent}

	_, err := handler.Handle(context.Background(), &RunContext{
		Config:      &Config{},
		AnalysisCtx: &AnalysisContext{WorkingDirectory: t.TempDir()},
	})
	if err == nil {
		t.Fatal("expected timeout error")
	}
	for _, expected := range []string{
		"phase \"collect\" timed out",
		"item/completed",
		"onequery api",
		"partial before timeout",
	} {
		if !strings.Contains(err.Error(), expected) {
			t.Fatalf("timeout error missing %q: %v", expected, err)
		}
	}
}

func TestAgentPhaseHandlerAdvancesScoreWhenBoundedAnalyticsUnavailable(t *testing.T) {
	t.Parallel()

	runner := &scriptedAgentRunner{
		responses: []AgentTurnResult{{
			Status: "completed",
			Response: `{
				"directive":"retry",
				"output":"Score phase remains blocked because the analytics source does not expose working bounded metric reads through the available OneQuery API paths.",
				"scored_data":{"PRImpacts":[],"ContributorStats":[]},
				"error":"Advancing without a working bounded endpoint would require fabricating analytics."
			}`,
		}},
	}
	agent := NewCodexAgentRuntimeWithRunner(CodexAgentConfig{CWD: "/repo"}, runner)
	handler := &AgentPhaseHandler{Phase: PhaseScore, Agent: agent}
	runCtx := &RunContext{
		Config:      &Config{},
		AnalysisCtx: &AnalysisContext{WorkingDirectory: "/repo"},
		LinkedData: &LinkedData{
			Deployments: []Deployment{{PRNumber: 142, Marker: "merge", Source: "merge_time", DeployedAt: time.Date(2026, 4, 13, 12, 41, 7, 0, time.UTC)}},
		},
	}

	result, err := handler.Handle(context.Background(), runCtx)
	if err != nil {
		t.Fatalf("handle phase: %v", err)
	}
	if result.Directive != DirectiveAdvancePhase {
		t.Fatalf("directive = %q, want %q", result.Directive, DirectiveAdvancePhase)
	}
	if runCtx.ScoredData == nil {
		t.Fatal("expected scored data placeholder")
	}
	if len(runCtx.ScoredData.PRImpacts) != 0 || len(runCtx.ScoredData.ContributorStats) != 0 {
		t.Fatalf("expected empty unavailable scoring payload, got %#v", runCtx.ScoredData)
	}
}

type timeoutAgentRunner struct{}

func (r *timeoutAgentRunner) StartThread(context.Context) (string, error) {
	return "thr-timeout", nil
}

func (r *timeoutAgentRunner) RunTurn(ctx context.Context, _ string, _ string, onDelta func(string), onEvent func(AgentRuntimeEvent)) (AgentTurnResult, error) {
	if onEvent != nil {
		onEvent(AgentRuntimeEvent{Method: "item/completed", Summary: "type=toolCall command=onequery api --source wordbricks-github"})
	}
	if onDelta != nil {
		onDelta("partial before timeout")
	}
	<-ctx.Done()
	return AgentTurnResult{Status: "interrupted", Response: "partial before timeout"}, ctx.Err()
}

func (r *timeoutAgentRunner) Close() error {
	return nil
}

func TestBuildAgentPhasePromptIncludesRuntimeContract(t *testing.T) {
	t.Parallel()

	prompt, err := BuildAgentPhasePrompt(&RunContext{
		Config: &Config{
			OneQuery: OneQueryConfig{
				Org: "jay",
				Sources: OneQuerySources{
					GitHub:    "wordbricks-github",
					Analytics: "getgpt-prod",
				},
			},
		},
		AnalysisCtx: &AnalysisContext{WorkingDirectory: "/repo"},
	}, PhaseSourceCheck)
	if err != nil {
		t.Fatalf("build prompt: %v", err)
	}

	for _, expected := range []string{
		"Codex app-server",
		"onequery --org <org> auth whoami",
		"Return exactly one JSON object",
		"\"Org\": \"jay\"",
		"GitHubRepository",
		"source show",
		"\"Tags\": [{\"Name\": \"\"",
	} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("prompt missing %q:\n%s", expected, prompt)
		}
	}
	if strings.Contains(prompt, "non-queryable") {
		t.Fatalf("source check prompt should not wait on non-queryable sources:\n%s", prompt)
	}
}

func TestBuildAgentPhasePromptCollectUsesConfiguredGitHubRepository(t *testing.T) {
	t.Parallel()

	prompt, err := BuildAgentPhasePrompt(&RunContext{
		Config: &Config{
			OneQuery: OneQueryConfig{
				Org:              "jay",
				GitHubRepository: "wordbricks/wordbricks",
				Sources: OneQuerySources{
					GitHub:    "wordbricks-github",
					Analytics: "getgpt-prod",
				},
			},
		},
		AnalysisCtx: &AnalysisContext{WorkingDirectory: "/repo"},
	}, PhaseCollect)
	if err != nil {
		t.Fatalf("build prompt: %v", err)
	}

	for _, expected := range []string{
		"Config.OneQuery.GitHubRepository is set",
		"analyze exactly that GitHub repository full name",
		"do not infer the repository from the current worktree",
		"onequery api",
		"wordbricks/wordbricks",
	} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("collect prompt missing %q:\n%s", expected, prompt)
		}
	}
}

func TestBuildAgentPhasePromptIncludesDetailedScoreReasoningContract(t *testing.T) {
	t.Parallel()

	prompt, err := BuildAgentPhasePrompt(&RunContext{
		Config: &Config{
			OneQuery: OneQueryConfig{
				Org: "jay",
				Sources: OneQuerySources{
					GitHub:    "wordbricks-github",
					Analytics: "getgpt-prod",
				},
			},
		},
		AnalysisCtx: &AnalysisContext{WorkingDirectory: "/repo"},
	}, PhaseScore)
	if err != nil {
		t.Fatalf("build prompt: %v", err)
	}

	for _, expected := range []string{
		"detailed multi-line Reasoning string",
		"PrimaryMetric, BeforeValue, AfterValue, DeltaValue",
		"BeforeWindowStart, BeforeWindowEnd, AfterWindowStart, and AfterWindowEnd",
		"Preserve the existing OneQuery auth context",
		"why the primary metric was chosen",
		"before/after analysis windows used, with concrete dates",
		"why that movement implies the assigned impact score",
		"Do not fabricate analytics values",
		"First inspect the configured analytics source metadata and source API descriptor",
		"Do not assume the analytics provider",
		"do not use provider-specific endpoints",
		"auth/config access is unavailable inside the score turn",
		"return scored_data with empty PRImpacts and ContributorStats",
		"use directive advance_phase instead of wait or retry",
	} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("score prompt missing %q:\n%s", expected, prompt)
		}
	}
	for _, unexpected := range []string{
		"Amplitude",
		"amplitude",
		"onequery api --help",
		"--dry-run",
		"Discover the correct OneQuery CLI",
	} {
		if strings.Contains(prompt, unexpected) {
			t.Fatalf("score prompt should not include hard-coded provider example %q:\n%s", unexpected, prompt)
		}
	}
}

func TestBuildAgentPhasePromptLinkInfersWithoutUserConfirmation(t *testing.T) {
	t.Parallel()

	prompt, err := BuildAgentPhasePrompt(&RunContext{
		Config:      &Config{},
		AnalysisCtx: &AnalysisContext{WorkingDirectory: "/repo"},
		CollectedData: &CollectedData{PRs: []PR{{
			Number:   6054,
			Title:    "feat: replace ChannelTalk with floating feature request button",
			Branch:   "codex/replace-channeltalk-feature-request",
			MergedAt: time.Date(2026, 4, 13, 12, 41, 7, 0, time.UTC),
		}}},
	}, PhaseLink)
	if err != nil {
		t.Fatalf("build prompt: %v", err)
	}

	for _, expected := range []string{
		"Do not ask the user to resolve ordinary ambiguity",
		"use each PR's MergedAt timestamp as the deployment marker",
		"fallback_merge_time",
		"split or name feature groups from the actual PR content",
		"without blocking",
	} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("link prompt missing %q:\n%s", expected, prompt)
		}
	}
}

func TestParseAgentPhasePayloadAcceptsLegacyStringTags(t *testing.T) {
	t.Parallel()

	payload, err := ParseAgentPhasePayload(`{
		"directive":"advance_phase",
		"collected_data":{
			"Tags":["v1.2.3|2026-01-04T08:00:00Z"]
		}
	}`)
	if err != nil {
		t.Fatalf("parse payload: %v", err)
	}
	if payload.CollectedData == nil || len(payload.CollectedData.Tags) != 1 {
		t.Fatalf("expected one tag, got %#v", payload.CollectedData)
	}
	tag := payload.CollectedData.Tags[0]
	if tag.Name != "v1.2.3" || !tag.CreatedAt.Equal(time.Date(2026, 1, 4, 8, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected tag: %#v", tag)
	}
}

func TestParseAgentPhasePayloadAcceptsTagObjectsWithoutCreatedAt(t *testing.T) {
	t.Parallel()

	payload, err := ParseAgentPhasePayload(`{
		"directive":"advance_phase",
		"collected_data":{
			"Tags":[{"Name":"v0.91.1","Sha":"794b91c400023ef14e6642c343093caac818c528"}]
		}
	}`)
	if err != nil {
		t.Fatalf("parse payload: %v", err)
	}
	if payload.CollectedData == nil || len(payload.CollectedData.Tags) != 1 {
		t.Fatalf("expected one tag, got %#v", payload.CollectedData)
	}
	tag := payload.CollectedData.Tags[0]
	if tag.Name != "v0.91.1" || tag.Sha != "794b91c400023ef14e6642c343093caac818c528" || !tag.CreatedAt.IsZero() {
		t.Fatalf("unexpected tag: %#v", tag)
	}
}

func TestParseAgentPhasePayloadAcceptsAnalysisResultPRNumbers(t *testing.T) {
	t.Parallel()

	payload, err := ParseAgentPhasePayload(`{
		"directive":"advance_phase",
		"analysis_result":{"PRs":[6054]}
	}`)
	if err != nil {
		t.Fatalf("parse payload: %v", err)
	}
	if payload.AnalysisResult == nil || len(payload.AnalysisResult.PRs) != 1 {
		t.Fatalf("expected one analysis result PR, got %#v", payload.AnalysisResult)
	}
	if got := payload.AnalysisResult.PRs[0].Number; got != 6054 {
		t.Fatalf("PR number = %d, want 6054", got)
	}
}

func TestParseAgentPhasePayloadAcceptsScoreAliases(t *testing.T) {
	t.Parallel()

	payload, err := ParseAgentPhasePayload(`{
		"directive":"advance_phase",
		"scored_data":{
			"Contributors":[{"Author":"siisee11","PRCount":4,"MeasuredPRCount":1,"AverageMeasuredImpactScore":20}],
			"PRImpacts":[{"PRNumber":6054,"ImpactScore":20,"Confidence":"low","Reason":"Direct event increased","Before":0,"After":3,"Delta":3}]
		}
	}`)
	if err != nil {
		t.Fatalf("parse payload: %v", err)
	}
	if payload.ScoredData == nil {
		t.Fatal("expected scored data")
	}
	if len(payload.ScoredData.PRImpacts) != 1 {
		t.Fatalf("expected one PR impact, got %#v", payload.ScoredData.PRImpacts)
	}
	impact := payload.ScoredData.PRImpacts[0]
	if impact.Score != 20 || impact.Reasoning != "Direct event increased" || impact.AfterValue != 3 || impact.DeltaValue != 3 {
		t.Fatalf("unexpected impact alias decode: %#v", impact)
	}
	if len(payload.ScoredData.ContributorStats) != 1 {
		t.Fatalf("expected one contributor, got %#v", payload.ScoredData.ContributorStats)
	}
	contributor := payload.ScoredData.ContributorStats[0]
	if contributor.Author != "siisee11" || contributor.PRCount != 4 || contributor.AverageScore != 20 {
		t.Fatalf("unexpected contributor alias decode: %#v", contributor)
	}
}

func TestApplyAgentPhasePayloadDoesNotOverwriteExistingCollectedDataWithPartialAnalysisResult(t *testing.T) {
	t.Parallel()

	runCtx := &RunContext{
		CollectedData: &CollectedData{PRs: []PR{{Number: 6054, Title: "full collected PR"}}},
		LinkedData:    &LinkedData{Deployments: []Deployment{{PRNumber: 6054, Source: "existing"}}},
	}
	applyAgentPhasePayload(runCtx, AgentPhasePayload{
		AnalysisResult: &AnalysisResult{
			PRs:         []PR{{Number: 6054}},
			Deployments: []Deployment{{PRNumber: 6054, Source: "partial"}},
		},
	})

	if got := runCtx.CollectedData.PRs[0].Title; got != "full collected PR" {
		t.Fatalf("collected data was overwritten: %#v", runCtx.CollectedData.PRs[0])
	}
	if got := runCtx.LinkedData.Deployments[0].Source; got != "existing" {
		t.Fatalf("linked data was overwritten: %#v", runCtx.LinkedData.Deployments[0])
	}
}

func TestResolveAgentModel(t *testing.T) {
	tests := []struct {
		name           string
		configModel    string
		gitImpactModel string
		wtlModel       string
		want           string
	}{
		{
			name:           "git impact env wins",
			configModel:    "configured-model",
			gitImpactModel: "env-model",
			wtlModel:       "wtl-model",
			want:           "env-model",
		},
		{
			name:        "wtl model fallback",
			configModel: "configured-model",
			wtlModel:    "wtl-model",
			want:        "wtl-model",
		},
		{
			name:        "configured model fallback",
			configModel: "configured-model",
			want:        "configured-model",
		},
		{
			name: "default model",
			want: "gpt-5.4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(agentModelEnv, tt.gitImpactModel)
			t.Setenv(agentFallbackModelEnv, tt.wtlModel)

			if got := ResolveAgentModel(tt.configModel); got != tt.want {
				t.Fatalf("ResolveAgentModel(%q) = %q, want %q", tt.configModel, got, tt.want)
			}
		})
	}
}

func TestAgentHandlersIncludesAllPhases(t *testing.T) {
	t.Parallel()

	handlers := AgentHandlers(NewCodexAgentRuntimeWithRunner(CodexAgentConfig{}, &scriptedAgentRunner{}))
	for _, phase := range phaseOrder {
		if handlers[phase] == nil {
			t.Fatalf("missing agent handler for phase %q", phase)
		}
	}
	if _, ok := handlers[PhaseSourceCheck].(*SourceCheckHandler); !ok {
		t.Fatalf("source check should use local SourceCheckHandler, got %T", handlers[PhaseSourceCheck])
	}
	if _, ok := handlers[PhaseCollect].(*CollectHandler); !ok {
		t.Fatalf("collect should use local CollectHandler, got %T", handlers[PhaseCollect])
	}
	if _, ok := handlers[PhaseLink].(*LinkHandler); !ok {
		t.Fatalf("link should use local LinkHandler, got %T", handlers[PhaseLink])
	}
	if _, ok := handlers[PhaseScore].(*AgentPhaseHandler); !ok {
		t.Fatalf("score should use AgentPhaseHandler, got %T", handlers[PhaseScore])
	}
	if _, ok := handlers[PhaseReport].(*ReportHandler); !ok {
		t.Fatalf("report should use local ReportHandler, got %T", handlers[PhaseReport])
	}
}
