package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRun_AnalyzeAcceptsRAliasForPR(t *testing.T) {
	dir := t.TempDir()
	writeMainTestConfig(t, dir)

	var stdout bytes.Buffer
	exitCode := withWorkingDirectory(t, dir, func() int {
		return run([]string{"analyze", "--output", "json", "--r", "-1"}, strings.NewReader(""), &stdout, &stdout)
	})
	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(stdout.String(), "--pr must be zero or a positive integer") {
		t.Fatalf("expected normalized --r flag to hit --pr validation, got %q", stdout.String())
	}
}

func TestRun_MissingDefaultConfigReturnsActionableError(t *testing.T) {
	dir := t.TempDir()

	var stdout bytes.Buffer
	exitCode := withWorkingDirectory(t, dir, func() int {
		return run([]string{"analyze", "--output", "json"}, strings.NewReader(""), &stdout, &stdout)
	})
	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(stdout.String(), "config file") {
		t.Fatalf("expected missing-config error in output, got %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "pass --config") {
		t.Fatalf("expected setup guidance in output, got %q", stdout.String())
	}
}

func TestRun_AnalyzeRejectsRemovedAgentRuntimeFlag(t *testing.T) {
	dir := t.TempDir()
	writeMainTestConfig(t, dir)

	var stdout bytes.Buffer
	exitCode := withWorkingDirectory(t, dir, func() int {
		return run([]string{"analyze", "--output", "json", "--agent-runtime", "nope"}, strings.NewReader(""), &stdout, &stdout)
	})
	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(stdout.String(), "unknown flag: --agent-runtime") {
		t.Fatalf("expected removed runtime flag error in output, got %q", stdout.String())
	}
}

func TestNewPromptWaitHandler_ReleasesAndRestoresTerminal(t *testing.T) {
	var stdout bytes.Buffer
	terminal := &stubTerminalController{}
	handler := newPromptWaitHandler(strings.NewReader("y\n"), &stdout, terminal)

	response, err := handler("Continue anyway? (y/n)")
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if response != "y" {
		t.Fatalf("response = %q, want %q", response, "y")
	}
	if terminal.releaseCalls != 1 {
		t.Fatalf("release calls = %d, want 1", terminal.releaseCalls)
	}
	if terminal.restoreCalls != 1 {
		t.Fatalf("restore calls = %d, want 1", terminal.restoreCalls)
	}
	if !strings.Contains(stdout.String(), "Continue anyway? (y/n)") {
		t.Fatalf("expected prompt message in output, got %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "> ") {
		t.Fatalf("expected prompt marker in output, got %q", stdout.String())
	}
}

func TestNewPromptWaitHandler_ReturnsReleaseError(t *testing.T) {
	terminal := &stubTerminalController{releaseErr: errors.New("boom")}
	handler := newPromptWaitHandler(strings.NewReader("y\n"), &bytes.Buffer{}, terminal)

	_, err := handler("Continue anyway? (y/n)")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "release terminal for prompt") {
		t.Fatalf("unexpected error: %v", err)
	}
	if terminal.restoreCalls != 0 {
		t.Fatalf("restore calls = %d, want 0", terminal.restoreCalls)
	}
}

func writeMainTestConfig(t *testing.T, dir string) {
	t.Helper()

	path := filepath.Join(dir, "impact-analyzer.yaml")
	content := `onequery:
  org: impactable
  sources:
    github: github-main
    analytics: amplitude-prod
feature_grouping:
  strategies:
    - label_prefix
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

func withWorkingDirectory(t *testing.T, dir string, fn func() int) int {
	t.Helper()

	previous, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir to %q: %v", dir, err)
	}
	defer func() {
		if err := os.Chdir(previous); err != nil {
			t.Fatalf("restore working directory %q: %v", previous, err)
		}
	}()

	return fn()
}

type stubTerminalController struct {
	releaseCalls int
	restoreCalls int
	releaseErr   error
	restoreErr   error
}

func (s *stubTerminalController) ReleaseTerminal() error {
	s.releaseCalls++
	return s.releaseErr
}

func (s *stubTerminalController) RestoreTerminal() error {
	s.restoreCalls++
	return s.restoreErr
}
