package ralphloop

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveOutputPathRejectsEscape(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if _, err := resolveOutputPath(root, "../escape.json"); err == nil {
		t.Fatal("expected output path escape to fail")
	}
}

func TestResolveOutputPathAllowsNestedFile(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	path, err := resolveOutputPath(root, filepath.Join("artifacts", "result.json"))
	if err != nil {
		t.Fatalf("resolveOutputPath returned error: %v", err)
	}
	expected, err := canonicalPathWithMissingTail(filepath.Join(root, "artifacts", "result.json"))
	if err != nil {
		t.Fatalf("canonicalPathWithMissingTail returned error: %v", err)
	}
	if path != expected {
		t.Fatalf("expected %q, got %q", expected, path)
	}
}

func TestDeriveWorktreeIDStable(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	first, err := deriveWorktreeID(root)
	if err != nil {
		t.Fatalf("deriveWorktreeID returned error: %v", err)
	}
	second, err := deriveWorktreeID(root)
	if err != nil {
		t.Fatalf("deriveWorktreeID returned error: %v", err)
	}
	if first != second {
		t.Fatalf("expected stable worktree id, got %q and %q", first, second)
	}
}

func TestExtractAgentTextPreservesCompletionTokenForLoopState(t *testing.T) {
	t.Parallel()

	item := map[string]any{
		"text": "done\n\n<promise>COMPLETE</promise>",
	}

	raw := extractAgentTextRaw(item)
	if !containsCompletionSignal(raw) {
		t.Fatalf("expected raw text to preserve completion token, got %q", raw)
	}

	display := extractAgentTextDisplay(item)
	if containsCompletionSignal(display) {
		t.Fatalf("expected display text to strip completion token, got %q", display)
	}
}

func TestResolveOutputPathRejectsSymlinkEscape(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	outside := t.TempDir()
	linkPath := filepath.Join(root, "link-outside")
	if err := os.Symlink(outside, linkPath); err != nil {
		t.Skipf("symlink not supported: %v", err)
	}

	if _, err := resolveOutputPath(root, filepath.Join("link-outside", "result.json")); err == nil {
		t.Fatal("expected symlink escape to fail")
	}
}

func TestResolveOutputPathAllowsSymlinkInsideRoot(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	inside := filepath.Join(root, "artifacts")
	if err := os.MkdirAll(inside, 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	linkPath := filepath.Join(root, "link-inside")
	if err := os.Symlink(inside, linkPath); err != nil {
		t.Skipf("symlink not supported: %v", err)
	}

	resolved, err := resolveOutputPath(root, filepath.Join("link-inside", "result.json"))
	if err != nil {
		t.Fatalf("resolveOutputPath returned error: %v", err)
	}
	expected, err := canonicalPathWithMissingTail(filepath.Join(inside, "result.json"))
	if err != nil {
		t.Fatalf("canonicalPathWithMissingTail returned error: %v", err)
	}
	if resolved != expected {
		t.Fatalf("expected %q, got %q", expected, resolved)
	}
}
