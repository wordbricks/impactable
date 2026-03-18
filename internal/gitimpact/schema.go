package gitimpact

type argumentDescriptor struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
	Required    bool   `json:"required,omitempty"`
}

type optionDescriptor struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Type        string   `json:"type"`
	Aliases     []string `json:"aliases,omitempty"`
	Required    bool     `json:"required,omitempty"`
	Default     any      `json:"default,omitempty"`
	Enum        []string `json:"enum,omitempty"`
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
	RawPayloadSchema payloadSchema        `json:"raw_payload_schema"`
}

func commandDescriptors() []commandDescriptor {
	baseReadOptions := []optionDescriptor{
		{Name: "--json", Description: "Raw JSON payload or - for stdin", Type: "string"},
		{Name: "--output", Description: "Envelope output format", Type: "string", Default: "text (tty) / json (non-tty)", Enum: []string{"text", "json", "ndjson"}},
		{Name: "--output-file", Description: "Write envelope output to file under the current directory", Type: "string"},
	}

	return []commandDescriptor{
		{
			Name:        commandAnalyze,
			Description: "Analyze Git impact scope (foundation contract only in Phase 1 M1).",
			Options: append([]optionDescriptor{}, append(baseReadOptions,
				optionDescriptor{Name: "--config", Description: "Path to analyzer config", Type: "string", Default: defaultConfigPath},
				optionDescriptor{Name: "--pr", Description: "Single pull request number", Type: "integer"},
				optionDescriptor{Name: "--since", Description: "Lower bound date/time for candidate PR discovery", Type: "string"},
			)...),
			RawPayloadSchema: payloadSchema{
				Type: "object",
				Properties: map[string]propertyDef{
					"command":     {Type: "string", Description: "Command name", Default: commandAnalyze},
					"config":      {Type: "string", Description: "Path to analyzer config", Default: defaultConfigPath},
					"pr":          {Type: "integer", Description: "Single pull request number"},
					"since":       {Type: "string", Description: "Lower bound date/time for candidate PR discovery"},
					"output":      {Type: "string", Description: "Envelope output format", Enum: []string{"text", "json", "ndjson"}},
					"output_file": {Type: "string", Description: "Output file path"},
				},
			},
		},
		{
			Name:        commandCheckSources,
			Description: "Check required data-source contract for impact analysis (foundation contract only in Phase 1 M1).",
			Options: append([]optionDescriptor{}, append(baseReadOptions,
				optionDescriptor{Name: "--config", Description: "Path to analyzer config", Type: "string", Default: defaultConfigPath},
				optionDescriptor{Name: "--require", Description: "Comma-separated required source roles", Type: "string", Default: "github,warehouse,analytics"},
			)...),
			RawPayloadSchema: payloadSchema{
				Type: "object",
				Properties: map[string]propertyDef{
					"command":        {Type: "string", Description: "Command name", Default: commandCheckSources},
					"config":         {Type: "string", Description: "Path to analyzer config", Default: defaultConfigPath},
					"required_roles": {Type: "array", Description: "Required source roles", Items: &propertyDef{Type: "string", Description: "Source role"}},
					"output":         {Type: "string", Description: "Envelope output format", Enum: []string{"text", "json", "ndjson"}},
					"output_file":    {Type: "string", Description: "Output file path"},
				},
			},
		},
		{
			Name:        commandReportScaffold,
			Description: "Generate report output-mode scaffolding contract for downstream writers.",
			Options: append([]optionDescriptor{}, append(baseReadOptions,
				optionDescriptor{Name: "--config", Description: "Path to analyzer config", Type: "string", Default: defaultConfigPath},
				optionDescriptor{Name: "--mode", Description: "Requested report output mode", Type: "string", Enum: []string{"terminal", "json", "markdown", "html"}},
				optionDescriptor{Name: "--output-dir", Description: "Directory for generated report artifacts", Type: "string", Default: "reports"},
			)...),
			RawPayloadSchema: payloadSchema{
				Type: "object",
				Properties: map[string]propertyDef{
					"command":     {Type: "string", Description: "Command name", Default: commandReportScaffold},
					"config":      {Type: "string", Description: "Path to analyzer config", Default: defaultConfigPath},
					"modes":       {Type: "array", Description: "Requested report output modes", Items: &propertyDef{Type: "string", Description: "Output mode", Enum: []string{"terminal", "json", "markdown", "html"}}},
					"output_dir":  {Type: "string", Description: "Directory for report artifacts", Default: "reports"},
					"output":      {Type: "string", Description: "Envelope output format", Enum: []string{"text", "json", "ndjson"}},
					"output_file": {Type: "string", Description: "Output file path"},
				},
			},
		},
		{
			Name:        commandSchema,
			Description: "Describe the live git-impact command surface in machine-readable form.",
			Positionals: []argumentDescriptor{{Name: "command", Description: "Optional command to describe", Type: "string"}},
			Options: append([]optionDescriptor{}, append(baseReadOptions,
				optionDescriptor{Name: "--command", Description: "Command name", Type: "string"},
			)...),
			RawPayloadSchema: payloadSchema{
				Type: "object",
				Properties: map[string]propertyDef{
					"command":        {Type: "string", Description: "Command name", Default: commandSchema},
					"target_command": {Type: "string", Description: "Optional command to describe"},
					"command_name":   {Type: "string", Description: "Legacy alias for target command"},
					"output":         {Type: "string", Description: "Envelope output format", Enum: []string{"text", "json", "ndjson"}},
					"output_file":    {Type: "string", Description: "Output file path"},
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
