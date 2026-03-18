package ralphloop

import (
	"fmt"
	"path/filepath"
	"strings"
)

func defaultPlanFilename(prompt string) string {
	slug := slugify(prompt)
	if len(slug) > 80 {
		slug = strings.Trim(slug[:80], "-")
	}
	if slug == "" {
		slug = "task"
	}
	return slug + ".md"
}

func buildSetupPrompt(prompt string, planPath string, worktree worktreeInfo) string {
	planName := strings.TrimSuffix(filepath.Base(planPath), filepath.Ext(planPath))
	return fmt.Sprintf(`You are the setup agent for an automated coding loop.

Task: %s

Prepared environment:
- Worktree path: %s
- Worktree ID: %s
- Working branch: %s
- Base branch: %s

Do the following in order:
1. Read AGENTS.md and relevant repository documentation.
2. Create an execution plan at %s with sections for Goal, Background, Milestones, Current progress, Key decisions, Remaining issues, and Links.
3. Break the work into 3-7 concrete milestones.
4. Mark every milestone as not started.
5. Stage and commit the plan with message: plan: %s
6. Print the absolute plan path.

Output %s when done.`, prompt, worktree.WorktreePath, worktree.WorktreeID, worktree.WorkBranch, worktree.BaseBranch, planPath, planName, completeToken)
}

func buildCodingPrompt(prompt string, planPath string) string {
	return fmt.Sprintf(`You are a coding agent working in an automated loop.

Task:
%s

Execution plan:
Read %s and continue from the current milestone.

Rules:
- Complete exactly one milestone per iteration.
- Update the plan file with progress and decisions.
- Stage and commit all changes from the iteration.
- If blocked, document the blocker in the plan and commit the current state.

When all milestones are complete, output %s.`, prompt, planPath, completeToken)
}

func buildRecoveryPrompt(planPath string) string {
	return fmt.Sprintf("The previous iteration failed. Repair the workspace, review the plan at %s, and continue with the next safe milestone. Commit recovery work before stopping.", planPath)
}

func buildPRPrompt(planPath string, baseBranch string) string {
	return fmt.Sprintf(`You are the PR agent for a completed Ralph Loop run.

Instructions:
1. Read the plan at %s.
2. Review commits with git log %s..HEAD --oneline.
3. Move the plan from docs/exec-plans/active/ to docs/exec-plans/completed/ and commit that move.
4. Create a pull request with a concise title and sections for Summary, Milestones completed, Key decisions, and Test plan.
5. Print the PR URL.

Output %s when done.`, planPath, baseBranch, completeToken)
}
