package gitimpact

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	mvpMetricName   = "conversion_rate"
	mvpMetricWeight = 1.0
)

type singlePRImpactResult struct {
	PR         prMetadata        `json:"pr"`
	Deployment deploymentLink    `json:"deployment"`
	Metric     metricComparison  `json:"metric"`
	Score      impactScoreResult `json:"impact_score"`
}

type prMetadata struct {
	Number   int    `json:"number"`
	Title    string `json:"title,omitempty"`
	Author   string `json:"author,omitempty"`
	MergedAt string `json:"merged_at"`
}

type deploymentLink struct {
	DeployedAt string `json:"deployed_at"`
	Source     string `json:"source"`
}

type metricComparison struct {
	Name       string  `json:"name"`
	Before     float64 `json:"before"`
	After      float64 `json:"after"`
	Delta      float64 `json:"delta"`
	DeltaRatio float64 `json:"delta_ratio"`
	Confidence float64 `json:"confidence"`
	Weight     float64 `json:"weight"`
}

type impactScoreResult struct {
	Score  float64 `json:"score"`
	Scale  string  `json:"scale"`
	Method string  `json:"method"`
}

func analyzeSinglePR(ctx context.Context, client VelenClient, cfg Config, prNumber int) (singlePRImpactResult, map[string]any, error) {
	pr, err := collectPRMetadata(ctx, client, cfg.Velen.Sources.GitHub, prNumber)
	if err != nil {
		return singlePRImpactResult{}, nil, err
	}

	link, deployedAt, err := linkDeployment(ctx, client, cfg.Velen.Sources.Warehouse, pr.Number, pr.MergedAt, cfg.Analysis.CooldownHours)
	if err != nil {
		return singlePRImpactResult{}, nil, err
	}

	metric, err := compareMetricBeforeAfter(ctx, client, cfg.Velen.Sources.Analytics, deployedAt, cfg.Analysis.BeforeWindowDays, cfg.Analysis.AfterWindowDays, cfg.Analysis.MinConfidence)
	if err != nil {
		return singlePRImpactResult{}, nil, err
	}
	score := computeImpactScore(metric)

	result := singlePRImpactResult{
		PR: prMetadata{
			Number:   pr.Number,
			Title:    pr.Title,
			Author:   pr.Author,
			MergedAt: pr.MergedAt.Format(time.RFC3339),
		},
		Deployment: link,
		Metric:     metric,
		Score:      score,
	}
	meta := map[string]any{
		"collector": "completed",
		"linker":    "completed",
		"scorer":    "completed",
	}
	return result, meta, nil
}

type collectedPR struct {
	Number   int
	Title    string
	Author   string
	MergedAt time.Time
}

func collectPRMetadata(ctx context.Context, client VelenClient, sourceKey string, prNumber int) (collectedPR, error) {
	sql := fmt.Sprintf(`-- phase1-mvp collector: PR metadata
SELECT
  number AS pr_number,
  title,
  author,
  merged_at
FROM pull_requests
WHERE number = %d
LIMIT 1;
`, prNumber)
	rows, err := runQueryRows(ctx, client, sourceKey, sql)
	if err != nil {
		return collectedPR{}, fmt.Errorf("collector query failed: %w", err)
	}
	if len(rows) == 0 {
		return collectedPR{}, fmt.Errorf("collector returned no rows for pr=%d", prNumber)
	}
	row := rows[0]
	number := intFromAny(row["pr_number"])
	if number == 0 {
		number = prNumber
	}
	mergedAt, err := timeFromAny(row["merged_at"])
	if err != nil {
		return collectedPR{}, fmt.Errorf("collector returned invalid merged_at: %w", err)
	}
	return collectedPR{
		Number:   number,
		Title:    stringFromAny(row["title"]),
		Author:   stringFromAny(row["author"]),
		MergedAt: mergedAt,
	}, nil
}

func linkDeployment(ctx context.Context, client VelenClient, sourceKey string, prNumber int, mergedAt time.Time, cooldownHours int) (deploymentLink, time.Time, error) {
	sql := fmt.Sprintf(`-- phase1-mvp linker: deployment lookup
SELECT
  deployed_at
FROM deployments
WHERE pr_number = %d
ORDER BY deployed_at ASC
LIMIT 1;
`, prNumber)
	rows, err := runQueryRows(ctx, client, sourceKey, sql)
	if err != nil {
		return deploymentLink{}, time.Time{}, fmt.Errorf("linker query failed: %w", err)
	}

	var deployedAt time.Time
	source := "warehouse.deployments"
	if len(rows) == 0 {
		deployedAt = mergedAt.Add(time.Duration(cooldownHours) * time.Hour)
		source = "fallback.merged_at_plus_cooldown"
	} else {
		value, parseErr := timeFromAny(rows[0]["deployed_at"])
		if parseErr != nil {
			return deploymentLink{}, time.Time{}, fmt.Errorf("linker returned invalid deployed_at: %w", parseErr)
		}
		deployedAt = value
	}

	return deploymentLink{
		DeployedAt: deployedAt.Format(time.RFC3339),
		Source:     source,
	}, deployedAt, nil
}

func compareMetricBeforeAfter(ctx context.Context, client VelenClient, sourceKey string, deployedAt time.Time, beforeWindowDays int, afterWindowDays int, minConfidence float64) (metricComparison, error) {
	beforeStart := deployedAt.AddDate(0, 0, -beforeWindowDays)
	beforeEnd := deployedAt
	afterStart := deployedAt
	afterEnd := deployedAt.AddDate(0, 0, afterWindowDays)

	beforeSQL := fmt.Sprintf(`-- phase1-mvp scorer: metric before window
-- phase: before
SELECT
  AVG(metric_value) AS metric_value,
  COUNT(1) AS sample_size
FROM metric_events
WHERE metric_name = '%s'
  AND event_time >= TIMESTAMP '%s'
  AND event_time < TIMESTAMP '%s';
`, mvpMetricName, beforeStart.Format(time.RFC3339), beforeEnd.Format(time.RFC3339))
	afterSQL := fmt.Sprintf(`-- phase1-mvp scorer: metric after window
-- phase: after
SELECT
  AVG(metric_value) AS metric_value,
  COUNT(1) AS sample_size
FROM metric_events
WHERE metric_name = '%s'
  AND event_time >= TIMESTAMP '%s'
  AND event_time < TIMESTAMP '%s';
`, mvpMetricName, afterStart.Format(time.RFC3339), afterEnd.Format(time.RFC3339))

	beforeRows, err := runQueryRows(ctx, client, sourceKey, beforeSQL)
	if err != nil {
		return metricComparison{}, fmt.Errorf("scorer before query failed: %w", err)
	}
	afterRows, err := runQueryRows(ctx, client, sourceKey, afterSQL)
	if err != nil {
		return metricComparison{}, fmt.Errorf("scorer after query failed: %w", err)
	}
	if len(beforeRows) == 0 || len(afterRows) == 0 {
		return metricComparison{}, fmt.Errorf("scorer query returned no rows for metric comparison")
	}

	beforeValue := floatFromAny(beforeRows[0]["metric_value"])
	afterValue := floatFromAny(afterRows[0]["metric_value"])
	beforeSample := floatFromAny(beforeRows[0]["sample_size"])
	afterSample := floatFromAny(afterRows[0]["sample_size"])
	confidence := deriveConfidence(beforeSample, afterSample, minConfidence)

	delta := afterValue - beforeValue
	deltaRatio := 0.0
	if beforeValue != 0 {
		deltaRatio = delta / abs(beforeValue)
	} else if afterValue != 0 {
		deltaRatio = 1
	}

	return metricComparison{
		Name:       mvpMetricName,
		Before:     beforeValue,
		After:      afterValue,
		Delta:      delta,
		DeltaRatio: deltaRatio,
		Confidence: confidence,
		Weight:     mvpMetricWeight,
	}, nil
}

func computeImpactScore(metric metricComparison) impactScoreResult {
	score := metric.DeltaRatio * metric.Weight * metric.Confidence * 10
	if score > 10 {
		score = 10
	}
	if score < -10 {
		score = -10
	}
	return impactScoreResult{
		Score:  score,
		Scale:  "[-10,10]",
		Method: "delta_ratio*weight*confidence*10",
	}
}

func runQueryRows(ctx context.Context, client VelenClient, sourceKey string, sql string) ([]map[string]any, error) {
	queryPath, err := writeTempQuery(sql)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = os.RemoveAll(filepath.Dir(queryPath))
	}()

	body, err := client.Query(ctx, sourceKey, queryPath)
	if err != nil {
		return nil, err
	}
	rows, err := parseRows(body)
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func writeTempQuery(sql string) (string, error) {
	dir, err := os.MkdirTemp("", "git-impact-queries-*")
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, "query.sql")
	if err := os.WriteFile(path, []byte(sql), 0o600); err != nil {
		return "", err
	}
	return path, nil
}

func parseRows(body []byte) ([]map[string]any, error) {
	object := map[string]any{}
	if err := json.Unmarshal(body, &object); err == nil {
		for _, key := range []string{"rows", "items", "data"} {
			if list, ok := object[key].([]any); ok {
				return anySliceToRows(list), nil
			}
		}
		if row, ok := object["row"].(map[string]any); ok {
			return []map[string]any{row}, nil
		}
	}

	list := []any{}
	if err := json.Unmarshal(body, &list); err == nil {
		return anySliceToRows(list), nil
	}
	return nil, fmt.Errorf("unable to parse query result rows")
}

func anySliceToRows(list []any) []map[string]any {
	rows := make([]map[string]any, 0, len(list))
	for _, item := range list {
		if row, ok := item.(map[string]any); ok {
			rows = append(rows, row)
		}
	}
	return rows
}

func timeFromAny(value any) (time.Time, error) {
	text := stringFromAny(value)
	if strings.TrimSpace(text) == "" {
		return time.Time{}, fmt.Errorf("empty time value")
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05", "2006-01-02"} {
		parsed, err := time.Parse(layout, text)
		if err == nil {
			return parsed.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("unsupported time format %q", text)
}

func floatFromAny(value any) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case float32:
		return float64(typed)
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		if err == nil {
			return parsed
		}
	}
	return 0
}

func intFromAny(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(typed))
		if err == nil {
			return parsed
		}
	}
	return 0
}

func stringFromAny(value any) string {
	return strings.TrimSpace(fmt.Sprint(value))
}

func deriveConfidence(beforeSample float64, afterSample float64, minConfidence float64) float64 {
	minSample := beforeSample
	if afterSample < minSample {
		minSample = afterSample
	}
	if minSample < 0 {
		minSample = 0
	}
	confidence := minSample / 1000
	if confidence < minConfidence {
		confidence = minConfidence
	}
	if confidence > 1 {
		confidence = 1
	}
	return confidence
}

func abs(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
}
