package ralphloop

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type commandResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

func nowUTC() time.Time {
	return time.Now().UTC()
}

func resolveRepoRoot(cwd string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = cwd
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to resolve repository root from %s: %w", cwd, err)
	}
	return strings.TrimSpace(string(output)), nil
}

func runCommand(ctx context.Context, dir string, name string, args ...string) (commandResult, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	result := commandResult{Stdout: stdout.String(), Stderr: stderr.String()}
	if exitErr, ok := err.(*exec.ExitError); ok {
		result.ExitCode = exitErr.ExitCode()
	} else if err == nil {
		result.ExitCode = 0
	} else {
		result.ExitCode = -1
	}
	return result, err
}

func currentBranch(ctx context.Context, dir string) string {
	result, err := runCommand(ctx, dir, "git", "branch", "--show-current")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(result.Stdout)
}

func currentHead(ctx context.Context, dir string) string {
	result, err := runCommand(ctx, dir, "git", "rev-parse", "HEAD")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(result.Stdout)
}

func isLinkedWorktree(worktreePath string) bool {
	info, err := os.Lstat(filepath.Join(worktreePath, ".git"))
	if err != nil {
		return false
	}
	return info.Mode().IsRegular()
}

func detectProjectCommands(root string) (projectCommands, error) {
	switch {
	case fileExists(filepath.Join(root, "go.mod")):
		return projectCommands{
			ProjectType: "go",
			Install:     []string{"go", "mod", "download"},
			Verify:      []string{"go", "test", "./..."},
		}, nil
	case fileExists(filepath.Join(root, "Cargo.toml")):
		return projectCommands{
			ProjectType: "cargo",
			Verify:      []string{"cargo", "build"},
		}, nil
	case fileExists(filepath.Join(root, "package.json")):
		return detectNodeCommands(root)
	default:
		return projectCommands{}, fmt.Errorf("unsupported repository type under %s", root)
	}
}

func detectNodeCommands(root string) (projectCommands, error) {
	body, err := os.ReadFile(filepath.Join(root, "package.json"))
	if err != nil {
		return projectCommands{}, err
	}
	pkg := struct {
		Scripts map[string]string `json:"scripts"`
	}{}
	if err := json.Unmarshal(body, &pkg); err != nil {
		return projectCommands{}, err
	}
	installer := []string{"npm", "install"}
	runner := "npm"
	if fileExists(filepath.Join(root, "bun.lock")) || fileExists(filepath.Join(root, "bun.lockb")) {
		installer = []string{"bun", "install"}
		runner = "bun"
	}
	verify := []string{}
	switch {
	case pkg.Scripts["build"] != "":
		verify = []string{runner, "run", "build"}
	case pkg.Scripts["test"] != "":
		verify = []string{runner, "run", "test"}
	default:
		return projectCommands{}, fmt.Errorf("package.json does not define a build or test script")
	}
	return projectCommands{
		ProjectType: "node",
		Install:     installer,
		Verify:      verify,
	}, nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
