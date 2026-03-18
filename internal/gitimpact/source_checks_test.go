package gitimpact

import (
	"context"
	"strings"
	"testing"
)

func TestCheckRequiredSources_AllCapabilitiesPresent(t *testing.T) {
	t.Parallel()

	cfg := defaultConfig()
	cfg.Velen.Org = "impactable"
	cfg.Velen.Sources.GitHub = "github-main"
	cfg.Velen.Sources.Warehouse = "prod-warehouse"
	cfg.Velen.Sources.Analytics = "amplitude-prod"

	checks, summary, meta, err := checkRequiredSources(context.Background(), cfg, []string{"github", "warehouse", "analytics"}, fakeVelenClient{
		identity:   VelenIdentity{Handle: "ci-user"},
		currentOrg: "impactable",
		sources: []VelenSource{
			{Key: "github-main", Provider: "github", SupportsQuery: true},
			{Key: "prod-warehouse", Provider: "bigquery", SupportsQuery: true},
			{Key: "amplitude-prod", Provider: "amplitude", SupportsQuery: true},
		},
		showByKey: map[string]VelenSource{
			"github-main":    {Key: "github-main", Provider: "github", SupportsQuery: true},
			"prod-warehouse": {Key: "prod-warehouse", Provider: "bigquery", SupportsQuery: true},
			"amplitude-prod": {Key: "amplitude-prod", Provider: "amplitude", SupportsQuery: true},
		},
	})
	if err != nil {
		t.Fatalf("checkRequiredSources returned error: %v", err)
	}
	if summary.Ready != true {
		t.Fatalf("expected ready=true summary: %#v", summary)
	}
	if summary.OK != 3 || summary.Missing != 0 || summary.Failed != 0 {
		t.Fatalf("unexpected summary: %#v", summary)
	}
	if len(checks) != 3 {
		t.Fatalf("expected 3 checks, got %d", len(checks))
	}
	if meta["discovered_sources"] != 3 {
		t.Fatalf("expected discovered_sources=3, got %#v", meta["discovered_sources"])
	}
}

func TestCheckRequiredSources_FlagsMissingAndUnsupportedSources(t *testing.T) {
	t.Parallel()

	cfg := defaultConfig()
	cfg.Velen.Org = "impactable"
	cfg.Velen.Sources.GitHub = "github-main"
	cfg.Velen.Sources.Warehouse = "prod-warehouse"
	cfg.Velen.Sources.Analytics = "amplitude-prod"

	checks, summary, _, err := checkRequiredSources(context.Background(), cfg, []string{"github", "warehouse", "analytics"}, fakeVelenClient{
		identity:   VelenIdentity{Handle: "ci-user"},
		currentOrg: "impactable",
		sources: []VelenSource{
			{Key: "github-main", Provider: "github", SupportsQuery: true},
			{Key: "prod-warehouse", Provider: "bigquery", SupportsQuery: false},
		},
		showByKey: map[string]VelenSource{
			"github-main":    {Key: "github-main", Provider: "github", SupportsQuery: true},
			"prod-warehouse": {Key: "prod-warehouse", Provider: "bigquery", SupportsQuery: false},
		},
	})
	if err != nil {
		t.Fatalf("checkRequiredSources returned error: %v", err)
	}
	if summary.Ready {
		t.Fatalf("expected ready=false summary: %#v", summary)
	}
	if summary.OK != 1 || summary.Missing != 1 || summary.Failed != 1 {
		t.Fatalf("unexpected summary: %#v", summary)
	}
	if len(checks) != 3 {
		t.Fatalf("expected 3 checks, got %d", len(checks))
	}
	statusByRole := map[string]string{}
	for _, check := range checks {
		statusByRole[check.Role] = check.Status
	}
	if statusByRole["github"] != "ok" {
		t.Fatalf("expected github status ok, got %q", statusByRole["github"])
	}
	if statusByRole["warehouse"] != "failed" {
		t.Fatalf("expected warehouse status failed, got %q", statusByRole["warehouse"])
	}
	if statusByRole["analytics"] != "missing" {
		t.Fatalf("expected analytics status missing, got %q", statusByRole["analytics"])
	}
}

func TestCheckRequiredSources_OrgMismatchReducesReadiness(t *testing.T) {
	t.Parallel()

	cfg := defaultConfig()
	cfg.Velen.Org = "impactable"
	cfg.Velen.Sources.GitHub = "github-main"
	cfg.Velen.Sources.Warehouse = "prod-warehouse"
	cfg.Velen.Sources.Analytics = "amplitude-prod"

	_, summary, meta, err := checkRequiredSources(context.Background(), cfg, []string{"github"}, fakeVelenClient{
		identity:   VelenIdentity{Handle: "ci-user"},
		currentOrg: "another-org",
		sources: []VelenSource{
			{Key: "github-main", Provider: "github", SupportsQuery: true},
		},
		showByKey: map[string]VelenSource{
			"github-main": {Key: "github-main", Provider: "github", SupportsQuery: true},
		},
	})
	if err != nil {
		t.Fatalf("checkRequiredSources returned error: %v", err)
	}
	if summary.Ready {
		t.Fatalf("expected ready=false due to org mismatch")
	}
	if summary.Failed != 1 {
		t.Fatalf("expected summary failed count to include org mismatch, got %#v", summary)
	}
	orgMeta, _ := meta["org"].(map[string]any)
	if orgMeta["match"] != false {
		t.Fatalf("expected org match false, got %#v", orgMeta["match"])
	}
}

func TestCheckRequiredSources_UsesSourceDetailForCapabilityValidation(t *testing.T) {
	t.Parallel()

	cfg := defaultConfig()
	cfg.Velen.Org = "impactable"
	cfg.Velen.Sources.GitHub = "github-main"
	cfg.Velen.Sources.Warehouse = "prod-warehouse"
	cfg.Velen.Sources.Analytics = "amplitude-prod"

	checks, summary, _, err := checkRequiredSources(context.Background(), cfg, []string{"github"}, fakeVelenClient{
		identity:   VelenIdentity{Handle: "ci-user"},
		currentOrg: "impactable",
		sources: []VelenSource{
			// Simulate list payload with missing QUERY capability metadata.
			{Key: "github-main", Provider: "github", SupportsQuery: false},
		},
		showByKey: map[string]VelenSource{
			"github-main": {Key: "github-main", Provider: "github", SupportsQuery: true},
		},
	})
	if err != nil {
		t.Fatalf("checkRequiredSources returned error: %v", err)
	}
	if summary.Ready != true {
		t.Fatalf("expected ready=true summary: %#v", summary)
	}
	if len(checks) != 1 {
		t.Fatalf("expected one check result, got %d", len(checks))
	}
	if checks[0].Status != "ok" {
		t.Fatalf("expected status ok, got %q", checks[0].Status)
	}
	if !checks[0].QuerySupported {
		t.Fatalf("expected query_supported=true from source detail")
	}
}

func TestCheckRequiredSources_FailsWhenListedSourceDetailLookupFails(t *testing.T) {
	t.Parallel()

	cfg := defaultConfig()
	cfg.Velen.Org = "impactable"
	cfg.Velen.Sources.GitHub = "github-main"
	cfg.Velen.Sources.Warehouse = "prod-warehouse"
	cfg.Velen.Sources.Analytics = "amplitude-prod"

	checks, summary, _, err := checkRequiredSources(context.Background(), cfg, []string{"github"}, fakeVelenClient{
		identity:   VelenIdentity{Handle: "ci-user"},
		currentOrg: "impactable",
		sources: []VelenSource{
			{Key: "github-main", Provider: "github", SupportsQuery: true},
		},
		showByKey: map[string]VelenSource{},
	})
	if err != nil {
		t.Fatalf("checkRequiredSources returned error: %v", err)
	}
	if summary.Ready {
		t.Fatalf("expected ready=false when source detail lookup fails")
	}
	if summary.Failed != 1 {
		t.Fatalf("expected failed=1, got %#v", summary)
	}
	if len(checks) != 1 {
		t.Fatalf("expected one check result, got %d", len(checks))
	}
	if checks[0].Status != "failed" {
		t.Fatalf("expected status failed, got %q", checks[0].Status)
	}
	if !strings.Contains(checks[0].Message, "unable to verify source detail") {
		t.Fatalf("unexpected check message: %q", checks[0].Message)
	}
}

func TestCheckRequiredSources_PropagatesWhoAmIError(t *testing.T) {
	t.Parallel()

	cfg := defaultConfig()
	cfg.Velen.Org = "impactable"
	cfg.Velen.Sources.GitHub = "github-main"
	cfg.Velen.Sources.Warehouse = "prod-warehouse"
	cfg.Velen.Sources.Analytics = "amplitude-prod"

	_, _, _, err := checkRequiredSources(context.Background(), cfg, []string{"github"}, fakeVelenClient{
		whoAmIErr: assertErr("boom"),
	})
	if err == nil {
		t.Fatalf("expected checkRequiredSources error")
	}
	if !strings.Contains(err.Error(), "velen auth whoami failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}
