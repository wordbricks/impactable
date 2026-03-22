package gitimpact

import tea "github.com/charmbracelet/bubbletea"

// TurnStartedMsg notifies the TUI that a turn has started.
type TurnStartedMsg struct {
	Phase     Phase
	Iteration int
}

// PhaseAdvancedMsg notifies the TUI that the run moved to another phase.
type PhaseAdvancedMsg struct {
	From Phase
	To   Phase
}

// WaitEnteredMsg notifies the TUI that execution is paused for user input.
type WaitEnteredMsg struct {
	Message string
}

// WaitResolvedMsg notifies the TUI that a paused wait state was resolved.
type WaitResolvedMsg struct {
	Response string
}

// RunCompletedMsg notifies the TUI that the run finished successfully.
type RunCompletedMsg struct {
	Result *AnalysisResult
}

// RunExhaustedMsg notifies the TUI that the run ended due to exhaustion/error.
type RunExhaustedMsg struct {
	Err error
}

// TUIObserver bridges engine observer callbacks into Bubble Tea messages.
type TUIObserver struct {
	program *tea.Program
}

var _ Observer = (*TUIObserver)(nil)

// NewTUIObserver constructs an Observer that forwards callbacks to Bubble Tea.
func NewTUIObserver(program *tea.Program) *TUIObserver {
	return &TUIObserver{program: program}
}

func (o *TUIObserver) OnTurnStarted(phase Phase, iteration int) {
	o.send(TurnStartedMsg{Phase: phase, Iteration: iteration})
}

func (o *TUIObserver) OnPhaseAdvanced(from, to Phase) {
	o.send(PhaseAdvancedMsg{From: from, To: to})
}

func (o *TUIObserver) OnWaitEntered(message string) {
	o.send(WaitEnteredMsg{Message: message})
}

func (o *TUIObserver) OnWaitResolved(response string) {
	o.send(WaitResolvedMsg{Response: response})
}

func (o *TUIObserver) OnRunCompleted(result *AnalysisResult) {
	o.send(RunCompletedMsg{Result: result})
}

func (o *TUIObserver) OnRunExhausted(err error) {
	o.send(RunExhaustedMsg{Err: err})
}

func (o *TUIObserver) send(msg tea.Msg) {
	if o == nil || o.program == nil {
		return
	}
	o.program.Send(msg)
}
