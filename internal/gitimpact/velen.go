package gitimpact

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

type commandRunner interface {
	Run(ctx context.Context, name string, args ...string) ([]byte, error)
}

type execRunner struct{}

func (r execRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		trimmed := strings.TrimSpace(string(out))
		if trimmed == "" {
			return nil, fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
		}
		return nil, fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, trimmed)
	}
	return out, nil
}

type CLIClient struct {
	runner commandRunner
}

func NewCLIClient() *CLIClient {
	return &CLIClient{runner: execRunner{}}
}

func (c *CLIClient) AuthWhoAmI(ctx context.Context) error {
	_, err := c.runner.Run(ctx, "velen", "auth", "whoami")
	return err
}

func (c *CLIClient) OrgCurrent(ctx context.Context) (string, error) {
	out, err := c.runner.Run(ctx, "velen", "org", "current")
	if err != nil {
		return "", err
	}
	return parseOrgName(out), nil
}

func (c *CLIClient) SourceList(ctx context.Context) ([]Source, error) {
	out, err := c.runner.Run(ctx, "velen", "source", "list", "--output", "json")
	if err != nil {
		return nil, err
	}
	return parseSourceList(out)
}

func parseOrgName(raw []byte) string {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		return ""
	}

	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return strings.Trim(trimmed, "\"")
	}

	for _, key := range []string{"org", "name", "slug"} {
		value, ok := payload[key]
		if !ok {
			continue
		}
		text := strings.TrimSpace(fmt.Sprint(value))
		if text != "" {
			return strings.Trim(text, "\"")
		}
	}

	return strings.Trim(trimmed, "\"")
}

func parseSourceList(raw []byte) ([]Source, error) {
	var direct []Source
	if err := json.Unmarshal(raw, &direct); err == nil {
		return direct, nil
	}

	var envelope struct {
		Sources []Source `json:"sources"`
		Items   []Source `json:"items"`
		Data    []Source `json:"data"`
	}
	if err := json.Unmarshal(raw, &envelope); err == nil {
		switch {
		case len(envelope.Sources) > 0:
			return envelope.Sources, nil
		case len(envelope.Items) > 0:
			return envelope.Items, nil
		case len(envelope.Data) > 0:
			return envelope.Data, nil
		}
	}

	var generic []map[string]any
	if err := json.Unmarshal(raw, &generic); err == nil {
		sources := make([]Source, 0, len(generic))
		for _, item := range generic {
			sources = append(sources, sourceFromMap(item))
		}
		return sources, nil
	}

	return nil, fmt.Errorf("unable to parse velen source list output")
}

func sourceFromMap(item map[string]any) Source {
	source := Source{}
	source.Key = mapString(item, "key")
	source.Name = mapString(item, "name")
	source.ProviderType = mapString(item, "provider_type")

	source.Capabilities = mapStringSlice(item, "capabilities")
	source.Operations = mapStringSlice(item, "operations")
	source.Actions = mapStringSlice(item, "actions")
	return source
}

func mapString(item map[string]any, key string) string {
	value, ok := item[key]
	if !ok {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func mapStringSlice(item map[string]any, key string) []string {
	value, ok := item[key]
	if !ok {
		return nil
	}
	anySlice, ok := value.([]any)
	if !ok {
		return nil
	}
	result := make([]string, 0, len(anySlice))
	for _, entry := range anySlice {
		text := strings.TrimSpace(fmt.Sprint(entry))
		if text == "" {
			continue
		}
		result = append(result, text)
	}
	return result
}
