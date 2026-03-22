package gitimpact

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

const (
	phaseStatusWaiting = "waiting"
	phaseStatusRunning = "running"
	phaseStatusDone    = "done"
)

// PhaseStatus tracks progress status for each analysis phase.
type PhaseStatus struct {
	Phase       Phase
	DisplayName string
	Status      string
	Detail      string
}

// AnalysisModel is the Bubble Tea model for analysis progress.
type AnalysisModel struct {
	phases       []PhaseStatus
	currentPhase Phase
	iteration    int
	totalPhases  int
	isWaiting    bool
	waitMessage  string
	spinner      spinner.Model
	done         bool
	result       *AnalysisResult
	showResults  bool
	err          error
}

var _ tea.Model = (*AnalysisModel)(nil)

// DefaultAnalysisPhases returns default status rows for the progress view.
func DefaultAnalysisPhases() []PhaseStatus {
	return []PhaseStatus{
		{Phase: PhaseSourceCheck, DisplayName: "Sources", Status: phaseStatusWaiting},
		{Phase: PhaseCollect, DisplayName: "Collect", Status: phaseStatusWaiting},
		{Phase: PhaseLink, DisplayName: "Link", Status: phaseStatusWaiting},
		{Phase: PhaseScore, DisplayName: "Score", Status: phaseStatusWaiting},
		{Phase: PhaseReport, DisplayName: "Report", Status: phaseStatusWaiting},
	}
}

// NewAnalysisModel builds progress state from the supplied phase rows.
func NewAnalysisModel(phases []PhaseStatus) AnalysisModel {
	if len(phases) == 0 {
		phases = DefaultAnalysisPhases()
	}

	phaseCopy := make([]PhaseStatus, len(phases))
	copy(phaseCopy, phases)

	spin := spinner.New()
	spin.Spinner = spinner.Dot

	return AnalysisModel{
		phases:      phaseCopy,
		totalPhases: len(phaseCopy),
		spinner:     spin,
	}
}

// Init implements tea.Model.
func (m *AnalysisModel) Init() tea.Cmd {
	return m.spinner.Tick
}

// Update implements tea.Model.
func (m *AnalysisModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case TurnStartedMsg:
		m.currentPhase = typed.Phase
		m.iteration = typed.Iteration
		m.isWaiting = false
		m.waitMessage = ""
		m.err = nil
		m.setRunning(typed.Phase)
	case PhaseAdvancedMsg:
		m.currentPhase = typed.To
		m.setStatus(typed.From, phaseStatusDone)
		m.setRunning(typed.To)
	case WaitEnteredMsg:
		m.isWaiting = true
		m.waitMessage = typed.Message
	case WaitResolvedMsg:
		m.isWaiting = false
		m.waitMessage = ""
	case RunCompletedMsg:
		m.isWaiting = false
		m.waitMessage = ""
		m.currentPhase = PhaseReport
		m.iteration = m.totalPhases
		m.setAllDone()
		m.done = true
		m.result = typed.Result
		m.showResults = true
		m.err = nil
		return m, tea.Quit
	case RunExhaustedMsg:
		m.isWaiting = false
		m.waitMessage = ""
		m.showResults = false
		m.err = typed.Err
		return m, tea.Quit
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(typed)
		return m, cmd
	}
	return m, nil
}

// View implements tea.Model.
func (m *AnalysisModel) View() string {
	var b strings.Builder

	b.WriteString("Git Impact Analyzer\n")
	b.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	b.WriteString(fmt.Sprintf("[%s]  Turn %d/%d - %s\n\n", m.progressBar(18), m.iteration, maxInt(m.totalPhases, 1), m.currentPhaseDisplayName()))

	labelWidth := m.maxDisplayNameWidth()
	for _, phase := range m.phases {
		prefix := " "
		detail := strings.TrimSpace(phase.Detail)
		switch phase.Status {
		case phaseStatusDone:
			prefix = "✓"
			if detail == "" {
				detail = "Done"
			}
		case phaseStatusRunning:
			prefix = "→"
			if detail == "" {
				detail = fmt.Sprintf("%s Running", m.spinner.View())
			}
		default:
			if detail == "" {
				detail = "Waiting"
			}
		}
		b.WriteString(fmt.Sprintf("%s %-*s  %s\n", prefix, labelWidth, phase.DisplayName, detail))
	}

	if m.isWaiting {
		b.WriteString("\n")
		b.WriteString(strings.TrimSpace(m.waitMessage))
		b.WriteString("\n")
	}
	if m.err != nil {
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("Error: %v\n", m.err))
	}

	return strings.TrimRight(b.String(), "\n")
}

func (m *AnalysisModel) setStatus(target Phase, status string) {
	for i := range m.phases {
		if m.phases[i].Phase == target {
			m.phases[i].Status = status
			return
		}
	}
}

func (m *AnalysisModel) setRunning(target Phase) {
	for i := range m.phases {
		switch {
		case m.phases[i].Phase == target:
			m.phases[i].Status = phaseStatusRunning
		case m.phases[i].Status != phaseStatusDone:
			m.phases[i].Status = phaseStatusWaiting
		}
	}
}

func (m *AnalysisModel) setAllDone() {
	for i := range m.phases {
		m.phases[i].Status = phaseStatusDone
	}
}

func (m *AnalysisModel) maxDisplayNameWidth() int {
	width := 0
	for _, phase := range m.phases {
		if len(phase.DisplayName) > width {
			width = len(phase.DisplayName)
		}
	}
	if width == 0 {
		return len("Phase")
	}
	return width
}

func (m *AnalysisModel) progressBar(width int) string {
	if width <= 0 {
		width = 10
	}
	total := maxInt(m.totalPhases, 1)
	completed := m.completedPhases()
	filled := (completed * width) / total
	if filled < 0 {
		filled = 0
	}
	if filled > width {
		filled = width
	}
	return strings.Repeat("■", filled) + strings.Repeat("░", width-filled)
}

func (m *AnalysisModel) completedPhases() int {
	if m.done {
		return maxInt(m.totalPhases, 1)
	}

	completed := 0
	hasRunning := false
	for _, phase := range m.phases {
		switch phase.Status {
		case phaseStatusDone:
			completed++
		case phaseStatusRunning:
			hasRunning = true
		}
	}
	if hasRunning {
		completed++
	}
	return completed
}

func (m *AnalysisModel) currentPhaseDisplayName() string {
	if m.currentPhase == "" {
		return "Starting"
	}
	for _, phase := range m.phases {
		if phase.Phase == m.currentPhase {
			if strings.TrimSpace(phase.DisplayName) != "" {
				return phase.DisplayName
			}
			break
		}
	}
	return string(m.currentPhase)
}

func maxInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

// Result returns the final analysis result received through RunCompletedMsg.
func (m *AnalysisModel) Result() *AnalysisResult {
	if m == nil {
		return nil
	}
	return m.result
}

// ShouldShowResults reports whether the model reached completion with a result payload.
func (m *AnalysisModel) ShouldShowResults() bool {
	if m == nil {
		return false
	}
	return m.showResults && m.result != nil
}
