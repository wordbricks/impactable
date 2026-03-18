package gitimpact

const (
	commandAnalyze        = "analyze"
	commandCheckSources   = "check-sources"
	commandReportScaffold = "report-scaffold"
	commandSchema         = "schema"

	defaultConfigPath = "impact-analyzer.yaml"
)

var knownCommands = map[string]struct{}{
	commandAnalyze:        {},
	commandCheckSources:   {},
	commandReportScaffold: {},
	commandSchema:         {},
}

type analyzeRequest struct {
	Command    string `json:"command,omitempty"`
	ConfigPath string `json:"config,omitempty"`
	PRNumber   int    `json:"pr,omitempty"`
	Since      string `json:"since,omitempty"`
	Output     string `json:"output,omitempty"`
	OutputFile string `json:"output_file,omitempty"`
}

type checkSourcesRequest struct {
	Command       string   `json:"command,omitempty"`
	ConfigPath    string   `json:"config,omitempty"`
	RequiredRoles []string `json:"required_roles,omitempty"`
	Output        string   `json:"output,omitempty"`
	OutputFile    string   `json:"output_file,omitempty"`
}

type reportScaffoldRequest struct {
	Command    string   `json:"command,omitempty"`
	ConfigPath string   `json:"config,omitempty"`
	Modes      []string `json:"modes,omitempty"`
	OutputDir  string   `json:"output_dir,omitempty"`
	Output     string   `json:"output,omitempty"`
	OutputFile string   `json:"output_file,omitempty"`
}

type schemaRequest struct {
	TargetCommand string `json:"target_command,omitempty"`
	CommandName   string `json:"command_name,omitempty"` // Legacy alias for payload compatibility.
	Output        string `json:"output,omitempty"`
	OutputFile    string `json:"output_file,omitempty"`
}

type parsedCommand struct {
	Kind           string
	Analyze        analyzeRequest
	CheckSources   checkSourcesRequest
	ReportScaffold reportScaffoldRequest
	Schema         schemaRequest
}

type structuredError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type sourceCheckContract struct {
	Role     string `json:"role"`
	Status   string `json:"status"`
	Provider string `json:"provider,omitempty"`
	Message  string `json:"message,omitempty"`
}

type reportScaffoldContract struct {
	Mode   string `json:"mode"`
	Status string `json:"status"`
	Path   string `json:"path,omitempty"`
}
