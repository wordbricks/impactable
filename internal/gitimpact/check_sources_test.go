package gitimpact

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestCheckSources_TableDriven(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                  string
		cfg                   Config
		sources               []Source
		wantGitHubKey         string
		wantAnalyticsKey      string
		wantGitHubOK          bool
		wantAnalyticsOK       bool
		wantOrgName           string
		wantErrorContains     []string
		wantNoErrorSubstrings []string
	}{
		{
			name: "provider type matching succeeds",
			sources: []Source{
				{Key: "github-main", ProviderType: "GitHub", Capabilities: []string{"QUERY", "SYNC"}},
				{Key: "amplitude-prod", ProviderType: "Amplitude", Capabilities: []string{"QUERY"}},
			},
			wantGitHubKey:         "github-main",
			wantAnalyticsKey:      "amplitude-prod",
			wantGitHubOK:          true,
			wantAnalyticsOK:       true,
			wantOrgName:           "Impactable",
			wantNoErrorSubstrings: []string{"not found", "does not support QUERY"},
		},
		{
			name: "analytics detected but query unsupported",
			sources: []Source{
				{Key: "gh-enterprise", ProviderType: "github-enterprise", Capabilities: []string{"QUERY"}},
				{Key: "mixpanel-main", ProviderType: "MIXPANEL", Capabilities: []string{"SYNC"}},
			},
			wantGitHubKey:     "gh-enterprise",
			wantAnalyticsKey:  "mixpanel-main",
			wantGitHubOK:      true,
			wantAnalyticsOK:   false,
			wantOrgName:       "Impactable",
			wantErrorContains: []string{"analytics source \"mixpanel-main\" does not support QUERY"},
		},
		{
			name: "provider and query fields match current velen payloads",
			cfg: Config{
				Velen: VelenConfig{
					Sources: VelenSources{
						GitHub:    "github-main",
						Analytics: "amplitude-prod",
					},
				},
			},
			sources: []Source{
				{Name: "getgpt-repo", Provider: "github", Query: "yes", Status: "active"},
				{Name: "getgpt-ga", Provider: "ga", Query: true, Status: "active"},
			},
			wantGitHubKey:         "getgpt-repo",
			wantAnalyticsKey:      "getgpt-ga",
			wantGitHubOK:          true,
			wantAnalyticsOK:       true,
			wantOrgName:           "Impactable",
			wantNoErrorSubstrings: []string{"not found", "does not support QUERY"},
		},
		{
			name: "fallback to configured keys when provider types do not match",
			cfg: Config{
				Velen: VelenConfig{
					Sources: VelenSources{
						GitHub:    "gh-fallback",
						Analytics: "analytics-fallback",
					},
				},
			},
			sources: []Source{
				{Key: "gh-fallback", ProviderType: "custom", Capabilities: []string{"QUERY"}},
				{Key: "analytics-fallback", ProviderType: "internal", Capabilities: []string{"QUERY"}},
			},
			wantGitHubKey:         "gh-fallback",
			wantAnalyticsKey:      "analytics-fallback",
			wantGitHubOK:          true,
			wantAnalyticsOK:       true,
			wantOrgName:           "Impactable",
			wantNoErrorSubstrings: []string{"not found", "does not support QUERY"},
		},
		{
			name: "missing required sources surfaces errors",
			cfg: Config{
				Velen: VelenConfig{
					Sources: VelenSources{
						GitHub:    "missing-github",
						Analytics: "missing-analytics",
					},
				},
			},
			sources: []Source{
				{Key: "warehouse-main", ProviderType: "warehouse", Capabilities: []string{"QUERY"}},
			},
			wantGitHubOK:      false,
			wantAnalyticsOK:   false,
			wantOrgName:       "Impactable",
			wantErrorContains: []string{"github source not found", "analytics source not found"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client := newCheckSourcesHelperClient(t, checkSourcesHelperPayload{
				WhoAmI: WhoAmIResult{
					Email: "agent@example.com",
					Org:   "impactable",
				},
				Org: OrgResult{
					Slug: "impactable",
					Name: "Impactable",
				},
				Sources: tt.sources,
			})

			result, err := CheckSources(context.Background(), client, &tt.cfg)
			if err != nil {
				t.Fatalf("CheckSources returned error: %v", err)
			}

			if got := sourceKey(result.GitHubSource); got != tt.wantGitHubKey {
				t.Fatalf("unexpected github source key: got %q, want %q", got, tt.wantGitHubKey)
			}
			if got := sourceKey(result.AnalyticsSource); got != tt.wantAnalyticsKey {
				t.Fatalf("unexpected analytics source key: got %q, want %q", got, tt.wantAnalyticsKey)
			}
			if result.GitHubOK != tt.wantGitHubOK {
				t.Fatalf("unexpected github ok: got %t, want %t", result.GitHubOK, tt.wantGitHubOK)
			}
			if result.AnalyticsOK != tt.wantAnalyticsOK {
				t.Fatalf("unexpected analytics ok: got %t, want %t", result.AnalyticsOK, tt.wantAnalyticsOK)
			}
			if result.OrgName != tt.wantOrgName {
				t.Fatalf("unexpected org name: got %q, want %q", result.OrgName, tt.wantOrgName)
			}
			if tt.wantGitHubKey != "" && tt.cfg.Velen.Sources.GitHub != "" && tt.cfg.Velen.Sources.GitHub != tt.wantGitHubKey {
				t.Fatalf("expected config github source to be updated to %q, got %q", tt.wantGitHubKey, tt.cfg.Velen.Sources.GitHub)
			}
			if tt.wantAnalyticsKey != "" && tt.cfg.Velen.Sources.Analytics != "" && tt.cfg.Velen.Sources.Analytics != tt.wantAnalyticsKey {
				t.Fatalf("expected config analytics source to be updated to %q, got %q", tt.wantAnalyticsKey, tt.cfg.Velen.Sources.Analytics)
			}

			for _, expected := range tt.wantErrorContains {
				if !containsString(result.Errors, expected) {
					t.Fatalf("expected errors to contain %q, got %#v", expected, result.Errors)
				}
			}
			for _, unexpected := range tt.wantNoErrorSubstrings {
				for _, msg := range result.Errors {
					if strings.Contains(msg, unexpected) {
						t.Fatalf("did not expect errors to contain substring %q, got %#v", unexpected, result.Errors)
					}
				}
			}
		})
	}
}

func TestCheckSources_ReturnsCommandErrors(t *testing.T) {
	t.Parallel()

	client := newCheckSourcesHelperClient(t, checkSourcesHelperPayload{
		FailCommand: "velen org current",
		FailMessage: "boom",
	})

	_, err := CheckSources(context.Background(), client, &Config{})
	if err == nil {
		t.Fatalf("expected command failure error")
	}
	if !strings.Contains(err.Error(), "velen org current") {
		t.Fatalf("expected wrapped org current error, got %q", err.Error())
	}
}

type checkSourcesHelperPayload struct {
	WhoAmI      WhoAmIResult
	Org         OrgResult
	Sources     []Source
	FailCommand string
	FailMessage string
}

func newCheckSourcesHelperClient(t *testing.T, payload checkSourcesHelperPayload) *VelenClient {
	t.Helper()

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal helper payload: %v", err)
	}

	client := NewVelenClient(time.Second)
	client.cmdFactory = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		helperArgs := []string{"-test.run=TestCheckSourcesHelperProcess", "--", name}
		helperArgs = append(helperArgs, args...)
		cmd := exec.CommandContext(ctx, os.Args[0], helperArgs...)
		cmd.Env = append(os.Environ(),
			"GO_WANT_CHECK_SOURCES_HELPER_PROCESS=1",
			"CHECK_SOURCES_HELPER_PAYLOAD="+string(payloadBytes),
		)
		return cmd
	}
	return client
}

func TestCheckSourcesHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_CHECK_SOURCES_HELPER_PROCESS") != "1" {
		return
	}

	separator := -1
	for idx, arg := range os.Args {
		if arg == "--" {
			separator = idx
			break
		}
	}
	if separator == -1 || separator+1 >= len(os.Args) {
		_, _ = os.Stderr.WriteString("missing helper args")
		os.Exit(2)
	}

	var payload checkSourcesHelperPayload
	if err := json.Unmarshal([]byte(os.Getenv("CHECK_SOURCES_HELPER_PAYLOAD")), &payload); err != nil {
		_, _ = os.Stderr.WriteString("invalid helper payload")
		os.Exit(2)
	}

	args := os.Args[separator+1:]
	filteredArgs := make([]string, 0, len(args))
	for idx := 0; idx < len(args); idx++ {
		if args[idx] == "--output" && idx+1 < len(args) {
			idx++
			continue
		}
		filteredArgs = append(filteredArgs, args[idx])
	}

	cmdText := strings.Join(filteredArgs, " ")
	if payload.FailCommand != "" && payload.FailCommand == cmdText {
		_, _ = os.Stderr.WriteString(payload.FailMessage)
		os.Exit(7)
	}

	switch cmdText {
	case "velen auth whoami":
		_ = json.NewEncoder(os.Stdout).Encode(payload.WhoAmI)
		os.Exit(0)
	case "velen org current":
		_ = json.NewEncoder(os.Stdout).Encode(payload.Org)
		os.Exit(0)
	case "velen source list":
		_ = json.NewEncoder(os.Stdout).Encode(payload.Sources)
		os.Exit(0)
	default:
		_, _ = os.Stderr.WriteString("unknown command")
		os.Exit(2)
	}
}

func sourceKey(source *Source) string {
	if source == nil {
		return ""
	}
	return source.SourceKey()
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
