package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"impactable/internal/gitimpact"

	"github.com/spf13/cobra"
)

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
		RunE: func(_ *cobra.Command, _ []string) error {
			analysisCtx, err := gitimpact.NewAnalysisContext(since, prNum, feature, state.configPath)
			if err != nil {
				return err
			}
			cfg, err := gitimpact.LoadConfig(analysisCtx.ConfigPath)
			if err != nil {
				return err
			}

			payload := map[string]any{
				"command":        "analyze",
				"status":         "not_implemented",
				"message":        "analysis not yet implemented",
				"context":        analysisCtx,
				"initial_prompt": gitimpact.BuildInitialPrompt(analysisCtx, &cfg),
			}
			return emitAnalyzeResult(state.output, stdout, payload, analysisCtx)
		},
	}
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

func emitAnalyzeResult(output string, stdout io.Writer, payload map[string]any, analysisCtx *gitimpact.AnalysisContext) error {
	if output == "json" {
		return emitJSON(stdout, payload)
	}

	contextBody, err := json.MarshalIndent(analysisCtx, "", "  ")
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintln(stdout, "analysis not yet implemented")
	_, _ = fmt.Fprintln(stdout, string(contextBody))
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

func sourceLabel(source *gitimpact.Source) string {
	if source == nil {
		return "missing"
	}
	if strings.TrimSpace(source.Key) != "" {
		return source.Key
	}
	if strings.TrimSpace(source.Name) != "" {
		return source.Name
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
