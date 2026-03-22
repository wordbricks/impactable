package gitimpact

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const sinceDateLayout = "2006-01-02"

type CLIArgs struct {
	ConfigPath string
	Since      string
	PR         int
	Feature    string
}

// NewAnalysisContext converts parsed CLI arguments into a runtime context.
func NewAnalysisContext(cwd string, args CLIArgs) (AnalysisContext, error) {
	workingDirectory := strings.TrimSpace(cwd)
	if workingDirectory == "" {
		wd, err := os.Getwd()
		if err != nil {
			return AnalysisContext{}, fmt.Errorf("resolve working directory: %w", err)
		}
		workingDirectory = wd
	}

	configPath, err := resolveConfigPath(workingDirectory, args.ConfigPath)
	if err != nil {
		return AnalysisContext{}, err
	}

	since, err := parseSince(args.Since)
	if err != nil {
		return AnalysisContext{}, err
	}

	feature := strings.TrimSpace(args.Feature)
	if args.PR < 0 {
		return AnalysisContext{}, fmt.Errorf("--pr must be zero or a positive integer")
	}
	if args.PR > 0 && feature != "" {
		return AnalysisContext{}, fmt.Errorf("--pr and --feature cannot be set together")
	}

	return AnalysisContext{
		WorkingDirectory: workingDirectory,
		ConfigPath:       configPath,
		Since:            since,
		PRNumber:         args.PR,
		Feature:          feature,
	}, nil
}

func resolveConfigPath(cwd string, configPath string) (string, error) {
	trimmed := strings.TrimSpace(configPath)
	if trimmed == "" {
		trimmed = DefaultConfigFile
	}
	if filepath.IsAbs(trimmed) {
		return filepath.Clean(trimmed), nil
	}
	if strings.TrimSpace(cwd) == "" {
		return "", fmt.Errorf("working directory is required to resolve relative --config")
	}
	return filepath.Clean(filepath.Join(cwd, trimmed)), nil
}

func parseSince(value string) (*time.Time, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, nil
	}
	parsed, err := time.Parse(sinceDateLayout, trimmed)
	if err != nil {
		return nil, fmt.Errorf("invalid --since value %q (expected YYYY-MM-DD)", trimmed)
	}
	return &parsed, nil
}
