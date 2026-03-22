package gitimpact

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

const (
	phaseStatusWaiting = "waiting"
	phaseStatusRunning = "running"
	phaseStatusDone    = "done"
)

// PhaseStatus tracks progress status for each analysis phase.
type PhaseStatus struct {
	Name   Phase
	Status string
}

// AnalysisModel is a minimal Bubble Tea model for analysis progress.
type AnalysisModel struct {
	currentPhase Phase
	iteration    int
	phases       []PhaseStatus
	waitMessage  string
	isWaiting    bool
}

var _ tea.Model = AnalysisModel{}

// NewAnalysisModel builds the default progress state for all phases.
func NewAnalysisModel() AnalysisModel {
	return AnalysisModel{
		phases: []PhaseStatus{
			{Name: PhaseSourceCheck, Status: phaseStatusWaiting},
			{Name: PhaseCollect, Status: phaseStatusWaiting},
			{Name: PhaseLink, Status: phaseStatusWaiting},
			{Name: PhaseScore, Status: phaseStatusWaiting},
			{Name: PhaseReport, Status: phaseStatusWaiting},
		},
	}
}

// Init implements tea.Model.
func (m AnalysisModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m AnalysisModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case TurnStartedMsg:
		m.currentPhase = typed.Phase
		m.iteration = typed.Iteration
		m.isWaiting = false
		m.waitMessage = ""
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
		m.setAllDone()
	case RunExhaustedMsg:
		m.isWaiting = false
	}
	return m, nil
}

// View implements tea.Model.
func (m AnalysisModel) View() string {
	var b strings.Builder

	b.WriteString("Git Impact Analyzer\n")
	b.WriteString(fmt.Sprintf("Iteration: %d\n", m.iteration))
	if m.currentPhase != "" {
		b.WriteString(fmt.Sprintf("Current phase: %s\n", m.currentPhase))
	}
	b.WriteString("\nPhase Progress:\n")
	for _, phase := range m.phases {
		b.WriteString(fmt.Sprintf("%s %s (%s)\n", phaseMarker(phase.Status), phase.Name, phase.Status))
	}

	if m.isWaiting {
		b.WriteString("\nWaiting for input:\n")
		b.WriteString(m.waitMessage)
		b.WriteString("\n")
	}

	return strings.TrimRight(b.String(), "\n")
}

func (m *AnalysisModel) setStatus(target Phase, status string) {
	for i := range m.phases {
		if m.phases[i].Name == target {
			m.phases[i].Status = status
			return
		}
	}
}

func (m *AnalysisModel) setRunning(target Phase) {
	for i := range m.phases {
		switch {
		case m.phases[i].Name == target:
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

func phaseMarker(status string) string {
	switch status {
	case phaseStatusDone:
		return "[x]"
	case phaseStatusRunning:
		return "[>]"
	default:
		return "[ ]"
	}
}
