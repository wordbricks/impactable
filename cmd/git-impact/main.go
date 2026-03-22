package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"impactable/internal/gitimpact"
)

type appOptions struct {
	configPath string
	output     string
}

func main() {
	root := newRootCmd(os.Stdout, os.Stderr)
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func newRootCmd(stdout io.Writer, stderr io.Writer) *cobra.Command {
	opts := &appOptions{
		configPath: gitimpact.DefaultConfigPath,
		output:     defaultOutput(stdout),
	}

	root := &cobra.Command{
		Use:           "git-impact",
		Short:         "Analyze product impact from Git and analytics sources",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			opts.output = strings.ToLower(strings.TrimSpace(opts.output))
			if opts.output == "" {
				opts.output = defaultOutput(stdout)
			}
			if opts.output != "text" && opts.output != "json" {
				return fmt.Errorf("invalid --output value %q (expected text or json)", opts.output)
			}
			return nil
		},
	}

	root.PersistentFlags().StringVar(&opts.configPath, "config", gitimpact.DefaultConfigPath, "Path to config file")
	root.PersistentFlags().StringVar(&opts.output, "output", opts.output, "Output format: text|json")

	root.AddCommand(newAnalyzeCmd(opts, stdout))
	root.AddCommand(newCheckSourcesCmd(opts, stdout, stderr))

	return root
}

func newAnalyzeCmd(opts *appOptions, stdout io.Writer) *cobra.Command {
	var since string
	var pr int
	var feature string

	cmd := &cobra.Command{
		Use:   "analyze",
		Short: "Run a git-impact analysis",
		RunE: func(_ *cobra.Command, _ []string) error {
			analysisCtx, err := gitimpact.NewAnalysisContext(since, pr, feature, opts.configPath)
			if err != nil {
				emitCommandError(stdout, opts.output, "analyze", err)
				return err
			}
			prompt := gitimpact.BuildInitialPrompt(analysisCtx)

			if opts.output == "json" {
				payload := map[string]any{
					"command":          "analyze",
					"status":           "ok",
					"analysis_context": analysisCtx,
					"initial_prompt":   prompt,
				}
				return encodeJSON(stdout, payload)
			}

			_, _ = fmt.Fprintf(stdout, "Analysis context loaded.\n")
			_, _ = fmt.Fprintf(stdout, "Since: %s\n", displayValue(analysisCtx.Since))
			if analysisCtx.PRNumber != nil {
				_, _ = fmt.Fprintf(stdout, "PR: #%d\n", *analysisCtx.PRNumber)
			} else {
				_, _ = fmt.Fprintf(stdout, "PR: not provided\n")
			}
			_, _ = fmt.Fprintf(stdout, "Feature: %s\n", displayValue(analysisCtx.FeatureName))
			_, _ = fmt.Fprintf(stdout, "Config: %s\n\n", analysisCtx.ConfigPath)
			_, _ = fmt.Fprintln(stdout, "Initial prompt:")
			_, _ = fmt.Fprintln(stdout, prompt)
			return nil
		},
	}

	cmd.Flags().StringVar(&since, "since", "", "Analyze changes since date (YYYY-MM-DD)")
	cmd.Flags().IntVar(&pr, "pr", 0, "Analyze a single PR number")
	cmd.Flags().StringVar(&feature, "feature", "", "Analyze a feature by name")
	return cmd
}

func newCheckSourcesCmd(opts *appOptions, stdout io.Writer, stderr io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "check-sources",
		Short: "Check required Velen sources",
		RunE: func(_ *cobra.Command, _ []string) error {
			cfg, _, err := gitimpact.LoadConfig(opts.configPath)
			if err != nil {
				emitCommandError(stdout, opts.output, "check-sources", err)
				return err
			}

			result, checkErr := gitimpact.CheckSources(context.Background(), cfg)
			if result == nil {
				result = &gitimpact.SourceCheckResult{}
			}

			if opts.output == "json" {
				if err := encodeJSON(stdout, result); err != nil {
					return err
				}
			} else {
				printCheckSourcesText(stdout, result)
			}

			if checkErr != nil {
				if opts.output == "text" {
					_, _ = fmt.Fprintf(stderr, "Source check failed: %s\n", checkErr.Error())
				}
				return checkErr
			}
			return nil
		},
	}
	return cmd
}

func printCheckSourcesText(writer io.Writer, result *gitimpact.SourceCheckResult) {
	_, _ = fmt.Fprintln(writer, "Source check")
	_, _ = fmt.Fprintf(writer, "Org: %s\n", displayValue(result.OrgName))
	if result.GitHubSource != nil {
		_, _ = fmt.Fprintf(writer, "GitHub source: %s (%s)\n", displayValue(result.GitHubSource.Key), displayValue(result.GitHubSource.ProviderType))
	} else {
		_, _ = fmt.Fprintln(writer, "GitHub source: not found")
	}
	_, _ = fmt.Fprintf(writer, "GitHub QUERY support: %s\n", statusLabel(result.GitHubOK))

	if result.AnalyticsSource != nil {
		_, _ = fmt.Fprintf(writer, "Analytics source: %s (%s)\n", displayValue(result.AnalyticsSource.Key), displayValue(result.AnalyticsSource.ProviderType))
	} else {
		_, _ = fmt.Fprintln(writer, "Analytics source: not found")
	}
	_, _ = fmt.Fprintf(writer, "Analytics QUERY support: %s\n", statusLabel(result.AnalyticsOK))

	if strings.TrimSpace(result.Error) != "" {
		_, _ = fmt.Fprintf(writer, "Error: %s\n", result.Error)
	}
}

func statusLabel(ok bool) string {
	if ok {
		return "ok"
	}
	return "failed"
}

func displayValue(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "not provided"
	}
	return trimmed
}

func defaultOutput(stdout io.Writer) string {
	file, ok := stdout.(*os.File)
	if !ok {
		return "json"
	}
	info, err := file.Stat()
	if err != nil {
		return "json"
	}
	if info.Mode()&os.ModeCharDevice != 0 {
		return "text"
	}
	return "json"
}

func encodeJSON(writer io.Writer, payload any) error {
	encoder := json.NewEncoder(writer)
	return encoder.Encode(payload)
}

func emitCommandError(stdout io.Writer, output string, command string, err error) {
	if output != "json" {
		return
	}
	_ = encodeJSON(stdout, map[string]any{
		"command": command,
		"status":  "failed",
		"error":   err.Error(),
	})
}
