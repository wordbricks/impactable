package gitimpact

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveOutputPath_RejectsPathTraversal(t *testing.T) {
	t.Parallel()

	cwd := t.TempDir()
	_, err := resolveOutputPath(cwd, "../escape/report.json")
	if err == nil {
		t.Fatalf("expected resolveOutputPath to reject traversal")
	}
	if !strings.Contains(err.Error(), "output file must stay under") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveOutputPath_AllowsNestedRelativePath(t *testing.T) {
	t.Parallel()

	cwd := t.TempDir()
	path, err := resolveOutputPath(cwd, "reports/nested/impact.json")
	if err != nil {
		t.Fatalf("resolveOutputPath returned error: %v", err)
	}
	canonicalCWD, err := canonicalPathWithMissingTail(cwd)
	if err != nil {
		t.Fatalf("canonicalPathWithMissingTail returned error: %v", err)
	}
	expected := filepath.Join(canonicalCWD, "reports", "nested", "impact.json")
	if path != expected {
		t.Fatalf("expected %q, got %q", expected, path)
	}
}
