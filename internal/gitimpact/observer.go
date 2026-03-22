package gitimpact

// Observer receives lifecycle callbacks from the git-impact engine.
type Observer interface {
	TurnStarted(phase Phase, iteration int)
	PhaseAdvanced(from, to Phase)
	WaitEntered(message string)
	WaitResolved(response string)
	RunCompleted(result *AnalysisResult)
	RunExhausted(err error)
}
