package gitimpact

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

var errHelpRequested = errors.New("help requested")

type parseState struct {
	command       string
	seenOptions   map[string]struct{}
	jsonSource    *string
	output        *string
	outputFile    *string
	configPath    *string
	prNumber      *int
	since         *string
	requiredRoles []string
	reportModes   []string
	reportDir     *string
	commandName   *string
	positionals   []string
}

func parseCLI(args []string, stdin io.Reader) (parsedCommand, error) {
	if len(args) == 0 {
		return parsedCommand{}, fmt.Errorf("command is required (analyze, check-sources, report-scaffold, schema)")
	}

	state := parseState{seenOptions: map[string]struct{}{}}
	if args[0] == "--help" || args[0] == "-h" {
		return parsedCommand{}, errHelpRequested
	}
	state.command = strings.TrimSpace(args[0])
	if _, ok := knownCommands[state.command]; !ok {
		return parsedCommand{}, fmt.Errorf("unknown command %q", state.command)
	}
	args = args[1:]

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--help", "-h":
			return parsedCommand{}, errHelpRequested
		case "--json":
			state.seenOptions[arg] = struct{}{}
			value, err := requireValue(args, &i, arg)
			if err != nil {
				return parsedCommand{}, err
			}
			state.jsonSource = &value
		case "--output":
			state.seenOptions[arg] = struct{}{}
			value, err := requireValue(args, &i, arg)
			if err != nil {
				return parsedCommand{}, err
			}
			state.output = &value
		case "--output-file":
			state.seenOptions[arg] = struct{}{}
			value, err := requireValue(args, &i, arg)
			if err != nil {
				return parsedCommand{}, err
			}
			state.outputFile = &value
		case "--config":
			state.seenOptions[arg] = struct{}{}
			value, err := requireValue(args, &i, arg)
			if err != nil {
				return parsedCommand{}, err
			}
			state.configPath = &value
		case "--pr":
			state.seenOptions[arg] = struct{}{}
			value, err := requireIntValue(args, &i, arg)
			if err != nil {
				return parsedCommand{}, err
			}
			if value <= 0 {
				return parsedCommand{}, fmt.Errorf("%s requires a positive integer value", arg)
			}
			state.prNumber = &value
		case "--since":
			state.seenOptions[arg] = struct{}{}
			value, err := requireValue(args, &i, arg)
			if err != nil {
				return parsedCommand{}, err
			}
			state.since = &value
		case "--require":
			state.seenOptions[arg] = struct{}{}
			value, err := requireValue(args, &i, arg)
			if err != nil {
				return parsedCommand{}, err
			}
			state.requiredRoles = append(state.requiredRoles, parseCSV(value)...)
		case "--mode":
			state.seenOptions[arg] = struct{}{}
			value, err := requireValue(args, &i, arg)
			if err != nil {
				return parsedCommand{}, err
			}
			state.reportModes = append(state.reportModes, strings.TrimSpace(value))
		case "--output-dir":
			state.seenOptions[arg] = struct{}{}
			value, err := requireValue(args, &i, arg)
			if err != nil {
				return parsedCommand{}, err
			}
			state.reportDir = &value
		case "--command":
			state.seenOptions[arg] = struct{}{}
			value, err := requireValue(args, &i, arg)
			if err != nil {
				return parsedCommand{}, err
			}
			state.commandName = &value
		default:
			if strings.HasPrefix(arg, "-") {
				return parsedCommand{}, fmt.Errorf("unknown option %q", arg)
			}
			state.positionals = append(state.positionals, arg)
		}
	}

	if err := validateOptionsForCommand(state); err != nil {
		return parsedCommand{}, err
	}

	if state.jsonSource != nil {
		body, err := readJSONPayload(stdin, *state.jsonSource)
		if err != nil {
			return parsedCommand{}, err
		}
		jsonCommand, err := commandFromJSON(body)
		if err != nil {
			return parsedCommand{}, err
		}
		if jsonCommand != "" && jsonCommand != state.command {
			return parsedCommand{}, fmt.Errorf("json payload command %q does not match selected command %q", jsonCommand, state.command)
		}
		return buildFromJSON(state, body)
	}
	return buildFromFlags(state)
}

func buildFromJSON(state parseState, body []byte) (parsedCommand, error) {
	switch state.command {
	case commandAnalyze:
		req := analyzeRequest{Command: commandAnalyze, ConfigPath: defaultConfigPath}
		if err := json.Unmarshal(body, &req); err != nil {
			return parsedCommand{}, err
		}
		applyBaseOverrides(&req.Output, &req.OutputFile, state)
		if state.configPath != nil {
			req.ConfigPath = *state.configPath
		}
		if state.prNumber != nil {
			req.PRNumber = *state.prNumber
		}
		if state.since != nil {
			req.Since = *state.since
		}
		if err := validateAnalyzeRequest(req, state.positionals); err != nil {
			return parsedCommand{}, err
		}
		return parsedCommand{Kind: commandAnalyze, Analyze: req}, nil
	case commandCheckSources:
		req := checkSourcesRequest{Command: commandCheckSources, ConfigPath: defaultConfigPath, RequiredRoles: defaultRequiredRoles()}
		if err := json.Unmarshal(body, &req); err != nil {
			return parsedCommand{}, err
		}
		applyBaseOverrides(&req.Output, &req.OutputFile, state)
		if state.configPath != nil {
			req.ConfigPath = *state.configPath
		}
		if len(state.requiredRoles) > 0 {
			req.RequiredRoles = state.requiredRoles
		}
		if err := validateCheckSourcesRequest(req, state.positionals); err != nil {
			return parsedCommand{}, err
		}
		return parsedCommand{Kind: commandCheckSources, CheckSources: req}, nil
	case commandReportScaffold:
		req := reportScaffoldRequest{Command: commandReportScaffold, ConfigPath: defaultConfigPath, Modes: defaultReportModes(), OutputDir: "reports"}
		if err := json.Unmarshal(body, &req); err != nil {
			return parsedCommand{}, err
		}
		applyBaseOverrides(&req.Output, &req.OutputFile, state)
		if state.configPath != nil {
			req.ConfigPath = *state.configPath
		}
		if len(state.reportModes) > 0 {
			req.Modes = state.reportModes
		}
		if state.reportDir != nil {
			req.OutputDir = *state.reportDir
		}
		if err := validateReportScaffoldRequest(req, state.positionals); err != nil {
			return parsedCommand{}, err
		}
		return parsedCommand{Kind: commandReportScaffold, ReportScaffold: req}, nil
	case commandSchema:
		req := schemaRequest{}
		if err := json.Unmarshal(body, &req); err != nil {
			return parsedCommand{}, err
		}
		applyBaseOverrides(&req.Output, &req.OutputFile, state)
		if state.commandName != nil {
			req.TargetCommand = *state.commandName
		} else if len(state.positionals) > 0 {
			req.TargetCommand = state.positionals[0]
		}
		if strings.TrimSpace(req.TargetCommand) == "" && strings.TrimSpace(req.CommandName) != "" {
			req.TargetCommand = req.CommandName
		}
		if err := validateSchemaRequest(req, state.positionals); err != nil {
			return parsedCommand{}, err
		}
		return parsedCommand{Kind: commandSchema, Schema: req}, nil
	default:
		return parsedCommand{}, fmt.Errorf("unknown command %q", state.command)
	}
}

func buildFromFlags(state parseState) (parsedCommand, error) {
	switch state.command {
	case commandAnalyze:
		req := analyzeRequest{Command: commandAnalyze, ConfigPath: defaultConfigPath}
		applyBaseOverrides(&req.Output, &req.OutputFile, state)
		if state.configPath != nil {
			req.ConfigPath = *state.configPath
		}
		if state.prNumber != nil {
			req.PRNumber = *state.prNumber
		}
		if state.since != nil {
			req.Since = *state.since
		}
		if err := validateAnalyzeRequest(req, state.positionals); err != nil {
			return parsedCommand{}, err
		}
		return parsedCommand{Kind: commandAnalyze, Analyze: req}, nil
	case commandCheckSources:
		req := checkSourcesRequest{Command: commandCheckSources, ConfigPath: defaultConfigPath, RequiredRoles: defaultRequiredRoles()}
		applyBaseOverrides(&req.Output, &req.OutputFile, state)
		if state.configPath != nil {
			req.ConfigPath = *state.configPath
		}
		if len(state.requiredRoles) > 0 {
			req.RequiredRoles = state.requiredRoles
		}
		if err := validateCheckSourcesRequest(req, state.positionals); err != nil {
			return parsedCommand{}, err
		}
		return parsedCommand{Kind: commandCheckSources, CheckSources: req}, nil
	case commandReportScaffold:
		req := reportScaffoldRequest{Command: commandReportScaffold, ConfigPath: defaultConfigPath, Modes: defaultReportModes(), OutputDir: "reports"}
		applyBaseOverrides(&req.Output, &req.OutputFile, state)
		if state.configPath != nil {
			req.ConfigPath = *state.configPath
		}
		if len(state.reportModes) > 0 {
			req.Modes = state.reportModes
		}
		if state.reportDir != nil {
			req.OutputDir = *state.reportDir
		}
		if err := validateReportScaffoldRequest(req, state.positionals); err != nil {
			return parsedCommand{}, err
		}
		return parsedCommand{Kind: commandReportScaffold, ReportScaffold: req}, nil
	case commandSchema:
		req := schemaRequest{}
		applyBaseOverrides(&req.Output, &req.OutputFile, state)
		if state.commandName != nil {
			req.TargetCommand = *state.commandName
		} else if len(state.positionals) > 0 {
			req.TargetCommand = state.positionals[0]
		}
		if err := validateSchemaRequest(req, state.positionals); err != nil {
			return parsedCommand{}, err
		}
		return parsedCommand{Kind: commandSchema, Schema: req}, nil
	default:
		return parsedCommand{}, fmt.Errorf("unknown command %q", state.command)
	}
}

func validateOptionsForCommand(state parseState) error {
	allowed := commandOptionSet(state.command)
	for option := range state.seenOptions {
		if _, ok := allowed[option]; !ok {
			return fmt.Errorf("%s does not support %s", state.command, option)
		}
	}
	return nil
}

func validateAnalyzeRequest(req analyzeRequest, positionals []string) error {
	if len(positionals) > 0 {
		return fmt.Errorf("analyze does not accept positional arguments")
	}
	if err := validateBaseRequest(req.Output); err != nil {
		return err
	}
	if strings.TrimSpace(req.ConfigPath) == "" {
		req.ConfigPath = defaultConfigPath
	}
	if req.PRNumber < 0 {
		return fmt.Errorf("pr must be a positive integer")
	}
	return nil
}

func validateCheckSourcesRequest(req checkSourcesRequest, positionals []string) error {
	if len(positionals) > 0 {
		return fmt.Errorf("check-sources does not accept positional arguments")
	}
	if err := validateBaseRequest(req.Output); err != nil {
		return err
	}
	if len(req.RequiredRoles) == 0 {
		return fmt.Errorf("check-sources requires at least one role")
	}
	for _, role := range req.RequiredRoles {
		if strings.TrimSpace(role) == "" {
			return fmt.Errorf("check-sources role cannot be blank")
		}
	}
	return nil
}

func validateReportScaffoldRequest(req reportScaffoldRequest, positionals []string) error {
	if len(positionals) > 0 {
		return fmt.Errorf("report-scaffold does not accept positional arguments")
	}
	if err := validateBaseRequest(req.Output); err != nil {
		return err
	}
	if strings.TrimSpace(req.OutputDir) == "" {
		return fmt.Errorf("report-scaffold requires --output-dir")
	}
	if _, err := normalizeModes(req.Modes); err != nil {
		return err
	}
	return nil
}

func validateSchemaRequest(req schemaRequest, positionals []string) error {
	if len(positionals) > 1 {
		return fmt.Errorf("schema accepts at most one command positional argument")
	}
	if err := validateBaseRequest(req.Output); err != nil {
		return err
	}
	target := strings.TrimSpace(req.TargetCommand)
	if target == "" {
		target = strings.TrimSpace(req.CommandName)
	}
	if target != "" {
		if _, ok := commandDescriptorByName(target); !ok {
			return fmt.Errorf("unknown command %q", target)
		}
	}
	return nil
}

func validateBaseRequest(output string) error {
	if output == "" {
		return nil
	}
	switch strings.TrimSpace(output) {
	case "text", "json", "ndjson":
		return nil
	default:
		return fmt.Errorf("invalid --output value %q (expected text, json, or ndjson)", output)
	}
}

func applyBaseOverrides(output *string, outputFile *string, state parseState) {
	if state.output != nil {
		*output = *state.output
	}
	if state.outputFile != nil {
		*outputFile = *state.outputFile
	}
}

func requireValue(args []string, index *int, name string) (string, error) {
	if *index+1 >= len(args) {
		return "", fmt.Errorf("%s requires a value", name)
	}
	*index++
	return args[*index], nil
}

func requireIntValue(args []string, index *int, name string) (int, error) {
	value, err := requireValue(args, index, name)
	if err != nil {
		return 0, err
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s requires an integer value", name)
	}
	return parsed, nil
}

func readJSONPayload(stdin io.Reader, source string) ([]byte, error) {
	if source == "-" {
		return io.ReadAll(stdin)
	}
	return []byte(source), nil
}

func commandFromJSON(body []byte) (string, error) {
	record := map[string]any{}
	if err := json.Unmarshal(body, &record); err != nil {
		return "", err
	}
	command, _ := record["command"].(string)
	command = strings.TrimSpace(command)
	if command == "" {
		return "", nil
	}
	if _, ok := knownCommands[command]; !ok {
		return "", fmt.Errorf("unknown command %q in JSON payload", command)
	}
	return command, nil
}

func parseCSV(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	roles := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			roles = append(roles, trimmed)
		}
	}
	return roles
}

func defaultRequiredRoles() []string {
	return []string{"github", "warehouse", "analytics"}
}

func defaultReportModes() []string {
	return []string{"terminal", "json"}
}

func normalizeModes(modes []string) ([]string, error) {
	if len(modes) == 0 {
		modes = defaultReportModes()
	}
	seen := map[string]struct{}{}
	normalized := make([]string, 0, len(modes))
	for _, mode := range modes {
		trimmed := strings.TrimSpace(strings.ToLower(mode))
		switch trimmed {
		case "terminal", "json", "markdown", "html":
		default:
			return nil, fmt.Errorf("unsupported report mode %q", mode)
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}
	if len(normalized) == 0 {
		return nil, fmt.Errorf("at least one report mode is required")
	}
	return normalized, nil
}
