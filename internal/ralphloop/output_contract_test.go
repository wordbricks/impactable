package ralphloop

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunSchemaJSONEnvelope(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	if err := runSchema(t.TempDir(), schemaRequest{Output: "json", TargetCommand: commandTail}, &stdout); err != nil {
		t.Fatalf("runSchema returned error: %v", err)
	}

	response := map[string]any{}
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse schema output: %v (%q)", err, stdout.String())
	}
	if response["command"] != commandSchema {
		t.Fatalf("expected command schema, got %#v", response["command"])
	}
	if response["status"] != "ok" {
		t.Fatalf("expected status ok, got %#v", response["status"])
	}
	items, _ := response["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("expected one schema item, got %d", len(items))
	}
	item, _ := items[0].(map[string]any)
	if item["name"] != commandTail {
		t.Fatalf("expected tail schema item, got %#v", item["name"])
	}
}

func TestRunListNDJSONPerRecord(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	writeSessionMetadata(t, repoRoot, "w1", sessionRecord{
		PID:          os.Getpid(),
		WorktreeID:   "w1",
		WorktreePath: repoRoot,
		WorkBranch:   "branch-1",
		LogPath:      filepath.Join(repoRoot, ".worktree", "w1", "logs", "ralph-loop.log"),
		StartedAt:    "2026-03-18T07:00:00Z",
	})

	var stdout bytes.Buffer
	req := listRequest{Output: "ndjson", Page: 1, PageSize: 50}
	if err := runList(repoRoot, repoRoot, req, &stdout); err != nil {
		t.Fatalf("runList returned error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected one ndjson line, got %d (%q)", len(lines), stdout.String())
	}
	record := map[string]any{}
	if err := json.Unmarshal([]byte(lines[0]), &record); err != nil {
		t.Fatalf("failed to parse ndjson record: %v", err)
	}
	if _, ok := record["items"]; ok {
		t.Fatalf("expected per-record ndjson, got envelope with items: %#v", record)
	}
	if record["worktree_id"] != "w1" {
		t.Fatalf("unexpected worktree_id: %#v", record["worktree_id"])
	}
}

func TestRunListNDJSONPageAllEnvelope(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	writeSessionMetadata(t, repoRoot, "w1", sessionRecord{
		PID:          os.Getpid(),
		WorktreeID:   "w1",
		WorktreePath: repoRoot,
		WorkBranch:   "branch-1",
		LogPath:      filepath.Join(repoRoot, ".worktree", "w1", "logs", "ralph-loop.log"),
		StartedAt:    "2026-03-18T07:00:00Z",
	})
	writeSessionMetadata(t, repoRoot, "w2", sessionRecord{
		PID:          os.Getpid(),
		WorktreeID:   "w2",
		WorktreePath: repoRoot,
		WorkBranch:   "branch-2",
		LogPath:      filepath.Join(repoRoot, ".worktree", "w2", "logs", "ralph-loop.log"),
		StartedAt:    "2026-03-18T07:01:00Z",
	})

	var stdout bytes.Buffer
	req := listRequest{Output: "ndjson", Page: 1, PageSize: 1, PageAll: true}
	if err := runList(repoRoot, repoRoot, req, &stdout); err != nil {
		t.Fatalf("runList returned error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected two page envelopes, got %d (%q)", len(lines), stdout.String())
	}
	first := map[string]any{}
	if err := json.Unmarshal([]byte(lines[0]), &first); err != nil {
		t.Fatalf("failed to parse first page envelope: %v", err)
	}
	if first["command"] != commandList {
		t.Fatalf("expected ls command envelope, got %#v", first["command"])
	}
	if first["page_all"] != true {
		t.Fatalf("expected page_all=true, got %#v", first["page_all"])
	}
	items, _ := first["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("expected one item in paged envelope, got %d", len(items))
	}
}

func TestRunTailJSONIncludesMetadata(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	logPath := filepath.Join(repoRoot, ".worktree", "w1", "logs", "ralph-loop.log")
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(logPath, []byte("line-1\nline-2\nline-3\n"), 0o644); err != nil {
		t.Fatalf("write log failed: %v", err)
	}

	var stdout bytes.Buffer
	req := tailRequest{Output: "json", Lines: 2, Page: 1, PageSize: 50}
	if err := runTail(context.Background(), repoRoot, repoRoot, req, &stdout); err != nil {
		t.Fatalf("runTail returned error: %v", err)
	}

	response := map[string]any{}
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse tail json output: %v (%q)", err, stdout.String())
	}
	if response["command"] != commandTail {
		t.Fatalf("expected tail command, got %#v", response["command"])
	}
	if response["log_path"] != logPath {
		t.Fatalf("expected log_path %q, got %#v", logPath, response["log_path"])
	}
	if response["total_items"] != float64(2) {
		t.Fatalf("expected total_items=2, got %#v", response["total_items"])
	}
}

func TestRunInitDryRunIncludesRequestAndSideEffects(t *testing.T) {
	t.Parallel()

	repoRoot := initTestGitRepo(t)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"init", "--dry-run", "--output", "json"}, repoRoot, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q", code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr in json dry-run mode, got %q", stderr.String())
	}

	response := map[string]any{}
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse init dry-run output: %v (%q)", err, stdout.String())
	}
	if response["dry_run"] != true {
		t.Fatalf("expected dry_run=true, got %#v", response["dry_run"])
	}
	if _, ok := response["request"]; !ok {
		t.Fatalf("expected request in dry-run response")
	}
	if _, ok := response["side_effects"]; !ok {
		t.Fatalf("expected side_effects in dry-run response")
	}
}

func TestRunMainDryRunIncludesRequestAndSideEffects(t *testing.T) {
	t.Parallel()

	repoRoot := initTestGitRepo(t)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"ship", "it", "--dry-run", "--output", "json"}, repoRoot, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q", code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr in json dry-run mode, got %q", stderr.String())
	}

	response := map[string]any{}
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse main dry-run output: %v (%q)", err, stdout.String())
	}
	if _, ok := response["result"]; !ok {
		t.Fatalf("expected result envelope in dry-run response")
	}
	if _, ok := response["request"]; !ok {
		t.Fatalf("expected request in dry-run response")
	}
	if _, ok := response["side_effects"]; !ok {
		t.Fatalf("expected side_effects in dry-run response")
	}
}

func writeSessionMetadata(t *testing.T, repoRoot string, worktreeID string, record sessionRecord) {
	t.Helper()

	metadataPath := filepath.Join(repoRoot, ".worktree", worktreeID, "run", "ralph-loop.json")
	if err := os.MkdirAll(filepath.Dir(metadataPath), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	body, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if err := os.WriteFile(metadataPath, body, 0o644); err != nil {
		t.Fatalf("write metadata failed: %v", err)
	}
}

func initTestGitRepo(t *testing.T) string {
	t.Helper()

	repoRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(repoRoot, "go.mod"), []byte("module example.com/testralph\n\ngo 1.22\n"), 0o644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	cmd := exec.Command("git", "init")
	cmd.Dir = repoRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git init failed: %v (%s)", err, string(output))
	}
	return repoRoot
}
