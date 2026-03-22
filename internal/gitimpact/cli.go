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
	_ = cwd
	var configPath string
	var analyzeSince string
	var analyzePR int
	var analyzeFeature string

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
			ctx, err := NewAnalysisContext(analyzeSince, analyzePR, analyzeFeature, configPath)
			if err != nil {
				return err
			}
			_ = ctx
			return ErrAnalyzeNotImplemented
		},
	}
	analyzeCmd.Flags().StringVar(&analyzeSince, "since", "", "Analyze changes since YYYY-MM-DD")
	analyzeCmd.Flags().IntVar(&analyzePR, "pr", 0, "Analyze a specific PR number")
	analyzeCmd.Flags().StringVar(&analyzeFeature, "feature", "", "Analyze a specific feature group")

	checkSourcesCmd := &cobra.Command{
		Use:   "check-sources",
		Short: "Validate configured Velen sources",
		RunE: func(_ *cobra.Command, _ []string) error {
			ctx, err := NewAnalysisContext("", 0, "", configPath)
			if err != nil {
				return err
			}
			_ = ctx
			return ErrCheckSourcesNotImplemented
		},
	}

	root.PersistentFlags().StringVar(&configPath, "config", DefaultConfigFile, "Path to impact analyzer config file")

	root.AddCommand(analyzeCmd)
	root.AddCommand(checkSourcesCmd)
	return root
}
