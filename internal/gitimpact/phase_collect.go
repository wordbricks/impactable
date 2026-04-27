package gitimpact

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	collectDateLayout = "2006-01-02"
	defaultSinceDate  = "1970-01-01"
)

type collectQueryFn func(client *OneQueryClient, sourceKey string, sql string) (*QueryResult, error)
type collectAPIFn func(client *OneQueryClient, sourceKey string, target string, fields []string, jq string) ([]byte, error)

// CollectHandler fetches GitHub PR, tag, and release metadata through OneQuery.
type CollectHandler struct {
	Query collectQueryFn
	API   collectAPIFn
}

func (h *CollectHandler) Handle(_ context.Context, runCtx *RunContext) (*TurnResult, error) {
	if runCtx == nil {
		return nil, fmt.Errorf("run context is required")
	}
	if runCtx.OneQueryClient == nil {
		return nil, fmt.Errorf("onequery client is required")
	}
	client := oneQueryClientForConfig(runCtx.OneQueryClient, runCtx.Config)

	sourceKey := ""
	if runCtx.Config != nil {
		sourceKey = strings.TrimSpace(runCtx.Config.OneQuery.Sources.GitHub)
	}
	if sourceKey == "" {
		return nil, fmt.Errorf("github source key is required")
	}

	query := h.Query
	if query == nil {
		query = func(client *OneQueryClient, sourceKey string, sql string) (*QueryResult, error) {
			return client.Query(sourceKey, sql)
		}
	}

	since := defaultSinceDate
	if runCtx.AnalysisCtx != nil && runCtx.AnalysisCtx.Since != nil {
		since = runCtx.AnalysisCtx.Since.Format(collectDateLayout)
	}

	prSQL := fmt.Sprintf(
		"SELECT number, title, author, merged_at, head_branch, labels FROM pull_requests WHERE merged_at > '%s' ORDER BY merged_at DESC LIMIT 100",
		since,
	)
	prResult, err := query(client, sourceKey, prSQL)
	if err != nil {
		if isSourceNotQueryable(err) {
			return h.collectViaAPI(client, sourceKey, since, runCtx)
		}
		return nil, fmt.Errorf("collect prs: %w", err)
	}
	prs, err := parsePRRows(prResult)
	if err != nil {
		return nil, err
	}

	tagSQL := "SELECT name, created_at FROM tags ORDER BY created_at DESC LIMIT 50"
	tagResult, err := query(client, sourceKey, tagSQL)
	if err != nil {
		return nil, fmt.Errorf("collect tags: %w", err)
	}
	tags, err := parseTagRows(tagResult)
	if err != nil {
		return nil, err
	}

	releaseSQL := "SELECT name, tag_name, published_at FROM releases ORDER BY published_at DESC LIMIT 20"
	releaseResult, err := query(client, sourceKey, releaseSQL)
	if err != nil {
		return nil, fmt.Errorf("collect releases: %w", err)
	}
	releases, err := parseReleaseRows(releaseResult)
	if err != nil {
		return nil, err
	}

	runCtx.CollectedData = &CollectedData{
		PRs:      prs,
		Tags:     tags,
		Releases: releases,
	}

	return &TurnResult{Directive: DirectiveAdvancePhase}, nil
}

func (h *CollectHandler) collectViaAPI(client *OneQueryClient, sourceKey string, since string, runCtx *RunContext) (*TurnResult, error) {
	repo := ""
	if runCtx != nil && runCtx.Config != nil {
		repo = strings.TrimSpace(runCtx.Config.OneQuery.GitHubRepository)
	}
	if repo == "" {
		return nil, fmt.Errorf("github repository is required for api collection")
	}

	apiCall := h.API
	if apiCall == nil {
		apiCall = func(client *OneQueryClient, sourceKey string, target string, fields []string, jq string) ([]byte, error) {
			return client.API(sourceKey, target, fields, jq)
		}
	}

	sinceTime, err := time.Parse(collectDateLayout, since)
	if err != nil {
		return nil, fmt.Errorf("collect api: invalid since date %q: %w", since, err)
	}
	sinceRFC3339 := sinceTime.UTC().Format(time.RFC3339)

	prJQ := fmt.Sprintf(`[.[] | select(.merged_at != null and .merged_at >= %q) | {Number: .number, Title: .title, Author: .user.login, MergedAt: .merged_at, Branch: .head.ref, Labels: [.labels[].name]}]`, sinceRFC3339)
	prPayload, err := apiCall(client, sourceKey, repo+"/pulls", []string{
		"params[state]=closed",
		"params[sort]=updated",
		"params[direction]=desc",
		"params[per_page]=100",
	}, prJQ)
	if err != nil {
		return nil, fmt.Errorf("collect prs via api: %w", err)
	}
	prs, err := parseAPIPrs(prPayload)
	if err != nil {
		return nil, err
	}
	for idx := range prs {
		filesPayload, err := apiCall(client, sourceKey, fmt.Sprintf("%s/pulls/%d/files", repo, prs[idx].Number), []string{"params[per_page]=100"}, `[.[].filename]`)
		if err != nil {
			return nil, fmt.Errorf("collect changed files for pr %d: %w", prs[idx].Number, err)
		}
		files, err := parseStringArray(filesPayload)
		if err != nil {
			return nil, fmt.Errorf("collect changed files for pr %d: %w", prs[idx].Number, err)
		}
		prs[idx].ChangedFile = files
	}

	tagPayload, err := apiCall(client, sourceKey, repo+"/tags", []string{"params[per_page]=100"}, `[.[] | {Name: .name, Sha: .commit.sha}]`)
	if err != nil {
		return nil, fmt.Errorf("collect tags via api: %w", err)
	}
	tags, err := parseAPITags(tagPayload)
	if err != nil {
		return nil, err
	}

	releasePayload, err := apiCall(client, sourceKey, repo+"/releases", []string{"params[per_page]=100"}, `[.[] | {Name: .name, TagName: .tag_name, PublishedAt: .published_at}]`)
	if err != nil {
		return nil, fmt.Errorf("collect releases via api: %w", err)
	}
	releases, err := parseAPIReleases(releasePayload)
	if err != nil {
		return nil, err
	}

	runCtx.CollectedData = &CollectedData{
		PRs:      prs,
		Tags:     tags,
		Releases: releases,
	}
	return &TurnResult{Directive: DirectiveAdvancePhase}, nil
}

func isSourceNotQueryable(err error) bool {
	var oneQueryErr *OneQueryError
	if !errors.As(err, &oneQueryErr) {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(oneQueryErr.Code), "SOURCE_NOT_QUERYABLE")
}

func parseAPIPrs(payload []byte) ([]PR, error) {
	var rows []struct {
		Number   int      `json:"Number"`
		Title    string   `json:"Title"`
		Author   string   `json:"Author"`
		MergedAt string   `json:"MergedAt"`
		Branch   string   `json:"Branch"`
		Labels   []string `json:"Labels"`
	}
	if err := json.Unmarshal(payload, &rows); err != nil {
		return nil, fmt.Errorf("collect prs via api: decode response: %w", err)
	}
	prs := make([]PR, 0, len(rows))
	for idx, row := range rows {
		mergedAt, err := asTime(row.MergedAt)
		if err != nil {
			return nil, fmt.Errorf("collect prs via api: row %d invalid merged_at: %w", idx, err)
		}
		prs = append(prs, PR{
			Number:   row.Number,
			Title:    row.Title,
			Author:   row.Author,
			MergedAt: mergedAt,
			Branch:   row.Branch,
			Labels:   row.Labels,
		})
	}
	return prs, nil
}

func parseAPITags(payload []byte) ([]Tag, error) {
	var rows []struct {
		Name string `json:"Name"`
		Sha  string `json:"Sha"`
	}
	if err := json.Unmarshal(payload, &rows); err != nil {
		return nil, fmt.Errorf("collect tags via api: decode response: %w", err)
	}
	tags := make([]Tag, 0, len(rows))
	for _, row := range rows {
		tags = append(tags, Tag{Name: strings.TrimSpace(row.Name), Sha: strings.TrimSpace(row.Sha)})
	}
	return tags, nil
}

func parseAPIReleases(payload []byte) ([]Release, error) {
	var rows []struct {
		Name        string `json:"Name"`
		TagName     string `json:"TagName"`
		PublishedAt string `json:"PublishedAt"`
	}
	if err := json.Unmarshal(payload, &rows); err != nil {
		return nil, fmt.Errorf("collect releases via api: decode response: %w", err)
	}
	releases := make([]Release, 0, len(rows))
	for idx, row := range rows {
		publishedAt, err := asTime(row.PublishedAt)
		if err != nil {
			return nil, fmt.Errorf("collect releases via api: row %d invalid published_at: %w", idx, err)
		}
		releases = append(releases, Release{Name: row.Name, TagName: row.TagName, PublishedAt: publishedAt})
	}
	return releases, nil
}

func parseStringArray(payload []byte) ([]string, error) {
	var values []string
	if err := json.Unmarshal(payload, &values); err != nil {
		return nil, fmt.Errorf("decode string array: %w", err)
	}
	return values, nil
}

func parsePRRows(result *QueryResult) ([]PR, error) {
	if result == nil {
		return nil, fmt.Errorf("collect prs: query result is nil")
	}

	prs := make([]PR, 0, len(result.Rows))
	for idx, row := range result.Rows {
		if len(row) < 6 {
			return nil, fmt.Errorf("collect prs: row %d has %d columns, expected 6", idx, len(row))
		}

		number, err := asInt(row[0])
		if err != nil {
			return nil, fmt.Errorf("collect prs: row %d invalid number: %w", idx, err)
		}
		mergedAt, err := asTime(row[3])
		if err != nil {
			return nil, fmt.Errorf("collect prs: row %d invalid merged_at: %w", idx, err)
		}
		labels, err := asStringSlice(row[5])
		if err != nil {
			return nil, fmt.Errorf("collect prs: row %d invalid labels: %w", idx, err)
		}

		prs = append(prs, PR{
			Number:   number,
			Title:    asString(row[1]),
			Author:   asString(row[2]),
			MergedAt: mergedAt,
			Branch:   asString(row[4]),
			Labels:   labels,
		})
	}

	return prs, nil
}

func parseTagRows(result *QueryResult) ([]Tag, error) {
	if result == nil {
		return nil, fmt.Errorf("collect tags: query result is nil")
	}

	tags := make([]Tag, 0, len(result.Rows))
	for idx, row := range result.Rows {
		if len(row) < 2 {
			return nil, fmt.Errorf("collect tags: row %d has %d columns, expected 2", idx, len(row))
		}
		createdAt, err := asTime(row[1])
		if err != nil {
			return nil, fmt.Errorf("collect tags: row %d invalid created_at: %w", idx, err)
		}
		tags = append(tags, newTag(asString(row[0]), createdAt))
	}
	return tags, nil
}

func newTag(name string, createdAt time.Time) Tag {
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return Tag{}
	}
	tag := Tag{Name: trimmedName}
	if !createdAt.IsZero() {
		tag.CreatedAt = createdAt.UTC()
	}
	return tag
}

func parseReleaseRows(result *QueryResult) ([]Release, error) {
	if result == nil {
		return nil, fmt.Errorf("collect releases: query result is nil")
	}

	releases := make([]Release, 0, len(result.Rows))
	for idx, row := range result.Rows {
		if len(row) < 3 {
			return nil, fmt.Errorf("collect releases: row %d has %d columns, expected 3", idx, len(row))
		}

		publishedAt, err := asTime(row[2])
		if err != nil {
			return nil, fmt.Errorf("collect releases: row %d invalid published_at: %w", idx, err)
		}

		releases = append(releases, Release{
			Name:        asString(row[0]),
			TagName:     asString(row[1]),
			PublishedAt: publishedAt,
		})
	}

	return releases, nil
}

func asString(value interface{}) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	default:
		return fmt.Sprintf("%v", typed)
	}
}

func asInt(value interface{}) (int, error) {
	switch typed := value.(type) {
	case int:
		return typed, nil
	case int32:
		return int(typed), nil
	case int64:
		return int(typed), nil
	case float64:
		return int(typed), nil
	case json.Number:
		parsed, err := typed.Int64()
		if err != nil {
			return 0, err
		}
		return int(parsed), nil
	case string:
		trimmed := strings.TrimSpace(typed)
		if trimmed == "" {
			return 0, fmt.Errorf("empty string")
		}
		parsed, err := strconv.Atoi(trimmed)
		if err != nil {
			return 0, err
		}
		return parsed, nil
	default:
		return 0, fmt.Errorf("unsupported type %T", typed)
	}
}

func asTime(value interface{}) (time.Time, error) {
	switch typed := value.(type) {
	case time.Time:
		return typed, nil
	case string:
		trimmed := strings.TrimSpace(typed)
		if trimmed == "" {
			return time.Time{}, fmt.Errorf("empty string")
		}
		layouts := []string{
			time.RFC3339Nano,
			time.RFC3339,
			"2006-01-02 15:04:05",
			"2006-01-02T15:04:05",
			collectDateLayout,
		}
		for _, layout := range layouts {
			if parsed, err := time.Parse(layout, trimmed); err == nil {
				return parsed, nil
			}
		}
		return time.Time{}, fmt.Errorf("unsupported time format %q", trimmed)
	default:
		return time.Time{}, fmt.Errorf("unsupported type %T", typed)
	}
}

func asStringSlice(value interface{}) ([]string, error) {
	switch typed := value.(type) {
	case nil:
		return nil, nil
	case []string:
		return typed, nil
	case []interface{}:
		values := make([]string, 0, len(typed))
		for _, item := range typed {
			values = append(values, asString(item))
		}
		return values, nil
	case string:
		trimmed := strings.TrimSpace(typed)
		if trimmed == "" {
			return nil, nil
		}

		var decoded []string
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			if err := json.Unmarshal([]byte(trimmed), &decoded); err == nil {
				return decoded, nil
			}
		}

		parts := strings.Split(trimmed, ",")
		values := make([]string, 0, len(parts))
		for _, part := range parts {
			item := strings.TrimSpace(part)
			if item != "" {
				values = append(values, item)
			}
		}
		return values, nil
	default:
		return nil, fmt.Errorf("unsupported type %T", typed)
	}
}
