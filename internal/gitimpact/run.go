package gitimpact

import (
	"context"
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
	cfg, resolvedConfig, err := loadConfig(cwd, req.ConfigPath)
	if err != nil {
		return err
	}
	if req.PRNumber <= 0 {
		return fmt.Errorf("analyze requires --pr for the Phase 1 single-PR path")
	}
	analysis, stages, err := analyzeSinglePR(context.Background(), newVelenClient(), cfg, req.PRNumber)
	if err != nil {
		return err
	}
	format := resolveOutput(req.Output, stdout)
	response := successEnvelope(
		commandAnalyze,
		map[string]any{
			"config": resolvedConfig,
			"pr":     req.PRNumber,
			"since":  req.Since,
		},
		map[string]any{
			"velen_org": cfg.Velen.Org,
			"sources": map[string]any{
				"github":    cfg.Velen.Sources.GitHub,
				"warehouse": cfg.Velen.Sources.Warehouse,
				"analytics": cfg.Velen.Sources.Analytics,
			},
			"analysis": map[string]any{
				"before_window_days": cfg.Analysis.BeforeWindowDays,
				"after_window_days":  cfg.Analysis.AfterWindowDays,
				"cooldown_hours":     cfg.Analysis.CooldownHours,
				"min_confidence":     cfg.Analysis.MinConfidence,
			},
		},
		map[string]any{
			"phase":                  "phase-1-foundation",
			"mode":                   "contract",
			"analysis_path":          "single_pr",
			"analysis_status":        "completed",
			"next_runtime_milestone": "M5_report_scaffolding",
			"stages":                 stages,
			"single_pr":              analysis,
		},
	)
	text := renderAnalyzeText(req, resolvedConfig)
	return emitSingle(cwd, format, req.OutputFile, stdout, response, text)
}

func runCheckSources(cwd string, req checkSourcesRequest, stdout io.Writer) error {
	cfg, resolvedConfig, err := loadConfig(cwd, req.ConfigPath)
	if err != nil {
		return err
	}
	checks, summary, checkContext, err := checkRequiredSources(context.Background(), cfg, req.RequiredRoles, newVelenClient())
	if err != nil {
		return err
	}
	format := resolveOutput(req.Output, stdout)
	response := successEnvelope(
		commandCheckSources,
		map[string]any{
			"config":         resolvedConfig,
			"required_roles": req.RequiredRoles,
		},
		map[string]any{
			"velen_org": cfg.Velen.Org,
		},
		map[string]any{
			"velen":   checkContext,
			"summary": summary,
			"sources": checks,
		},
	)
	text := renderCheckSourcesText(resolvedConfig, checks, summary)
	return emitSingle(cwd, format, req.OutputFile, stdout, response, text)
}

func runReportScaffold(cwd string, req reportScaffoldRequest, stdout io.Writer) error {
	cfg, resolvedConfig, err := loadConfig(cwd, req.ConfigPath)
	if err != nil {
		return err
	}
	format := resolveOutput(req.Output, stdout)
	modes, err := normalizeModes(req.Modes)
	if err != nil {
		return err
	}
	scaffolds, err := buildReportScaffolds(cwd, req.OutputDir, modes, cfg, resolvedConfig)
	if err != nil {
		return err
	}
	response := successEnvelope(
		commandReportScaffold,
		map[string]any{
			"config":     resolvedConfig,
			"modes":      modes,
			"output_dir": req.OutputDir,
		},
		map[string]any{
			"velen_org": cfg.Velen.Org,
			"analysis": map[string]any{
				"before_window_days": cfg.Analysis.BeforeWindowDays,
				"after_window_days":  cfg.Analysis.AfterWindowDays,
				"cooldown_hours":     cfg.Analysis.CooldownHours,
			},
		},
		map[string]any{
			"reports": scaffolds,
		},
	)
	text := renderReportScaffoldText(req, resolvedConfig, scaffolds)
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
	response := successEnvelope(commandSchema, nil, nil, map[string]any{"items": descriptors})
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
		payload := failureEnvelope(command, err)
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

func renderAnalyzeText(req analyzeRequest, configPath string) string {
	return fmt.Sprintf("git-impact analyze (contract mode)\nconfig: %s\npr: %d\nsince: %s", configPath, req.PRNumber, emptyAsNone(req.Since))
}

func renderCheckSourcesText(configPath string, sources []sourceCheckContract, summary sourceCheckSummary) string {
	lines := []string{
		"git-impact check-sources (contract mode)",
		fmt.Sprintf("config: %s", configPath),
		fmt.Sprintf("summary: ok=%d missing=%d failed=%d ready=%t", summary.OK, summary.Missing, summary.Failed, summary.Ready),
	}
	for _, source := range sources {
		lines = append(lines, fmt.Sprintf("- %s: %s", source.Role, source.Status))
	}
	return strings.Join(lines, "\n")
}

func renderReportScaffoldText(req reportScaffoldRequest, configPath string, reports []reportScaffoldContract) string {
	lines := []string{
		"git-impact report-scaffold (contract mode)",
		fmt.Sprintf("config: %s", configPath),
		fmt.Sprintf("output_dir: %s", req.OutputDir),
	}
	for _, report := range reports {
		lines = append(lines, fmt.Sprintf("- %s (%s): %s", report.Mode, report.Status, report.Path))
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

func buildReportScaffolds(cwd string, outputDir string, modes []string, cfg Config, resolvedConfig string) ([]reportScaffoldContract, error) {
	scaffolds := make([]reportScaffoldContract, 0, len(modes))
	for _, mode := range modes {
		scaffold := reportScaffoldContract{
			Mode: mode,
			Path: scaffoldPathForMode(outputDir, mode),
		}
		content, writable, err := reportScaffoldContent(mode, cfg, resolvedConfig)
		if err != nil {
			return nil, err
		}
		if !writable {
			scaffold.Status = "ready"
			scaffolds = append(scaffolds, scaffold)
			continue
		}

		path, err := resolveOutputPath(cwd, scaffold.Path)
		if err != nil {
			return nil, err
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return nil, err
		}
		scaffold.Status = "written"
		scaffolds = append(scaffolds, scaffold)
	}
	return scaffolds, nil
}

func reportScaffoldContent(mode string, cfg Config, resolvedConfig string) (string, bool, error) {
	switch mode {
	case "terminal":
		return "", false, nil
	case "json":
		type analysisWindow struct {
			BeforeWindowDays int `json:"before_window_days"`
			AfterWindowDays  int `json:"after_window_days"`
			CooldownHours    int `json:"cooldown_hours"`
		}
		type jsonReportScaffold struct {
			Version  string         `json:"version"`
			Format   string         `json:"format"`
			Config   string         `json:"config"`
			VelenOrg string         `json:"velen_org"`
			Metric   string         `json:"metric"`
			Analysis analysisWindow `json:"analysis"`
			Notes    []string       `json:"notes"`
		}
		payload := jsonReportScaffold{
			Version:  "phase-1-foundation",
			Format:   "json",
			Config:   resolvedConfig,
			VelenOrg: cfg.Velen.Org,
			Metric:   mvpMetricName,
			Analysis: analysisWindow{
				BeforeWindowDays: cfg.Analysis.BeforeWindowDays,
				AfterWindowDays:  cfg.Analysis.AfterWindowDays,
				CooldownHours:    cfg.Analysis.CooldownHours,
			},
			Notes: []string{
				"Report scaffold generated by git-impact Phase 1.",
				"Populate with analyze command output when report generation is implemented.",
			},
		}
		body, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			return "", false, err
		}
		return string(body) + "\n", true, nil
	case "markdown":
		lines := []string{
			"# Git Impact Report Scaffold",
			"",
			"- Version: phase-1-foundation",
			"- Format: markdown",
			fmt.Sprintf("- Config: %s", resolvedConfig),
			fmt.Sprintf("- Velen Org: %s", cfg.Velen.Org),
			fmt.Sprintf("- Metric: %s", mvpMetricName),
			fmt.Sprintf("- Analysis Window: before=%d days, after=%d days, cooldown=%d hours", cfg.Analysis.BeforeWindowDays, cfg.Analysis.AfterWindowDays, cfg.Analysis.CooldownHours),
			"",
			"## Summary",
			"",
			"_TODO: populate summary from analyze output._",
			"",
			"## Single PR Impact",
			"",
			"_TODO: render PR-level impact section._",
		}
		return strings.Join(lines, "\n") + "\n", true, nil
	case "html":
		html := strings.Join([]string{
			"<!doctype html>",
			"<html lang=\"en\">",
			"<head>",
			"  <meta charset=\"utf-8\" />",
			"  <meta name=\"viewport\" content=\"width=device-width, initial-scale=1\" />",
			"  <title>Git Impact Report Scaffold</title>",
			"</head>",
			"<body>",
			"  <main>",
			"    <h1>Git Impact Report Scaffold</h1>",
			"    <p>Version: phase-1-foundation</p>",
			fmt.Sprintf("    <p>Config: %s</p>", resolvedConfig),
			fmt.Sprintf("    <p>Velen Org: %s</p>", cfg.Velen.Org),
			fmt.Sprintf("    <p>Metric: %s</p>", mvpMetricName),
			fmt.Sprintf("    <p>Analysis Window: before=%d days, after=%d days, cooldown=%d hours</p>", cfg.Analysis.BeforeWindowDays, cfg.Analysis.AfterWindowDays, cfg.Analysis.CooldownHours),
			"    <section>",
			"      <h2>Summary</h2>",
			"      <p>TODO: populate summary from analyze output.</p>",
			"    </section>",
			"  </main>",
			"</body>",
			"</html>",
		}, "\n")
		return html + "\n", true, nil
	default:
		return "", false, fmt.Errorf("unsupported report mode %q", mode)
	}
}

func successEnvelope(command string, request any, config any, result any) map[string]any {
	envelope := map[string]any{
		"command": command,
		"status":  "ok",
		"result":  result,
	}
	if request != nil {
		envelope["request"] = request
	}
	if config != nil {
		envelope["config"] = config
	}
	return envelope
}

func failureEnvelope(command string, err error) map[string]any {
	return map[string]any{
		"command": command,
		"status":  "failed",
		"error": structuredError{
			Code:    "command_failed",
			Message: err.Error(),
		},
	}
}
