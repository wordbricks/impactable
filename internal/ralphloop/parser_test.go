package ralphloop

import (
	"strings"
	"testing"
)

func TestParseCLI_MainFlags(t *testing.T) {
	t.Parallel()

	parsed, err := parseCLI([]string{"ship", "the", "feature", "--work-branch", "ralph-custom", "--output", "json"}, strings.NewReader(""))
	if err != nil {
		t.Fatalf("parseCLI returned error: %v", err)
	}
	if parsed.Kind != commandMain {
		t.Fatalf("expected main command, got %q", parsed.Kind)
	}
	if parsed.Main.Prompt != "ship the feature" {
		t.Fatalf("unexpected prompt: %q", parsed.Main.Prompt)
	}
	if parsed.Main.WorkBranch != "ralph-custom" {
		t.Fatalf("unexpected work branch: %q", parsed.Main.WorkBranch)
	}
	if parsed.Main.Output != "json" {
		t.Fatalf("unexpected output: %q", parsed.Main.Output)
	}
}

func TestParseCLI_JSONOverridesCommand(t *testing.T) {
	t.Parallel()

	body := `{"command":"init","base_branch":"develop","work_branch":"ralph-agent"}`
	parsed, err := parseCLI([]string{"--json", body}, strings.NewReader(""))
	if err != nil {
		t.Fatalf("parseCLI returned error: %v", err)
	}
	if parsed.Kind != commandInit {
		t.Fatalf("expected init command, got %q", parsed.Kind)
	}
	if parsed.Init.BaseBranch != "develop" {
		t.Fatalf("unexpected base branch: %q", parsed.Init.BaseBranch)
	}
	if parsed.Init.WorkBranch != "ralph-agent" {
		t.Fatalf("unexpected work branch: %q", parsed.Init.WorkBranch)
	}
}

func TestParseCLI_TailShortAliases(t *testing.T) {
	t.Parallel()

	parsed, err := parseCLI([]string{"tail", "-n", "15", "-f", "session-a"}, strings.NewReader(""))
	if err != nil {
		t.Fatalf("parseCLI returned error: %v", err)
	}
	if parsed.Kind != commandTail {
		t.Fatalf("expected tail command, got %q", parsed.Kind)
	}
	if parsed.Tail.Lines != 15 {
		t.Fatalf("expected lines=15, got %d", parsed.Tail.Lines)
	}
	if !parsed.Tail.Follow {
		t.Fatalf("expected follow=true")
	}
	if parsed.Tail.Selector != "session-a" {
		t.Fatalf("unexpected selector: %q", parsed.Tail.Selector)
	}
}

func TestParseCLI_RejectsUnknownOption(t *testing.T) {
	t.Parallel()

	_, err := parseCLI([]string{"ls", "--unknown-option"}, strings.NewReader(""))
	if err == nil {
		t.Fatalf("expected unknown option error")
	}
	if !strings.Contains(err.Error(), "unknown option") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseCLI_RejectsOptionNotSupportedByCommand(t *testing.T) {
	t.Parallel()

	_, err := parseCLI([]string{"ls", "--max-iterations", "3"}, strings.NewReader(""))
	if err == nil {
		t.Fatalf("expected command option compatibility error")
	}
	if !strings.Contains(err.Error(), "ls does not support --max-iterations") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseCLI_SchemaJSONTargetCommand(t *testing.T) {
	t.Parallel()

	body := `{"command":"schema","target_command":"tail","output":"json"}`
	parsed, err := parseCLI([]string{"schema", "--json", body}, strings.NewReader(""))
	if err != nil {
		t.Fatalf("parseCLI returned error: %v", err)
	}
	if parsed.Kind != commandSchema {
		t.Fatalf("expected schema command, got %q", parsed.Kind)
	}
	if parsed.Schema.TargetCommand != "tail" {
		t.Fatalf("unexpected target command: %q", parsed.Schema.TargetCommand)
	}
}

func TestParseCLI_SchemaJSONLegacyCommandName(t *testing.T) {
	t.Parallel()

	body := `{"command":"schema","command_name":"main","output":"json"}`
	parsed, err := parseCLI([]string{"schema", "--json", body}, strings.NewReader(""))
	if err != nil {
		t.Fatalf("parseCLI returned error: %v", err)
	}
	if parsed.Schema.TargetCommand != "main" {
		t.Fatalf("unexpected target command: %q", parsed.Schema.TargetCommand)
	}
}

func TestParseCLI_RejectsInvalidOutputValue(t *testing.T) {
	t.Parallel()

	_, err := parseCLI([]string{"schema", "--output", "yaml"}, strings.NewReader(""))
	if err == nil {
		t.Fatalf("expected invalid output error")
	}
	if !strings.Contains(err.Error(), "invalid --output value") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseCLI_SchemaRejectsUnknownTargetCommand(t *testing.T) {
	t.Parallel()

	_, err := parseCLI([]string{"schema", "--command", "unknown"}, strings.NewReader(""))
	if err == nil {
		t.Fatalf("expected unknown command error")
	}
	if !strings.Contains(err.Error(), `unknown command "unknown"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}
