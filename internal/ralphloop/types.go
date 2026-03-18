package ralphloop

import "time"

const (
	commandMain   = "main"
	commandInit   = "init"
	commandList   = "ls"
	commandTail   = "tail"
	commandSchema = "schema"

	defaultModel          = "gpt-5.3-codex"
	defaultBaseBranch     = "main"
	defaultMaxIterations  = 20
	defaultTimeoutSeconds = 21600
	defaultApprovalPolicy = "never"
	defaultSandbox        = "workspace-write"
	defaultTailLines      = 40
	defaultPage           = 1
	defaultPageSize       = 50

	completeToken = "<promise>COMPLETE</promise>"
)

var knownCommands = map[string]struct{}{
	commandInit:   {},
	commandList:   {},
	commandTail:   {},
	commandSchema: {},
}

type baseRequest struct {
	Output     string   `json:"output,omitempty"`
	OutputFile string   `json:"output_file,omitempty"`
	Fields     []string `json:"fields,omitempty"`
	Page       int      `json:"page,omitempty"`
	PageSize   int      `json:"page_size,omitempty"`
	PageAll    bool     `json:"page_all,omitempty"`
}

type mainRequest struct {
	Command        string   `json:"command,omitempty"`
	Prompt         string   `json:"prompt,omitempty"`
	Model          string   `json:"model,omitempty"`
	BaseBranch     string   `json:"base_branch,omitempty"`
	MaxIterations  int      `json:"max_iterations,omitempty"`
	WorkBranch     string   `json:"work_branch,omitempty"`
	TimeoutSeconds int      `json:"timeout,omitempty"`
	ApprovalPolicy string   `json:"approval_policy,omitempty"`
	Sandbox        string   `json:"sandbox,omitempty"`
	PreserveTree   bool     `json:"preserve_worktree,omitempty"`
	DryRun         bool     `json:"dry_run,omitempty"`
	Output         string   `json:"output,omitempty"`
	OutputFile     string   `json:"output_file,omitempty"`
	Fields         []string `json:"fields,omitempty"`
	Page           int      `json:"page,omitempty"`
	PageSize       int      `json:"page_size,omitempty"`
	PageAll        bool     `json:"page_all,omitempty"`
}

type initRequest struct {
	Command    string   `json:"command,omitempty"`
	BaseBranch string   `json:"base_branch,omitempty"`
	WorkBranch string   `json:"work_branch,omitempty"`
	DryRun     bool     `json:"dry_run,omitempty"`
	Output     string   `json:"output,omitempty"`
	OutputFile string   `json:"output_file,omitempty"`
	Fields     []string `json:"fields,omitempty"`
	Page       int      `json:"page,omitempty"`
	PageSize   int      `json:"page_size,omitempty"`
	PageAll    bool     `json:"page_all,omitempty"`
}

type listRequest struct {
	Command    string   `json:"command,omitempty"`
	Selector   string   `json:"selector,omitempty"`
	Output     string   `json:"output,omitempty"`
	OutputFile string   `json:"output_file,omitempty"`
	Fields     []string `json:"fields,omitempty"`
	Page       int      `json:"page,omitempty"`
	PageSize   int      `json:"page_size,omitempty"`
	PageAll    bool     `json:"page_all,omitempty"`
}

type tailRequest struct {
	Command    string   `json:"command,omitempty"`
	Selector   string   `json:"selector,omitempty"`
	Lines      int      `json:"lines,omitempty"`
	Follow     bool     `json:"follow,omitempty"`
	Raw        bool     `json:"raw,omitempty"`
	Output     string   `json:"output,omitempty"`
	OutputFile string   `json:"output_file,omitempty"`
	Fields     []string `json:"fields,omitempty"`
	Page       int      `json:"page,omitempty"`
	PageSize   int      `json:"page_size,omitempty"`
	PageAll    bool     `json:"page_all,omitempty"`
}

type schemaRequest struct {
	CommandName string   `json:"command,omitempty"`
	Output      string   `json:"output,omitempty"`
	OutputFile  string   `json:"output_file,omitempty"`
	Fields      []string `json:"fields,omitempty"`
	Page        int      `json:"page,omitempty"`
	PageSize    int      `json:"page_size,omitempty"`
	PageAll     bool     `json:"page_all,omitempty"`
}

type parsedCommand struct {
	Kind   string
	Main   mainRequest
	Init   initRequest
	List   listRequest
	Tail   tailRequest
	Schema schemaRequest
}

type structuredError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type sessionRecord struct {
	Command      string `json:"command,omitempty"`
	Status       string `json:"status,omitempty"`
	PID          int    `json:"pid"`
	WorktreeID   string `json:"worktree_id,omitempty"`
	WorktreePath string `json:"worktree_path,omitempty"`
	WorkBranch   string `json:"work_branch,omitempty"`
	BaseBranch   string `json:"base_branch,omitempty"`
	RuntimeRoot  string `json:"runtime_root,omitempty"`
	LogPath      string `json:"log_path,omitempty"`
	StartedAt    string `json:"started_at,omitempty"`
}

type sessionView struct {
	sessionRecord
	RelativeWorktreePath string `json:"relative_worktree_path,omitempty"`
	RelativeLogPath      string `json:"relative_log_path,omitempty"`
}

type logRecord struct {
	Line       string `json:"line,omitempty"`
	Rendered   string `json:"rendered,omitempty"`
	Raw        bool   `json:"raw,omitempty"`
	LineNumber int    `json:"line_number,omitempty"`
}

type worktreeInfo struct {
	RepoRoot       string `json:"repo_root"`
	WorktreeID     string `json:"worktree_id"`
	WorktreePath   string `json:"worktree_path"`
	WorkBranch     string `json:"work_branch"`
	BaseBranch     string `json:"base_branch"`
	RuntimeRoot    string `json:"runtime_root"`
	LinkedWorktree bool   `json:"linked_worktree"`
	Reused         bool   `json:"reused"`
}

type projectCommands struct {
	ProjectType string   `json:"project_type"`
	Install     []string `json:"install"`
	Verify      []string `json:"verify"`
}

type planSummary struct {
	PlanPath string `json:"plan_path"`
	Name     string `json:"name"`
}

type runResult struct {
	Command      string           `json:"command"`
	Status       string           `json:"status"`
	Phase        string           `json:"phase,omitempty"`
	Iterations   int              `json:"iterations,omitempty"`
	WorktreeID   string           `json:"worktree_id,omitempty"`
	WorktreePath string           `json:"worktree_path,omitempty"`
	WorkBranch   string           `json:"work_branch,omitempty"`
	RuntimeRoot  string           `json:"runtime_root,omitempty"`
	LogPath      string           `json:"log_path,omitempty"`
	PlanPath     string           `json:"plan_path,omitempty"`
	PRURL        string           `json:"pr_url,omitempty"`
	Error        *structuredError `json:"error,omitempty"`
}

type jsonRPCNotification struct {
	Method string
	Params map[string]any
}

type commandEvent struct {
	Command      string           `json:"command"`
	Event        string           `json:"event"`
	Status       string           `json:"status,omitempty"`
	Phase        string           `json:"phase,omitempty"`
	Iteration    int              `json:"iteration,omitempty"`
	WorktreePath string           `json:"worktree_path,omitempty"`
	WorkBranch   string           `json:"work_branch,omitempty"`
	PlanPath     string           `json:"plan_path,omitempty"`
	LogPath      string           `json:"log_path,omitempty"`
	PRURL        string           `json:"pr_url,omitempty"`
	Message      string           `json:"message,omitempty"`
	Error        *structuredError `json:"error,omitempty"`
	TS           string           `json:"ts"`
}

func newEvent(command string, event string) commandEvent {
	return commandEvent{
		Command: command,
		Event:   event,
		TS:      time.Now().UTC().Format(time.RFC3339),
	}
}
