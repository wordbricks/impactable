package gitimpact

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
)

type mockVelenClient struct {
	authErr error
	orgErr  error
	listErr error

	orgName string
	sources []Source
	calls   []string
}

func (m *mockVelenClient) AuthWhoAmI(context.Context) error {
	m.calls = append(m.calls, "auth whoami")
	return m.authErr
}

func (m *mockVelenClient) OrgCurrent(context.Context) (string, error) {
	m.calls = append(m.calls, "org current")
	if m.orgErr != nil {
		return "", m.orgErr
	}
	return m.orgName, nil
}

func (m *mockVelenClient) SourceList(context.Context) ([]Source, error) {
	m.calls = append(m.calls, "source list")
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.sources, nil
}

func TestCheckSourcesWithClientSuccess(t *testing.T) {
	cfg := &Config{}
	client := &mockVelenClient{
		orgName: "acme",
		sources: []Source{
			{Key: "gh-main", ProviderType: "github", Capabilities: []string{"QUERY"}},
			{Key: "amp-prod", ProviderType: "amplitude", Capabilities: []string{"QUERY"}},
		},
	}

	result, err := CheckSourcesWithClient(context.Background(), cfg, client)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatalf("expected result")
	}
	if !result.GitHubOK {
		t.Fatalf("expected github check to pass")
	}
	if !result.AnalyticsOK {
		t.Fatalf("expected analytics check to pass")
	}
	if result.OrgName != "acme" {
		t.Fatalf("expected org acme, got %q", result.OrgName)
	}
	if result.GitHubSource == nil || result.GitHubSource.Key != "gh-main" {
		t.Fatalf("expected github source gh-main, got %#v", result.GitHubSource)
	}
	if result.AnalyticsSource == nil || result.AnalyticsSource.Key != "amp-prod" {
		t.Fatalf("expected analytics source amp-prod, got %#v", result.AnalyticsSource)
	}

	expectedCalls := []string{"auth whoami", "org current", "source list"}
	if !reflect.DeepEqual(expectedCalls, client.calls) {
		t.Fatalf("unexpected call order: got %v want %v", client.calls, expectedCalls)
	}
}

func TestCheckSourcesWithClientMissingAnalytics(t *testing.T) {
	cfg := &Config{}
	client := &mockVelenClient{
		orgName: "acme",
		sources: []Source{
			{Key: "gh-main", ProviderType: "github", Capabilities: []string{"QUERY"}},
		},
	}

	result, err := CheckSourcesWithClient(context.Background(), cfg, client)
	if err == nil {
		t.Fatalf("expected error for missing analytics source")
	}
	if result == nil {
		t.Fatalf("expected result")
	}
	if !result.GitHubOK {
		t.Fatalf("expected github to remain ok")
	}
	if result.AnalyticsSource != nil {
		t.Fatalf("expected no analytics source, got %#v", result.AnalyticsSource)
	}
	if !strings.Contains(result.Error, "analytics source not found") {
		t.Fatalf("expected analytics missing message, got %q", result.Error)
	}
}

func TestCheckSourcesWithClientQueryCapabilityFailure(t *testing.T) {
	cfg := &Config{}
	client := &mockVelenClient{
		orgName: "acme",
		sources: []Source{
			{Key: "gh-main", ProviderType: "github", Capabilities: []string{"READ"}},
			{Key: "mxp", ProviderType: "mixpanel", Capabilities: []string{"QUERY"}},
		},
	}

	result, err := CheckSourcesWithClient(context.Background(), cfg, client)
	if err == nil {
		t.Fatalf("expected error for github without QUERY")
	}
	if result == nil {
		t.Fatalf("expected result")
	}
	if result.GitHubOK {
		t.Fatalf("expected github check to fail")
	}
	if !result.AnalyticsOK {
		t.Fatalf("expected analytics check to pass")
	}
	if !strings.Contains(result.Error, "github source does not support QUERY") {
		t.Fatalf("expected query capability message, got %q", result.Error)
	}
}

func TestCheckSourcesWithClientAuthFailure(t *testing.T) {
	cfg := &Config{}
	client := &mockVelenClient{authErr: errors.New("not logged in")}

	result, err := CheckSourcesWithClient(context.Background(), cfg, client)
	if err == nil {
		t.Fatalf("expected auth error")
	}
	if result == nil {
		t.Fatalf("expected result")
	}
	if !strings.Contains(result.Error, "velen auth whoami failed") {
		t.Fatalf("unexpected error message: %q", result.Error)
	}
	expectedCalls := []string{"auth whoami"}
	if !reflect.DeepEqual(expectedCalls, client.calls) {
		t.Fatalf("unexpected call order: got %v want %v", client.calls, expectedCalls)
	}
}
