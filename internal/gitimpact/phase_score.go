package gitimpact

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	defaultScoreWindowDays = 7
	metricDateLayout       = "2006-01-02"
	analyticsSchemaSQL     = "SELECT table_name, column_name, data_type FROM information_schema.columns WHERE table_schema = current_schema() LIMIT 200"
)

type scoreQueryFn func(client *VelenClient, sourceKey string, sql string) (*QueryResult, error)

// ScoreHandler calculates PR impact scores from analytics metrics and deployment windows.
type ScoreHandler struct{}

var scoreQueryOverride scoreQueryFn

// Handle runs Turn 3 scoring and advances to the reporting phase.
func (h *ScoreHandler) Handle(_ context.Context, runCtx *RunContext) (*TurnResult, error) {
	if runCtx == nil {
		return nil, fmt.Errorf("run context is required")
	}
	if runCtx.VelenClient == nil {
		return nil, fmt.Errorf("velen client is required")
	}
	if runCtx.LinkedData == nil {
		return nil, fmt.Errorf("linked data is required")
	}

	sourceKey := ""
	if runCtx.Config != nil {
		sourceKey = strings.TrimSpace(runCtx.Config.Velen.Sources.Analytics)
	}
	if sourceKey == "" {
		return nil, fmt.Errorf("analytics source key is required")
	}

	query := scoreQueryOverride
	if query == nil {
		query = func(client *VelenClient, sourceKey string, sql string) (*QueryResult, error) {
			return client.Query(sourceKey, sql)
		}
	}

	schemaResult, err := query(runCtx.VelenClient, sourceKey, analyticsSchemaSQL)
	if err != nil {
		return nil, fmt.Errorf("score schema discovery: %w", err)
	}
	tableName, metricCol := selectMetricFromSchema(schemaResult)

	beforeDays, afterDays := resolveScoreWindows(runCtx.Config)
	overlapWindowDays := scoreWindowForConfidence(beforeDays, afterDays)
	deployments := runCtx.LinkedData.Deployments
	impacts := make([]PRImpact, 0, len(deployments))
	for _, deployment := range deployments {
		if deployment.DeployedAt.IsZero() {
			impacts = append(impacts, PRImpact{
				PRNumber:   deployment.PRNumber,
				Score:      0,
				Confidence: "low",
				Reasoning:  "Deployment timestamp is unavailable. Assigned neutral score 0.0 with low confidence.",
			})
			continue
		}

		beforeStart := deployment.DeployedAt.AddDate(0, 0, -beforeDays)
		afterEnd := deployment.DeployedAt.AddDate(0, 0, afterDays)
		overlapCount := countOverlappingDeployments(deployment.DeployedAt, deployments, overlapWindowDays)
		confidence := assessConfidenceWithWindow(deployment.DeployedAt, deployments, overlapWindowDays)
		confoundingContext := buildConfoundingContext(overlapCount, overlapWindowDays)

		impact := PRImpact{
			PRNumber:   deployment.PRNumber,
			Score:      0,
			Confidence: confidence,
			Reasoning:  "",
		}

		if tableName == "" || metricCol == "" {
			impact.Reasoning = fmt.Sprintf(
				"No analytics metric discovered from source %q. Assigned neutral score 0.0 with %s confidence (%s).",
				sourceKey,
				confidence,
				confoundingContext,
			)
			impacts = append(impacts, impact)
			continue
		}

		beforeSQL := buildMetricQuery(tableName, metricCol, sourceKey, beforeStart, deployment.DeployedAt)
		afterSQL := buildMetricQuery(tableName, metricCol, sourceKey, deployment.DeployedAt, afterEnd)

		beforeResult, err := query(runCtx.VelenClient, sourceKey, beforeSQL)
		if err != nil {
			impact.Reasoning = fmt.Sprintf(
				"Metric query failed for %s.%s before deployment: %v. Assigned neutral score 0.0 with %s confidence (%s).",
				tableName,
				metricCol,
				err,
				confidence,
				confoundingContext,
			)
			impacts = append(impacts, impact)
			continue
		}

		afterResult, err := query(runCtx.VelenClient, sourceKey, afterSQL)
		if err != nil {
			impact.Reasoning = fmt.Sprintf(
				"Metric query failed for %s.%s after deployment: %v. Assigned neutral score 0.0 with %s confidence (%s).",
				tableName,
				metricCol,
				err,
				confidence,
				confoundingContext,
			)
			impacts = append(impacts, impact)
			continue
		}

		beforeValue, beforeOk, err := extractAverage(beforeResult)
		if err != nil {
			return nil, fmt.Errorf("parse before metric for PR #%d: %w", deployment.PRNumber, err)
		}
		afterValue, afterOk, err := extractAverage(afterResult)
		if err != nil {
			return nil, fmt.Errorf("parse after metric for PR #%d: %w", deployment.PRNumber, err)
		}

		if !beforeOk || !afterOk {
			impact.Reasoning = fmt.Sprintf(
				"Metric %s.%s had insufficient data between %s and %s. Assigned neutral score 0.0 with %s confidence (%s).",
				tableName,
				metricCol,
				beforeStart.UTC().Format(metricDateLayout),
				afterEnd.UTC().Format(metricDateLayout),
				confidence,
				confoundingContext,
			)
			impacts = append(impacts, impact)
			continue
		}

		impact.Score = calculateScore(beforeValue, afterValue)
		delta := afterValue - beforeValue
		impact.Reasoning = fmt.Sprintf(
			"Metric %s.%s moved from %.4f to %.4f (delta %+0.4f) between %s and %s. Confidence %s due to %s.",
			tableName,
			metricCol,
			beforeValue,
			afterValue,
			delta,
			beforeStart.UTC().Format(metricDateLayout),
			afterEnd.UTC().Format(metricDateLayout),
			confidence,
			confoundingContext,
		)
		impacts = append(impacts, impact)
	}

	var prs []PR
	if runCtx.CollectedData != nil {
		prs = runCtx.CollectedData.PRs
	}

	runCtx.ScoredData = &ScoredData{
		PRImpacts:        impacts,
		ContributorStats: buildContributorStats(impacts, prs),
	}

	return &TurnResult{Directive: DirectiveAdvancePhase}, nil
}

func resolveScoreWindows(cfg *Config) (beforeDays int, afterDays int) {
	beforeDays = defaultScoreWindowDays
	afterDays = defaultScoreWindowDays
	if cfg == nil {
		return beforeDays, afterDays
	}
	if cfg.Analysis.BeforeWindowDays > 0 {
		beforeDays = cfg.Analysis.BeforeWindowDays
	}
	if cfg.Analysis.AfterWindowDays > 0 {
		afterDays = cfg.Analysis.AfterWindowDays
	}
	return beforeDays, afterDays
}

func scoreWindowForConfidence(beforeDays, afterDays int) int {
	windowDays := beforeDays
	if afterDays > windowDays {
		windowDays = afterDays
	}
	if windowDays <= 0 {
		return defaultScoreWindowDays
	}
	return windowDays
}

func buildConfoundingContext(overlapCount, windowDays int) string {
	if windowDays <= 0 {
		windowDays = defaultScoreWindowDays
	}
	if overlapCount <= 0 {
		return fmt.Sprintf("no confounding deployments in +/- %d days", windowDays)
	}
	return fmt.Sprintf("%d confounding deployments in +/- %d days", overlapCount, windowDays)
}

func selectMetricFromSchema(schema *QueryResult) (tableName string, metricCol string) {
	if schema == nil || len(schema.Rows) == 0 {
		return "", ""
	}

	firstTable := ""
	firstColumn := ""
	for _, row := range schema.Rows {
		if len(row) < 2 {
			continue
		}

		table := strings.TrimSpace(asString(row[0]))
		column := strings.TrimSpace(asString(row[1]))
		if table == "" || column == "" {
			continue
		}

		if firstTable == "" {
			firstTable = table
			firstColumn = column
		}

		dataType := ""
		if len(row) > 2 {
			dataType = strings.ToLower(strings.TrimSpace(asString(row[2])))
		}
		if isLikelyMetricColumn(column, dataType) {
			return table, column
		}
	}

	return firstTable, firstColumn
}

func isLikelyMetricColumn(column string, dataType string) bool {
	columnLower := strings.ToLower(strings.TrimSpace(column))
	if strings.Contains(columnLower, "date") || strings.Contains(columnLower, "time") {
		return false
	}

	typeLower := strings.ToLower(strings.TrimSpace(dataType))
	if typeLower == "" {
		return true
	}
	if strings.Contains(typeLower, "int") ||
		strings.Contains(typeLower, "float") ||
		strings.Contains(typeLower, "double") ||
		strings.Contains(typeLower, "numeric") ||
		strings.Contains(typeLower, "decimal") ||
		strings.Contains(typeLower, "real") {
		return true
	}
	if strings.Contains(typeLower, "date") || strings.Contains(typeLower, "time") {
		return false
	}

	return true
}

func extractAverage(result *QueryResult) (float64, bool, error) {
	if result == nil || len(result.Rows) == 0 || len(result.Rows[0]) == 0 {
		return 0, false, nil
	}
	value := result.Rows[0][0]
	if value == nil {
		return 0, false, nil
	}
	parsed, err := asFloat(value)
	if err != nil {
		return 0, false, err
	}
	return parsed, true, nil
}

func asFloat(value interface{}) (float64, error) {
	switch typed := value.(type) {
	case float64:
		return typed, nil
	case float32:
		return float64(typed), nil
	case int:
		return float64(typed), nil
	case int64:
		return float64(typed), nil
	case int32:
		return float64(typed), nil
	case json.Number:
		parsed, err := typed.Float64()
		if err != nil {
			return 0, err
		}
		return parsed, nil
	case string:
		trimmed := strings.TrimSpace(typed)
		if trimmed == "" {
			return 0, fmt.Errorf("empty string")
		}
		parsed, err := strconv.ParseFloat(trimmed, 64)
		if err != nil {
			return 0, err
		}
		return parsed, nil
	default:
		return 0, fmt.Errorf("unsupported type %T", typed)
	}
}

// calculateScore converts before/after metric movement into a 0-10 score.
func calculateScore(before, after float64) float64 {
	if before == 0 && after == 0 {
		return 0
	}
	baseline := math.Max(math.Abs(before), 1)
	relativeChange := math.Abs(after-before) / baseline
	score := relativeChange * 10
	if score > 10 {
		return 10
	}
	if score < 0 {
		return 0
	}
	return score
}

// assessConfidence estimates confidence from deployment overlap density.
func assessConfidence(deployedAt time.Time, allDeployments []Deployment) string {
	return assessConfidenceWithWindow(deployedAt, allDeployments, defaultScoreWindowDays)
}

func assessConfidenceWithWindow(deployedAt time.Time, allDeployments []Deployment, windowDays int) string {
	overlapCount := countOverlappingDeployments(deployedAt, allDeployments, windowDays)
	switch {
	case overlapCount == 0:
		return "high"
	case overlapCount <= 2:
		return "medium"
	default:
		return "low"
	}
}

func countOverlappingDeployments(deployedAt time.Time, allDeployments []Deployment, windowDays int) int {
	if deployedAt.IsZero() || len(allDeployments) == 0 {
		return 0
	}
	window := time.Duration(windowDays) * 24 * time.Hour
	withinWindow := 0
	for _, deployment := range allDeployments {
		if deployment.DeployedAt.IsZero() {
			continue
		}
		delta := deployment.DeployedAt.Sub(deployedAt)
		if delta < 0 {
			delta = -delta
		}
		if delta <= window {
			withinWindow++
		}
	}
	if withinWindow > 0 {
		withinWindow--
	}
	return withinWindow
}

// buildMetricQuery constructs a simple average metric query for a time window.
func buildMetricQuery(tableName, metricCol, source string, from, to time.Time) string {
	tableExpr := quoteIdentifier(tableName)
	metricExpr := quoteIdentifier(metricCol)
	fromValue := from.UTC().Format(metricDateLayout)
	toValue := to.UTC().Format(metricDateLayout)

	sourceComment := ""
	trimmedSource := strings.TrimSpace(source)
	if trimmedSource != "" {
		sanitized := strings.ReplaceAll(trimmedSource, "*/", "")
		sourceComment = fmt.Sprintf("/* source:%s */ ", sanitized)
	}

	return fmt.Sprintf(
		"%sSELECT avg(%s) FROM %s WHERE date BETWEEN '%s' AND '%s'",
		sourceComment,
		metricExpr,
		tableExpr,
		fromValue,
		toValue,
	)
}

func quoteIdentifier(name string) string {
	parts := strings.Split(strings.TrimSpace(name), ".")
	quotedParts := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		escaped := strings.ReplaceAll(trimmed, `"`, `""`)
		quotedParts = append(quotedParts, fmt.Sprintf(`"%s"`, escaped))
	}
	if len(quotedParts) == 0 {
		return `""`
	}
	return strings.Join(quotedParts, ".")
}

func buildContributorStats(impacts []PRImpact, prs []PR) []ContributorStats {
	if len(impacts) == 0 {
		return nil
	}

	authorByPR := map[int]string{}
	for _, pr := range prs {
		author := strings.TrimSpace(pr.Author)
		existing := strings.TrimSpace(authorByPR[pr.Number])
		// Keep the first non-empty author when duplicate PR rows appear.
		if existing != "" {
			continue
		}
		authorByPR[pr.Number] = author
	}

	type contributorBucket struct {
		count    int
		total    float64
		topPR    int
		topScore float64
	}

	buckets := map[string]*contributorBucket{}
	for _, impact := range impacts {
		author := strings.TrimSpace(authorByPR[impact.PRNumber])
		if author == "" {
			author = "unknown"
		}
		bucket, exists := buckets[author]
		if !exists {
			bucket = &contributorBucket{topPR: impact.PRNumber, topScore: impact.Score}
			buckets[author] = bucket
		}
		bucket.count++
		bucket.total += impact.Score
		if impact.Score > bucket.topScore || (impact.Score == bucket.topScore && impact.PRNumber < bucket.topPR) {
			bucket.topScore = impact.Score
			bucket.topPR = impact.PRNumber
		}
	}

	authors := make([]string, 0, len(buckets))
	for author := range buckets {
		authors = append(authors, author)
	}
	sort.Strings(authors)

	stats := make([]ContributorStats, 0, len(authors))
	for _, author := range authors {
		bucket := buckets[author]
		stats = append(stats, ContributorStats{
			Author:       author,
			PRCount:      bucket.count,
			AverageScore: bucket.total / float64(bucket.count),
			TopPRNumber:  bucket.topPR,
		})
	}
	return stats
}
