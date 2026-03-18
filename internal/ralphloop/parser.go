package ralphloop

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type parseState struct {
	command       string
	seenOptions   map[string]struct{}
	jsonSource    *string
	output        *string
	outputFile    *string
	fields        *string
	page          *int
	pageSize      *int
	pageAll       bool
	commandName   *string
	lines         *int
	follow        bool
	raw           bool
	model         *string
	baseBranch    *string
	maxIterations *int
	workBranch    *string
	timeout       *int
	approval      *string
	sandbox       *string
	preserve      bool
	dryRun        bool
	positionals   []string
}

func parseCLI(args []string, stdin io.Reader) (parsedCommand, error) {
	state := parseState{
		command:     commandMain,
		seenOptions: map[string]struct{}{},
	}
	if len(args) > 0 {
		if _, ok := knownCommands[args[0]]; ok {
			state.command = args[0]
			args = args[1:]
		}
	}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--help", "-h":
			return parsedCommand{}, fmt.Errorf("help requested")
		case "--json":
			state.seenOptions["--json"] = struct{}{}
			value, err := requireValue(args, &i, arg)
			if err != nil {
				return parsedCommand{}, err
			}
			state.jsonSource = &value
		case "--output":
			state.seenOptions["--output"] = struct{}{}
			value, err := requireValue(args, &i, arg)
			if err != nil {
				return parsedCommand{}, err
			}
			state.output = &value
		case "--output-file":
			state.seenOptions["--output-file"] = struct{}{}
			value, err := requireValue(args, &i, arg)
			if err != nil {
				return parsedCommand{}, err
			}
			state.outputFile = &value
		case "--fields":
			state.seenOptions["--fields"] = struct{}{}
			value, err := requireValue(args, &i, arg)
			if err != nil {
				return parsedCommand{}, err
			}
			state.fields = &value
		case "--page":
			state.seenOptions["--page"] = struct{}{}
			value, err := requireIntValue(args, &i, arg)
			if err != nil {
				return parsedCommand{}, err
			}
			if value <= 0 {
				return parsedCommand{}, fmt.Errorf("%s requires a positive integer value", arg)
			}
			state.page = &value
		case "--page-size":
			state.seenOptions["--page-size"] = struct{}{}
			value, err := requireIntValue(args, &i, arg)
			if err != nil {
				return parsedCommand{}, err
			}
			if value <= 0 {
				return parsedCommand{}, fmt.Errorf("%s requires a positive integer value", arg)
			}
			state.pageSize = &value
		case "--page-all":
			state.seenOptions["--page-all"] = struct{}{}
			state.pageAll = true
		case "--command":
			state.seenOptions["--command"] = struct{}{}
			value, err := requireValue(args, &i, arg)
			if err != nil {
				return parsedCommand{}, err
			}
			state.commandName = &value
		case "--lines", "-n":
			state.seenOptions["--lines"] = struct{}{}
			value, err := requireIntValue(args, &i, arg)
			if err != nil {
				return parsedCommand{}, err
			}
			if value <= 0 {
				return parsedCommand{}, fmt.Errorf("%s requires a positive integer value", arg)
			}
			state.lines = &value
		case "--follow", "-f":
			state.seenOptions["--follow"] = struct{}{}
			state.follow = true
		case "--raw":
			state.seenOptions["--raw"] = struct{}{}
			state.raw = true
		case "--model":
			state.seenOptions["--model"] = struct{}{}
			value, err := requireValue(args, &i, arg)
			if err != nil {
				return parsedCommand{}, err
			}
			state.model = &value
		case "--base-branch":
			state.seenOptions["--base-branch"] = struct{}{}
			value, err := requireValue(args, &i, arg)
			if err != nil {
				return parsedCommand{}, err
			}
			state.baseBranch = &value
		case "--max-iterations":
			state.seenOptions["--max-iterations"] = struct{}{}
			value, err := requireIntValue(args, &i, arg)
			if err != nil {
				return parsedCommand{}, err
			}
			if value <= 0 {
				return parsedCommand{}, fmt.Errorf("%s requires a positive integer value", arg)
			}
			state.maxIterations = &value
		case "--work-branch":
			state.seenOptions["--work-branch"] = struct{}{}
			value, err := requireValue(args, &i, arg)
			if err != nil {
				return parsedCommand{}, err
			}
			state.workBranch = &value
		case "--timeout":
			state.seenOptions["--timeout"] = struct{}{}
			value, err := requireIntValue(args, &i, arg)
			if err != nil {
				return parsedCommand{}, err
			}
			if value <= 0 {
				return parsedCommand{}, fmt.Errorf("%s requires a positive integer value", arg)
			}
			state.timeout = &value
		case "--approval-policy":
			state.seenOptions["--approval-policy"] = struct{}{}
			value, err := requireValue(args, &i, arg)
			if err != nil {
				return parsedCommand{}, err
			}
			state.approval = &value
		case "--sandbox":
			state.seenOptions["--sandbox"] = struct{}{}
			value, err := requireValue(args, &i, arg)
			if err != nil {
				return parsedCommand{}, err
			}
			state.sandbox = &value
		case "--preserve-worktree":
			state.seenOptions["--preserve-worktree"] = struct{}{}
			state.preserve = true
		case "--dry-run":
			state.seenOptions["--dry-run"] = struct{}{}
			state.dryRun = true
		default:
			if strings.HasPrefix(arg, "-") {
				return parsedCommand{}, fmt.Errorf("unknown option %q", arg)
			}
			state.positionals = append(state.positionals, arg)
		}
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
		if state.command == commandMain && jsonCommand != "" && jsonCommand != commandMain {
			state.command = jsonCommand
		}
		return buildFromJSON(state, body)
	}
	return buildFromFlags(state)
}

func buildFromJSON(state parseState, body []byte) (parsedCommand, error) {
	switch state.command {
	case commandInit:
		if err := validateOptionsForCommand(state, commandInit); err != nil {
			return parsedCommand{}, err
		}
		req := initRequest{
			Command:    commandInit,
			BaseBranch: defaultBaseBranch,
			WorkBranch: "",
			Output:     "",
			Page:       defaultPage,
			PageSize:   defaultPageSize,
		}
		if err := json.Unmarshal(body, &req); err != nil {
			return parsedCommand{}, err
		}
		applyBaseOverrides(&req.Output, &req.OutputFile, &req.Fields, &req.Page, &req.PageSize, &req.PageAll, state)
		if state.baseBranch != nil {
			req.BaseBranch = *state.baseBranch
		}
		if state.workBranch != nil {
			req.WorkBranch = *state.workBranch
		}
		if state.dryRun {
			req.DryRun = true
		}
		if strings.TrimSpace(req.BaseBranch) == "" {
			req.BaseBranch = defaultBaseBranch
		}
		if len(state.positionals) > 0 {
			return parsedCommand{}, fmt.Errorf("init does not accept positional arguments")
		}
		if err := validateBaseRequest(req.Output, req.Page, req.PageSize); err != nil {
			return parsedCommand{}, err
		}
		return parsedCommand{Kind: commandInit, Init: req}, nil
	case commandList:
		if err := validateOptionsForCommand(state, commandList); err != nil {
			return parsedCommand{}, err
		}
		req := listRequest{Command: commandList, Page: defaultPage, PageSize: defaultPageSize}
		if err := json.Unmarshal(body, &req); err != nil {
			return parsedCommand{}, err
		}
		applyBaseOverrides(&req.Output, &req.OutputFile, &req.Fields, &req.Page, &req.PageSize, &req.PageAll, state)
		if len(state.positionals) > 1 {
			return parsedCommand{}, fmt.Errorf("ls accepts at most one selector positional argument")
		}
		if strings.TrimSpace(req.Selector) == "" && len(state.positionals) == 1 {
			req.Selector = state.positionals[0]
		}
		if err := validateBaseRequest(req.Output, req.Page, req.PageSize); err != nil {
			return parsedCommand{}, err
		}
		return parsedCommand{Kind: commandList, List: req}, nil
	case commandTail:
		if err := validateOptionsForCommand(state, commandTail); err != nil {
			return parsedCommand{}, err
		}
		req := tailRequest{Command: commandTail, Lines: defaultTailLines, Page: defaultPage, PageSize: defaultPageSize}
		if err := json.Unmarshal(body, &req); err != nil {
			return parsedCommand{}, err
		}
		applyBaseOverrides(&req.Output, &req.OutputFile, &req.Fields, &req.Page, &req.PageSize, &req.PageAll, state)
		if len(state.positionals) > 1 {
			return parsedCommand{}, fmt.Errorf("tail accepts at most one selector positional argument")
		}
		if strings.TrimSpace(req.Selector) == "" && len(state.positionals) == 1 {
			req.Selector = state.positionals[0]
		}
		if state.lines != nil {
			req.Lines = *state.lines
		}
		if state.follow {
			req.Follow = true
		}
		if state.raw {
			req.Raw = true
		}
		if req.Lines <= 0 {
			req.Lines = defaultTailLines
		}
		if err := validateBaseRequest(req.Output, req.Page, req.PageSize); err != nil {
			return parsedCommand{}, err
		}
		return parsedCommand{Kind: commandTail, Tail: req}, nil
	case commandSchema:
		if err := validateOptionsForCommand(state, commandSchema); err != nil {
			return parsedCommand{}, err
		}
		req := schemaRequest{Page: defaultPage, PageSize: defaultPageSize}
		if err := json.Unmarshal(body, &req); err != nil {
			return parsedCommand{}, err
		}
		applyBaseOverrides(&req.Output, &req.OutputFile, &req.Fields, &req.Page, &req.PageSize, &req.PageAll, state)
		if state.commandName != nil {
			req.TargetCommand = *state.commandName
		} else if len(state.positionals) > 0 {
			req.TargetCommand = state.positionals[0]
		}
		if strings.TrimSpace(req.TargetCommand) == "" && strings.TrimSpace(req.CommandName) != "" {
			req.TargetCommand = req.CommandName
		}
		if len(state.positionals) > 1 {
			return parsedCommand{}, fmt.Errorf("schema accepts at most one command positional argument")
		}
		if strings.TrimSpace(req.TargetCommand) != "" {
			if _, ok := commandDescriptorByName(req.TargetCommand); !ok {
				return parsedCommand{}, fmt.Errorf("unknown command %q", req.TargetCommand)
			}
		}
		if err := validateBaseRequest(req.Output, req.Page, req.PageSize); err != nil {
			return parsedCommand{}, err
		}
		return parsedCommand{Kind: commandSchema, Schema: req}, nil
	default:
		if err := validateOptionsForCommand(state, commandMain); err != nil {
			return parsedCommand{}, err
		}
		req := mainRequest{
			Command:        commandMain,
			Model:          defaultModel,
			BaseBranch:     defaultBaseBranch,
			MaxIterations:  defaultMaxIterations,
			TimeoutSeconds: defaultTimeoutSeconds,
			ApprovalPolicy: defaultApprovalPolicy,
			Sandbox:        defaultSandbox,
			Page:           defaultPage,
			PageSize:       defaultPageSize,
		}
		if err := json.Unmarshal(body, &req); err != nil {
			return parsedCommand{}, err
		}
		applyBaseOverrides(&req.Output, &req.OutputFile, &req.Fields, &req.Page, &req.PageSize, &req.PageAll, state)
		if state.model != nil {
			req.Model = *state.model
		}
		if state.baseBranch != nil {
			req.BaseBranch = *state.baseBranch
		}
		if state.maxIterations != nil {
			req.MaxIterations = *state.maxIterations
		}
		if state.workBranch != nil {
			req.WorkBranch = *state.workBranch
		}
		if state.timeout != nil {
			req.TimeoutSeconds = *state.timeout
		}
		if state.approval != nil {
			req.ApprovalPolicy = *state.approval
		}
		if state.sandbox != nil {
			req.Sandbox = *state.sandbox
		}
		if state.preserve {
			req.PreserveTree = true
		}
		if state.dryRun {
			req.DryRun = true
		}
		if strings.TrimSpace(req.Prompt) == "" && len(state.positionals) > 0 {
			req.Prompt = strings.TrimSpace(strings.Join(state.positionals, " "))
		}
		if strings.TrimSpace(req.Prompt) == "" {
			return parsedCommand{}, fmt.Errorf("main command requires a prompt")
		}
		if strings.TrimSpace(req.WorkBranch) == "" {
			req.WorkBranch = defaultWorkBranch(req.Prompt)
		}
		if err := validateBaseRequest(req.Output, req.Page, req.PageSize); err != nil {
			return parsedCommand{}, err
		}
		return parsedCommand{Kind: commandMain, Main: req}, nil
	}
}

func buildFromFlags(state parseState) (parsedCommand, error) {
	switch state.command {
	case commandInit:
		if err := validateOptionsForCommand(state, commandInit); err != nil {
			return parsedCommand{}, err
		}
		req := initRequest{
			Command:    commandInit,
			BaseBranch: defaultBaseBranch,
			Page:       defaultPage,
			PageSize:   defaultPageSize,
		}
		applyBaseOverrides(&req.Output, &req.OutputFile, &req.Fields, &req.Page, &req.PageSize, &req.PageAll, state)
		if state.baseBranch != nil {
			req.BaseBranch = *state.baseBranch
		}
		if state.workBranch != nil {
			req.WorkBranch = *state.workBranch
		}
		req.DryRun = state.dryRun
		if strings.TrimSpace(req.WorkBranch) == "" {
			req.WorkBranch = defaultInitBranch()
		}
		if len(state.positionals) > 0 {
			return parsedCommand{}, fmt.Errorf("init does not accept positional arguments")
		}
		if err := validateBaseRequest(req.Output, req.Page, req.PageSize); err != nil {
			return parsedCommand{}, err
		}
		return parsedCommand{Kind: commandInit, Init: req}, nil
	case commandList:
		if err := validateOptionsForCommand(state, commandList); err != nil {
			return parsedCommand{}, err
		}
		req := listRequest{Command: commandList, Page: defaultPage, PageSize: defaultPageSize}
		applyBaseOverrides(&req.Output, &req.OutputFile, &req.Fields, &req.Page, &req.PageSize, &req.PageAll, state)
		if len(state.positionals) > 1 {
			return parsedCommand{}, fmt.Errorf("ls accepts at most one selector positional argument")
		}
		if len(state.positionals) > 0 {
			req.Selector = state.positionals[0]
		}
		if err := validateBaseRequest(req.Output, req.Page, req.PageSize); err != nil {
			return parsedCommand{}, err
		}
		return parsedCommand{Kind: commandList, List: req}, nil
	case commandTail:
		if err := validateOptionsForCommand(state, commandTail); err != nil {
			return parsedCommand{}, err
		}
		req := tailRequest{Command: commandTail, Lines: defaultTailLines, Page: defaultPage, PageSize: defaultPageSize}
		applyBaseOverrides(&req.Output, &req.OutputFile, &req.Fields, &req.Page, &req.PageSize, &req.PageAll, state)
		if len(state.positionals) > 1 {
			return parsedCommand{}, fmt.Errorf("tail accepts at most one selector positional argument")
		}
		if len(state.positionals) > 0 {
			req.Selector = state.positionals[0]
		}
		if state.lines != nil {
			req.Lines = *state.lines
		}
		req.Follow = state.follow
		req.Raw = state.raw
		if err := validateBaseRequest(req.Output, req.Page, req.PageSize); err != nil {
			return parsedCommand{}, err
		}
		return parsedCommand{Kind: commandTail, Tail: req}, nil
	case commandSchema:
		if err := validateOptionsForCommand(state, commandSchema); err != nil {
			return parsedCommand{}, err
		}
		req := schemaRequest{Page: defaultPage, PageSize: defaultPageSize}
		applyBaseOverrides(&req.Output, &req.OutputFile, &req.Fields, &req.Page, &req.PageSize, &req.PageAll, state)
		if state.commandName != nil {
			req.TargetCommand = *state.commandName
		} else if len(state.positionals) > 0 {
			req.TargetCommand = state.positionals[0]
		}
		if len(state.positionals) > 1 {
			return parsedCommand{}, fmt.Errorf("schema accepts at most one command positional argument")
		}
		if strings.TrimSpace(req.TargetCommand) != "" {
			if _, ok := commandDescriptorByName(req.TargetCommand); !ok {
				return parsedCommand{}, fmt.Errorf("unknown command %q", req.TargetCommand)
			}
		}
		if err := validateBaseRequest(req.Output, req.Page, req.PageSize); err != nil {
			return parsedCommand{}, err
		}
		return parsedCommand{Kind: commandSchema, Schema: req}, nil
	default:
		if err := validateOptionsForCommand(state, commandMain); err != nil {
			return parsedCommand{}, err
		}
		req := mainRequest{
			Command:        commandMain,
			Model:          defaultModel,
			BaseBranch:     defaultBaseBranch,
			MaxIterations:  defaultMaxIterations,
			TimeoutSeconds: defaultTimeoutSeconds,
			ApprovalPolicy: defaultApprovalPolicy,
			Sandbox:        defaultSandbox,
			Page:           defaultPage,
			PageSize:       defaultPageSize,
		}
		applyBaseOverrides(&req.Output, &req.OutputFile, &req.Fields, &req.Page, &req.PageSize, &req.PageAll, state)
		if state.model != nil {
			req.Model = *state.model
		}
		if state.baseBranch != nil {
			req.BaseBranch = *state.baseBranch
		}
		if state.maxIterations != nil {
			req.MaxIterations = *state.maxIterations
		}
		if state.workBranch != nil {
			req.WorkBranch = *state.workBranch
		}
		if state.timeout != nil {
			req.TimeoutSeconds = *state.timeout
		}
		if state.approval != nil {
			req.ApprovalPolicy = *state.approval
		}
		if state.sandbox != nil {
			req.Sandbox = *state.sandbox
		}
		req.PreserveTree = state.preserve
		req.DryRun = state.dryRun
		req.Prompt = strings.TrimSpace(strings.Join(state.positionals, " "))
		if strings.TrimSpace(req.Prompt) == "" {
			return parsedCommand{}, fmt.Errorf("main command requires a prompt")
		}
		if strings.TrimSpace(req.WorkBranch) == "" {
			req.WorkBranch = defaultWorkBranch(req.Prompt)
		}
		if err := validateBaseRequest(req.Output, req.Page, req.PageSize); err != nil {
			return parsedCommand{}, err
		}
		return parsedCommand{Kind: commandMain, Main: req}, nil
	}
}

func applyBaseOverrides(output *string, outputFile *string, fields *[]string, page *int, pageSize *int, pageAll *bool, state parseState) {
	if state.output != nil {
		*output = *state.output
	}
	if state.outputFile != nil {
		*outputFile = *state.outputFile
	}
	if state.fields != nil {
		*fields = parseFields(*state.fields)
	}
	if state.page != nil {
		*page = *state.page
	}
	if state.pageSize != nil {
		*pageSize = *state.pageSize
	}
	if state.pageAll {
		*pageAll = true
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
	if command == commandMain {
		return command, nil
	}
	if _, ok := knownCommands[command]; !ok {
		return "", fmt.Errorf("unknown command %q in JSON payload", command)
	}
	return command, nil
}

func validateOptionsForCommand(state parseState, command string) error {
	allowed := commandOptionSet(command)
	for option := range state.seenOptions {
		if _, ok := allowed[option]; !ok {
			return fmt.Errorf("%s does not support %s", command, option)
		}
	}
	return nil
}

func validateBaseRequest(output string, page int, pageSize int) error {
	if output != "" {
		switch strings.TrimSpace(output) {
		case "text", "json", "ndjson":
		default:
			return fmt.Errorf("invalid --output value %q (expected text, json, or ndjson)", output)
		}
	}
	if page <= 0 {
		return fmt.Errorf("page must be greater than 0")
	}
	if pageSize <= 0 {
		return fmt.Errorf("page_size must be greater than 0")
	}
	return nil
}
