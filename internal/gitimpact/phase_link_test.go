package gitimpact

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestInferDeployment_PrefersReleaseOverTag(t *testing.T) {
	t.Parallel()

	mergedAt := time.Date(2026, 2, 10, 10, 0, 0, 0, time.UTC)
	pr := PR{Number: 42, MergedAt: mergedAt}

	deployment, ok := inferDeployment(pr, []Release{
		{TagName: "v2.0.1", PublishedAt: mergedAt.Add(4 * time.Hour)},
	}, []Tag{
		newTag("v2.0.0", mergedAt.Add(30*time.Minute)),
	})
	if !ok {
		t.Fatal("expected deployment inference to succeed")
	}
	if deployment.Source != "release" {
		t.Fatalf("expected release source, got %q", deployment.Source)
	}
	if deployment.Marker != "v2.0.1" {
		t.Fatalf("expected release marker v2.0.1, got %q", deployment.Marker)
	}
	if !deployment.DeployedAt.Equal(mergedAt.Add(4 * time.Hour)) {
		t.Fatalf("unexpected deployed_at: %s", deployment.DeployedAt)
	}
}

func TestInferDeployment_UsesVersionTagWhenReleaseUnavailable(t *testing.T) {
	t.Parallel()

	mergedAt := time.Date(2026, 2, 10, 10, 0, 0, 0, time.UTC)
	pr := PR{Number: 77, MergedAt: mergedAt}

	tagCreatedAt := mergedAt.Add(2 * time.Hour)
	deployment, ok := inferDeployment(pr, []Release{
		{TagName: "v2.1.0", PublishedAt: mergedAt.Add(72 * time.Hour)},
	}, []Tag{
		newTag("build-20260210", tagCreatedAt),
		newTag("release-2.0.0", tagCreatedAt),
	})
	if !ok {
		t.Fatal("expected deployment inference to succeed")
	}
	if deployment.Source != "tag" {
		t.Fatalf("expected tag source, got %q", deployment.Source)
	}
	if deployment.Marker != "release-2.0.0" {
		t.Fatalf("expected version tag marker, got %q", deployment.Marker)
	}
	if !deployment.DeployedAt.Equal(tagCreatedAt) {
		t.Fatalf("expected deployed_at to equal tag timestamp, got %s", deployment.DeployedAt)
	}
}

func TestInferDeployment_FallsBackToMergeTime(t *testing.T) {
	t.Parallel()

	mergedAt := time.Date(2026, 2, 10, 10, 0, 0, 0, time.UTC)
	pr := PR{Number: 101, MergedAt: mergedAt}

	deployment, ok := inferDeployment(pr, []Release{
		{TagName: "v2.0.2", PublishedAt: mergedAt.Add(72 * time.Hour)},
	}, []Tag{
		newTag("v2.0.1", mergedAt.Add(72*time.Hour)),
		{Name: "v2.0.0"},
	})
	if !ok {
		t.Fatal("expected fallback inference to be considered successful")
	}
	if deployment.Source != "fallback_merge_time" {
		t.Fatalf("expected fallback_merge_time source, got %q", deployment.Source)
	}
	if !deployment.DeployedAt.Equal(mergedAt) {
		t.Fatalf("expected merge timestamp fallback, got %s", deployment.DeployedAt)
	}
}

func TestDetectAmbiguousDeployments(t *testing.T) {
	t.Parallel()

	releases := []Release{
		{Name: "Release A", TagName: "v1.0.0", PublishedAt: time.Date(2026, 2, 10, 10, 0, 0, 0, time.UTC)},
		{Name: "Release B", TagName: "v1.0.1", PublishedAt: time.Date(2026, 2, 10, 20, 0, 0, 0, time.UTC)},
	}
	prs := []PR{
		{Number: 1, MergedAt: time.Date(2026, 2, 10, 11, 0, 0, 0, time.UTC)},
		{Number: 2, MergedAt: time.Date(2026, 2, 10, 19, 0, 0, 0, time.UTC)},
	}

	items := detectAmbiguousDeployments(prs, releases)
	if len(items) != 2 {
		t.Fatalf("expected 2 ambiguous deployments, got %d", len(items))
	}
	for _, item := range items {
		if !reflect.DeepEqual(item.Options, []string{"v1.0.0", "v1.0.1"}) {
			t.Fatalf("unexpected options for PR #%d: %#v", item.PRNumber, item.Options)
		}
	}
}

func TestLinkHandlerHandle_AdvancePhaseAndPopulateLinkedData(t *testing.T) {
	t.Parallel()

	mergedA := time.Date(2026, 2, 10, 10, 0, 0, 0, time.UTC)
	mergedB := time.Date(2026, 2, 11, 10, 0, 0, 0, time.UTC)
	releaseAt := mergedA.Add(6 * time.Hour)
	tagAt := mergedB.Add(2 * time.Hour)

	runCtx := &RunContext{
		CollectedData: &CollectedData{
			PRs: []PR{
				{
					Number:   10,
					MergedAt: mergedA,
					Branch:   "feature/checkout",
					Labels:   []string{"feature/checkout"},
				},
				{
					Number:   11,
					MergedAt: mergedB,
					Branch:   "feature/onboarding",
					Labels:   []string{"feature/onboarding"},
				},
			},
			Releases: []Release{
				{TagName: "v3.0.0", PublishedAt: releaseAt},
			},
			Tags: []Tag{
				newTag("release-3.1.0", tagAt),
			},
		},
	}

	handler := &LinkHandler{}
	result, err := handler.Handle(context.Background(), runCtx)
	if err != nil {
		t.Fatalf("handle returned error: %v", err)
	}
	if result == nil || result.Directive != DirectiveAdvancePhase {
		t.Fatalf("expected advance_phase, got %+v", result)
	}
	if runCtx.LinkedData == nil {
		t.Fatal("expected linked data to be populated")
	}
	if len(runCtx.LinkedData.Deployments) != 2 {
		t.Fatalf("expected 2 deployments, got %d", len(runCtx.LinkedData.Deployments))
	}
	if len(runCtx.LinkedData.FeatureGroups) != 2 {
		t.Fatalf("expected 2 feature groups, got %d", len(runCtx.LinkedData.FeatureGroups))
	}
}

func TestLinkHandlerHandle_RecordsAmbiguityWithoutWaiting(t *testing.T) {
	t.Parallel()

	releases := []Release{
		{TagName: "v4.0.0", PublishedAt: time.Date(2026, 2, 10, 10, 0, 0, 0, time.UTC)},
		{TagName: "v4.0.1", PublishedAt: time.Date(2026, 2, 10, 16, 0, 0, 0, time.UTC)},
	}
	runCtx := &RunContext{
		CollectedData: &CollectedData{
			PRs: []PR{
				{Number: 20, MergedAt: time.Date(2026, 2, 10, 11, 0, 0, 0, time.UTC)},
				{Number: 21, MergedAt: time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC)},
			},
			Releases: releases,
		},
	}

	handler := &LinkHandler{}
	result, err := handler.Handle(context.Background(), runCtx)
	if err != nil {
		t.Fatalf("handle returned error: %v", err)
	}
	if result == nil || result.Directive != DirectiveAdvancePhase {
		t.Fatalf("expected advance_phase directive, got %+v", result)
	}
	if !strings.Contains(result.Output, "AmbiguousItems") {
		t.Fatalf("unexpected output: %q", result.Output)
	}
	if runCtx.LinkedData == nil || len(runCtx.LinkedData.AmbiguousItems) != 2 {
		t.Fatalf("expected ambiguity to be retained in linked data, got %#v", runCtx.LinkedData)
	}
}

func TestProposeFeatureGroupsInfersFromGenericBranchContent(t *testing.T) {
	t.Parallel()

	groups := proposeFeatureGroups([]PR{
		{
			Number: 6054,
			Title:  "feat: replace ChannelTalk with floating feature request button",
			Branch: "codex/replace-channeltalk-feature-request",
		},
	})
	if len(groups) != 1 {
		t.Fatalf("expected one inferred group, got %#v", groups)
	}
	if groups[0].Name != "replace_channeltalk_feature_request" {
		t.Fatalf("unexpected inferred group name: %q", groups[0].Name)
	}
	if !reflect.DeepEqual(groups[0].PRNumbers, []int{6054}) {
		t.Fatalf("unexpected PR numbers: %#v", groups[0].PRNumbers)
	}
}

func TestIsVersionTag(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		expected bool
	}{
		{name: "v1.2.3", expected: true},
		{name: "release-2026-02", expected: true},
		{name: "hotfix-1", expected: false},
		{name: "build-999", expected: false},
	}

	for _, tc := range cases {
		if got := isVersionTag(tc.name); got != tc.expected {
			t.Fatalf("isVersionTag(%q) = %t, want %t", tc.name, got, tc.expected)
		}
	}
}
