package gitimpact

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

const (
	DefaultConfigFile          = "impact-analyzer.yaml"
	DefaultBeforeWindowDays    = 7
	DefaultAfterWindowDays     = 7
	DefaultCooldownHours       = 24
	DefaultFeatureMappingsFile = "feature-map.yaml"
)

// LoadConfig reads and decodes impact-analyzer.yaml configuration.
func LoadConfig(configPath string) (Config, error) {
	resolvedPath := strings.TrimSpace(configPath)
	if resolvedPath == "" {
		resolvedPath = DefaultConfigFile
	}

	v := viper.New()
	v.SetConfigFile(resolvedPath)
	v.SetConfigType("yaml")
	v.SetDefault("analysis.before_window_days", DefaultBeforeWindowDays)
	v.SetDefault("analysis.after_window_days", DefaultAfterWindowDays)
	v.SetDefault("analysis.cooldown_hours", DefaultCooldownHours)
	v.SetDefault("feature_grouping.custom_mappings_file", DefaultFeatureMappingsFile)

	if err := v.ReadInConfig(); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Config{}, fmt.Errorf("config file %q not found; create it in the repo root or pass --config /path/to/impact-analyzer.yaml", resolvedPath)
		}
		return Config{}, fmt.Errorf("read config %q: %w", resolvedPath, err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return Config{}, fmt.Errorf("decode config %q: %w", resolvedPath, err)
	}
	return cfg, nil
}
