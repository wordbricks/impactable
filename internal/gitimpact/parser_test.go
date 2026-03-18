package gitimpact

import (
	"strings"
	"testing"
)

func TestParseCLI_AnalyzeFlags(t *testing.T) {
	t.Parallel()

	parsed, err := parseCLI([]string{"analyze", "--config", "custom.yaml", "--pr", "142", "--since", "2026-01-01", "--output", "json"}, strings.NewReader(""))
	if err != nil {
		t.Fatalf("parseCLI returned error: %v", err)
	}
	if parsed.Kind != commandAnalyze {
		t.Fatalf("expected command %q, got %q", commandAnalyze, parsed.Kind)
	}
	if parsed.Analyze.ConfigPath != "custom.yaml" {
		t.Fatalf("unexpected config path: %q", parsed.Analyze.ConfigPath)
	}
	if parsed.Analyze.PRNumber != 142 {
		t.Fatalf("unexpected pr value: %d", parsed.Analyze.PRNumber)
	}
	if parsed.Analyze.Since != "2026-01-01" {
		t.Fatalf("unexpected since: %q", parsed.Analyze.Since)
	}
	if parsed.Analyze.Output != "json" {
		t.Fatalf("unexpected output value: %q", parsed.Analyze.Output)
	}
}

func TestParseCLI_CheckSourcesJSON(t *testing.T) {
	t.Parallel()

	payload := `{"command":"check-sources","required_roles":["github","warehouse"],"config":"impact.yaml"}`
	parsed, err := parseCLI([]string{"check-sources", "--json", payload}, strings.NewReader(""))
	if err != nil {
		t.Fatalf("parseCLI returned error: %v", err)
	}
	if parsed.Kind != commandCheckSources {
		t.Fatalf("expected command %q, got %q", commandCheckSources, parsed.Kind)
	}
	if len(parsed.CheckSources.RequiredRoles) != 2 {
		t.Fatalf("expected 2 required roles, got %d", len(parsed.CheckSources.RequiredRoles))
	}
	if parsed.CheckSources.RequiredRoles[0] != "github" || parsed.CheckSources.RequiredRoles[1] != "warehouse" {
		t.Fatalf("unexpected roles: %#v", parsed.CheckSources.RequiredRoles)
	}
}

func TestParseCLI_ReportScaffoldModes(t *testing.T) {
	t.Parallel()

	parsed, err := parseCLI([]string{"report-scaffold", "--mode", "markdown", "--mode", "html", "--output-dir", "out/reports"}, strings.NewReader(""))
	if err != nil {
		t.Fatalf("parseCLI returned error: %v", err)
	}
	if parsed.Kind != commandReportScaffold {
		t.Fatalf("expected command %q, got %q", commandReportScaffold, parsed.Kind)
	}
	if len(parsed.ReportScaffold.Modes) != 2 {
		t.Fatalf("expected 2 modes, got %d", len(parsed.ReportScaffold.Modes))
	}
	if parsed.ReportScaffold.Modes[0] != "markdown" || parsed.ReportScaffold.Modes[1] != "html" {
		t.Fatalf("unexpected modes: %#v", parsed.ReportScaffold.Modes)
	}
	if parsed.ReportScaffold.OutputDir != "out/reports" {
		t.Fatalf("unexpected output dir: %q", parsed.ReportScaffold.OutputDir)
	}
}

func TestParseCLI_RejectsUnsupportedOptionForCommand(t *testing.T) {
	t.Parallel()

	_, err := parseCLI([]string{"check-sources", "--pr", "9"}, strings.NewReader(""))
	if err == nil {
		t.Fatalf("expected unsupported option error")
	}
	if !strings.Contains(err.Error(), "check-sources does not support --pr") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseCLI_RejectsInvalidReportMode(t *testing.T) {
	t.Parallel()

	_, err := parseCLI([]string{"report-scaffold", "--mode", "pdf"}, strings.NewReader(""))
	if err == nil {
		t.Fatalf("expected invalid mode error")
	}
	if !strings.Contains(err.Error(), "unsupported report mode") {
		t.Fatalf("unexpected error: %v", err)
	}
}
