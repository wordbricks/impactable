package ralphloop

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func Run(args []string, cwd string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	parsed, err := parseCLI(args, stdin)
	if err != nil {
		hint := outputHintFromArgs(args, stdout)
		return emitFailure(cwd, hint.Command, hint.Format, hint.OutputFile, stdout, stderr, err)
	}
	selected := selectedOutputForParsed(parsed, stdout)

	switch parsed.Kind {
	case commandSchema:
		if err := runSchema(cwd, parsed.Schema, stdout); err != nil {
			return emitFailure(cwd, selected.Command, selected.Format, selected.OutputFile, stdout, stderr, err)
		}
		return 0
	}

	repoRoot, err := resolveRepoRoot(cwd)
	if err != nil {
		return emitFailure(cwd, selected.Command, selected.Format, selected.OutputFile, stdout, stderr, err)
	}

	switch parsed.Kind {
	case commandInit:
		if err := runInit(context.Background(), cwd, repoRoot, parsed.Init, stdout, stderr); err != nil {
			return emitFailure(cwd, selected.Command, selected.Format, selected.OutputFile, stdout, stderr, err)
		}
	case commandList:
		if err := runList(cwd, repoRoot, parsed.List, stdout); err != nil {
			return emitFailure(cwd, selected.Command, selected.Format, selected.OutputFile, stdout, stderr, err)
		}
	case commandTail:
		if err := runTail(context.Background(), cwd, repoRoot, parsed.Tail, stdout); err != nil {
			return emitFailure(cwd, selected.Command, selected.Format, selected.OutputFile, stdout, stderr, err)
		}
	default:
		if err := runMain(context.Background(), cwd, repoRoot, parsed.Main, stdout, stderr); err != nil {
			return emitFailure(cwd, selected.Command, selected.Format, selected.OutputFile, stdout, stderr, err)
		}
	}
	return 0
}

type outputSelection struct {
	Command    string
	Format     string
	OutputFile string
}

func selectedOutputForParsed(parsed parsedCommand, stdout io.Writer) outputSelection {
	switch parsed.Kind {
	case commandInit:
		return outputSelection{
			Command:    commandInit,
			Format:     resolveOutput(parsed.Init.Output, stdout),
			OutputFile: parsed.Init.OutputFile,
		}
	case commandList:
		return outputSelection{
			Command:    commandList,
			Format:     resolveOutput(parsed.List.Output, stdout),
			OutputFile: parsed.List.OutputFile,
		}
	case commandTail:
		return outputSelection{
			Command:    commandTail,
			Format:     resolveOutput(parsed.Tail.Output, stdout),
			OutputFile: parsed.Tail.OutputFile,
		}
	case commandSchema:
		return outputSelection{
			Command:    commandSchema,
			Format:     resolveOutput(parsed.Schema.Output, stdout),
			OutputFile: parsed.Schema.OutputFile,
		}
	default:
		return outputSelection{
			Command:    commandMain,
			Format:     resolveOutput(parsed.Main.Output, stdout),
			OutputFile: parsed.Main.OutputFile,
		}
	}
}

func outputHintFromArgs(args []string, stdout io.Writer) outputSelection {
	selection := outputSelection{
		Command: commandMain,
		Format:  resolveOutput("", stdout),
	}
	if len(args) > 0 {
		if _, ok := knownCommands[args[0]]; ok {
			selection.Command = args[0]
		}
	}
	for index := 0; index < len(args); index++ {
		switch args[index] {
		case "--output":
			if index+1 < len(args) {
				selection.Format = args[index+1]
				index++
			}
		case "--output-file":
			if index+1 < len(args) {
				selection.OutputFile = args[index+1]
				index++
			}
		}
	}
	return selection
}

func emitFailure(cwd string, command string, format string, outputFile string, stdout io.Writer, stderr io.Writer, err error) int {
	if strings.TrimSpace(format) == "json" || strings.TrimSpace(format) == "ndjson" {
		payload := map[string]any{
			"command": command,
			"status":  "failed",
			"error": structuredError{
				Code:    "command_failed",
				Message: err.Error(),
			},
		}
		text := mustJSON(payload)
		emitErr := emitSingle(cwd, format, outputFile, stdout, payload, text)
		if emitErr == nil {
			return 1
		}
		// If machine-readable rendering itself fails, fallback to stderr as process-level failure.
		_, _ = fmt.Fprintln(stderr, emitErr.Error())
		return 1
	}
	_, _ = fmt.Fprintln(stderr, err.Error())
	return 1
}

func runSchema(cwd string, req schemaRequest, stdout io.Writer) error {
	descriptors := commandDescriptors()
	target := strings.TrimSpace(req.TargetCommand)
	if target == "" {
		target = strings.TrimSpace(req.CommandName)
	}
	if target != "" {
		filtered := make([]commandDescriptor, 0, 1)
		for _, descriptor := range descriptors {
			if descriptor.Name == target {
				filtered = append(filtered, descriptor)
			}
		}
		descriptors = filtered
	}
	format := resolveOutput(req.Output, stdout)
	envelope := map[string]any{
		"command": "schema",
		"status":  "ok",
		"items":   descriptors,
	}
	return emitSingle(cwd, format, req.OutputFile, stdout, envelope, renderSchemaText(descriptors))
}

func runList(cwd string, repoRoot string, req listRequest, stdout io.Writer) error {
	sessions, err := listSessions(repoRoot, req.Selector)
	if err != nil {
		return err
	}
	records := make([]map[string]any, 0, len(sessions))
	for _, session := range sessions {
		body := map[string]any{
			"pid":                    session.PID,
			"worktree_id":            session.WorktreeID,
			"worktree_path":          session.WorktreePath,
			"work_branch":            session.WorkBranch,
			"log_path":               session.LogPath,
			"started_at":             session.StartedAt,
			"relative_worktree_path": session.RelativeWorktreePath,
			"relative_log_path":      session.RelativeLogPath,
		}
		records = append(records, applyFieldMaskMap(body, req.Fields))
	}
	format := resolveOutput(req.Output, stdout)
	return emitReadResult(cwd, format, req.OutputFile, stdout, commandList, records, req.Page, req.PageSize, req.PageAll, renderSessionText(sessions))
}

func runTail(ctx context.Context, cwd string, repoRoot string, req tailRequest, stdout io.Writer) error {
	format := resolveOutput(req.Output, stdout)
	paths, err := findLogs(repoRoot, req.Selector)
	if err != nil {
		return err
	}
	if len(paths) == 0 {
		return fmt.Errorf("no Ralph Loop logs found")
	}
	if req.Follow {
		if format != "ndjson" {
			format = "ndjson"
		}
		return followLog(ctx, paths[0], req.Lines, req.Raw, stdout)
	}
	records, err := readTail(paths[0], req.Lines, req.Raw)
	if err != nil {
		return err
	}
	items := make([]map[string]any, 0, len(records))
	for _, record := range records {
		body := map[string]any{
			"line":        record.Line,
			"rendered":    record.Rendered,
			"raw":         record.Raw,
			"line_number": record.LineNumber,
		}
		items = append(items, applyFieldMaskMap(body, req.Fields))
	}
	textLines := make([]string, 0, len(records)+1)
	textLines = append(textLines, paths[0])
	for _, record := range records {
		textLines = append(textLines, record.Rendered)
	}
	return emitReadResult(cwd, format, req.OutputFile, stdout, commandTail, items, req.Page, req.PageSize, req.PageAll, strings.Join(textLines, "\n"))
}

func runInit(ctx context.Context, cwd string, repoRoot string, req initRequest, stdout io.Writer, stderr io.Writer) error {
	format := resolveOutput(req.Output, stdout)
	if req.DryRun {
		worktree, commands, err := prepareWorktree(ctx, cwd, repoRoot, req)
		if err != nil {
			return err
		}
		envelope := map[string]any{
			"command":   commandInit,
			"status":    "ok",
			"dry_run":   true,
			"worktree":  worktree,
			"project":   commands,
			"repo_root": repoRoot,
		}
		return emitSingle(cwd, format, req.OutputFile, stdout, envelope, mustJSON(envelope))
	}

	if format == "text" {
		_, _ = fmt.Fprintln(stderr, "Preparing worktree")
	}
	worktree, commands, err := prepareWorktree(ctx, cwd, repoRoot, req)
	if err != nil {
		return err
	}
	envelope := map[string]any{
		"command":       commandInit,
		"status":        "ok",
		"worktree_id":   worktree.WorktreeID,
		"worktree_path": worktree.WorktreePath,
		"work_branch":   worktree.WorkBranch,
		"base_branch":   worktree.BaseBranch,
		"runtime_root":  worktree.RuntimeRoot,
		"project":       commands,
	}
	return emitSingle(cwd, format, req.OutputFile, stdout, envelope, mustJSON(envelope))
}

func runMain(ctx context.Context, cwd string, repoRoot string, req mainRequest, stdout io.Writer, stderr io.Writer) error {
	format := resolveOutput(req.Output, stdout)
	initReq := initRequest{
		Command:    commandInit,
		BaseBranch: req.BaseBranch,
		WorkBranch: req.WorkBranch,
		DryRun:     req.DryRun,
	}
	if req.DryRun {
		worktree, commands, err := prepareWorktree(ctx, cwd, repoRoot, initReq)
		if err != nil {
			return err
		}
		planPath := filepath.Join(worktree.WorktreePath, "docs", "exec-plans", "active", defaultPlanFilename(req.Prompt))
		result := runResult{
			Command:      commandMain,
			Status:       "ok",
			Phase:        "dry-run",
			WorktreeID:   worktree.WorktreeID,
			WorktreePath: worktree.WorktreePath,
			WorkBranch:   worktree.WorkBranch,
			RuntimeRoot:  worktree.RuntimeRoot,
			PlanPath:     planPath,
		}
		envelope := map[string]any{
			"result":  result,
			"project": commands,
		}
		return emitSingle(cwd, format, req.OutputFile, stdout, envelope, mustJSON(envelope))
	}

	worktree, _, err := prepareWorktree(ctx, cwd, repoRoot, initReq)
	if err != nil {
		return err
	}
	logPath := filepath.Join(worktree.RuntimeRoot, "logs", "ralph-loop.log")
	logger := newLoopLogger(logPath)
	cleanupSession, err := registerSession(worktree, logPath)
	if err != nil {
		return err
	}
	defer cleanupSession()
	if !req.PreserveTree {
		defer func() {
			_ = cleanupWorktree(context.Background(), repoRoot, worktree)
		}()
	}

	planPath := filepath.Join(worktree.WorktreePath, "docs", "exec-plans", "active", defaultPlanFilename(req.Prompt))
	if err := os.MkdirAll(filepath.Dir(planPath), 0o755); err != nil {
		return err
	}

	emitEvent(stdout, format, newEvent(commandMain, "run.started"), map[string]any{
		"status":        "running",
		"worktree_path": worktree.WorktreePath,
		"work_branch":   worktree.WorkBranch,
		"log_path":      logPath,
		"plan_path":     planPath,
	})

	setupClient, err := newAppServerClient(logger)
	if err != nil {
		return err
	}
	defer setupClient.Close()
	setupClient.SetNotificationHandler(func(notification jsonRPCNotification) {
		if message := agentMessage(notification); message != "" && format == "ndjson" {
			event := newEvent(commandMain, "agent.message")
			event.Status = "running"
			event.Phase = "setup"
			event.Message = message
			event.WorktreePath = worktree.WorktreePath
			event.WorkBranch = worktree.WorkBranch
			event.PlanPath = planPath
			emitEvent(stdout, format, event, nil)
		}
	})
	if err := setupClient.Initialize(ctx); err != nil {
		return err
	}
	threadID, err := setupClient.StartThread(ctx, req.Model, worktree.WorktreePath, req.ApprovalPolicy, resolveSandbox(req.Sandbox))
	if err != nil {
		return err
	}
	status, agentText, err := setupClient.RunTurn(ctx, threadID, buildSetupPrompt(req.Prompt, planPath, worktree), turnTimeout(req.TimeoutSeconds))
	if err != nil {
		return err
	}
	if strings.EqualFold(status, "failed") || !containsCompletionSignal(agentText) {
		return fmt.Errorf("setup agent did not complete successfully")
	}

	codingClient, err := newAppServerClient(logger)
	if err != nil {
		return err
	}
	defer codingClient.Close()
	if err := codingClient.Initialize(ctx); err != nil {
		return err
	}
	codingThreadID, err := codingClient.StartThread(ctx, req.Model, worktree.WorktreePath, req.ApprovalPolicy, resolveSandbox(req.Sandbox))
	if err != nil {
		return err
	}
	iterations := 0
	nextPrompt := buildCodingPrompt(req.Prompt, planPath)
	for iterations < req.MaxIterations {
		iterations++
		event := newEvent(commandMain, "iteration.started")
		event.Status = "running"
		event.Phase = "coding"
		event.Iteration = iterations
		event.WorktreePath = worktree.WorktreePath
		event.WorkBranch = worktree.WorkBranch
		event.PlanPath = planPath
		emitEvent(stdout, format, event, nil)

		status, agentText, err = codingClient.RunTurn(ctx, codingThreadID, nextPrompt, turnTimeout(req.TimeoutSeconds))
		if err != nil {
			return err
		}
		if strings.EqualFold(status, "failed") {
			nextPrompt = buildRecoveryPrompt(planPath)
			continue
		}
		if containsCompletionSignal(agentText) {
			break
		}
		nextPrompt = buildCodingPrompt(req.Prompt, planPath)
	}
	if iterations >= req.MaxIterations && !containsCompletionSignal(agentText) {
		return fmt.Errorf("reached max iterations without completion")
	}

	prClient, err := newAppServerClient(logger)
	if err != nil {
		return err
	}
	defer prClient.Close()
	if err := prClient.Initialize(ctx); err != nil {
		return err
	}
	prThreadID, err := prClient.StartThread(ctx, req.Model, worktree.WorktreePath, req.ApprovalPolicy, resolvePRSandbox(req.Sandbox, worktree.WorktreePath))
	if err != nil {
		return err
	}
	status, agentText, err = prClient.RunTurn(ctx, prThreadID, buildPRPrompt(planPath, req.BaseBranch), turnTimeout(req.TimeoutSeconds))
	if err != nil {
		return err
	}
	if strings.EqualFold(status, "failed") {
		return fmt.Errorf("pr agent failed")
	}
	result := runResult{
		Command:      commandMain,
		Status:       "completed",
		Phase:        "completed",
		Iterations:   iterations,
		WorktreeID:   worktree.WorktreeID,
		WorktreePath: worktree.WorktreePath,
		WorkBranch:   worktree.WorkBranch,
		RuntimeRoot:  worktree.RuntimeRoot,
		LogPath:      logPath,
		PlanPath:     planPath,
		PRURL:        extractPullURL(agentText),
	}
	emitEvent(stdout, format, newEvent(commandMain, "run.completed"), map[string]any{
		"status":        "completed",
		"phase":         "completed",
		"iterations":    iterations,
		"worktree_path": worktree.WorktreePath,
		"work_branch":   worktree.WorkBranch,
		"plan_path":     planPath,
		"log_path":      logPath,
		"pr_url":        result.PRURL,
	})
	return emitSingle(cwd, "json", req.OutputFile, stdout, result, mustJSON(result))
}

func resolveOutput(requested string, stdout io.Writer) string {
	requested = strings.TrimSpace(requested)
	if requested != "" {
		return requested
	}
	if file, ok := stdout.(*os.File); ok {
		if info, err := file.Stat(); err == nil && (info.Mode()&os.ModeCharDevice) != 0 {
			return "text"
		}
	}
	return "json"
}

func emitSingle(cwd string, format string, outputFile string, stdout io.Writer, payload any, text string) error {
	data, err := marshalForFormat(format, payload, text)
	if err != nil {
		return err
	}
	if err := emitPayload(cwd, outputFile, stdout, data); err != nil {
		return err
	}
	return nil
}

func emitReadResult(cwd string, format string, outputFile string, stdout io.Writer, command string, items []map[string]any, page int, pageSize int, pageAll bool, text string) error {
	if page <= 0 {
		page = defaultPage
	}
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	pages := paginate(items, pageSize)
	if len(pages) == 0 {
		pages = [][]map[string]any{{}}
	}
	if pageAll {
		if format == "ndjson" {
			lines := make([]string, 0, len(pages))
			for index, items := range pages {
				envelope := map[string]any{
					"command":   command,
					"status":    "ok",
					"page":      index + 1,
					"page_size": pageSize,
					"total":     len(items),
					"page_all":  true,
					"items":     items,
				}
				body, _ := json.Marshal(envelope)
				lines = append(lines, string(body))
			}
			return emitPayload(cwd, outputFile, stdout, strings.Join(lines, "\n")+"\n")
		}
		envelope := map[string]any{"command": command, "status": "ok", "page_all": true, "items": items}
		return emitSingle(cwd, format, outputFile, stdout, envelope, text)
	}
	if page > len(pages) {
		page = len(pages)
	}
	selected := pages[page-1]
	envelope := map[string]any{
		"command":   command,
		"status":    "ok",
		"page":      page,
		"page_size": pageSize,
		"total":     len(items),
		"items":     selected,
	}
	return emitSingle(cwd, format, outputFile, stdout, envelope, text)
}

func paginate(items []map[string]any, pageSize int) [][]map[string]any {
	if len(items) == 0 {
		return nil
	}
	pages := [][]map[string]any{}
	for start := 0; start < len(items); start += pageSize {
		end := start + pageSize
		if end > len(items) {
			end = len(items)
		}
		pages = append(pages, items[start:end])
	}
	return pages
}

func marshalForFormat(format string, payload any, text string) (string, error) {
	switch format {
	case "text":
		return text + "\n", nil
	case "ndjson":
		body, err := json.Marshal(payload)
		if err != nil {
			return "", err
		}
		return string(body) + "\n", nil
	default:
		body, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			return "", err
		}
		return string(body) + "\n", nil
	}
}

func emitPayload(cwd string, outputFile string, stdout io.Writer, data string) error {
	if strings.TrimSpace(outputFile) != "" {
		path, err := resolveOutputPath(cwd, outputFile)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
			return err
		}
	}
	_, err := io.WriteString(stdout, data)
	return err
}

func renderSchemaText(items []commandDescriptor) string {
	lines := make([]string, 0, len(items)+1)
	lines = append(lines, "ralph-loop schema")
	for _, item := range items {
		lines = append(lines, fmt.Sprintf("%s: %s", item.Name, item.Description))
	}
	return strings.Join(lines, "\n")
}

func renderSessionText(sessions []sessionView) string {
	if len(sessions) == 0 {
		return "No running Ralph Loop sessions found."
	}
	lines := make([]string, 0, len(sessions)+1)
	lines = append(lines, "Running Ralph Loop sessions:")
	for _, session := range sessions {
		lines = append(lines, fmt.Sprintf("%d %s %s %s", session.PID, session.WorktreeID, session.WorkBranch, session.LogPath))
	}
	return strings.Join(lines, "\n")
}

func resolveSandbox(mode string) any {
	switch mode {
	case "read-only", "readOnly":
		return "read-only"
	case "danger-full-access", "dangerFullAccess":
		return "danger-full-access"
	default:
		return "workspace-write"
	}
}

func resolvePRSandbox(mode string, worktreePath string) any {
	_ = worktreePath
	return resolveSandbox(mode)
}

func emitEvent(stdout io.Writer, format string, event commandEvent, overrides map[string]any) {
	if format != "ndjson" {
		return
	}
	if overrides != nil {
		if status, ok := overrides["status"].(string); ok {
			event.Status = status
		}
		if phase, ok := overrides["phase"].(string); ok {
			event.Phase = phase
		}
		if iteration, ok := overrides["iteration"].(int); ok {
			event.Iteration = iteration
		}
		if worktreePath, ok := overrides["worktree_path"].(string); ok {
			event.WorktreePath = worktreePath
		}
		if workBranch, ok := overrides["work_branch"].(string); ok {
			event.WorkBranch = workBranch
		}
		if planPath, ok := overrides["plan_path"].(string); ok {
			event.PlanPath = planPath
		}
		if logPath, ok := overrides["log_path"].(string); ok {
			event.LogPath = logPath
		}
		if prURL, ok := overrides["pr_url"].(string); ok {
			event.PRURL = prURL
		}
		if message, ok := overrides["message"].(string); ok {
			event.Message = message
		}
	}
	body, _ := json.Marshal(event)
	_, _ = fmt.Fprintln(stdout, string(body))
}

func agentMessage(notification jsonRPCNotification) string {
	if strings.TrimSpace(notification.Method) != "item/completed" {
		return ""
	}
	return extractAgentTextDisplay(notification.Params["item"])
}

func turnTimeout(seconds int) time.Duration {
	if seconds <= 0 {
		return 30 * time.Minute
	}
	return time.Duration(seconds) * time.Second
}
