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
