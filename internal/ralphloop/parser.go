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
	jsonSource    *string
	output        *string
	outputFile    *string
	fields        *string
	page          *int
	pageSize      *int
	pageAll       bool
	commandName   *string
	selector      *string
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
	state := parseState{command: commandMain}
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
			value, err := requireValue(args, &i, arg)
			if err != nil {
				return parsedCommand{}, err
			}
			state.jsonSource = &value
		case "--output":
			value, err := requireValue(args, &i, arg)
			if err != nil {
				return parsedCommand{}, err
			}
			state.output = &value
		case "--output-file":
			value, err := requireValue(args, &i, arg)
			if err != nil {
				return parsedCommand{}, err
			}
			state.outputFile = &value
		case "--fields":
			value, err := requireValue(args, &i, arg)
			if err != nil {
				return parsedCommand{}, err
			}
			state.fields = &value
		case "--page":
			value, err := requireIntValue(args, &i, arg)
			if err != nil {
				return parsedCommand{}, err
			}
			state.page = &value
		case "--page-size":
			value, err := requireIntValue(args, &i, arg)
			if err != nil {
				return parsedCommand{}, err
			}
			state.pageSize = &value
		case "--page-all":
			state.pageAll = true
		case "--command":
			value, err := requireValue(args, &i, arg)
			if err != nil {
				return parsedCommand{}, err
			}
			state.commandName = &value
		case "--selector":
			value, err := requireValue(args, &i, arg)
			if err != nil {
				return parsedCommand{}, err
			}
			state.selector = &value
		case "--lines":
			value, err := requireIntValue(args, &i, arg)
			if err != nil {
				return parsedCommand{}, err
			}
			state.lines = &value
		case "--follow":
			state.follow = true
		case "--raw":
			state.raw = true
		case "--model":
			value, err := requireValue(args, &i, arg)
			if err != nil {
				return parsedCommand{}, err
			}
			state.model = &value
		case "--base-branch":
			value, err := requireValue(args, &i, arg)
			if err != nil {
				return parsedCommand{}, err
			}
			state.baseBranch = &value
		case "--max-iterations":
			value, err := requireIntValue(args, &i, arg)
			if err != nil {
				return parsedCommand{}, err
			}
			state.maxIterations = &value
		case "--work-branch":
			value, err := requireValue(args, &i, arg)
			if err != nil {
				return parsedCommand{}, err
			}
			state.workBranch = &value
		case "--timeout":
			value, err := requireIntValue(args, &i, arg)
			if err != nil {
				return parsedCommand{}, err
			}
			state.timeout = &value
		case "--approval-policy":
			value, err := requireValue(args, &i, arg)
			if err != nil {
				return parsedCommand{}, err
			}
			state.approval = &value
		case "--sandbox":
			value, err := requireValue(args, &i, arg)
			if err != nil {
				return parsedCommand{}, err
			}
			state.sandbox = &value
		case "--preserve-worktree":
			state.preserve = true
		case "--dry-run":
			state.dryRun = true
		default:
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
		return parsedCommand{Kind: commandInit, Init: req}, nil
	case commandList:
		req := listRequest{Command: commandList, Page: defaultPage, PageSize: defaultPageSize}
		if err := json.Unmarshal(body, &req); err != nil {
			return parsedCommand{}, err
		}
		applyBaseOverrides(&req.Output, &req.OutputFile, &req.Fields, &req.Page, &req.PageSize, &req.PageAll, state)
		if state.selector != nil {
			req.Selector = *state.selector
		}
		return parsedCommand{Kind: commandList, List: req}, nil
	case commandTail:
		req := tailRequest{Command: commandTail, Lines: defaultTailLines, Page: defaultPage, PageSize: defaultPageSize}
		if err := json.Unmarshal(body, &req); err != nil {
			return parsedCommand{}, err
		}
		applyBaseOverrides(&req.Output, &req.OutputFile, &req.Fields, &req.Page, &req.PageSize, &req.PageAll, state)
		if state.selector != nil {
			req.Selector = *state.selector
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
		return parsedCommand{Kind: commandTail, Tail: req}, nil
	case commandSchema:
		req := schemaRequest{Page: defaultPage, PageSize: defaultPageSize}
		if err := json.Unmarshal(body, &req); err != nil {
			return parsedCommand{}, err
		}
		applyBaseOverrides(&req.Output, &req.OutputFile, &req.Fields, &req.Page, &req.PageSize, &req.PageAll, state)
		if state.commandName != nil {
			req.CommandName = *state.commandName
		}
		return parsedCommand{Kind: commandSchema, Schema: req}, nil
	default:
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
		return parsedCommand{Kind: commandMain, Main: req}, nil
	}
}

func buildFromFlags(state parseState) (parsedCommand, error) {
	switch state.command {
	case commandInit:
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
		return parsedCommand{Kind: commandInit, Init: req}, nil
	case commandList:
		req := listRequest{Command: commandList, Page: defaultPage, PageSize: defaultPageSize}
		applyBaseOverrides(&req.Output, &req.OutputFile, &req.Fields, &req.Page, &req.PageSize, &req.PageAll, state)
		if state.selector != nil {
			req.Selector = *state.selector
		} else if len(state.positionals) > 0 {
			req.Selector = state.positionals[0]
		}
		return parsedCommand{Kind: commandList, List: req}, nil
	case commandTail:
		req := tailRequest{Command: commandTail, Lines: defaultTailLines, Page: defaultPage, PageSize: defaultPageSize}
		applyBaseOverrides(&req.Output, &req.OutputFile, &req.Fields, &req.Page, &req.PageSize, &req.PageAll, state)
		if state.selector != nil {
			req.Selector = *state.selector
		} else if len(state.positionals) > 0 {
			req.Selector = state.positionals[0]
		}
		if state.lines != nil {
			req.Lines = *state.lines
		}
		req.Follow = state.follow
		req.Raw = state.raw
		return parsedCommand{Kind: commandTail, Tail: req}, nil
	case commandSchema:
		req := schemaRequest{Page: defaultPage, PageSize: defaultPageSize}
		applyBaseOverrides(&req.Output, &req.OutputFile, &req.Fields, &req.Page, &req.PageSize, &req.PageAll, state)
		if state.commandName != nil {
			req.CommandName = *state.commandName
		} else if len(state.positionals) > 0 {
			req.CommandName = state.positionals[0]
		}
		return parsedCommand{Kind: commandSchema, Schema: req}, nil
	default:
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
