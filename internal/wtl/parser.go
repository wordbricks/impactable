package wtl

import (
	"fmt"
	"strings"
)

func parseCLI(args []string) (request, error) {
	req := request{
		Command:  commandRun,
		MaxIter:  defaultMaxIterations,
		MaxRetry: defaultMaxRetry,
		Model:    defaultModel,
	}
	if len(args) == 0 {
		return req, fmt.Errorf("usage: wtl run [--max-iter N] [--max-retry N] [--output text|json|ndjson]")
	}
	if strings.TrimSpace(args[0]) != commandRun {
		return req, fmt.Errorf("unknown command %q", args[0])
	}
	for index := 1; index < len(args); index++ {
		arg := args[index]
		if !strings.HasPrefix(arg, "--") {
			return req, fmt.Errorf("unexpected positional argument %q", arg)
		}
		switch arg {
		case "--max-iter":
			value, next, err := parsePositiveIntFlag(args, index, arg)
			if err != nil {
				return req, err
			}
			req.MaxIter = value
			index = next
		case "--max-retry":
			value, next, err := parsePositiveIntFlag(args, index, arg)
			if err != nil {
				return req, err
			}
			req.MaxRetry = value
			index = next
		case "--output":
			if index+1 >= len(args) {
				return req, fmt.Errorf("%s requires a value", arg)
			}
			value := strings.TrimSpace(args[index+1])
			if value != "text" && value != "json" && value != "ndjson" {
				return req, fmt.Errorf("invalid --output value %q", value)
			}
			req.Output = value
			index++
		case "--model":
			if index+1 >= len(args) {
				return req, fmt.Errorf("%s requires a value", arg)
			}
			req.Model = strings.TrimSpace(args[index+1])
			if req.Model == "" {
				return req, fmt.Errorf("--model requires a non-empty value")
			}
			index++
		default:
			return req, fmt.Errorf("unknown option %q", arg)
		}
	}
	return req, nil
}

func parsePositiveIntFlag(args []string, index int, name string) (int, int, error) {
	if index+1 >= len(args) {
		return 0, index, fmt.Errorf("%s requires a value", name)
	}
	value := strings.TrimSpace(args[index+1])
	parsed := 0
	for _, ch := range value {
		if ch < '0' || ch > '9' {
			return 0, index, fmt.Errorf("%s requires a positive integer", name)
		}
		parsed = parsed*10 + int(ch-'0')
	}
	if parsed <= 0 {
		return 0, index, fmt.Errorf("%s requires a positive integer", name)
	}
	return parsed, index + 1, nil
}
