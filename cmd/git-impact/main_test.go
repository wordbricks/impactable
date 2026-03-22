package main

import (
	"bytes"
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

func writeMainTestConfig(t *testing.T, dir string) {
	t.Helper()

	path := filepath.Join(dir, "impact-analyzer.yaml")
	content := `velen:
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
