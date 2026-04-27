package gitimpact

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	linkInferenceWindow   = 48 * time.Hour
	ambiguousWindow       = 24 * time.Hour
	tagTimestampSeparator = "|"
)

// LinkHandler maps collected GitHub metadata to inferred deployments and feature groups.
type LinkHandler struct{}

// Handle infers deployment markers for collected PRs and advances to scoring when unambiguous.
func (h *LinkHandler) Handle(_ context.Context, runCtx *RunContext) (*TurnResult, error) {
	if runCtx == nil {
		return nil, fmt.Errorf("run context is required")
	}
	if runCtx.CollectedData == nil {
		return nil, fmt.Errorf("collected data is required")
	}

	prs := runCtx.CollectedData.PRs
	releases := runCtx.CollectedData.Releases
	tags := runCtx.CollectedData.Tags

	deployments := make([]Deployment, 0, len(prs))
	for _, pr := range prs {
		deployment, _ := inferDeployment(pr, releases, tags)
		deployments = append(deployments, deployment)
	}

	featureGroups := proposeFeatureGroups(prs)
	ambiguousItems := detectAmbiguousDeployments(prs, releases)

	runCtx.LinkedData = &LinkedData{
		Deployments:    deployments,
		FeatureGroups:  featureGroups,
		AmbiguousItems: ambiguousItems,
	}

	output := "Linked PRs using inferred deployment markers and feature groups."
	if len(ambiguousItems) > 0 {
		output = "Linked PRs without blocking; ambiguous release clusters were retained in AmbiguousItems."
	}
	return &TurnResult{Directive: DirectiveAdvancePhase, Output: output}, nil
}

func inferDeployment(pr PR, releases []Release, tags []Tag) (Deployment, bool) {
	deployment := Deployment{
		PRNumber:   pr.Number,
		Marker:     fmt.Sprintf("pr-%d-merge", pr.Number),
		Source:     "fallback_merge_time",
		DeployedAt: pr.MergedAt,
	}
	if pr.MergedAt.IsZero() {
		return deployment, false
	}

	if release, ok := nearestReleaseAfter(pr.MergedAt, releases); ok {
		deployment.Marker = releaseMarker(release)
		deployment.Source = "release"
		deployment.DeployedAt = release.PublishedAt
		return deployment, true
	}

	if tagName, createdAt, ok := nearestVersionTagAfter(pr.MergedAt, tags); ok {
		deployment.Marker = tagName
		deployment.Source = "tag"
		deployment.DeployedAt = createdAt
		return deployment, true
	}

	return deployment, true
}

func proposeFeatureGroups(prs []PR) []FeatureGroup {
	type groupBucket struct {
		Name      string
		PRNumbers map[int]struct{}
	}

	groups := map[string]*groupBucket{}
	addToGroup := func(name string, prNumber int, requireFeaturePrefix bool) bool {
		trimmed := strings.TrimSpace(name)
		if trimmed == "" {
			return false
		}
		if requireFeaturePrefix && !strings.HasPrefix(strings.ToLower(trimmed), "feature/") {
			return false
		}

		key := strings.ToLower(normalizeFeatureGroupName(trimmed))
		if key == "" {
			return false
		}
		bucket, ok := groups[key]
		if !ok {
			bucket = &groupBucket{
				Name:      key,
				PRNumbers: map[int]struct{}{},
			}
			groups[key] = bucket
		}
		bucket.PRNumbers[prNumber] = struct{}{}
		return true
	}

	for _, pr := range prs {
		added := false
		for _, label := range pr.Labels {
			if addToGroup(label, pr.Number, true) {
				added = true
			}
		}
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(pr.Branch)), "feature/") && addToGroup(pr.Branch, pr.Number, true) {
			added = true
		}
		if !added {
			addToGroup(featureNameFromPR(pr), pr.Number, false)
		}
	}

	groupNames := make([]string, 0, len(groups))
	for key := range groups {
		groupNames = append(groupNames, key)
	}
	sort.Strings(groupNames)

	result := make([]FeatureGroup, 0, len(groups))
	for _, key := range groupNames {
		bucket := groups[key]
		prNumbers := make([]int, 0, len(bucket.PRNumbers))
		for number := range bucket.PRNumbers {
			prNumbers = append(prNumbers, number)
		}
		sort.Ints(prNumbers)
		result = append(result, FeatureGroup{
			Name:      bucket.Name,
			PRNumbers: prNumbers,
		})
	}
	return result
}

func featureNameFromPR(pr PR) string {
	if candidate := branchFeatureCandidate(pr.Branch); candidate != "" {
		return candidate
	}
	if candidate := titleFeatureCandidate(pr.Title); candidate != "" {
		return candidate
	}
	for _, file := range pr.ChangedFile {
		if candidate := fileFeatureCandidate(file); candidate != "" {
			return candidate
		}
	}
	if pr.Number > 0 {
		return fmt.Sprintf("pr_%d", pr.Number)
	}
	return "unknown_feature"
}

func branchFeatureCandidate(branch string) string {
	trimmed := strings.Trim(strings.ToLower(strings.TrimSpace(branch)), "/")
	if trimmed == "" {
		return ""
	}
	parts := strings.Split(trimmed, "/")
	if len(parts) > 1 {
		return strings.Join(parts[1:], "_")
	}
	return trimmed
}

func titleFeatureCandidate(title string) string {
	trimmed := strings.TrimSpace(title)
	if trimmed == "" {
		return ""
	}
	if idx := strings.Index(trimmed, ":"); idx >= 0 && idx+1 < len(trimmed) {
		trimmed = trimmed[idx+1:]
	}
	return trimmed
}

func fileFeatureCandidate(path string) string {
	trimmed := strings.Trim(strings.ToLower(strings.TrimSpace(path)), "/")
	if trimmed == "" {
		return ""
	}
	parts := strings.Split(trimmed, "/")
	if len(parts) == 1 {
		return strings.TrimSuffix(parts[0], ".ts")
	}
	if len(parts) >= 2 && parts[0] == "packages" {
		return strings.Join(parts[:minInt(3, len(parts))], "_")
	}
	return strings.Join(parts[:minInt(2, len(parts))], "_")
}

func normalizeFeatureGroupName(value string) string {
	var b strings.Builder
	lastUnderscore := false
	for _, r := range strings.ToLower(value) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastUnderscore = false
			continue
		}
		if !lastUnderscore {
			b.WriteByte('_')
			lastUnderscore = true
		}
	}
	return strings.Trim(b.String(), "_")
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func detectAmbiguousDeployments(prs []PR, releases []Release) []AmbiguousDeployment {
	if len(prs) < 2 || len(releases) < 2 {
		return nil
	}

	sortedReleases := append([]Release(nil), releases...)
	sort.Slice(sortedReleases, func(i, j int) bool {
		return sortedReleases[i].PublishedAt.Before(sortedReleases[j].PublishedAt)
	})

	ambiguousByPR := map[int]AmbiguousDeployment{}
	for start := 0; start < len(sortedReleases); {
		end := start + 1
		for end < len(sortedReleases) {
			if sortedReleases[end].PublishedAt.Sub(sortedReleases[end-1].PublishedAt) > ambiguousWindow {
				break
			}
			end++
		}

		cluster := sortedReleases[start:end]
		if len(cluster) > 1 {
			windowStart := cluster[0].PublishedAt
			windowEnd := cluster[len(cluster)-1].PublishedAt
			prNumbers := prsMergedWithinWindow(prs, windowStart, windowEnd)
			if len(prNumbers) > 1 {
				options := clusterReleaseOptions(cluster)
				for _, prNumber := range prNumbers {
					current, ok := ambiguousByPR[prNumber]
					if !ok {
						ambiguousByPR[prNumber] = AmbiguousDeployment{
							PRNumber: prNumber,
							Options:  append([]string(nil), options...),
							Reason:   "multiple releases were published within 24h while multiple PRs merged in that window",
						}
						continue
					}

					current.Options = mergeDistinctOptions(current.Options, options)
					ambiguousByPR[prNumber] = current
				}
			}
		}

		start = end
	}

	if len(ambiguousByPR) == 0 {
		return nil
	}

	result := make([]AmbiguousDeployment, 0, len(ambiguousByPR))
	for _, item := range ambiguousByPR {
		sort.Strings(item.Options)
		result = append(result, item)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].PRNumber < result[j].PRNumber
	})
	return result
}

func isVersionTag(name string) bool {
	trimmed := strings.ToLower(strings.TrimSpace(name))
	return strings.HasPrefix(trimmed, "v") || strings.HasPrefix(trimmed, "release-")
}

func nearestReleaseAfter(mergedAt time.Time, releases []Release) (Release, bool) {
	var chosen Release
	found := false
	for _, release := range releases {
		if release.PublishedAt.IsZero() || release.PublishedAt.Before(mergedAt) {
			continue
		}
		if release.PublishedAt.Sub(mergedAt) > linkInferenceWindow {
			continue
		}
		if !found || release.PublishedAt.Before(chosen.PublishedAt) {
			chosen = release
			found = true
		}
	}
	return chosen, found
}

func nearestVersionTagAfter(mergedAt time.Time, tags []Tag) (string, time.Time, bool) {
	var (
		selectedName string
		selectedAt   time.Time
		found        bool
	)

	for _, tag := range tags {
		tagName := strings.TrimSpace(tag.Name)
		createdAt := tag.CreatedAt
		if tagName == "" || createdAt.IsZero() || !isVersionTag(tagName) {
			continue
		}
		if createdAt.Before(mergedAt) || createdAt.Sub(mergedAt) > linkInferenceWindow {
			continue
		}
		if !found || createdAt.Before(selectedAt) {
			selectedName = tagName
			selectedAt = createdAt
			found = true
		}
	}

	return selectedName, selectedAt, found
}

func releaseMarker(release Release) string {
	if strings.TrimSpace(release.TagName) != "" {
		return strings.TrimSpace(release.TagName)
	}
	if strings.TrimSpace(release.Name) != "" {
		return strings.TrimSpace(release.Name)
	}
	return "release"
}

func prsMergedWithinWindow(prs []PR, windowStart time.Time, windowEnd time.Time) []int {
	matches := make([]int, 0, len(prs))
	for _, pr := range prs {
		if pr.MergedAt.Before(windowStart) || pr.MergedAt.After(windowEnd) {
			continue
		}
		matches = append(matches, pr.Number)
	}
	sort.Ints(matches)
	return matches
}

func clusterReleaseOptions(cluster []Release) []string {
	options := make([]string, 0, len(cluster))
	seen := map[string]struct{}{}
	for _, release := range cluster {
		option := releaseMarker(release)
		if _, exists := seen[option]; exists {
			continue
		}
		seen[option] = struct{}{}
		options = append(options, option)
	}
	sort.Strings(options)
	return options
}

func mergeDistinctOptions(current []string, additions []string) []string {
	seen := map[string]struct{}{}
	merged := make([]string, 0, len(current)+len(additions))
	for _, option := range current {
		if _, exists := seen[option]; exists {
			continue
		}
		seen[option] = struct{}{}
		merged = append(merged, option)
	}
	for _, option := range additions {
		if _, exists := seen[option]; exists {
			continue
		}
		seen[option] = struct{}{}
		merged = append(merged, option)
	}
	sort.Strings(merged)
	return merged
}

func buildAmbiguityWaitMessage(items []AmbiguousDeployment) string {
	if len(items) == 0 {
		return ""
	}

	descriptions := make([]string, 0, len(items))
	for _, item := range items {
		descriptions = append(
			descriptions,
			fmt.Sprintf("PR #%d -> [%s]", item.PRNumber, strings.Join(item.Options, ", ")),
		)
	}
	return fmt.Sprintf(
		"Ambiguous deployment mapping detected (%d PRs). Multiple releases were published within 24h. Please confirm mappings: %s",
		len(items),
		strings.Join(descriptions, "; "),
	)
}
