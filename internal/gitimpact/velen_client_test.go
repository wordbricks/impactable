package gitimpact

import (
	"context"
	"testing"
)

func TestVelenCLIClient_Primitives(t *testing.T) {
	t.Parallel()

	executor := fakeCommandExecutor{
		responses: map[string][]byte{
			"velen auth whoami --output json":                                      []byte(`{"handle":"impact-bot"}`),
			"velen org current --output json":                                      []byte(`{"org":"impactable"}`),
			"velen source list --output json":                                      []byte(`{"items":[{"key":"github-main","provider":"github","capabilities":["QUERY"]}]}`),
			"velen source show github-main --output json":                          []byte(`{"source":{"key":"github-main","provider":"github","capabilities":{"QUERY":true}}}`),
			"velen query --source github-main --file queries/pr.sql --output json": []byte(`{"rows":[{"pr":142}]}`),
		},
	}

	client := NewVelenCLIClient(executor)
	identity, err := client.WhoAmI(context.Background())
	if err != nil {
		t.Fatalf("WhoAmI returned error: %v", err)
	}
	if identity.Handle != "impact-bot" {
		t.Fatalf("unexpected handle: %q", identity.Handle)
	}

	org, err := client.CurrentOrg(context.Background())
	if err != nil {
		t.Fatalf("CurrentOrg returned error: %v", err)
	}
	if org != "impactable" {
		t.Fatalf("unexpected org: %q", org)
	}

	sources, err := client.ListSources(context.Background())
	if err != nil {
		t.Fatalf("ListSources returned error: %v", err)
	}
	if len(sources) != 1 {
		t.Fatalf("expected one source, got %d", len(sources))
	}
	if !sources[0].SupportsQuery {
		t.Fatalf("expected source QUERY capability")
	}

	source, err := client.ShowSource(context.Background(), "github-main")
	if err != nil {
		t.Fatalf("ShowSource returned error: %v", err)
	}
	if source.Key != "github-main" || !source.SupportsQuery {
		t.Fatalf("unexpected source detail: %#v", source)
	}

	queryBody, err := client.Query(context.Background(), "github-main", "queries/pr.sql")
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}
	if string(queryBody) != `{"rows":[{"pr":142}]}` {
		t.Fatalf("unexpected query body: %s", string(queryBody))
	}
}

func TestVelenCLIClient_ParseSourceListFromArrayPayload(t *testing.T) {
	t.Parallel()

	executor := fakeCommandExecutor{
		responses: map[string][]byte{
			"velen source list --output json": []byte(`[{"key":"prod-warehouse","provider":"bigquery","supports_query":true}]`),
		},
	}
	client := NewVelenCLIClient(executor)

	sources, err := client.ListSources(context.Background())
	if err != nil {
		t.Fatalf("ListSources returned error: %v", err)
	}
	if len(sources) != 1 {
		t.Fatalf("expected one source, got %d", len(sources))
	}
	if sources[0].Key != "prod-warehouse" || !sources[0].SupportsQuery {
		t.Fatalf("unexpected source payload: %#v", sources[0])
	}
}

type fakeCommandExecutor struct {
	responses map[string][]byte
	errs      map[string]error
}

func (executor fakeCommandExecutor) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	key := name
	for _, arg := range args {
		key += " " + arg
	}
	if err, ok := executor.errs[key]; ok {
		return nil, err
	}
	if response, ok := executor.responses[key]; ok {
		return response, nil
	}
	return nil, assertErr("unexpected command: " + key)
}
