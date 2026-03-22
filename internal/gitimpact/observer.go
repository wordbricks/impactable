package gitimpact

// Observer receives run lifecycle notifications from the phased engine.
type Observer interface {
	OnTurnStarted(phase Phase, iteration int)
	OnPhaseAdvanced(from, to Phase)
	OnWaitEntered(message string)
	OnWaitResolved(response string)
	OnRunCompleted(result *AnalysisResult)
	OnRunExhausted(err error)
}

// WaitHandler requests external input when the engine enters wait state.
type WaitHandler func(message string) (string, error)
