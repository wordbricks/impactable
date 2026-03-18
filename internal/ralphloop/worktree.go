package ralphloop

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func prepareWorktree(ctx context.Context, cwd string, repoRoot string, req initRequest) (worktreeInfo, projectCommands, error) {
	baseBranch := strings.TrimSpace(req.BaseBranch)
	if baseBranch == "" {
		baseBranch = defaultBaseBranch
	}
	workBranch := sanitizeBranch(req.WorkBranch)
	if workBranch == "" {
		workBranch = defaultInitBranch()
	}

	currentRoot, err := resolveRepoRoot(cwd)
	if err != nil {
		return worktreeInfo{}, projectCommands{}, err
	}

	var info worktreeInfo
	linked := isLinkedWorktree(currentRoot)
	if linked {
		info = worktreeInfo{
			RepoRoot:       repoRoot,
			WorktreePath:   currentRoot,
			WorkBranch:     workBranch,
			BaseBranch:     baseBranch,
			LinkedWorktree: true,
			Reused:         true,
		}
	} else {
		info = worktreeInfo{
			RepoRoot:       repoRoot,
			WorktreePath:   filepath.Join(repoRoot, ".worktrees", workBranch),
			WorkBranch:     workBranch,
			BaseBranch:     baseBranch,
			LinkedWorktree: false,
		}
	}

	if req.DryRun {
		info.WorktreeID, err = deriveWorktreeID(info.WorktreePath)
		if err != nil {
			return worktreeInfo{}, projectCommands{}, err
		}
		info.RuntimeRoot = runtimeRoot(info.WorktreePath, info.WorktreeID)
		commands, err := detectProjectCommands(repoRoot)
		return info, commands, err
	}

	if linked {
		if err := ensureWorktreeBranch(ctx, info.WorktreePath, workBranch); err != nil {
			return worktreeInfo{}, projectCommands{}, err
		}
		if err := stashIfDirty(ctx, info.WorktreePath); err != nil {
			return worktreeInfo{}, projectCommands{}, err
		}
	} else {
		created, err := createOrReuseLinkedWorktree(ctx, repoRoot, info.WorktreePath, workBranch, baseBranch)
		if err != nil {
			return worktreeInfo{}, projectCommands{}, err
		}
		info.Reused = !created
	}

	info.WorktreeID, err = deriveWorktreeID(info.WorktreePath)
	if err != nil {
		return worktreeInfo{}, projectCommands{}, err
	}
	info.RuntimeRoot = runtimeRoot(info.WorktreePath, info.WorktreeID)
	if err := os.MkdirAll(filepath.Join(info.RuntimeRoot, "logs"), 0o755); err != nil {
		return worktreeInfo{}, projectCommands{}, err
	}
	if err := os.MkdirAll(filepath.Join(info.RuntimeRoot, "run"), 0o755); err != nil {
		return worktreeInfo{}, projectCommands{}, err
	}

	commands, err := detectProjectCommands(info.WorktreePath)
	if err != nil {
		return worktreeInfo{}, projectCommands{}, err
	}
	if len(commands.Install) > 0 {
		if _, err := runCommand(ctx, info.WorktreePath, commands.Install[0], commands.Install[1:]...); err != nil {
			return worktreeInfo{}, projectCommands{}, fmt.Errorf("dependency install failed: %w", err)
		}
	}
	if len(commands.Verify) > 0 {
		if _, err := runCommand(ctx, info.WorktreePath, commands.Verify[0], commands.Verify[1:]...); err != nil {
			return worktreeInfo{}, projectCommands{}, fmt.Errorf("verification failed: %w", err)
		}
	}
	return info, commands, nil
}

func ensureWorktreeBranch(ctx context.Context, worktreePath string, branch string) error {
	current := currentBranch(ctx, worktreePath)
	if current == branch {
		return nil
	}
	if branchExists(ctx, worktreePath, branch) {
		_, err := runCommand(ctx, worktreePath, "git", "checkout", branch)
		return err
	}
	_, err := runCommand(ctx, worktreePath, "git", "checkout", "-b", branch)
	return err
}

func createOrReuseLinkedWorktree(ctx context.Context, repoRoot string, worktreePath string, workBranch string, baseBranch string) (bool, error) {
	if fileExists(filepath.Join(worktreePath, ".git")) {
		return false, nil
	}
	if err := os.MkdirAll(filepath.Dir(worktreePath), 0o755); err != nil {
		return false, err
	}
	if branchExists(ctx, repoRoot, workBranch) {
		_, err := runCommand(ctx, repoRoot, "git", "worktree", "add", worktreePath, workBranch)
		return true, err
	}
	_, err := runCommand(ctx, repoRoot, "git", "worktree", "add", "-b", workBranch, worktreePath, baseBranch)
	return true, err
}

func branchExists(ctx context.Context, repoRoot string, branch string) bool {
	result, err := runCommand(ctx, repoRoot, "git", "show-ref", "--verify", "--quiet", "refs/heads/"+branch)
	return err == nil && result.ExitCode == 0
}

func stashIfDirty(ctx context.Context, worktreePath string) error {
	result, err := runCommand(ctx, worktreePath, "git", "status", "--porcelain")
	if err != nil {
		return err
	}
	if strings.TrimSpace(result.Stdout) == "" {
		return nil
	}
	_, err = runCommand(ctx, worktreePath, "git", "stash", "push", "--include-untracked", "-m", "ralph-loop init autostash")
	return err
}

func cleanupWorktree(ctx context.Context, repoRoot string, worktree worktreeInfo) error {
	if worktree.LinkedWorktree {
		return nil
	}
	_, err := runCommand(ctx, repoRoot, "git", "worktree", "remove", "--force", worktree.WorktreePath)
	return err
}
