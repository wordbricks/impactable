package wtl

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"time"
)

var newRunner = func(cfg runConfig) (turnRunner, error) {
	return newAppServerRunner(cfg)
}

func Run(args []string, cwd string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	req, err := parseCLI(args)
	output := resolveOutput("", stdout)
	if err == nil {
		output = resolveOutput(req.Output, stdout)
	}
	if err != nil {
		return emitFailure(output, stdout, stderr, err)
	}

	prompt, err := readPrompt(stdin, stdout, output)
	if err != nil {
		return emitFailure(output, stdout, stderr, err)
	}

	cfg := runConfig{
		CWD:      cwd,
		Model:    modelForRequest(req),
		MaxIter:  req.MaxIter,
		MaxRetry: req.MaxRetry,
	}
	runner, err := newRunner(cfg)
	if err != nil {
		return emitFailure(output, stdout, stderr, err)
	}
	defer runner.Close()

	var observers []observer
	var collector *jsonCollector
	switch output {
	case "text":
		observers = append(observers, &textObserver{writer: stdout, lastDeltaEnded: true})
	case "ndjson":
		observers = append(observers, &ndjsonObserver{writer: stdout})
	default:
		collector = &jsonCollector{}
		observers = append(observers, collector)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	runID := fmt.Sprintf("wtl-%d", time.Now().UTC().UnixNano())
	loop := &engine{
		runner:    runner,
		policy:    simpleLoopPolicy{},
		observers: observers,
		maxIter:   req.MaxIter,
		maxRetry:  req.MaxRetry,
		runID:     runID,
		phase:     simpleLoopPolicy{}.InitialPhase(),
	}

	summary, runErr := loop.run(ctx, cfg, prompt)
	if output == "json" {
		return emitJSONSummary(stdout, collector, summary, runErr)
	}
	if summary.Status == statusInterrupted {
		return 130
	}
	if summary.Status == statusCompleted {
		return 0
	}
	if runErr != nil {
		if output == "text" {
			_, _ = fmt.Fprintln(stderr, runErr.Error())
		}
		return 1
	}
	return 1
}

func modelForRequest(req request) string {
	if value := strings.TrimSpace(os.Getenv("WTL_MODEL")); value != "" {
		return value
	}
	if strings.TrimSpace(req.Model) != "" {
		return req.Model
	}
	return defaultModel
}

func readPrompt(stdin io.Reader, stdout io.Writer, output string) (string, error) {
	if isTerminalReader(stdin) && output == "text" {
		_, _ = fmt.Fprint(stdout, "> Enter your request: ")
		line, err := bufio.NewReader(stdin).ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return "", err
		}
		prompt := strings.TrimSpace(line)
		if prompt == "" {
			return "", fmt.Errorf("prompt is required")
		}
		return prompt, nil
	}
	body, err := io.ReadAll(stdin)
	if err != nil {
		return "", err
	}
	prompt := strings.TrimSpace(string(body))
	if prompt == "" {
		return "", fmt.Errorf("prompt is required on stdin")
	}
	return prompt, nil
}

func emitFailure(output string, stdout io.Writer, stderr io.Writer, err error) int {
	if output == "json" || output == "ndjson" {
		payload := map[string]any{
			"command": commandRun,
			"status":  statusFailed,
			"error": structuredError{
				Code:    "command_failed",
				Message: err.Error(),
			},
		}
		body, marshalErr := json.Marshal(payload)
		if marshalErr == nil {
			_, _ = fmt.Fprintf(stdout, "%s\n", body)
			return 1
		}
	}
	_, _ = fmt.Fprintln(stderr, err.Error())
	return 1
}

func emitJSONSummary(stdout io.Writer, collector *jsonCollector, summary runSummary, runErr error) int {
	payload := map[string]any{
		"command": commandRun,
		"status":  summary.Status,
		"run":     summary,
		"events":  collector.events,
	}
	if runErr != nil && summary.Status == statusFailed {
		payload["error"] = structuredError{
			Code:    "command_failed",
			Message: runErr.Error(),
		}
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return 1
	}
	_, _ = fmt.Fprintf(stdout, "%s\n", body)
	if summary.Status == statusCompleted {
		return 0
	}
	if summary.Status == statusInterrupted {
		return 130
	}
	return 1
}
