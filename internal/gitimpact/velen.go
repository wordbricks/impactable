package gitimpact

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

type VelenClient interface {
	WhoAmI(ctx context.Context) (VelenIdentity, error)
	CurrentOrg(ctx context.Context) (string, error)
	ListSources(ctx context.Context) ([]VelenSource, error)
	ShowSource(ctx context.Context, sourceKey string) (VelenSource, error)
	Query(ctx context.Context, sourceKey string, queryFile string) ([]byte, error)
}

type VelenIdentity struct {
	Handle string `json:"handle,omitempty"`
}

type VelenSource struct {
	Key           string `json:"key"`
	Provider      string `json:"provider,omitempty"`
	SupportsQuery bool   `json:"supports_query"`
}

type commandExecutor interface {
	Run(ctx context.Context, name string, args ...string) ([]byte, error)
}

type osCommandExecutor struct{}

func (runner osCommandExecutor) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	body, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%s %s failed: %w (%s)", name, strings.Join(args, " "), err, strings.TrimSpace(string(body)))
	}
	return body, nil
}

type velenCLIClient struct {
	executor commandExecutor
}

func NewVelenCLIClient(executor commandExecutor) VelenClient {
	if executor == nil {
		executor = osCommandExecutor{}
	}
	return &velenCLIClient{executor: executor}
}

var newVelenClient = func() VelenClient {
	return NewVelenCLIClient(nil)
}

func (client *velenCLIClient) WhoAmI(ctx context.Context) (VelenIdentity, error) {
	body, err := client.executor.Run(ctx, "velen", "auth", "whoami", "--output", "json")
	if err != nil {
		return VelenIdentity{}, err
	}
	record := map[string]any{}
	if err := json.Unmarshal(body, &record); err != nil {
		// Fallback to non-JSON output.
		text := strings.TrimSpace(string(body))
		if text == "" {
			return VelenIdentity{}, fmt.Errorf("unexpected empty whoami output")
		}
		return VelenIdentity{Handle: text}, nil
	}
	return VelenIdentity{
		Handle: firstString(record, "handle", "user", "email", "id"),
	}, nil
}

func (client *velenCLIClient) CurrentOrg(ctx context.Context) (string, error) {
	body, err := client.executor.Run(ctx, "velen", "org", "current", "--output", "json")
	if err != nil {
		return "", err
	}
	record := map[string]any{}
	if err := json.Unmarshal(body, &record); err != nil {
		text := strings.TrimSpace(string(body))
		if text == "" {
			return "", fmt.Errorf("unexpected empty org output")
		}
		return text, nil
	}
	org := firstString(record, "org", "slug", "name")
	if strings.TrimSpace(org) == "" {
		return "", fmt.Errorf("unable to resolve current org from velen output")
	}
	return org, nil
}

func (client *velenCLIClient) ListSources(ctx context.Context) ([]VelenSource, error) {
	body, err := client.executor.Run(ctx, "velen", "source", "list", "--output", "json")
	if err != nil {
		return nil, err
	}
	return parseSourceList(body)
}

func (client *velenCLIClient) ShowSource(ctx context.Context, sourceKey string) (VelenSource, error) {
	body, err := client.executor.Run(ctx, "velen", "source", "show", sourceKey, "--output", "json")
	if err != nil {
		return VelenSource{}, err
	}
	return parseSourceObject(body)
}

func (client *velenCLIClient) Query(ctx context.Context, sourceKey string, queryFile string) ([]byte, error) {
	return client.executor.Run(ctx, "velen", "query", "--source", sourceKey, "--file", queryFile, "--output", "json")
}

func parseSourceList(body []byte) ([]VelenSource, error) {
	items := []map[string]any{}

	// Accept top-level array.
	var listPayload []map[string]any
	if err := json.Unmarshal(body, &listPayload); err == nil {
		items = listPayload
	} else {
		// Accept object envelope.
		object := map[string]any{}
		if unmarshalErr := json.Unmarshal(body, &object); unmarshalErr != nil {
			return nil, fmt.Errorf("unable to parse source list: %w", unmarshalErr)
		}
		if rawItems, ok := object["items"].([]any); ok {
			for _, item := range rawItems {
				if sourceMap, castOk := item.(map[string]any); castOk {
					items = append(items, sourceMap)
				}
			}
		}
		if rawSources, ok := object["sources"].([]any); ok {
			for _, item := range rawSources {
				if sourceMap, castOk := item.(map[string]any); castOk {
					items = append(items, sourceMap)
				}
			}
		}
	}

	sources := make([]VelenSource, 0, len(items))
	for _, item := range items {
		source := sourceFromRecord(item)
		if strings.TrimSpace(source.Key) != "" {
			sources = append(sources, source)
		}
	}
	return sources, nil
}

func parseSourceObject(body []byte) (VelenSource, error) {
	record := map[string]any{}
	if err := json.Unmarshal(body, &record); err != nil {
		return VelenSource{}, fmt.Errorf("unable to parse source detail: %w", err)
	}
	if nested, ok := record["source"].(map[string]any); ok {
		record = nested
	}
	source := sourceFromRecord(record)
	if strings.TrimSpace(source.Key) == "" {
		return VelenSource{}, fmt.Errorf("source key is missing from detail payload")
	}
	return source, nil
}

func sourceFromRecord(record map[string]any) VelenSource {
	source := VelenSource{
		Key:      firstString(record, "key", "source_key", "name"),
		Provider: firstString(record, "provider", "type"),
	}

	// Capabilities can appear as a list of strings, object map, or bool fields.
	if capabilities, ok := record["capabilities"].([]any); ok {
		for _, capability := range capabilities {
			if strings.EqualFold(strings.TrimSpace(fmt.Sprint(capability)), "query") {
				source.SupportsQuery = true
				break
			}
		}
	}
	if capabilityMap, ok := record["capabilities"].(map[string]any); ok {
		if value, exists := capabilityMap["QUERY"]; exists && truthy(value) {
			source.SupportsQuery = true
		}
		if value, exists := capabilityMap["query"]; exists && truthy(value) {
			source.SupportsQuery = true
		}
	}
	if value, exists := record["query"]; exists && truthy(value) {
		source.SupportsQuery = true
	}
	if value, exists := record["supports_query"]; exists && truthy(value) {
		source.SupportsQuery = true
	}
	if value, exists := record["query_enabled"]; exists && truthy(value) {
		source.SupportsQuery = true
	}

	return source
}

func truthy(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		normalized := strings.TrimSpace(strings.ToLower(typed))
		return normalized == "true" || normalized == "yes" || normalized == "1" || normalized == "y"
	case float64:
		return typed != 0
	case int:
		return typed != 0
	default:
		return false
	}
}

func firstString(record map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := record[key]; ok {
			text := strings.TrimSpace(fmt.Sprint(value))
			if text != "" && text != "<nil>" {
				return text
			}
		}
	}
	return ""
}
