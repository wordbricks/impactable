package gitimpact

import (
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

var (
	ErrAnalyzeNotImplemented      = errors.New("analyze command is not implemented yet")
	ErrCheckSourcesNotImplemented = errors.New("check-sources command is not implemented yet")
)

// Run executes git-impact CLI commands.
func Run(args []string, cwd string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	root := NewRootCommand(cwd, stdin, stdout, stderr)
	root.SetArgs(args)
	if err := root.Execute(); err != nil {
		if stderr != nil {
			_, _ = fmt.Fprintln(stderr, err)
		}
		return 1
	}
	return 0
}

// NewRootCommand builds the Cobra command tree for git-impact.
func NewRootCommand(cwd string, stdin io.Reader, stdout io.Writer, stderr io.Writer) *cobra.Command {
	var analyzeArgs CLIArgs
	var checkSourcesConfigPath string

	root := &cobra.Command{
		Use:           "git-impact",
		Short:         "Analyze git change impact against product metrics",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.SetIn(stdin)
	root.SetOut(stdout)
	root.SetErr(stderr)

	analyzeCmd := &cobra.Command{
		Use:   "analyze",
		Short: "Run impact analysis",
		RunE: func(_ *cobra.Command, _ []string) error {
			ctx, err := NewAnalysisContext(cwd, analyzeArgs)
			if err != nil {
				return err
			}
			if _, err := LoadConfig(ctx.ConfigPath); err != nil {
				return err
			}
			return ErrAnalyzeNotImplemented
		},
	}
	analyzeCmd.Flags().StringVar(&analyzeArgs.ConfigPath, "config", DefaultConfigFile, "Path to impact analyzer config file")
	analyzeCmd.Flags().StringVar(&analyzeArgs.Since, "since", "", "Analyze changes since YYYY-MM-DD")
	analyzeCmd.Flags().IntVar(&analyzeArgs.PR, "pr", 0, "Analyze a specific PR number")
	analyzeCmd.Flags().StringVar(&analyzeArgs.Feature, "feature", "", "Analyze a specific feature group")

	checkSourcesCmd := &cobra.Command{
		Use:   "check-sources",
		Short: "Validate configured Velen sources",
		RunE: func(_ *cobra.Command, _ []string) error {
			ctx, err := NewAnalysisContext(cwd, CLIArgs{ConfigPath: checkSourcesConfigPath})
			if err != nil {
				return err
			}
			if _, err := LoadConfig(ctx.ConfigPath); err != nil {
				return err
			}
			return ErrCheckSourcesNotImplemented
		},
	}
	checkSourcesCmd.Flags().StringVar(&checkSourcesConfigPath, "config", DefaultConfigFile, "Path to impact analyzer config file")

	root.AddCommand(analyzeCmd)
	root.AddCommand(checkSourcesCmd)
	return root
}
