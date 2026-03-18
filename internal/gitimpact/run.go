package gitimpact

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func Run(args []string, cwd string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	parsed, err := parseCLI(args, stdin)
	if err != nil {
		if errors.Is(err, errHelpRequested) {
			_, _ = io.WriteString(stdout, renderHelp())
			return 0
		}
		hint := outputHintFromArgs(args, stdout)
		return emitFailure(cwd, hint.Command, hint.Format, hint.OutputFile, stdout, stderr, err)
	}

	selected := selectedOutputForParsed(parsed, stdout)
	switch parsed.Kind {
	case commandAnalyze:
		if err := runAnalyze(cwd, parsed.Analyze, stdout); err != nil {
			return emitFailure(cwd, selected.Command, selected.Format, selected.OutputFile, stdout, stderr, err)
		}
	case commandCheckSources:
		if err := runCheckSources(cwd, parsed.CheckSources, stdout); err != nil {
			return emitFailure(cwd, selected.Command, selected.Format, selected.OutputFile, stdout, stderr, err)
		}
	case commandReportScaffold:
		if err := runReportScaffold(cwd, parsed.ReportScaffold, stdout); err != nil {
			return emitFailure(cwd, selected.Command, selected.Format, selected.OutputFile, stdout, stderr, err)
		}
	case commandSchema:
		if err := runSchema(cwd, parsed.Schema, stdout); err != nil {
			return emitFailure(cwd, selected.Command, selected.Format, selected.OutputFile, stdout, stderr, err)
		}
	default:
		return emitFailure(cwd, selected.Command, selected.Format, selected.OutputFile, stdout, stderr, fmt.Errorf("unknown command %q", parsed.Kind))
	}
	return 0
}

func runAnalyze(cwd string, req analyzeRequest, stdout io.Writer) error {
	format := resolveOutput(req.Output, stdout)
	response := map[string]any{
		"command": commandAnalyze,
		"status":  "ok",
		"request": map[string]any{
			"config": req.ConfigPath,
			"pr":     req.PRNumber,
			"since":  req.Since,
		},
		"result": map[string]any{
			"phase":                  "phase-1-foundation",
			"mode":                   "contract",
			"analysis_path":          "single_pr",
			"analysis_status":        "not_started",
			"next_runtime_milestone": "M2_config_loading",
		},
	}
	text := renderAnalyzeText(req)
	return emitSingle(cwd, format, req.OutputFile, stdout, response, text)
}

func runCheckSources(cwd string, req checkSourcesRequest, stdout io.Writer) error {
	format := resolveOutput(req.Output, stdout)
	sources := make([]sourceCheckContract, 0, len(req.RequiredRoles))
	for _, role := range req.RequiredRoles {
		sources = append(sources, sourceCheckContract{
			Role:    role,
			Status:  "pending",
			Message: "velen integration abstraction not wired yet",
		})
	}
	response := map[string]any{
		"command": commandCheckSources,
		"status":  "ok",
		"request": map[string]any{
			"config":         req.ConfigPath,
			"required_roles": req.RequiredRoles,
		},
		"summary": map[string]any{
			"required": len(req.RequiredRoles),
			"pending":  len(req.RequiredRoles),
		},
		"sources": sources,
	}
	text := renderCheckSourcesText(req, sources)
	return emitSingle(cwd, format, req.OutputFile, stdout, response, text)
}

func runReportScaffold(cwd string, req reportScaffoldRequest, stdout io.Writer) error {
	format := resolveOutput(req.Output, stdout)
	modes, err := normalizeModes(req.Modes)
	if err != nil {
		return err
	}
	scaffolds := make([]reportScaffoldContract, 0, len(modes))
	for _, mode := range modes {
		scaffolds = append(scaffolds, reportScaffoldContract{
			Mode:   mode,
			Status: "planned",
			Path:   scaffoldPathForMode(req.OutputDir, mode),
		})
	}
	response := map[string]any{
		"command": commandReportScaffold,
		"status":  "ok",
		"request": map[string]any{
			"config":     req.ConfigPath,
			"modes":      modes,
			"output_dir": req.OutputDir,
		},
		"reports": scaffolds,
	}
	text := renderReportScaffoldText(req, scaffolds)
	return emitSingle(cwd, format, req.OutputFile, stdout, response, text)
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
	response := map[string]any{
		"command": commandSchema,
		"status":  "ok",
		"items":   descriptors,
	}
	return emitSingle(cwd, format, req.OutputFile, stdout, response, renderSchemaText(descriptors))
}

type outputSelection struct {
	Command    string
	Format     string
	OutputFile string
}

func selectedOutputForParsed(parsed parsedCommand, stdout io.Writer) outputSelection {
	switch parsed.Kind {
	case commandAnalyze:
		return outputSelection{Command: commandAnalyze, Format: resolveOutput(parsed.Analyze.Output, stdout), OutputFile: parsed.Analyze.OutputFile}
	case commandCheckSources:
		return outputSelection{Command: commandCheckSources, Format: resolveOutput(parsed.CheckSources.Output, stdout), OutputFile: parsed.CheckSources.OutputFile}
	case commandReportScaffold:
		return outputSelection{Command: commandReportScaffold, Format: resolveOutput(parsed.ReportScaffold.Output, stdout), OutputFile: parsed.ReportScaffold.OutputFile}
	case commandSchema:
		return outputSelection{Command: commandSchema, Format: resolveOutput(parsed.Schema.Output, stdout), OutputFile: parsed.Schema.OutputFile}
	default:
		return outputSelection{Command: commandAnalyze, Format: resolveOutput("", stdout)}
	}
}

func outputHintFromArgs(args []string, stdout io.Writer) outputSelection {
	selection := outputSelection{Command: commandAnalyze, Format: resolveOutput("", stdout)}
	if len(args) > 0 {
		selection.Command = strings.TrimSpace(args[0])
	}
	for index := 0; index < len(args); index++ {
		switch args[index] {
		case "--output":
			if index+1 < len(args) {
				selection.Format = strings.TrimSpace(args[index+1])
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
		emitErr := emitSingle(cwd, format, outputFile, stdout, payload, mustJSON(payload))
		if emitErr == nil {
			return 1
		}
		_, _ = fmt.Fprintln(stderr, emitErr.Error())
		return 1
	}
	_, _ = fmt.Fprintln(stderr, err.Error())
	return 1
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
	return emitPayload(cwd, outputFile, stdout, data)
}

func marshalForFormat(format string, payload any, text string) (string, error) {
	switch strings.TrimSpace(format) {
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

func renderAnalyzeText(req analyzeRequest) string {
	return fmt.Sprintf("git-impact analyze (contract mode)\nconfig: %s\npr: %d\nsince: %s", req.ConfigPath, req.PRNumber, emptyAsNone(req.Since))
}

func renderCheckSourcesText(req checkSourcesRequest, sources []sourceCheckContract) string {
	lines := []string{
		"git-impact check-sources (contract mode)",
		fmt.Sprintf("config: %s", req.ConfigPath),
	}
	for _, source := range sources {
		lines = append(lines, fmt.Sprintf("- %s: %s", source.Role, source.Status))
	}
	return strings.Join(lines, "\n")
}

func renderReportScaffoldText(req reportScaffoldRequest, reports []reportScaffoldContract) string {
	lines := []string{
		"git-impact report-scaffold (contract mode)",
		fmt.Sprintf("config: %s", req.ConfigPath),
		fmt.Sprintf("output_dir: %s", req.OutputDir),
	}
	for _, report := range reports {
		lines = append(lines, fmt.Sprintf("- %s: %s", report.Mode, report.Path))
	}
	return strings.Join(lines, "\n")
}

func renderSchemaText(items []commandDescriptor) string {
	lines := make([]string, 0, len(items)+1)
	lines = append(lines, "git-impact schema")
	for _, item := range items {
		lines = append(lines, fmt.Sprintf("%s: %s", item.Name, item.Description))
	}
	return strings.Join(lines, "\n")
}

func renderHelp() string {
	return strings.Join([]string{
		"git-impact commands:",
		"  analyze          Analyze git impact scope (foundation contract).",
		"  check-sources    Check required source roles (foundation contract).",
		"  report-scaffold  Build report output-mode scaffold contract.",
		"  schema           Describe command contracts as JSON schema metadata.",
	}, "\n") + "\n"
}

func emptyAsNone(value string) string {
	if strings.TrimSpace(value) == "" {
		return "none"
	}
	return value
}

func scaffoldPathForMode(outputDir string, mode string) string {
	switch mode {
	case "terminal":
		return "stdout"
	case "json":
		return filepath.Join(outputDir, "impact-report.json")
	case "markdown":
		return filepath.Join(outputDir, "impact-report.md")
	case "html":
		return filepath.Join(outputDir, "impact-report.html")
	default:
		return filepath.Join(outputDir, "impact-report.txt")
	}
}
