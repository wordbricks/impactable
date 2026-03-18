package gitimpact

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	Velen    VelenConfig    `json:"velen"`
	Analysis AnalysisConfig `json:"analysis"`
}

type VelenConfig struct {
	Org     string        `json:"org"`
	Sources SourceMapping `json:"sources"`
}

type SourceMapping struct {
	GitHub    string `json:"github"`
	Warehouse string `json:"warehouse"`
	Analytics string `json:"analytics"`
}

type AnalysisConfig struct {
	BeforeWindowDays int     `json:"before_window_days"`
	AfterWindowDays  int     `json:"after_window_days"`
	CooldownHours    int     `json:"cooldown_hours"`
	MinConfidence    float64 `json:"min_confidence"`
}

func defaultConfig() Config {
	return Config{
		Analysis: AnalysisConfig{
			BeforeWindowDays: 7,
			AfterWindowDays:  7,
			CooldownHours:    24,
			MinConfidence:    0.6,
		},
	}
}

func loadConfig(cwd string, path string) (Config, string, error) {
	resolved, err := resolveConfigPath(cwd, path)
	if err != nil {
		return Config{}, "", err
	}
	body, err := os.ReadFile(resolved)
	if err != nil {
		return Config{}, "", err
	}
	values, err := parseConfigYAML(body)
	if err != nil {
		return Config{}, "", err
	}

	cfg := defaultConfig()
	if value, ok := values["velen.org"]; ok {
		cfg.Velen.Org = value
	}
	if value, ok := values["velen.sources.github"]; ok {
		cfg.Velen.Sources.GitHub = value
	}
	if value, ok := values["velen.sources.warehouse"]; ok {
		cfg.Velen.Sources.Warehouse = value
	}
	if value, ok := values["velen.sources.analytics"]; ok {
		cfg.Velen.Sources.Analytics = value
	}
	if value, ok := values["analysis.before_window_days"]; ok {
		parsed, parseErr := strconv.Atoi(value)
		if parseErr != nil {
			return Config{}, "", fmt.Errorf("analysis.before_window_days must be an integer")
		}
		cfg.Analysis.BeforeWindowDays = parsed
	}
	if value, ok := values["analysis.after_window_days"]; ok {
		parsed, parseErr := strconv.Atoi(value)
		if parseErr != nil {
			return Config{}, "", fmt.Errorf("analysis.after_window_days must be an integer")
		}
		cfg.Analysis.AfterWindowDays = parsed
	}
	if value, ok := values["analysis.cooldown_hours"]; ok {
		parsed, parseErr := strconv.Atoi(value)
		if parseErr != nil {
			return Config{}, "", fmt.Errorf("analysis.cooldown_hours must be an integer")
		}
		cfg.Analysis.CooldownHours = parsed
	}
	if value, ok := values["analysis.min_confidence"]; ok {
		parsed, parseErr := strconv.ParseFloat(value, 64)
		if parseErr != nil {
			return Config{}, "", fmt.Errorf("analysis.min_confidence must be a number")
		}
		cfg.Analysis.MinConfidence = parsed
	}
	if err := validateConfig(cfg); err != nil {
		return Config{}, "", err
	}
	return cfg, resolved, nil
}

func resolveConfigPath(cwd string, path string) (string, error) {
	configPath := strings.TrimSpace(path)
	if configPath == "" {
		configPath = defaultConfigPath
	}
	if !filepath.IsAbs(configPath) {
		configPath = filepath.Join(cwd, configPath)
	}
	return filepath.Clean(configPath), nil
}

func validateConfig(cfg Config) error {
	if strings.TrimSpace(cfg.Velen.Org) == "" {
		return fmt.Errorf("velen.org is required")
	}
	if strings.TrimSpace(cfg.Velen.Sources.GitHub) == "" {
		return fmt.Errorf("velen.sources.github is required")
	}
	if strings.TrimSpace(cfg.Velen.Sources.Warehouse) == "" {
		return fmt.Errorf("velen.sources.warehouse is required")
	}
	if strings.TrimSpace(cfg.Velen.Sources.Analytics) == "" {
		return fmt.Errorf("velen.sources.analytics is required")
	}
	if cfg.Analysis.BeforeWindowDays <= 0 {
		return fmt.Errorf("analysis.before_window_days must be greater than zero")
	}
	if cfg.Analysis.AfterWindowDays <= 0 {
		return fmt.Errorf("analysis.after_window_days must be greater than zero")
	}
	if cfg.Analysis.CooldownHours < 0 {
		return fmt.Errorf("analysis.cooldown_hours must be zero or greater")
	}
	if math.IsNaN(cfg.Analysis.MinConfidence) || math.IsInf(cfg.Analysis.MinConfidence, 0) || cfg.Analysis.MinConfidence < 0 || cfg.Analysis.MinConfidence > 1 {
		return fmt.Errorf("analysis.min_confidence must be between 0 and 1")
	}
	return nil
}

func parseConfigYAML(body []byte) (map[string]string, error) {
	values := map[string]string{}
	scanner := bufio.NewScanner(strings.NewReader(string(body)))
	stack := []string{}

	for lineNo := 1; scanner.Scan(); lineNo++ {
		line := scanner.Text()
		if strings.ContainsRune(line, '\t') {
			return nil, fmt.Errorf("tabs are not supported in config (line %d)", lineNo)
		}
		trimmed := strings.TrimSpace(stripInlineComment(line))
		if trimmed == "" {
			continue
		}

		indent := leadingSpaces(line)
		if indent%2 != 0 {
			return nil, fmt.Errorf("invalid indentation on line %d", lineNo)
		}
		level := indent / 2
		if level > len(stack) {
			return nil, fmt.Errorf("unexpected indentation on line %d", lineNo)
		}
		stack = stack[:level]

		parts := strings.SplitN(trimmed, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid key-value entry on line %d", lineNo)
		}
		key := strings.TrimSpace(parts[0])
		if key == "" {
			return nil, fmt.Errorf("empty key on line %d", lineNo)
		}
		value := strings.TrimSpace(parts[1])
		if value == "" {
			stack = append(stack, key)
			continue
		}
		path := append(append([]string{}, stack...), key)
		values[strings.Join(path, ".")] = unquoteYAML(value)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return values, nil
}

func leadingSpaces(value string) int {
	count := 0
	for _, ch := range value {
		if ch != ' ' {
			break
		}
		count++
	}
	return count
}

func stripInlineComment(line string) string {
	var builder strings.Builder
	quote := rune(0)
	for _, ch := range line {
		if quote == 0 && ch == '#' {
			break
		}
		if ch == '\'' || ch == '"' {
			if quote == 0 {
				quote = ch
			} else if quote == ch {
				quote = 0
			}
		}
		builder.WriteRune(ch)
	}
	return builder.String()
}

func unquoteYAML(value string) string {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) >= 2 {
		if (trimmed[0] == '"' && trimmed[len(trimmed)-1] == '"') || (trimmed[0] == '\'' && trimmed[len(trimmed)-1] == '\'') {
			return trimmed[1 : len(trimmed)-1]
		}
	}
	return trimmed
}
