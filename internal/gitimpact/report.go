package gitimpact

import (
	"fmt"
	"html"
	"os"
	"sort"
	"strings"
	"time"
)

// SaveMarkdown writes an analysis report to a markdown file.
func SaveMarkdown(result *AnalysisResult, path string) error {
	if result == nil {
		return fmt.Errorf("analysis result is required")
	}
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("report path is required")
	}

	content := renderMarkdownReport(result)
	return os.WriteFile(path, []byte(content), 0o644)
}

// SaveHTML writes an analysis report to an HTML file.
func SaveHTML(result *AnalysisResult, path string) error {
	if result == nil {
		return fmt.Errorf("analysis result is required")
	}
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("report path is required")
	}

	content := renderHTMLReport(result)
	return os.WriteFile(path, []byte(content), 0o644)
}

func renderMarkdownReport(result *AnalysisResult) string {
	prMap := mapPRByNumber(result.PRs)
	featureRows := buildFeatureImpactRows(result.FeatureGroups, result.PRImpacts)
	contributors := sortedContributors(result.Contributors)

	var b strings.Builder
	b.WriteString("# Git Impact Report\n\n")
	if !result.GeneratedAt.IsZero() {
		b.WriteString(fmt.Sprintf("Generated: %s\n\n", result.GeneratedAt.UTC().Format(time.RFC3339)))
	}
	b.WriteString("Correlation note: these scores indicate correlation, not strict causation.\n\n")

	b.WriteString("## PR Impact Results\n\n")
	b.WriteString("| # | PR Title | Author | Score | Confidence |\n")
	b.WriteString("| --- | --- | --- | ---: | --- |\n")
	for _, impact := range result.PRImpacts {
		pr := prMap[impact.PRNumber]
		b.WriteString(fmt.Sprintf(
			"| %d | %s | %s | %.1f | %s |\n",
			impact.PRNumber,
			escapeMarkdownCell(fallbackText(pr.Title, fmt.Sprintf("PR #%d", impact.PRNumber))),
			escapeMarkdownCell(displayAuthor(pr.Author)),
			impact.Score,
			escapeMarkdownCell(fallbackText(strings.TrimSpace(impact.Confidence), "unknown")),
		))
	}
	if len(result.PRImpacts) == 0 {
		b.WriteString("| - | _No PR impacts_ | - | - | - |\n")
	}
	b.WriteString("\n")

	b.WriteString("## Feature Impact\n\n")
	b.WriteString("| Feature | PRs | Combined Score |\n")
	b.WriteString("| --- | --- | ---: |\n")
	for _, row := range featureRows {
		b.WriteString(fmt.Sprintf(
			"| %s | %s | %.1f |\n",
			escapeMarkdownCell(row.Name),
			escapeMarkdownCell(formatPRNumberList(row.PRNumbers)),
			row.CombinedScore,
		))
	}
	if len(featureRows) == 0 {
		b.WriteString("| _No feature groups_ | - | - |\n")
	}
	b.WriteString("\n")

	b.WriteString("## Contributor Leaderboard\n\n")
	b.WriteString("| Rank | Author | PRs | Avg Impact | Top PR |\n")
	b.WriteString("| ---: | --- | ---: | ---: | --- |\n")
	for idx, contributor := range contributors {
		topPRLabel := contributorTopPRLabel(contributor, prMap)
		b.WriteString(fmt.Sprintf(
			"| %d | %s | %d | %.1f | %s |\n",
			idx+1,
			escapeMarkdownCell(displayAuthor(contributor.Author)),
			contributor.PRCount,
			contributor.AverageScore,
			escapeMarkdownCell(topPRLabel),
		))
	}
	if len(contributors) == 0 {
		b.WriteString("| - | _No contributor stats_ | - | - | - |\n")
	}

	return b.String()
}

func renderHTMLReport(result *AnalysisResult) string {
	prMap := mapPRByNumber(result.PRs)
	featureRows := buildFeatureImpactRows(result.FeatureGroups, result.PRImpacts)
	contributors := sortedContributors(result.Contributors)

	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n")
	b.WriteString("<html lang=\"en\">\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n")
	b.WriteString("<title>Git Impact Report</title>\n")
	b.WriteString("<style>")
	b.WriteString("body{font-family:ui-sans-serif,system-ui,-apple-system,Segoe UI,Roboto,sans-serif;margin:24px;line-height:1.4;color:#1f2937;}")
	b.WriteString("table{border-collapse:collapse;width:100%;margin:12px 0 24px 0;}")
	b.WriteString("th,td{border:1px solid #d1d5db;padding:8px;text-align:left;vertical-align:top;}")
	b.WriteString("th{background:#f3f4f6;}")
	b.WriteString(".num{text-align:right;}")
	b.WriteString("code{background:#f3f4f6;padding:1px 4px;border-radius:4px;}")
	b.WriteString("</style>\n</head>\n<body>\n")
	b.WriteString("<h1>Git Impact Report</h1>\n")
	if !result.GeneratedAt.IsZero() {
		b.WriteString("<p><strong>Generated:</strong> ")
		b.WriteString(html.EscapeString(result.GeneratedAt.UTC().Format(time.RFC3339)))
		b.WriteString("</p>\n")
	}
	b.WriteString("<p><em>Correlation note: these scores indicate correlation, not strict causation.</em></p>\n")

	b.WriteString("<h2>PR Impact Results</h2>\n<table>\n")
	b.WriteString("<thead><tr><th>#</th><th>PR Title</th><th>Author</th><th class=\"num\">Score</th><th>Confidence</th></tr></thead>\n<tbody>\n")
	for _, impact := range result.PRImpacts {
		pr := prMap[impact.PRNumber]
		b.WriteString("<tr>")
		b.WriteString(fmt.Sprintf("<td>%d</td>", impact.PRNumber))
		b.WriteString("<td>" + html.EscapeString(fallbackText(pr.Title, fmt.Sprintf("PR #%d", impact.PRNumber))) + "</td>")
		b.WriteString("<td>" + html.EscapeString(displayAuthor(pr.Author)) + "</td>")
		b.WriteString(fmt.Sprintf("<td class=\"num\">%.1f</td>", impact.Score))
		b.WriteString("<td>" + html.EscapeString(fallbackText(strings.TrimSpace(impact.Confidence), "unknown")) + "</td>")
		b.WriteString("</tr>\n")
	}
	if len(result.PRImpacts) == 0 {
		b.WriteString("<tr><td>-</td><td>No PR impacts</td><td>-</td><td class=\"num\">-</td><td>-</td></tr>\n")
	}
	b.WriteString("</tbody>\n</table>\n")

	b.WriteString("<h2>Feature Impact</h2>\n<table>\n")
	b.WriteString("<thead><tr><th>Feature</th><th>PRs</th><th class=\"num\">Combined Score</th></tr></thead>\n<tbody>\n")
	for _, row := range featureRows {
		b.WriteString("<tr>")
		b.WriteString("<td>" + html.EscapeString(row.Name) + "</td>")
		b.WriteString("<td>" + html.EscapeString(formatPRNumberList(row.PRNumbers)) + "</td>")
		b.WriteString(fmt.Sprintf("<td class=\"num\">%.1f</td>", row.CombinedScore))
		b.WriteString("</tr>\n")
	}
	if len(featureRows) == 0 {
		b.WriteString("<tr><td>No feature groups</td><td>-</td><td class=\"num\">-</td></tr>\n")
	}
	b.WriteString("</tbody>\n</table>\n")

	b.WriteString("<h2>Contributor Leaderboard</h2>\n<table>\n")
	b.WriteString("<thead><tr><th class=\"num\">Rank</th><th>Author</th><th class=\"num\">PRs</th><th class=\"num\">Avg Impact</th><th>Top PR</th></tr></thead>\n<tbody>\n")
	for idx, contributor := range contributors {
		topPRLabel := contributorTopPRLabel(contributor, prMap)
		b.WriteString("<tr>")
		b.WriteString(fmt.Sprintf("<td class=\"num\">%d</td>", idx+1))
		b.WriteString("<td>" + html.EscapeString(displayAuthor(contributor.Author)) + "</td>")
		b.WriteString(fmt.Sprintf("<td class=\"num\">%d</td>", contributor.PRCount))
		b.WriteString(fmt.Sprintf("<td class=\"num\">%.1f</td>", contributor.AverageScore))
		b.WriteString("<td>" + html.EscapeString(topPRLabel) + "</td>")
		b.WriteString("</tr>\n")
	}
	if len(contributors) == 0 {
		b.WriteString("<tr><td class=\"num\">-</td><td>No contributor stats</td><td class=\"num\">-</td><td class=\"num\">-</td><td>-</td></tr>\n")
	}
	b.WriteString("</tbody>\n</table>\n")
	b.WriteString("</body>\n</html>\n")

	return b.String()
}

type featureImpactRow struct {
	Name          string
	PRNumbers     []int
	CombinedScore float64
}

func buildFeatureImpactRows(groups []FeatureGroup, impacts []PRImpact) []featureImpactRow {
	if len(groups) == 0 {
		return nil
	}
	impactByPR := mapImpactByPRNumber(impacts)
	rows := make([]featureImpactRow, 0, len(groups))
	for _, group := range groups {
		rows = append(rows, featureImpactRow{
			Name:          fallbackText(strings.TrimSpace(group.Name), "unnamed feature"),
			PRNumbers:     append([]int(nil), group.PRNumbers...),
			CombinedScore: featureCombinedScore(group, impactByPR),
		})
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].CombinedScore == rows[j].CombinedScore {
			return strings.ToLower(rows[i].Name) < strings.ToLower(rows[j].Name)
		}
		return rows[i].CombinedScore > rows[j].CombinedScore
	})
	return rows
}

func mapPRByNumber(prs []PR) map[int]PR {
	mapped := make(map[int]PR, len(prs))
	for _, pr := range prs {
		mapped[pr.Number] = pr
	}
	return mapped
}

func mapImpactByPRNumber(impacts []PRImpact) map[int]PRImpact {
	mapped := make(map[int]PRImpact, len(impacts))
	for _, impact := range impacts {
		mapped[impact.PRNumber] = impact
	}
	return mapped
}

func featureCombinedScore(feature FeatureGroup, impactByPR map[int]PRImpact) float64 {
	if len(feature.PRNumbers) == 0 {
		return 0
	}
	total := 0.0
	count := 0
	for _, prNumber := range feature.PRNumbers {
		impact, ok := impactByPR[prNumber]
		if !ok {
			continue
		}
		total += impact.Score
		count++
	}
	if count == 0 {
		return 0
	}
	return total / float64(count)
}

func sortedContributors(input []ContributorStats) []ContributorStats {
	contributors := append([]ContributorStats(nil), input...)
	sort.SliceStable(contributors, func(i, j int) bool {
		if contributors[i].AverageScore == contributors[j].AverageScore {
			if contributors[i].PRCount == contributors[j].PRCount {
				return strings.ToLower(contributors[i].Author) < strings.ToLower(contributors[j].Author)
			}
			return contributors[i].PRCount > contributors[j].PRCount
		}
		return contributors[i].AverageScore > contributors[j].AverageScore
	})
	return contributors
}

func contributorTopPRLabel(stat ContributorStats, prByNumber map[int]PR) string {
	if stat.TopPRNumber <= 0 {
		return "n/a"
	}
	pr, ok := prByNumber[stat.TopPRNumber]
	if !ok || strings.TrimSpace(pr.Title) == "" {
		return fmt.Sprintf("PR #%d", stat.TopPRNumber)
	}
	return pr.Title
}

func displayAuthor(author string) string {
	trimmed := strings.TrimSpace(author)
	if trimmed == "" {
		return "@unknown"
	}
	if strings.HasPrefix(trimmed, "@") {
		return trimmed
	}
	return "@" + trimmed
}

func fallbackText(value, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	return trimmed
}

func formatPRNumberList(prNumbers []int) string {
	if len(prNumbers) == 0 {
		return "-"
	}
	parts := make([]string, 0, len(prNumbers))
	for _, prNumber := range prNumbers {
		parts = append(parts, fmt.Sprintf("#%d", prNumber))
	}
	return strings.Join(parts, ", ")
}

func escapeMarkdownCell(value string) string {
	replacer := strings.NewReplacer("|", "\\|", "\n", " ")
	return replacer.Replace(value)
}
