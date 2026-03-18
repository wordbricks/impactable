package ralphloop

type argumentDescriptor struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
	Required    bool   `json:"required,omitempty"`
}

type optionDescriptor struct {
	Name           string   `json:"name"`
	Aliases        []string `json:"aliases,omitempty"`
	Description    string   `json:"description"`
	Type           string   `json:"type"`
	Required       bool     `json:"required,omitempty"`
	Default        any      `json:"default,omitempty"`
	Enum           []string `json:"enum,omitempty"`
	SupportsDryRun bool     `json:"supports_dry_run,omitempty"`
}

type payloadSchema struct {
	Type       string                 `json:"type"`
	Properties map[string]propertyDef `json:"properties"`
	Required   []string               `json:"required,omitempty"`
}

type propertyDef struct {
	Type        string       `json:"type"`
	Description string       `json:"description"`
	Default     any          `json:"default,omitempty"`
	Enum        []string     `json:"enum,omitempty"`
	Items       *propertyDef `json:"items,omitempty"`
}

type commandDescriptor struct {
	Name             string               `json:"name"`
	Description      string               `json:"description"`
	Positionals      []argumentDescriptor `json:"positionals,omitempty"`
	Options          []optionDescriptor   `json:"options"`
	MutatesState     bool                 `json:"mutates_state"`
	SupportsDryRun   bool                 `json:"supports_dry_run"`
	RawPayloadSchema payloadSchema        `json:"raw_payload_schema"`
}

func commandDescriptors() []commandDescriptor {
	baseReadOptions := []optionDescriptor{
		{Name: "--json", Description: "Raw JSON payload or - for stdin", Type: "string"},
		{Name: "--output", Description: "Output format", Type: "string", Default: "text (tty) / json (non-tty)", Enum: []string{"text", "json", "ndjson"}},
		{Name: "--output-file", Description: "Write the result to a file under the current working directory", Type: "string"},
		{Name: "--fields", Description: "Comma-separated field mask", Type: "string"},
		{Name: "--page", Description: "Page number", Type: "integer", Default: defaultPage},
		{Name: "--page-size", Description: "Page size", Type: "integer", Default: defaultPageSize},
		{Name: "--page-all", Description: "Return all pages", Type: "boolean", Default: false},
	}

	return []commandDescriptor{
		{
			Name:        commandMain,
			Description: "Run the full Ralph Loop lifecycle against a user prompt.",
			Positionals: []argumentDescriptor{{Name: "prompt", Description: "User prompt to execute", Type: "string", Required: true}},
			Options: append([]optionDescriptor{}, append(baseReadOptions,
				optionDescriptor{Name: "--model", Description: "Codex model", Type: "string", Default: defaultModel},
				optionDescriptor{Name: "--base-branch", Description: "Base branch", Type: "string", Default: defaultBaseBranch},
				optionDescriptor{Name: "--max-iterations", Description: "Safety cap", Type: "integer", Default: defaultMaxIterations},
				optionDescriptor{Name: "--work-branch", Description: "Working branch", Type: "string"},
				optionDescriptor{Name: "--timeout", Description: "Timeout in seconds", Type: "integer", Default: defaultTimeoutSeconds},
				optionDescriptor{Name: "--approval-policy", Description: "Codex approval policy", Type: "string", Default: defaultApprovalPolicy},
				optionDescriptor{Name: "--sandbox", Description: "Codex sandbox policy", Type: "string", Default: defaultSandbox},
				optionDescriptor{Name: "--preserve-worktree", Description: "Keep worktree on exit", Type: "boolean", Default: false},
				optionDescriptor{Name: "--dry-run", Description: "Validate only", Type: "boolean", Default: false, SupportsDryRun: true},
			)...),
			MutatesState:   true,
			SupportsDryRun: true,
			RawPayloadSchema: payloadSchema{
				Type: "object",
				Properties: map[string]propertyDef{
					"command":           {Type: "string", Description: "Command name", Default: commandMain},
					"prompt":            {Type: "string", Description: "User prompt"},
					"model":             {Type: "string", Description: "Codex model", Default: defaultModel},
					"base_branch":       {Type: "string", Description: "Base branch", Default: defaultBaseBranch},
					"max_iterations":    {Type: "integer", Description: "Safety cap", Default: defaultMaxIterations},
					"work_branch":       {Type: "string", Description: "Working branch"},
					"timeout":           {Type: "integer", Description: "Timeout in seconds", Default: defaultTimeoutSeconds},
					"approval_policy":   {Type: "string", Description: "Codex approval policy", Default: defaultApprovalPolicy},
					"sandbox":           {Type: "string", Description: "Codex sandbox policy", Default: defaultSandbox},
					"preserve_worktree": {Type: "boolean", Description: "Keep worktree on exit", Default: false},
					"dry_run":           {Type: "boolean", Description: "Validate only", Default: false},
					"output":            {Type: "string", Description: "Output format", Enum: []string{"text", "json", "ndjson"}},
					"output_file":       {Type: "string", Description: "Output file path"},
					"fields":            {Type: "array", Description: "Field mask", Items: &propertyDef{Type: "string", Description: "Field name"}},
					"page":              {Type: "integer", Description: "Page number", Default: defaultPage},
					"page_size":         {Type: "integer", Description: "Page size", Default: defaultPageSize},
					"page_all":          {Type: "boolean", Description: "Return all pages", Default: false},
				},
				Required: []string{"prompt"},
			},
		},
		{
			Name:        commandInit,
			Description: "Create or reuse a worktree, install dependencies, and verify the repository build.",
			Options: append([]optionDescriptor{}, append(baseReadOptions,
				optionDescriptor{Name: "--base-branch", Description: "Base branch", Type: "string", Default: defaultBaseBranch},
				optionDescriptor{Name: "--work-branch", Description: "Working branch", Type: "string"},
				optionDescriptor{Name: "--dry-run", Description: "Validate only", Type: "boolean", Default: false, SupportsDryRun: true},
			)...),
			MutatesState:   true,
			SupportsDryRun: true,
			RawPayloadSchema: payloadSchema{
				Type: "object",
				Properties: map[string]propertyDef{
					"command":     {Type: "string", Description: "Command name", Default: commandInit},
					"base_branch": {Type: "string", Description: "Base branch", Default: defaultBaseBranch},
					"work_branch": {Type: "string", Description: "Working branch"},
					"dry_run":     {Type: "boolean", Description: "Validate only", Default: false},
					"output":      {Type: "string", Description: "Output format", Enum: []string{"text", "json", "ndjson"}},
					"output_file": {Type: "string", Description: "Output file path"},
					"fields":      {Type: "array", Description: "Field mask", Items: &propertyDef{Type: "string", Description: "Field name"}},
					"page":        {Type: "integer", Description: "Page number", Default: defaultPage},
					"page_size":   {Type: "integer", Description: "Page size", Default: defaultPageSize},
					"page_all":    {Type: "boolean", Description: "Return all pages", Default: false},
				},
			},
		},
		{
			Name:           commandList,
			Description:    "List active Ralph Loop sessions discovered under repository worktree runtime roots.",
			Positionals:    []argumentDescriptor{{Name: "selector", Description: "Optional selector", Type: "string"}},
			Options:        baseReadOptions,
			MutatesState:   false,
			SupportsDryRun: false,
			RawPayloadSchema: payloadSchema{
				Type: "object",
				Properties: map[string]propertyDef{
					"command":     {Type: "string", Description: "Command name", Default: commandList},
					"selector":    {Type: "string", Description: "Optional selector"},
					"output":      {Type: "string", Description: "Output format", Enum: []string{"text", "json", "ndjson"}},
					"output_file": {Type: "string", Description: "Output file path"},
					"fields":      {Type: "array", Description: "Field mask", Items: &propertyDef{Type: "string", Description: "Field name"}},
					"page":        {Type: "integer", Description: "Page number", Default: defaultPage},
					"page_size":   {Type: "integer", Description: "Page size", Default: defaultPageSize},
					"page_all":    {Type: "boolean", Description: "Return all pages", Default: false},
				},
			},
		},
		{
			Name:        commandTail,
			Description: "Read or follow a Ralph Loop log file.",
			Positionals: []argumentDescriptor{{Name: "selector", Description: "Optional selector", Type: "string"}},
			Options: append([]optionDescriptor{}, append(baseReadOptions,
				optionDescriptor{Name: "--lines", Aliases: []string{"-n"}, Description: "Number of lines", Type: "integer", Default: defaultTailLines},
				optionDescriptor{Name: "--follow", Aliases: []string{"-f"}, Description: "Follow appended log lines", Type: "boolean", Default: false},
				optionDescriptor{Name: "--raw", Description: "Do not render pretty summaries", Type: "boolean", Default: false},
			)...),
			MutatesState:   false,
			SupportsDryRun: false,
			RawPayloadSchema: payloadSchema{
				Type: "object",
				Properties: map[string]propertyDef{
					"command":     {Type: "string", Description: "Command name", Default: commandTail},
					"selector":    {Type: "string", Description: "Optional selector"},
					"lines":       {Type: "integer", Description: "Number of lines", Default: defaultTailLines},
					"follow":      {Type: "boolean", Description: "Follow appended log lines", Default: false},
					"raw":         {Type: "boolean", Description: "Return raw log lines", Default: false},
					"output":      {Type: "string", Description: "Output format", Enum: []string{"text", "json", "ndjson"}},
					"output_file": {Type: "string", Description: "Output file path"},
					"fields":      {Type: "array", Description: "Field mask", Items: &propertyDef{Type: "string", Description: "Field name"}},
					"page":        {Type: "integer", Description: "Page number", Default: defaultPage},
					"page_size":   {Type: "integer", Description: "Page size", Default: defaultPageSize},
					"page_all":    {Type: "boolean", Description: "Return all pages", Default: false},
				},
			},
		},
		{
			Name:        commandSchema,
			Description: "Describe the live command surface in machine-readable form.",
			Positionals: []argumentDescriptor{{Name: "command", Description: "Optional command to describe", Type: "string"}},
			Options: append([]optionDescriptor{}, append(baseReadOptions,
				optionDescriptor{Name: "--command", Description: "Command name", Type: "string"},
			)...),
			MutatesState:   false,
			SupportsDryRun: false,
			RawPayloadSchema: payloadSchema{
				Type: "object",
				Properties: map[string]propertyDef{
					"command":        {Type: "string", Description: "Command name", Default: commandSchema},
					"target_command": {Type: "string", Description: "Optional command to describe"},
					"command_name":   {Type: "string", Description: "Legacy alias for target command"},
					"output":         {Type: "string", Description: "Output format", Enum: []string{"text", "json", "ndjson"}},
					"output_file":    {Type: "string", Description: "Output file path"},
					"fields":         {Type: "array", Description: "Field mask", Items: &propertyDef{Type: "string", Description: "Field name"}},
					"page":           {Type: "integer", Description: "Page number", Default: defaultPage},
					"page_size":      {Type: "integer", Description: "Page size", Default: defaultPageSize},
					"page_all":       {Type: "boolean", Description: "Return all pages", Default: false},
				},
			},
		},
	}
}

func commandDescriptorByName(name string) (commandDescriptor, bool) {
	for _, descriptor := range commandDescriptors() {
		if descriptor.Name == name {
			return descriptor, true
		}
	}
	return commandDescriptor{}, false
}

func commandOptionSet(command string) map[string]struct{} {
	descriptor, ok := commandDescriptorByName(command)
	if !ok {
		return map[string]struct{}{}
	}
	options := map[string]struct{}{
		"--help": {},
		"-h":     {},
	}
	for _, option := range descriptor.Options {
		options[option.Name] = struct{}{}
		for _, alias := range option.Aliases {
			options[alias] = struct{}{}
		}
	}
	return options
}
