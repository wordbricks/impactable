package gitimpact

// Phase represents a lifecycle stage in the git-impact analysis run.
type Phase string

const (
	PhaseSourceCheck Phase = "source_check"
	PhaseCollect     Phase = "collect"
	PhaseLink        Phase = "link"
	PhaseScore       Phase = "score"
	PhaseReport      Phase = "report"
)

// AnalysisResult is the terminal analysis payload produced by a completed run.
// Fields will be expanded in later milestones as report data is introduced.
type AnalysisResult struct{}
