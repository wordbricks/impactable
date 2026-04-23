package gitimpact

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRun_AnalyzeStub(t *testing.T) {
	t.Parallel()

	cwd := t.TempDir()
	configPath := writeTestConfig(t, cwd)

	var stderr bytes.Buffer
	exitCode := Run([]string{"analyze", "--config", configPath}, cwd, strings.NewReader(""), io.Discard, &stderr)
	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(stderr.String(), ErrAnalyzeNotImplemented.Error()) {
		t.Fatalf("expected analyze stub error, got %q", stderr.String())
	}
}

func TestRun_CheckSourcesStub(t *testing.T) {
	t.Parallel()

	cwd := t.TempDir()
	configPath := writeTestConfig(t, cwd)

	var stderr bytes.Buffer
	exitCode := Run([]string{"check-sources", "--config", configPath}, cwd, strings.NewReader(""), io.Discard, &stderr)
	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(stderr.String(), ErrCheckSourcesNotImplemented.Error()) {
		t.Fatalf("expected check-sources stub error, got %q", stderr.String())
	}
}

func writeTestConfig(t *testing.T, dir string) string {
	t.Helper()

	path := filepath.Join(dir, DefaultConfigFile)
	content := `onequery:
  org: my-company
  sources:
    github: github-main
    analytics: amplitude-prod
feature_grouping:
  strategies:
    - label_prefix
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write test config: %v", err)
	}
	return path
}
