package ralphloop

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func sessionPaths(worktree worktreeInfo) (string, string) {
	runDir := filepath.Join(worktree.RuntimeRoot, "run")
	return filepath.Join(runDir, "ralph-loop.pid"), filepath.Join(runDir, "ralph-loop.json")
}

func registerSession(worktree worktreeInfo, logPath string) (func(), error) {
	pidPath, metadataPath := sessionPaths(worktree)
	if err := os.MkdirAll(filepath.Dir(pidPath), 0o755); err != nil {
		return nil, err
	}
	pid := os.Getpid()
	if err := os.WriteFile(pidPath, []byte(strconv.Itoa(pid)+"\n"), 0o644); err != nil {
		return nil, err
	}
	record := sessionRecord{
		Command:      commandMain,
		Status:       "running",
		PID:          pid,
		WorktreeID:   worktree.WorktreeID,
		WorktreePath: worktree.WorktreePath,
		WorkBranch:   worktree.WorkBranch,
		BaseBranch:   worktree.BaseBranch,
		RuntimeRoot:  worktree.RuntimeRoot,
		LogPath:      logPath,
		StartedAt:    nowUTC().Format(timeRFC3339()),
	}
	body, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(metadataPath, body, 0o644); err != nil {
		return nil, err
	}
	return func() {
		_ = os.Remove(pidPath)
		_ = os.Remove(metadataPath)
	}, nil
}

func listSessions(repoRoot string, selector string) ([]sessionView, error) {
	metadataFiles := []string{}
	for _, root := range []string{filepath.Join(repoRoot, ".worktree"), filepath.Join(repoRoot, ".worktrees")} {
		if !fileExists(root) {
			continue
		}
		err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				return nil
			}
			if filepath.Base(path) != "ralph-loop.json" {
				return nil
			}
			metadataFiles = append(metadataFiles, path)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	sessions := make([]sessionView, 0, len(metadataFiles))
	for _, metadataPath := range metadataFiles {
		body, err := os.ReadFile(metadataPath)
		if err != nil {
			continue
		}
		record := sessionRecord{}
		if err := json.Unmarshal(body, &record); err != nil {
			continue
		}
		if !pidRunning(record.PID) {
			continue
		}
		view := sessionView{sessionRecord: record}
		if relative, err := filepath.Rel(repoRoot, record.WorktreePath); err == nil && relative != "." {
			view.RelativeWorktreePath = relative
		}
		if relative, err := filepath.Rel(repoRoot, record.LogPath); err == nil && relative != "." {
			view.RelativeLogPath = relative
		}
		if strings.TrimSpace(selector) != "" && !sessionMatches(view, selector) {
			continue
		}
		sessions = append(sessions, view)
	}
	sort.Slice(sessions, func(i, j int) bool {
		if sessions[i].StartedAt == sessions[j].StartedAt {
			return sessions[i].WorktreePath < sessions[j].WorktreePath
		}
		return sessions[i].StartedAt > sessions[j].StartedAt
	})
	return sessions, nil
}

func sessionMatches(view sessionView, selector string) bool {
	target := strings.TrimSpace(selector)
	if target == "" {
		return true
	}
	for _, value := range []string{
		view.WorktreeID,
		view.WorktreePath,
		view.WorkBranch,
		view.LogPath,
		view.RelativeLogPath,
		view.RelativeWorktreePath,
	} {
		if strings.Contains(value, target) {
			return true
		}
	}
	return false
}

func pidRunning(pid int) bool {
	if pid <= 0 {
		return false
	}
	return exec.Command("kill", "-0", strconv.Itoa(pid)).Run() == nil
}

func timeRFC3339() string {
	return "2006-01-02T15:04:05Z07:00"
}
