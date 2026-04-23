package gitimpact

import (
	"context"
	"strings"
	"testing"
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

func (r *scriptedAgentRunner) RunTurn(_ context.Context, _ string, prompt string, _ func(string)) (AgentTurnResult, error) {
	r.prompts = append(r.prompts, prompt)
	index := len(r.prompts) - 1
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
					"Tags":["v1.0.0|2026-02-15T00:00:00Z"],
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
	if runner.threadStarts != 1 {
		t.Fatalf("thread starts = %d, want 1", runner.threadStarts)
	}
	if len(runner.prompts) != 1 || !strings.Contains(runner.prompts[0], "phase \"collect\"") {
		t.Fatalf("expected collect phase prompt, got %#v", runner.prompts)
	}
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
		"source show",
	} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("prompt missing %q:\n%s", expected, prompt)
		}
	}
	if strings.Contains(prompt, "non-queryable") {
		t.Fatalf("source check prompt should not wait on non-queryable sources:\n%s", prompt)
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
}
