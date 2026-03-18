package gitimpact

import "testing"

func TestCommandDescriptors_ReportScaffoldModes(t *testing.T) {
	t.Parallel()

	descriptor, ok := commandDescriptorByName(commandReportScaffold)
	if !ok {
		t.Fatalf("expected descriptor for %q", commandReportScaffold)
	}

	var modeOption optionDescriptor
	found := false
	for _, option := range descriptor.Options {
		if option.Name == "--mode" {
			modeOption = option
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected --mode option")
	}
	if len(modeOption.Enum) != 4 {
		t.Fatalf("expected 4 mode enum values, got %d", len(modeOption.Enum))
	}
}

func TestCommandOptionSet_IsCommandSpecific(t *testing.T) {
	t.Parallel()

	analyzeOptions := commandOptionSet(commandAnalyze)
	if _, ok := analyzeOptions["--pr"]; !ok {
		t.Fatalf("expected analyze to support --pr")
	}
	if _, ok := analyzeOptions["--require"]; ok {
		t.Fatalf("did not expect analyze to support --require")
	}

	checkOptions := commandOptionSet(commandCheckSources)
	if _, ok := checkOptions["--require"]; !ok {
		t.Fatalf("expected check-sources to support --require")
	}
	if _, ok := checkOptions["--mode"]; ok {
		t.Fatalf("did not expect check-sources to support --mode")
	}
}
