package gitimpact

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestSourceCheckHandlerHandle_AdvancesWhenSourcesReady(t *testing.T) {
	t.Parallel()

	handler := &SourceCheckHandler{
		CheckSources: func(context.Context, *OneQueryClient, *Config) (*SourceCheckResult, error) {
			return &SourceCheckResult{GitHubOK: true, AnalyticsOK: true}, nil
		},
	}

	result, err := handler.Handle(context.Background(), &RunContext{
		OneQueryClient: &OneQueryClient{},
		Config:         &Config{},
	})
	if err != nil {
		t.Fatalf("handle returned error: %v", err)
	}
	if result == nil || result.Directive != DirectiveAdvancePhase {
		t.Fatalf("expected advance directive, got %+v", result)
	}
}

func TestSourceCheckHandlerHandle_WaitsWhenSourcesNotReady(t *testing.T) {
	t.Parallel()

	handler := &SourceCheckHandler{
		CheckSources: func(context.Context, *OneQueryClient, *Config) (*SourceCheckResult, error) {
			return &SourceCheckResult{
				GitHubOK:    false,
				AnalyticsOK: true,
				Errors:      []string{"github source not found"},
			}, nil
		},
	}

	result, err := handler.Handle(context.Background(), &RunContext{
		OneQueryClient: &OneQueryClient{},
		Config:         &Config{},
	})
	if err != nil {
		t.Fatalf("handle returned error: %v", err)
	}
	if result == nil || result.Directive != DirectiveWait {
		t.Fatalf("expected wait directive, got %+v", result)
	}
	if !strings.Contains(result.WaitMessage, "github source not found") {
		t.Fatalf("expected wait message to contain source error, got %q", result.WaitMessage)
	}
	if !strings.Contains(result.WaitMessage, "(y/n)") {
		t.Fatalf("expected wait message to ask for y/n confirmation, got %q", result.WaitMessage)
	}
}

func TestSourceCheckHandlerHandle_UsesWaitResponseYToAdvance(t *testing.T) {
	t.Parallel()

	called := false
	handler := &SourceCheckHandler{
		CheckSources: func(context.Context, *OneQueryClient, *Config) (*SourceCheckResult, error) {
			called = true
			return &SourceCheckResult{}, nil
		},
	}

	result, err := handler.Handle(context.Background(), &RunContext{
		AnalysisCtx: &AnalysisContext{LastWaitResponse: " y "},
	})
	if err != nil {
		t.Fatalf("handle returned error: %v", err)
	}
	if result == nil || result.Directive != DirectiveAdvancePhase {
		t.Fatalf("expected advance directive, got %+v", result)
	}
	if called {
		t.Fatalf("did not expect CheckSources to run when wait response already exists")
	}
}

func TestSourceCheckHandlerHandle_UsesWaitResponseNToError(t *testing.T) {
	t.Parallel()

	handler := &SourceCheckHandler{}
	_, err := handler.Handle(context.Background(), &RunContext{
		AnalysisCtx: &AnalysisContext{LastWaitResponse: "n"},
	})
	if err == nil {
		t.Fatal("expected error for wait response n")
	}
	if !strings.Contains(err.Error(), "aborted") {
		t.Fatalf("expected abort error, got %v", err)
	}
}

func TestSourceCheckHandlerHandle_PropagatesCheckSourcesError(t *testing.T) {
	t.Parallel()

	handler := &SourceCheckHandler{
		CheckSources: func(context.Context, *OneQueryClient, *Config) (*SourceCheckResult, error) {
			return nil, errors.New("boom")
		},
	}
	_, err := handler.Handle(context.Background(), &RunContext{
		OneQueryClient: &OneQueryClient{},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Fatalf("expected wrapped check-sources error, got %v", err)
	}
}
