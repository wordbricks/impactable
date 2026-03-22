package gitimpact

import (
	"io"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type bridgeCaptureModel struct {
	ready    chan struct{}
	expected int
	msgs     []tea.Msg
}

func (m *bridgeCaptureModel) Init() tea.Cmd {
	close(m.ready)
	return nil
}

func (m *bridgeCaptureModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if !isBridgeMsg(msg) {
		return m, nil
	}

	m.msgs = append(m.msgs, msg)
	if len(m.msgs) >= m.expected {
		return m, tea.Quit
	}
	return m, nil
}

func (m *bridgeCaptureModel) View() string {
	return ""
}

func isBridgeMsg(msg tea.Msg) bool {
	switch msg.(type) {
	case TurnStartedMsg, PhaseAdvancedMsg, WaitEnteredMsg, WaitResolvedMsg, RunCompletedMsg, RunExhaustedMsg:
		return true
	default:
		return false
	}
}

func TestTUIObserverSendsBubbleTeaMessages(t *testing.T) {
	model := &bridgeCaptureModel{
		ready:    make(chan struct{}),
		expected: 6,
	}

	program := tea.NewProgram(
		model,
		tea.WithInput(nil),
		tea.WithOutput(io.Discard),
		tea.WithoutRenderer(),
		tea.WithoutSignalHandler(),
	)

	type runResult struct {
		model tea.Model
		err   error
	}
	done := make(chan runResult, 1)
	go func() {
		finalModel, err := program.Run()
		done <- runResult{model: finalModel, err: err}
	}()

	select {
	case <-model.ready:
	case <-time.After(2 * time.Second):
		t.Fatal("bubble tea program did not initialize in time")
	}

	observer := NewTUIObserver(program)
	result := &AnalysisResult{}
	exhaustedErr := &testErr{msg: "max iterations reached"}

	observer.TurnStarted(PhaseCollect, 2)
	observer.PhaseAdvanced(PhaseCollect, PhaseLink)
	observer.WaitEntered("Need deployment mapping confirmation")
	observer.WaitResolved("Use release tags")
	observer.RunCompleted(result)
	observer.RunExhausted(exhaustedErr)

	var run runResult
	select {
	case run = <-done:
	case <-time.After(2 * time.Second):
		program.Kill()
		t.Fatal("bubble tea program did not stop in time")
	}

	if run.err != nil {
		t.Fatalf("program run failed: %v", run.err)
	}

	finalModel, ok := run.model.(*bridgeCaptureModel)
	if !ok {
		t.Fatalf("expected *bridgeCaptureModel, got %T", run.model)
	}

	if len(finalModel.msgs) != 6 {
		t.Fatalf("expected 6 bridge messages, got %d", len(finalModel.msgs))
	}

	turnStarted, ok := finalModel.msgs[0].(TurnStartedMsg)
	if !ok {
		t.Fatalf("msg 0 type = %T, want TurnStartedMsg", finalModel.msgs[0])
	}
	if turnStarted.Phase != PhaseCollect || turnStarted.Iteration != 2 {
		t.Fatalf("unexpected TurnStartedMsg payload: %+v", turnStarted)
	}

	phaseAdvanced, ok := finalModel.msgs[1].(PhaseAdvancedMsg)
	if !ok {
		t.Fatalf("msg 1 type = %T, want PhaseAdvancedMsg", finalModel.msgs[1])
	}
	if phaseAdvanced.From != PhaseCollect || phaseAdvanced.To != PhaseLink {
		t.Fatalf("unexpected PhaseAdvancedMsg payload: %+v", phaseAdvanced)
	}

	waitEntered, ok := finalModel.msgs[2].(WaitEnteredMsg)
	if !ok {
		t.Fatalf("msg 2 type = %T, want WaitEnteredMsg", finalModel.msgs[2])
	}
	if waitEntered.Message != "Need deployment mapping confirmation" {
		t.Fatalf("unexpected WaitEnteredMsg payload: %+v", waitEntered)
	}

	waitResolved, ok := finalModel.msgs[3].(WaitResolvedMsg)
	if !ok {
		t.Fatalf("msg 3 type = %T, want WaitResolvedMsg", finalModel.msgs[3])
	}
	if waitResolved.Response != "Use release tags" {
		t.Fatalf("unexpected WaitResolvedMsg payload: %+v", waitResolved)
	}

	runCompleted, ok := finalModel.msgs[4].(RunCompletedMsg)
	if !ok {
		t.Fatalf("msg 4 type = %T, want RunCompletedMsg", finalModel.msgs[4])
	}
	if runCompleted.Result != result {
		t.Fatalf("unexpected RunCompletedMsg result pointer: got %p want %p", runCompleted.Result, result)
	}

	runExhausted, ok := finalModel.msgs[5].(RunExhaustedMsg)
	if !ok {
		t.Fatalf("msg 5 type = %T, want RunExhaustedMsg", finalModel.msgs[5])
	}
	if runExhausted.Err != exhaustedErr {
		t.Fatalf("unexpected RunExhaustedMsg err pointer: got %v want %v", runExhausted.Err, exhaustedErr)
	}
}

type testErr struct {
	msg string
}

func (e *testErr) Error() string {
	return e.msg
}
