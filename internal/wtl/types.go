package wtl

import "time"

const (
	commandRun = "run"

	defaultModel         = "gpt-5.3-codex"
	defaultMaxIterations = 20
	defaultMaxRetry      = 3

	defaultApprovalPolicy = "never"
	defaultSandbox        = "workspace-write"

	completionMarker = "##WTL_DONE##"
)

type directive string

const (
	directiveContinue    directive = "continue"
	directiveWait        directive = "wait"
	directiveRetry       directive = "retry"
	directiveCompact     directive = "compact"
	directiveAdvance     directive = "advance_phase"
	directiveComplete    directive = "complete"
	directiveUnsupported directive = "unsupported"
)

type runStatus string

const (
	statusCompleted   runStatus = "completed"
	statusExhausted   runStatus = "exhausted"
	statusInterrupted runStatus = "interrupted"
	statusFailed      runStatus = "failed"
)

type eventType string

const (
	eventRunStarted     eventType = "run.started"
	eventPhaseChanged   eventType = "phase.changed"
	eventTurnStarted    eventType = "turn.started"
	eventTurnDelta      eventType = "turn.delta"
	eventTurnFinished   eventType = "turn.finished"
	eventWaitEntered    eventType = "run.wait_entered"
	eventRunCompleted   eventType = "run.completed"
	eventRunExhausted   eventType = "run.exhausted"
	eventRunInterrupted eventType = "run.interrupted"
	eventRunFailed      eventType = "run.failed"
)

type request struct {
	Command  string
	MaxIter  int
	MaxRetry int
	Output   string
	Model    string
}

type runConfig struct {
	CWD      string
	Model    string
	MaxIter  int
	MaxRetry int
}

type turnResult struct {
	TurnID       string
	Status       string
	Response     string
	ErrorMessage string
}

type turnOutcome struct {
	Response string
	Err      error
	Status   string
}

type runSummary struct {
	RunID          string    `json:"run_id,omitempty"`
	ThreadID       string    `json:"thread_id,omitempty"`
	Status         runStatus `json:"status"`
	Iterations     int       `json:"iterations,omitempty"`
	MaxIterations  int       `json:"max_iterations,omitempty"`
	MaxRetry       int       `json:"max_retry,omitempty"`
	FinalResponse  string    `json:"final_response,omitempty"`
	ExhaustedBy    string    `json:"exhausted_by,omitempty"`
	LastDirective  directive `json:"last_directive,omitempty"`
	LastError      string    `json:"last_error,omitempty"`
	LastPhase      string    `json:"phase,omitempty"`
	CompletedAtUTC string    `json:"completed_at,omitempty"`
}

type runEvent struct {
	Command    string    `json:"command"`
	Event      eventType `json:"event"`
	RunID      string    `json:"run_id,omitempty"`
	ThreadID   string    `json:"thread_id,omitempty"`
	Phase      string    `json:"phase,omitempty"`
	Iteration  int       `json:"iteration,omitempty"`
	Status     runStatus `json:"status,omitempty"`
	Directive  directive `json:"directive,omitempty"`
	Text       string    `json:"text,omitempty"`
	Response   string    `json:"response,omitempty"`
	Reason     string    `json:"reason,omitempty"`
	Error      string    `json:"error,omitempty"`
	OccurredAt string    `json:"ts"`
}

func newEvent(kind eventType) runEvent {
	return runEvent{
		Command:    commandRun,
		Event:      kind,
		OccurredAt: time.Now().UTC().Format(time.RFC3339),
	}
}

type structuredError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
