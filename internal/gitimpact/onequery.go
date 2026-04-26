package gitimpact

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const defaultOneQueryTimeout = 30 * time.Second

type cmdFactory func(ctx context.Context, name string, args ...string) *exec.Cmd

type OneQueryClient struct {
	binary     string
	timeout    time.Duration
	org        string
	cmdFactory cmdFactory
}

func NewOneQueryClient(timeout time.Duration) *OneQueryClient {
	if timeout <= 0 {
		timeout = defaultOneQueryTimeout
	}
	return &OneQueryClient{
		binary:     "onequery",
		timeout:    timeout,
		cmdFactory: exec.CommandContext,
	}
}

// WithOrg returns a shallow client copy that passes --org on org-scoped OneQuery calls.
func (c *OneQueryClient) WithOrg(org string) *OneQueryClient {
	if c == nil {
		return nil
	}
	clone := *c
	clone.org = strings.TrimSpace(org)
	return &clone
}

func (c *OneQueryClient) WhoAmI() (*WhoAmIResult, error) {
	result := &WhoAmIResult{}
	if err := c.runAndDecode(result, "auth", "whoami"); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *OneQueryClient) CurrentOrg() (*OrgResult, error) {
	result := &OrgResult{}
	if err := c.runAndDecode(result, "org", "current"); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *OneQueryClient) ListSources() ([]Source, error) {
	payload, err := c.run("source", "list", "--page-all")
	if err != nil {
		return nil, err
	}

	payload = extractOneQueryData(payload)

	var direct []Source
	if err := json.Unmarshal(payload, &direct); err == nil {
		return direct, nil
	}

	var envelope struct {
		Sources []Source `json:"sources"`
		Items   []Source `json:"items"`
	}
	if err := json.Unmarshal(payload, &envelope); err == nil {
		if len(envelope.Sources) > 0 {
			return envelope.Sources, nil
		}
		if len(envelope.Items) > 0 {
			return envelope.Items, nil
		}
	}

	return nil, fmt.Errorf("decode onequery \"source list\" response")
}

func (c *OneQueryClient) ShowSource(key string) (*Source, error) {
	payload, err := c.run("source", "show", key)
	if err != nil {
		return nil, err
	}

	payload = extractOneQueryData(payload)

	result := &Source{}
	if err := json.Unmarshal(payload, result); err == nil && result.SourceKey() != "" {
		return result, nil
	}

	var envelope struct {
		Source Source `json:"source"`
		Item   Source `json:"item"`
	}
	if err := json.Unmarshal(payload, &envelope); err == nil {
		if envelope.Source.SourceKey() != "" {
			return &envelope.Source, nil
		}
		if envelope.Item.SourceKey() != "" {
			return &envelope.Item, nil
		}
	}

	return nil, fmt.Errorf("decode onequery \"source show\" response")
}

func (c *OneQueryClient) Query(sourceKey, sql string) (*QueryResult, error) {
	result := &QueryResult{}
	if err := c.runAndDecode(result, "query", "exec", "--source", sourceKey, "--sql", sql, "--max-rows", "500"); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *OneQueryClient) runAndDecode(target any, args ...string) error {
	payload, err := c.run(args...)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(extractOneQueryData(payload), target); err != nil {
		return fmt.Errorf("decode onequery response: %w", err)
	}
	return nil
}

func (c *OneQueryClient) run(args ...string) ([]byte, error) {
	if c == nil {
		return nil, fmt.Errorf("onequery client is nil")
	}

	timeout := c.timeout
	if timeout <= 0 {
		timeout = defaultOneQueryTimeout
	}
	binary := strings.TrimSpace(c.binary)
	if binary == "" {
		binary = "onequery"
	}
	runner := c.cmdFactory
	if runner == nil {
		runner = exec.CommandContext
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	commandArgs := []string{"--json"}
	if org := strings.TrimSpace(c.org); org != "" {
		commandArgs = append(commandArgs, "--org", org)
	}
	commandArgs = append(commandArgs, args...)
	cmd := runner(ctx, binary, commandArgs...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, &OneQueryError{
				Code:    "timeout",
				Message: fmt.Sprintf("onequery command timed out after %s", timeout),
			}
		}

		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, buildOneQueryError(exitErr.ExitCode(), stdout.String(), stderr.String())
		}

		return nil, fmt.Errorf("run onequery command: %w", err)
	}

	return bytes.TrimSpace(stdout.Bytes()), nil
}

func extractOneQueryData(payload []byte) []byte {
	trimmed := bytes.TrimSpace(payload)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return trimmed
	}

	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(trimmed, &envelope); err != nil {
		return trimmed
	}
	if len(bytes.TrimSpace(envelope.Data)) == 0 {
		return trimmed
	}
	return bytes.TrimSpace(envelope.Data)
}

func buildOneQueryError(exitCode int, stdout string, stderr string) *OneQueryError {
	if parsed := parseOneQueryError(stderr); parsed != nil {
		if strings.TrimSpace(parsed.Code) == "" {
			parsed.Code = fmt.Sprintf("exit_%d", exitCode)
		}
		return parsed
	}
	if parsed := parseOneQueryError(stdout); parsed != nil {
		if strings.TrimSpace(parsed.Code) == "" {
			parsed.Code = fmt.Sprintf("exit_%d", exitCode)
		}
		return parsed
	}

	trimmedStderr := strings.TrimSpace(stderr)
	trimmedStdout := strings.TrimSpace(stdout)
	parts := []string{}
	if trimmedStderr != "" {
		parts = append(parts, trimmedStderr)
	}
	if trimmedStdout != "" {
		parts = append(parts, "stdout: "+trimmedStdout)
	}
	message := strings.Join(parts, " | ")
	if message == "" {
		message = fmt.Sprintf("onequery command failed with exit code %d", exitCode)
	}

	return &OneQueryError{
		Code:    fmt.Sprintf("exit_%d", exitCode),
		Message: message,
	}
}

func parseOneQueryError(payload string) *OneQueryError {
	trimmed := strings.TrimSpace(payload)
	if trimmed == "" {
		return nil
	}

	var direct OneQueryError
	if err := json.Unmarshal([]byte(trimmed), &direct); err == nil {
		if oneQueryErrorHasText(&direct) {
			return &direct
		}
	}

	var wrapped struct {
		Error *OneQueryError `json:"error"`
	}
	if err := json.Unmarshal([]byte(trimmed), &wrapped); err == nil {
		if wrapped.Error != nil {
			if oneQueryErrorHasText(wrapped.Error) {
				return wrapped.Error
			}
		}
	}

	return nil
}

func oneQueryErrorHasText(err *OneQueryError) bool {
	if err == nil {
		return false
	}
	return strings.TrimSpace(err.Code) != "" ||
		strings.TrimSpace(err.Message) != "" ||
		strings.TrimSpace(err.Detail) != "" ||
		strings.TrimSpace(err.Title) != ""
}
