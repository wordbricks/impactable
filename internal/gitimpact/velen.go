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

const defaultVelenTimeout = 30 * time.Second

type cmdFactory func(ctx context.Context, name string, args ...string) *exec.Cmd

type VelenClient struct {
	binary     string
	timeout    time.Duration
	cmdFactory cmdFactory
}

func NewVelenClient(timeout time.Duration) *VelenClient {
	if timeout <= 0 {
		timeout = defaultVelenTimeout
	}
	return &VelenClient{
		binary:     "velen",
		timeout:    timeout,
		cmdFactory: exec.CommandContext,
	}
}

func (c *VelenClient) WhoAmI() (*WhoAmIResult, error) {
	result := &WhoAmIResult{}
	if err := c.runAndDecode(result, "auth", "whoami"); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *VelenClient) CurrentOrg() (*OrgResult, error) {
	result := &OrgResult{}
	if err := c.runAndDecode(result, "org", "current"); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *VelenClient) ListSources() ([]Source, error) {
	payload, err := c.run("source", "list")
	if err != nil {
		return nil, err
	}

	payload = extractVelenData(payload)

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

	return nil, fmt.Errorf("decode velen \"source list\" response")
}

func (c *VelenClient) ShowSource(key string) (*Source, error) {
	payload, err := c.run("source", "show", key)
	if err != nil {
		return nil, err
	}

	payload = extractVelenData(payload)

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

	return nil, fmt.Errorf("decode velen \"source show\" response")
}

func (c *VelenClient) Query(sourceKey, sql string) (*QueryResult, error) {
	result := &QueryResult{}
	if err := c.runAndDecode(result, "query", "--source", sourceKey, "--sql", sql); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *VelenClient) runAndDecode(target any, args ...string) error {
	payload, err := c.run(args...)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(extractVelenData(payload), target); err != nil {
		return fmt.Errorf("decode velen response: %w", err)
	}
	return nil
}

func (c *VelenClient) run(args ...string) ([]byte, error) {
	if c == nil {
		return nil, fmt.Errorf("velen client is nil")
	}

	timeout := c.timeout
	if timeout <= 0 {
		timeout = defaultVelenTimeout
	}
	binary := strings.TrimSpace(c.binary)
	if binary == "" {
		binary = "velen"
	}
	runner := c.cmdFactory
	if runner == nil {
		runner = exec.CommandContext
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	commandArgs := append([]string{"--output", "json"}, args...)
	cmd := runner(ctx, binary, commandArgs...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, &VelenError{
				Code:    "timeout",
				Message: fmt.Sprintf("velen command timed out after %s", timeout),
			}
		}

		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, buildVelenError(exitErr.ExitCode(), stdout.String(), stderr.String())
		}

		return nil, fmt.Errorf("run velen command: %w", err)
	}

	return bytes.TrimSpace(stdout.Bytes()), nil
}

func extractVelenData(payload []byte) []byte {
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

func buildVelenError(exitCode int, stdout string, stderr string) *VelenError {
	if parsed := parseVelenError(stderr); parsed != nil {
		if strings.TrimSpace(parsed.Code) == "" {
			parsed.Code = fmt.Sprintf("exit_%d", exitCode)
		}
		return parsed
	}
	if parsed := parseVelenError(stdout); parsed != nil {
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
		message = fmt.Sprintf("velen command failed with exit code %d", exitCode)
	}

	return &VelenError{
		Code:    fmt.Sprintf("exit_%d", exitCode),
		Message: message,
	}
}

func parseVelenError(payload string) *VelenError {
	trimmed := strings.TrimSpace(payload)
	if trimmed == "" {
		return nil
	}

	var direct VelenError
	if err := json.Unmarshal([]byte(trimmed), &direct); err == nil {
		if strings.TrimSpace(direct.Code) != "" || strings.TrimSpace(direct.Message) != "" {
			return &direct
		}
	}

	var wrapped struct {
		Error *VelenError `json:"error"`
	}
	if err := json.Unmarshal([]byte(trimmed), &wrapped); err == nil {
		if wrapped.Error != nil {
			if strings.TrimSpace(wrapped.Error.Code) != "" || strings.TrimSpace(wrapped.Error.Message) != "" {
				return wrapped.Error
			}
		}
	}

	return nil
}
