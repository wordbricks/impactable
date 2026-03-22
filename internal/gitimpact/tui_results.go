package gitimpact

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

const (
	resultsViewPRs          = "prs"
	resultsViewFeatures     = "features"
	resultsViewContributors = "contributors"
)

// SaveReportFunc persists the current analysis result in the requested format.
type SaveReportFunc func(format string) (path string, err error)

type saveResultMsg struct {
	format string
	path   string
	err    error
}

// ResultsModel renders and navigates post-analysis interactive results.
type ResultsModel struct {
	prImpacts       []PRImpact
	prs             []PR
	featureGroups   []FeatureGroup
	contributors    []ContributorStats
	cursor          int
	activeView      string
	selectedPR      *PRImpact
	selectedFeature *FeatureGroup
	viewport        viewport.Model
	table           table.Model

	result             *AnalysisResult
	saveReport         SaveReportFunc
	prByNumber         map[int]PR
	deploymentByPR     map[int]Deployment
	featureRows        []featureImpactRow
	rankedContributors []ContributorStats
	savePrompt         bool
	saveMessage        string
	windowWidth        int
	windowHeight       int
}

var _ tea.Model = (*ResultsModel)(nil)

// NewResultsModel builds a results UI model from an analysis result.
func NewResultsModel(result *AnalysisResult, saveReport SaveReportFunc) ResultsModel {
	var (
		prImpacts     []PRImpact
		prs           []PR
		featureGroups []FeatureGroup
		contributors  []ContributorStats
		deployments   []Deployment
	)
	if result != nil {
		prImpacts = append(prImpacts, result.PRImpacts...)
		prs = append(prs, result.PRs...)
		featureGroups = append(featureGroups, result.FeatureGroups...)
		contributors = append(contributors, result.Contributors...)
		deployments = append(deployments, result.Deployments...)
	}

	tbl := table.New(
		table.WithColumns(prTableColumns()),
		table.WithRows(buildPRTableRows(prImpacts, prs)),
		table.WithHeight(12),
		table.WithFocused(true),
	)

	vp := viewport.New(100, 18)
	vp.SetContent("")

	m := ResultsModel{
		prImpacts:          prImpacts,
		prs:                prs,
		featureGroups:      featureGroups,
		contributors:       contributors,
		cursor:             0,
		activeView:         resultsViewPRs,
		viewport:           vp,
		table:              tbl,
		result:             result,
		saveReport:         saveReport,
		prByNumber:         mapPRByNumber(prs),
		deploymentByPR:     mapDeploymentByPRNumber(deployments),
		featureRows:        buildFeatureRowsPreservingOrder(featureGroups, prImpacts),
		rankedContributors: sortedContributors(contributors),
		windowWidth:        100,
		windowHeight:       30,
	}
	m.syncTableToActiveView()
	return m
}

// Init implements tea.Model.
func (m *ResultsModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m *ResultsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		m.windowWidth = typed.Width
		m.windowHeight = typed.Height
		m.resizeComponents()
		return m, nil
	case saveResultMsg:
		m.savePrompt = false
		if typed.err != nil {
			m.saveMessage = fmt.Sprintf("Save failed: %v", typed.err)
		} else {
			formatName := strings.ToUpper(typed.format)
			m.saveMessage = fmt.Sprintf("Saved %s report: %s", formatName, typed.path)
		}
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(typed)
	}

	if m.inDetailView() {
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}
	if m.activeView == resultsViewPRs || m.activeView == resultsViewContributors {
		var cmd tea.Cmd
		m.table, cmd = m.table.Update(msg)
		m.cursor = m.table.Cursor()
		return m, cmd
	}

	return m, nil
}

func (m *ResultsModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	if m.savePrompt {
		switch key {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "esc":
			m.savePrompt = false
			m.saveMessage = "Save cancelled"
			return m, nil
		case "m", "M":
			return m, m.saveCmd("md")
		case "h", "H":
			return m, m.saveCmd("html")
		default:
			return m, nil
		}
	}

	switch key {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "tab":
		m.cycleView()
		return m, nil
	case "up":
		if m.inDetailView() {
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		}
		m.moveCursor(-1)
		return m, nil
	case "down":
		if m.inDetailView() {
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		}
		m.moveCursor(1)
		return m, nil
	case "enter":
		m.drillIntoSelection()
		return m, nil
	case "esc":
		m.selectedPR = nil
		m.selectedFeature = nil
		m.viewport.GotoTop()
		m.savePrompt = false
		return m, nil
	case "s":
		m.savePrompt = true
		m.saveMessage = "Save format? press m for markdown or h for html"
		return m, nil
	}

	if m.inDetailView() {
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m *ResultsModel) saveCmd(format string) tea.Cmd {
	return func() tea.Msg {
		path, err := m.saveByFormat(format)
		return saveResultMsg{format: format, path: path, err: err}
	}
}

func (m *ResultsModel) saveByFormat(format string) (string, error) {
	if m.saveReport != nil {
		return m.saveReport(format)
	}
	if m.result == nil {
		return "", fmt.Errorf("analysis result is unavailable")
	}
	format = strings.ToLower(strings.TrimSpace(format))
	switch format {
	case "md":
		path := "git-impact-report.md"
		return path, SaveMarkdown(m.result, path)
	case "html":
		path := "git-impact-report.html"
		return path, SaveHTML(m.result, path)
	default:
		return "", fmt.Errorf("unsupported save format %q", format)
	}
}

func (m *ResultsModel) cycleView() {
	next := resultsViewPRs
	switch m.activeView {
	case resultsViewPRs:
		next = resultsViewFeatures
	case resultsViewFeatures:
		next = resultsViewContributors
	case resultsViewContributors:
		next = resultsViewPRs
	}
	m.activeView = next
	m.selectedPR = nil
	m.selectedFeature = nil
	m.savePrompt = false
	m.cursor = 0
	m.syncTableToActiveView()
}

func (m *ResultsModel) moveCursor(delta int) {
	rows := m.activeViewLength()
	if rows == 0 {
		m.cursor = 0
		if m.activeView == resultsViewPRs || m.activeView == resultsViewContributors {
			m.table.SetCursor(0)
		}
		return
	}

	m.cursor = clampResultCursor(m.cursor+delta, 0, rows-1)
	if m.activeView == resultsViewPRs || m.activeView == resultsViewContributors {
		m.table.SetCursor(m.cursor)
	}
}

func (m *ResultsModel) activeViewLength() int {
	switch m.activeView {
	case resultsViewPRs:
		return len(m.prImpacts)
	case resultsViewFeatures:
		return len(m.featureRows)
	case resultsViewContributors:
		return len(m.rankedContributors)
	default:
		return 0
	}
}

func (m *ResultsModel) drillIntoSelection() {
	if m.inDetailView() {
		return
	}
	switch m.activeView {
	case resultsViewPRs:
		if m.cursor < 0 || m.cursor >= len(m.prImpacts) {
			return
		}
		m.selectedPR = &m.prImpacts[m.cursor]
		m.viewport.SetContent(m.renderPRDetailContent())
		m.viewport.GotoTop()
	case resultsViewFeatures:
		if m.cursor < 0 || m.cursor >= len(m.featureRows) {
			return
		}
		row := m.featureRows[m.cursor]
		m.selectedFeature = &FeatureGroup{Name: row.Name, PRNumbers: append([]int(nil), row.PRNumbers...)}
		m.viewport.SetContent(m.renderFeatureDetailContent())
		m.viewport.GotoTop()
	}
}

func (m *ResultsModel) inDetailView() bool {
	return m.selectedPR != nil || m.selectedFeature != nil
}

// View implements tea.Model.
func (m *ResultsModel) View() string {
	if m.selectedPR != nil {
		return m.viewPRDetail()
	}
	if m.selectedFeature != nil {
		return m.viewFeatureDetail()
	}

	var b strings.Builder
	b.WriteString(m.listHeader())
	b.WriteString("\n")
	switch m.activeView {
	case resultsViewPRs:
		b.WriteString(m.table.View())
	case resultsViewFeatures:
		b.WriteString(m.featureListView())
	case resultsViewContributors:
		b.WriteString(m.table.View())
	}

	if m.savePrompt {
		b.WriteString("\n\nSave format: [m] markdown  [h] html  [esc] cancel")
	}
	if strings.TrimSpace(m.saveMessage) != "" {
		b.WriteString("\n")
		b.WriteString(m.saveMessage)
	}

	return strings.TrimRight(b.String(), "\n")
}

func (m *ResultsModel) listHeader() string {
	suffix := "(up/down navigate, Enter to expand, Tab switch view, s save, q quit)"
	switch m.activeView {
	case resultsViewFeatures:
		return "Feature Impact Results  " + suffix
	case resultsViewContributors:
		return "Contributor Leaderboard  " + suffix
	default:
		return "PR Impact Results  " + suffix
	}
}

func (m *ResultsModel) featureListView() string {
	if len(m.featureRows) == 0 {
		return "No feature groups available."
	}
	lines := make([]string, 0, len(m.featureRows)+1)
	for i, row := range m.featureRows {
		marker := " "
		if i == m.cursor {
			marker = "▶"
		}
		lines = append(lines, fmt.Sprintf("%s %s (%s)  Combined Score: %.1f", marker, row.Name, formatPRNumberList(row.PRNumbers), row.CombinedScore))
	}
	return strings.Join(lines, "\n")
}

func (m *ResultsModel) viewPRDetail() string {
	var b strings.Builder
	b.WriteString(m.viewport.View())
	b.WriteString("\n\n")
	b.WriteString("(up/down scroll, esc back, tab switch view, s save, q quit)")
	if m.savePrompt {
		b.WriteString("\nSave format: [m] markdown  [h] html  [esc] cancel")
	}
	if strings.TrimSpace(m.saveMessage) != "" {
		b.WriteString("\n")
		b.WriteString(m.saveMessage)
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m *ResultsModel) renderPRDetailContent() string {
	if m.selectedPR == nil {
		return "No PR selected"
	}

	impact := *m.selectedPR
	pr := m.prByNumber[impact.PRNumber]
	deployment, hasDeployment := m.deploymentByPR[impact.PRNumber]

	mergedText := formatOptionalTime(pr.MergedAt)
	deployedText := "unknown"
	if hasDeployment {
		deployedText = formatOptionalTime(deployment.DeployedAt)
	}
	viaText := ""
	if hasDeployment {
		viaText = fmt.Sprintf(" (via %s)", formatDeploymentSource(deployment))
	}

	breakdown := metricBreakdownLine(impact)

	var b strings.Builder
	b.WriteString(fmt.Sprintf("PR #%d - \"%s\"\n", impact.PRNumber, fallbackText(pr.Title, fmt.Sprintf("PR #%d", impact.PRNumber))))
	b.WriteString(fmt.Sprintf("Author: %s\n", displayAuthor(pr.Author)))
	b.WriteString(fmt.Sprintf("Merged: %s, Deployed: %s%s\n", mergedText, deployedText, viaText))
	b.WriteString(fmt.Sprintf("Impact Score: %.1f / 10\n", impact.Score))
	b.WriteString("─────────────────────────────────────────────\n")
	b.WriteString(breakdown)
	b.WriteString("\n\nAgent reasoning:\n")
	b.WriteString(fmt.Sprintf("\"%s\"", fallbackText(impact.Reasoning, "No reasoning provided by agent.")))
	return b.String()
}

func (m *ResultsModel) viewFeatureDetail() string {
	var b strings.Builder
	b.WriteString(m.viewport.View())
	b.WriteString("\n\n")
	b.WriteString("(up/down scroll, esc back, tab switch view, s save, q quit)")
	if m.savePrompt {
		b.WriteString("\nSave format: [m] markdown  [h] html  [esc] cancel")
	}
	if strings.TrimSpace(m.saveMessage) != "" {
		b.WriteString("\n")
		b.WriteString(m.saveMessage)
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m *ResultsModel) renderFeatureDetailContent() string {
	if m.selectedFeature == nil {
		return "No feature selected"
	}
	feature := *m.selectedFeature
	impactByPR := mapImpactByPRNumber(m.prImpacts)
	combinedScore := featureCombinedScore(feature, impactByPR)
	periodStart, periodEnd := featureMergedPeriod(feature, m.prByNumber)

	var topMetric string
	for _, prNumber := range feature.PRNumbers {
		impact, ok := impactByPR[prNumber]
		if !ok {
			continue
		}
		topMetric = parseMetricName(impact.Reasoning)
		if topMetric != "" {
			break
		}
	}
	if topMetric == "" {
		topMetric = "n/a"
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Feature: \"%s\" (%s)\n", feature.Name, formatPRNumberList(feature.PRNumbers)))
	b.WriteString(fmt.Sprintf("Period: %s ~ %s\n", periodStart, periodEnd))
	b.WriteString(fmt.Sprintf("Combined Impact Score: %.1f / 10\n", combinedScore))
	b.WriteString(fmt.Sprintf("Top Metric: %s\n", topMetric))
	if len(feature.PRNumbers) > 0 {
		b.WriteString("\nRelated PRs:\n")
		for _, prNumber := range feature.PRNumbers {
			pr := m.prByNumber[prNumber]
			b.WriteString(fmt.Sprintf("- #%d %s\n", prNumber, fallbackText(pr.Title, fmt.Sprintf("PR #%d", prNumber))))
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m *ResultsModel) resizeComponents() {
	width := m.windowWidth - 2
	if width < 60 {
		width = 60
	}
	tableHeight := m.windowHeight - 8
	if tableHeight < 8 {
		tableHeight = 8
	}
	m.table.SetWidth(width)
	m.table.SetHeight(tableHeight)

	m.viewport.Width = width
	vpHeight := m.windowHeight - 10
	if vpHeight < 6 {
		vpHeight = 6
	}
	m.viewport.Height = vpHeight
}

func (m *ResultsModel) syncTableToActiveView() {
	switch m.activeView {
	case resultsViewContributors:
		m.table.SetColumns(contributorTableColumns())
		m.table.SetRows(buildContributorTableRows(m.rankedContributors, m.prByNumber))
	default:
		m.table.SetColumns(prTableColumns())
		m.table.SetRows(buildPRTableRows(m.prImpacts, m.prs))
	}
	m.cursor = clampResultCursor(m.cursor, 0, maxResultIndex(m.activeViewLength()))
	m.table.SetCursor(m.cursor)
}

func buildPRTableRows(impacts []PRImpact, prs []PR) []table.Row {
	prByNumber := mapPRByNumber(prs)
	rows := make([]table.Row, 0, len(impacts))
	for _, impact := range impacts {
		pr := prByNumber[impact.PRNumber]
		rows = append(rows, table.Row{
			fmt.Sprintf("%d", impact.PRNumber),
			fallbackText(pr.Title, fmt.Sprintf("PR #%d", impact.PRNumber)),
			displayAuthor(pr.Author),
			fmt.Sprintf("%.1f", impact.Score),
			fallbackText(impact.Confidence, "unknown"),
		})
	}
	return rows
}

func buildContributorTableRows(contributors []ContributorStats, prByNumber map[int]PR) []table.Row {
	rows := make([]table.Row, 0, len(contributors))
	for i, contributor := range contributors {
		rows = append(rows, table.Row{
			fmt.Sprintf("%d", i+1),
			displayAuthor(contributor.Author),
			fmt.Sprintf("%d", contributor.PRCount),
			fmt.Sprintf("%.1f", contributor.AverageScore),
			contributorTopPRLabel(contributor, prByNumber),
		})
	}
	return rows
}

func prTableColumns() []table.Column {
	return []table.Column{
		{Title: "#", Width: 6},
		{Title: "PR Title", Width: 42},
		{Title: "Author", Width: 16},
		{Title: "Score", Width: 8},
		{Title: "Confidence", Width: 12},
	}
}

func contributorTableColumns() []table.Column {
	return []table.Column{
		{Title: "Rank", Width: 6},
		{Title: "Author", Width: 16},
		{Title: "PRs", Width: 8},
		{Title: "Avg Impact", Width: 12},
		{Title: "Top PR", Width: 42},
	}
}

func buildFeatureRowsPreservingOrder(groups []FeatureGroup, impacts []PRImpact) []featureImpactRow {
	impactByPR := mapImpactByPRNumber(impacts)
	rows := make([]featureImpactRow, 0, len(groups))
	for _, group := range groups {
		rows = append(rows, featureImpactRow{
			Name:          fallbackText(group.Name, "unnamed feature"),
			PRNumbers:     append([]int(nil), group.PRNumbers...),
			CombinedScore: featureCombinedScore(group, impactByPR),
		})
	}
	return rows
}

func mapDeploymentByPRNumber(deployments []Deployment) map[int]Deployment {
	mapped := make(map[int]Deployment, len(deployments))
	for _, deployment := range deployments {
		mapped[deployment.PRNumber] = deployment
	}
	return mapped
}

func metricBreakdownLine(impact PRImpact) string {
	confidence := fallbackText(impact.Confidence, "unknown")
	metricName := parseMetricName(impact.Reasoning)
	delta := parseMetricDelta(impact.Reasoning)
	if metricName == "" {
		metricName = "impact_signal"
	}
	if delta == "" {
		delta = fmt.Sprintf("score %.1f", impact.Score)
	}
	return fmt.Sprintf("%s: %s (confidence: %s)", metricName, delta, confidence)
}

func parseMetricName(reasoning string) string {
	trimmed := strings.TrimSpace(reasoning)
	if !strings.HasPrefix(trimmed, "Metric ") {
		return ""
	}
	const prefix = "Metric "
	const suffix = " moved from"
	start := strings.Index(trimmed, prefix)
	end := strings.Index(trimmed, suffix)
	if start < 0 || end <= start+len(prefix) {
		return ""
	}
	return strings.TrimSpace(trimmed[start+len(prefix) : end])
}

func parseMetricDelta(reasoning string) string {
	trimmed := strings.TrimSpace(reasoning)
	const prefix = "(delta "
	start := strings.Index(trimmed, prefix)
	if start < 0 {
		return ""
	}
	remaining := trimmed[start+len(prefix):]
	end := strings.Index(remaining, ")")
	if end < 0 {
		return ""
	}
	return strings.TrimSpace(remaining[:end])
}

func formatOptionalTime(ts time.Time) string {
	if ts.IsZero() {
		return "unknown"
	}
	return ts.UTC().Format("2006-01-02 15:04")
}

func formatDeploymentSource(deployment Deployment) string {
	marker := strings.TrimSpace(deployment.Marker)
	source := strings.TrimSpace(deployment.Source)
	if source == "" {
		source = "unknown"
	}
	if marker == "" {
		return source
	}
	switch source {
	case "release":
		return "release " + marker
	case "tag":
		return "tag " + marker
	default:
		return source + " " + marker
	}
}

func featureMergedPeriod(feature FeatureGroup, prByNumber map[int]PR) (string, string) {
	var (
		start time.Time
		end   time.Time
	)
	for _, prNumber := range feature.PRNumbers {
		pr, ok := prByNumber[prNumber]
		if !ok || pr.MergedAt.IsZero() {
			continue
		}
		if start.IsZero() || pr.MergedAt.Before(start) {
			start = pr.MergedAt
		}
		if end.IsZero() || pr.MergedAt.After(end) {
			end = pr.MergedAt
		}
	}
	if start.IsZero() {
		return "unknown", "unknown"
	}
	return start.UTC().Format("2006-01-02"), end.UTC().Format("2006-01-02")
}

func maxResultIndex(length int) int {
	if length <= 0 {
		return 0
	}
	return length - 1
}

func clampResultCursor(v, minValue, maxValue int) int {
	if maxValue < minValue {
		return minValue
	}
	if v < minValue {
		return minValue
	}
	if v > maxValue {
		return maxValue
	}
	return v
}
