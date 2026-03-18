package ralphloop

import (
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
	expected := filepath.Join(root, "artifacts", "result.json")
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
