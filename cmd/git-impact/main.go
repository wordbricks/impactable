package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"impactable/internal/gitimpact"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type terminalController interface {
	ReleaseTerminal() error
	RestoreTerminal() error
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

type cliState struct {
	configPath string
	output     string
}

func run(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	root, state := newRootCommand(stdin, stdout, stderr)
	root.SetArgs(args)
	if err := root.Execute(); err != nil {
		return emitCommandError(state.output, stdout, stderr, err)
	}
	return 0
}

func newRootCommand(stdin io.Reader, stdout io.Writer, stderr io.Writer) (*cobra.Command, *cliState) {
	state := &cliState{
		configPath: gitimpact.DefaultConfigFile,
		output:     defaultOutput(stdout),
	}

	root := &cobra.Command{
		Use:           "git-impact",
		Short:         "Analyze git change impact against product metrics",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			state.output = strings.ToLower(strings.TrimSpace(state.output))
			switch state.output {
			case "text", "json":
				return nil
			default:
				return fmt.Errorf("invalid --output value %q (expected text or json)", state.output)
			}
		},
	}
	root.SetIn(stdin)
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.PersistentFlags().StringVar(&state.configPath, "config", gitimpact.DefaultConfigFile, "Path to impact analyzer config file")
	root.PersistentFlags().StringVar(&state.output, "output", state.output, "Output format (text or json)")

	var since string
	var prNum int
	var feature string
	analyzeCmd := &cobra.Command{
		Use:   "analyze",
		Short: "Run impact analysis",
		RunE: func(cmd *cobra.Command, _ []string) error {
			analysisCtx, err := gitimpact.NewAnalysisContext(since, prNum, feature, state.configPath)
			if err != nil {
				return err
			}
			cfg, err := gitimpact.LoadConfig(analysisCtx.ConfigPath)
			if err != nil {
				return err
			}

			runCtx := &gitimpact.RunContext{
				Config:      &cfg,
				AnalysisCtx: analysisCtx,
				VelenClient: gitimpact.NewVelenClient(0),
			}

			interactiveTUI := state.output == "text" && isTerminalWriter(stdout)
			if interactiveTUI {
				result, err := runAnalyzeWithTUI(cmd.Context(), stdin, stdout, runCtx)
				if err != nil {
					return err
				}
				if result == nil {
					return fmt.Errorf("analysis completed without a result")
				}
				return nil
			}

			waitHandler := newNonInteractiveWaitHandler()
			if isTerminalReader(stdin) {
				waitHandler = newPromptWaitHandler(stdin, stdout, nil)
			}
			engine := gitimpact.NewDefaultEngine(runCtx.VelenClient, nil, waitHandler)
			result, err := engine.Run(cmd.Context(), runCtx)
			if err != nil {
				return err
			}
			if result == nil {
				return fmt.Errorf("analysis completed without a result")
			}

			payload := map[string]any{
				"command":        "analyze",
				"status":         "ok",
				"result":         result,
				"context":        analysisCtx,
				"initial_prompt": gitimpact.BuildInitialPrompt(analysisCtx, &cfg),
			}
			return emitAnalyzeResult(state.output, stdout, payload, result)
		},
	}
	analyzeCmd.Flags().SetNormalizeFunc(normalizeAnalyzeFlagName)
	analyzeCmd.Flags().StringVar(&since, "since", "", "Analyze changes since YYYY-MM-DD")
	analyzeCmd.Flags().IntVar(&prNum, "pr", 0, "Analyze a specific PR number")
	analyzeCmd.Flags().StringVar(&feature, "feature", "", "Analyze a specific feature group")

	checkSourcesCmd := &cobra.Command{
		Use:   "check-sources",
		Short: "Validate configured Velen sources",
		RunE: func(cmd *cobra.Command, _ []string) error {
			analysisCtx, err := gitimpact.NewAnalysisContext("", 0, "", state.configPath)
			if err != nil {
				return err
			}

			cfg, err := gitimpact.LoadConfig(analysisCtx.ConfigPath)
			if err != nil {
				return err
			}

			result, err := gitimpact.CheckSources(cmd.Context(), gitimpact.NewVelenClient(0), &cfg)
			if err != nil {
				return err
			}
			return emitSourceCheckResult(state.output, stdout, result)
		},
	}

	root.AddCommand(analyzeCmd)
	root.AddCommand(checkSourcesCmd)
	return root, state
}

func normalizeAnalyzeFlagName(_ *pflag.FlagSet, name string) pflag.NormalizedName {
	if strings.EqualFold(strings.TrimSpace(name), "r") {
		return pflag.NormalizedName("pr")
	}
	return pflag.NormalizedName(name)
}

func runAnalyzeWithTUI(ctx context.Context, stdin io.Reader, stdout io.Writer, runCtx *gitimpact.RunContext) (*gitimpact.AnalysisResult, error) {
	phases := gitimpact.DefaultAnalysisPhases()
	model := gitimpact.NewAnalysisModel(phases)
	programOptions := []tea.ProgramOption{
		tea.WithOutput(stdout),
		tea.WithoutSignalHandler(),
	}
	if isTerminalReader(stdin) {
		programOptions = append(programOptions, tea.WithInput(stdin))
	} else {
		programOptions = append(programOptions, tea.WithInput(nil))
	}
	program := tea.NewProgram(&model, programOptions...)

	type progressProgramResult struct {
		model tea.Model
		err   error
	}
	runDone := make(chan progressProgramResult, 1)
	go func() {
		finalModel, err := program.Run()
		runDone <- progressProgramResult{model: finalModel, err: err}
	}()

	observer := gitimpact.NewTUIObserver(program)
	waitHandler := newNonInteractiveWaitHandler()
	if isTerminalReader(stdin) {
		waitHandler = newPromptWaitHandler(stdin, stdout, program)
	}
	engine := gitimpact.NewDefaultEngine(runCtx.VelenClient, observer, waitHandler)

	result, runErr := engine.Run(ctx, runCtx)
	progressProgram := <-runDone
	if runErr != nil {
		return nil, runErr
	}
	if progressProgram.err != nil {
		return nil, fmt.Errorf("run analysis progress TUI: %w", progressProgram.err)
	}
	if progressModel, ok := progressProgram.model.(*gitimpact.AnalysisModel); ok && progressModel.ShouldShowResults() {
		result = progressModel.Result()
	}
	if result == nil {
		return nil, fmt.Errorf("analysis completed without a result")
	}

	saveHandler := newResultsSaveHandler(runCtx.AnalysisCtx, result)
	resultsModel := gitimpact.NewResultsModel(result, saveHandler)
	resultsProgram := tea.NewProgram(
		&resultsModel,
		tea.WithInput(stdin),
		tea.WithOutput(stdout),
		tea.WithoutSignalHandler(),
	)
	if _, err := resultsProgram.Run(); err != nil {
		return nil, fmt.Errorf("run analysis results TUI: %w", err)
	}

	return result, nil
}

func newResultsSaveHandler(analysisCtx *gitimpact.AnalysisContext, result *gitimpact.AnalysisResult) gitimpact.SaveReportFunc {
	baseDir := "."
	if analysisCtx != nil && strings.TrimSpace(analysisCtx.WorkingDirectory) != "" {
		baseDir = analysisCtx.WorkingDirectory
	}

	return func(format string) (string, error) {
		normalized := strings.ToLower(strings.TrimSpace(format))
		stamp := time.Now().UTC().Format("20060102-150405")
		switch normalized {
		case "md":
			path := filepath.Join(baseDir, fmt.Sprintf("git-impact-report-%s.md", stamp))
			return path, gitimpact.SaveMarkdown(result, path)
		case "html":
			path := filepath.Join(baseDir, fmt.Sprintf("git-impact-report-%s.html", stamp))
			return path, gitimpact.SaveHTML(result, path)
		default:
			return "", fmt.Errorf("unsupported report format %q", format)
		}
	}
}

func emitAnalyzeResult(output string, stdout io.Writer, payload map[string]any, result *gitimpact.AnalysisResult) error {
	if output == "json" {
		return emitJSON(stdout, payload)
	}

	body := map[string]any{
		"status": "ok",
		"result": result,
	}
	textBody, err := json.MarshalIndent(body, "", "  ")
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintln(stdout, string(textBody))
	return nil
}

func emitSourceCheckResult(output string, stdout io.Writer, result *gitimpact.SourceCheckResult) error {
	status := "ok"
	if !result.GitHubOK || !result.AnalyticsOK || len(result.Errors) > 0 {
		status = "issues"
	}

	if output == "json" {
		return emitJSON(stdout, map[string]any{
			"command": "check-sources",
			"status":  status,
			"result":  result,
		})
	}

	_, _ = fmt.Fprintf(stdout, "organization: %s\n", fallbackText(result.OrgName, "unknown"))
	_, _ = fmt.Fprintf(stdout, "github: %s (query=%t)\n", sourceLabel(result.GitHubSource), result.GitHubOK)
	_, _ = fmt.Fprintf(stdout, "analytics: %s (query=%t)\n", sourceLabel(result.AnalyticsSource), result.AnalyticsOK)
	if len(result.Errors) == 0 {
		_, _ = fmt.Fprintln(stdout, "status: ok")
		return nil
	}
	_, _ = fmt.Fprintln(stdout, "status: issues")
	for _, issue := range result.Errors {
		_, _ = fmt.Fprintf(stdout, "- %s\n", issue)
	}
	return nil
}

func emitJSON(stdout io.Writer, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(stdout, "%s\n", body)
	return nil
}

func emitCommandError(output string, stdout io.Writer, stderr io.Writer, err error) int {
	if strings.EqualFold(strings.TrimSpace(output), "json") {
		_ = emitJSON(stdout, map[string]any{
			"status": "failed",
			"error": map[string]string{
				"code":    "command_failed",
				"message": err.Error(),
			},
		})
		return 1
	}
	_, _ = fmt.Fprintln(stderr, err.Error())
	return 1
}

func defaultOutput(stdout io.Writer) string {
	if isTerminalWriter(stdout) {
		return "text"
	}
	return "json"
}

func isTerminalWriter(writer io.Writer) bool {
	file, ok := writer.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

func isTerminalReader(reader io.Reader) bool {
	file, ok := reader.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

func newPromptWaitHandler(stdin io.Reader, stdout io.Writer, terminal terminalController) gitimpact.WaitHandler {
	reader := bufio.NewReader(stdin)
	return func(message string) (string, error) {
		if terminal != nil {
			if err := terminal.ReleaseTerminal(); err != nil {
				return "", fmt.Errorf("release terminal for prompt: %w", err)
			}
			defer func() {
				_ = terminal.RestoreTerminal()
			}()
		}

		prompt := strings.TrimSpace(message)
		if prompt != "" {
			_, _ = fmt.Fprintln(stdout, prompt)
		}
		_, _ = fmt.Fprint(stdout, "> ")

		response, err := reader.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return "", err
		}
		return strings.TrimSpace(response), nil
	}
}

func newNonInteractiveWaitHandler() gitimpact.WaitHandler {
	return func(message string) (string, error) {
		return "", fmt.Errorf("analysis requires user input: %s", strings.TrimSpace(message))
	}
}

func sourceLabel(source *gitimpact.Source) string {
	if source == nil {
		return "missing"
	}
	if source.SourceKey() != "" {
		return source.SourceKey()
	}
	return "unknown"
}

func fallbackText(value string, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	return trimmed
}
