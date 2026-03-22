package gitimpact

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestNewVelenClientDefaultTimeout(t *testing.T) {
	client := NewVelenClient(0)
	if client.timeout != defaultVelenTimeout {
		t.Fatalf("expected default timeout %s, got %s", defaultVelenTimeout, client.timeout)
	}
	if client.binary != "velen" {
		t.Fatalf("expected binary velen, got %q", client.binary)
	}
}

func TestSourceSupportsQuery(t *testing.T) {
	source := Source{Capabilities: []string{"SYNC", "query"}}
	if !source.SupportsQuery() {
		t.Fatalf("expected SupportsQuery to be true")
	}

	source = Source{Capabilities: []string{"SYNC"}}
	if source.SupportsQuery() {
		t.Fatalf("expected SupportsQuery to be false")
	}

	source = Source{Query: "yes"}
	if !source.SupportsQuery() {
		t.Fatalf("expected SupportsQuery to be true for query=yes")
	}

	source = Source{Query: true}
	if !source.SupportsQuery() {
		t.Fatalf("expected SupportsQuery to be true for query=true")
	}
}

func TestWhoAmISuccess(t *testing.T) {
	argsFile := filepath.Join(t.TempDir(), "args.txt")
	client := newHelperClient(t, "whoami_success", time.Second, argsFile)

	result, err := client.WhoAmI()
	if err != nil {
		t.Fatalf("WhoAmI returned error: %v", err)
	}
	if result.Email != "agent@example.com" || result.Org != "impactable" {
		t.Fatalf("unexpected whoami result: %+v", result)
	}

	expectArgs(t, argsFile, []string{"velen", "--output", "json", "auth", "whoami"})
}

func TestCurrentOrgSuccess(t *testing.T) {
	argsFile := filepath.Join(t.TempDir(), "args.txt")
	client := newHelperClient(t, "org_success", time.Second, argsFile)

	result, err := client.CurrentOrg()
	if err != nil {
		t.Fatalf("CurrentOrg returned error: %v", err)
	}
	if result.Slug != "impactable" || result.Name != "Impactable" {
		t.Fatalf("unexpected org result: %+v", result)
	}

	expectArgs(t, argsFile, []string{"velen", "--output", "json", "org", "current"})
}

func TestListSourcesSuccess(t *testing.T) {
	argsFile := filepath.Join(t.TempDir(), "args.txt")
	client := newHelperClient(t, "source_list_success", time.Second, argsFile)

	result, err := client.ListSources()
	if err != nil {
		t.Fatalf("ListSources returned error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 source, got %d", len(result))
	}
	if result[0].Key != "github-main" || !result[0].SupportsQuery() {
		t.Fatalf("unexpected source result: %+v", result[0])
	}

	expectArgs(t, argsFile, []string{"velen", "--output", "json", "source", "list"})
}

func TestShowSourceSuccess(t *testing.T) {
	argsFile := filepath.Join(t.TempDir(), "args.txt")
	client := newHelperClient(t, "source_show_success", time.Second, argsFile)

	result, err := client.ShowSource("amplitude-prod")
	if err != nil {
		t.Fatalf("ShowSource returned error: %v", err)
	}
	if result.Key != "amplitude-prod" || result.ProviderType != "ANALYTICS" {
		t.Fatalf("unexpected source result: %+v", result)
	}

	expectArgs(t, argsFile, []string{"velen", "--output", "json", "source", "show", "amplitude-prod"})
}

func TestQuerySuccessAndSafeArgs(t *testing.T) {
	argsFile := filepath.Join(t.TempDir(), "args.txt")
	client := newHelperClient(t, "query_success", time.Second, argsFile)

	sql := "SELECT * FROM events WHERE note = '; rm -rf /' LIMIT 10"
	result, err := client.Query("github-main", sql)
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}
	if result.RowCount != 1 || len(result.Rows) != 1 || len(result.Columns) != 1 {
		t.Fatalf("unexpected query result: %+v", result)
	}

	expectArgs(t, argsFile, []string{"velen", "--output", "json", "query", "--source", "github-main", "--sql", sql})
}

func TestCurrentOrgEnvelopeSuccess(t *testing.T) {
	argsFile := filepath.Join(t.TempDir(), "args.txt")
	client := newHelperClient(t, "org_envelope_success", time.Second, argsFile)

	result, err := client.CurrentOrg()
	if err != nil {
		t.Fatalf("CurrentOrg returned error: %v", err)
	}
	if result.Org != "impactable" {
		t.Fatalf("unexpected org result: %+v", result)
	}
}

func TestListSourcesEnvelopeSuccess(t *testing.T) {
	argsFile := filepath.Join(t.TempDir(), "args.txt")
	client := newHelperClient(t, "source_list_envelope_success", time.Second, argsFile)

	result, err := client.ListSources()
	if err != nil {
		t.Fatalf("ListSources returned error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 sources, got %d", len(result))
	}
	if result[0].SourceKey() != "getgpt-repo" || result[0].ProviderLabel() != "github" || !result[0].SupportsQuery() {
		t.Fatalf("unexpected first source result: %+v", result[0])
	}
	if result[1].SourceKey() != "getgpt-ga" || result[1].ProviderLabel() != "ga" || !result[1].SupportsQuery() {
		t.Fatalf("unexpected second source result: %+v", result[1])
	}
}

func TestQueryEnvelopeSuccess(t *testing.T) {
	client := newHelperClient(t, "query_envelope_success", time.Second, "")

	result, err := client.Query("github-main", "SELECT 1")
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}
	if result.RowCount != 1 || len(result.Rows) != 1 || len(result.Columns) != 1 {
		t.Fatalf("unexpected query result: %+v", result)
	}
}

func TestNonZeroExitReturnsStructuredVelenError(t *testing.T) {
	client := newHelperClient(t, "nonzero_json_error", time.Second, "")

	_, err := client.WhoAmI()
	if err == nil {
		t.Fatalf("expected error")
	}

	var velenErr *VelenError
	if !errors.As(err, &velenErr) {
		t.Fatalf("expected VelenError, got %T", err)
	}
	if velenErr.Code != "unauthorized" || velenErr.Message != "bad token" {
		t.Fatalf("unexpected velen error: %+v", velenErr)
	}
}

func TestNonZeroExitFallbackVelenError(t *testing.T) {
	client := newHelperClient(t, "nonzero_plain_error", time.Second, "")

	_, err := client.WhoAmI()
	if err == nil {
		t.Fatalf("expected error")
	}

	var velenErr *VelenError
	if !errors.As(err, &velenErr) {
		t.Fatalf("expected VelenError, got %T", err)
	}
	if velenErr.Code != "exit_7" {
		t.Fatalf("expected code exit_7, got %q", velenErr.Code)
	}
	if !strings.Contains(velenErr.Message, "permission denied") || !strings.Contains(velenErr.Message, "partial output") {
		t.Fatalf("unexpected velen error message: %q", velenErr.Message)
	}
}

func TestInvalidJSONReturnsDecodeError(t *testing.T) {
	client := newHelperClient(t, "invalid_json", time.Second, "")

	_, err := client.WhoAmI()
	if err == nil {
		t.Fatalf("expected error")
	}

	var velenErr *VelenError
	if errors.As(err, &velenErr) {
		t.Fatalf("expected non-VelenError decode error, got %+v", velenErr)
	}
}

func TestTimeoutReturnsVelenError(t *testing.T) {
	client := newHelperClient(t, "sleep", 25*time.Millisecond, "")

	_, err := client.WhoAmI()
	if err == nil {
		t.Fatalf("expected timeout error")
	}

	var velenErr *VelenError
	if !errors.As(err, &velenErr) {
		t.Fatalf("expected VelenError, got %T", err)
	}
	if velenErr.Code != "timeout" {
		t.Fatalf("expected timeout code, got %q", velenErr.Code)
	}
}

func newHelperClient(t *testing.T, scenario string, timeout time.Duration, argsFile string) *VelenClient {
	t.Helper()
	client := NewVelenClient(timeout)
	client.cmdFactory = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		helperArgs := []string{"-test.run=TestVelenHelperProcess", "--", name}
		helperArgs = append(helperArgs, args...)
		cmd := exec.CommandContext(ctx, os.Args[0], helperArgs...)

		env := append(os.Environ(),
			"GO_WANT_VELEN_HELPER_PROCESS=1",
			"VELEN_HELPER_SCENARIO="+scenario,
		)
		if argsFile != "" {
			env = append(env, "VELEN_ARGS_FILE="+argsFile)
		}
		cmd.Env = env
		return cmd
	}
	return client
}

func expectArgs(t *testing.T, argsFile string, expected []string) {
	t.Helper()
	payload, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("read args file: %v", err)
	}
	content := strings.TrimSpace(string(payload))
	if content == "" {
		t.Fatalf("args file is empty")
	}
	actual := strings.Split(content, "\n")
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("unexpected args: got %#v, want %#v", actual, expected)
	}
}

func TestVelenHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_VELEN_HELPER_PROCESS") != "1" {
		return
	}

	separatorIndex := -1
	for idx, arg := range os.Args {
		if arg == "--" {
			separatorIndex = idx
			break
		}
	}
	if separatorIndex == -1 || separatorIndex+1 >= len(os.Args) {
		_, _ = os.Stderr.WriteString("missing helper args")
		os.Exit(2)
	}

	args := os.Args[separatorIndex+1:]
	if path := os.Getenv("VELEN_ARGS_FILE"); path != "" {
		if err := os.WriteFile(path, []byte(strings.Join(args, "\n")), 0o600); err != nil {
			_, _ = os.Stderr.WriteString(fmt.Sprintf("write args: %v", err))
			os.Exit(2)
		}
	}

	switch os.Getenv("VELEN_HELPER_SCENARIO") {
	case "whoami_success":
		_, _ = os.Stdout.WriteString(`{"email":"agent@example.com","org":"impactable"}`)
		os.Exit(0)
	case "org_success":
		_, _ = os.Stdout.WriteString(`{"slug":"impactable","name":"Impactable"}`)
		os.Exit(0)
	case "org_envelope_success":
		_, _ = os.Stdout.WriteString(`{"command":"org current","data":{"org":"impactable","resolved":true,"source":"config"},"ok":true,"warnings":[]}`)
		os.Exit(0)
	case "source_list_success":
		_, _ = os.Stdout.WriteString(`[{"key":"github-main","name":"GitHub","provider_type":"GITHUB","capabilities":["QUERY","SYNC"]}]`)
		os.Exit(0)
	case "source_list_envelope_success":
		_, _ = os.Stdout.WriteString(`{"command":"source list","data":{"items":[{"name":"getgpt-repo","provider":"github","query":"yes","status":"active"},{"name":"getgpt-ga","provider":"ga","query":true,"status":"active"}]},"ok":true,"warnings":[]}`)
		os.Exit(0)
	case "source_show_success":
		_, _ = os.Stdout.WriteString(`{"key":"amplitude-prod","name":"Amplitude","provider_type":"ANALYTICS","capabilities":["QUERY"]}`)
		os.Exit(0)
	case "query_success":
		_, _ = os.Stdout.WriteString(`{"columns":["id"],"rows":[[1]],"row_count":1}`)
		os.Exit(0)
	case "query_envelope_success":
		_, _ = os.Stdout.WriteString(`{"command":"query","data":{"columns":["id"],"rows":[[1]],"row_count":1},"ok":true,"warnings":[]}`)
		os.Exit(0)
	case "nonzero_json_error":
		_, _ = os.Stderr.WriteString(`{"code":"unauthorized","message":"bad token"}`)
		os.Exit(3)
	case "nonzero_plain_error":
		_, _ = os.Stderr.WriteString("permission denied")
		_, _ = os.Stdout.WriteString("partial output")
		os.Exit(7)
	case "invalid_json":
		_, _ = os.Stdout.WriteString("not-json")
		os.Exit(0)
	case "sleep":
		time.Sleep(200 * time.Millisecond)
		_, _ = os.Stdout.WriteString(`{"email":"late@example.com","org":"late"}`)
		os.Exit(0)
	default:
		_, _ = os.Stderr.WriteString("unknown helper scenario")
		os.Exit(2)
	}
}
