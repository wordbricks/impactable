package gitimpact

import (
	"context"
	"strings"
)

const DefaultConfigPath = "impact-analyzer.yaml"

type Config struct {
	Velen           VelenConfig           `json:"velen" mapstructure:"velen"`
	Analysis        AnalysisConfig        `json:"analysis" mapstructure:"analysis"`
	FeatureGrouping FeatureGroupingConfig `json:"feature_grouping" mapstructure:"feature_grouping"`
}

type VelenConfig struct {
	Org     string             `json:"org" mapstructure:"org"`
	Sources VelenSourcesConfig `json:"sources" mapstructure:"sources"`
}

type VelenSourcesConfig struct {
	GitHub    string `json:"github" mapstructure:"github"`
	Analytics string `json:"analytics" mapstructure:"analytics"`
}

type AnalysisConfig struct {
	BeforeWindowDays int `json:"before_window_days" mapstructure:"before_window_days"`
	AfterWindowDays  int `json:"after_window_days" mapstructure:"after_window_days"`
	CooldownHours    int `json:"cooldown_hours" mapstructure:"cooldown_hours"`
}

type FeatureGroupingConfig struct {
	Strategies         []string `json:"strategies" mapstructure:"strategies"`
	CustomMappingsFile string   `json:"custom_mappings_file" mapstructure:"custom_mappings_file"`
}

type AnalysisContext struct {
	Since       string  `json:"since,omitempty"`
	PRNumber    *int    `json:"pr_number,omitempty"`
	FeatureName string  `json:"feature_name,omitempty"`
	ConfigPath  string  `json:"config_path"`
	Config      *Config `json:"config"`
}

type Source struct {
	Key          string   `json:"key" mapstructure:"key"`
	Name         string   `json:"name,omitempty" mapstructure:"name"`
	ProviderType string   `json:"provider_type" mapstructure:"provider_type"`
	Capabilities []string `json:"capabilities,omitempty" mapstructure:"capabilities"`
	Operations   []string `json:"operations,omitempty" mapstructure:"operations"`
	Actions      []string `json:"actions,omitempty" mapstructure:"actions"`
}

func (s Source) SupportsQuery() bool {
	for _, capability := range s.Capabilities {
		if strings.EqualFold(strings.TrimSpace(capability), "QUERY") {
			return true
		}
	}
	for _, operation := range s.Operations {
		if strings.EqualFold(strings.TrimSpace(operation), "QUERY") {
			return true
		}
	}
	for _, action := range s.Actions {
		if strings.EqualFold(strings.TrimSpace(action), "QUERY") {
			return true
		}
	}
	return false
}

type VelenClient interface {
	AuthWhoAmI(ctx context.Context) error
	OrgCurrent(ctx context.Context) (string, error)
	SourceList(ctx context.Context) ([]Source, error)
}

type SourceCheckResult struct {
	GitHubSource    *Source `json:"github_source,omitempty"`
	AnalyticsSource *Source `json:"analytics_source,omitempty"`
	GitHubOK        bool    `json:"github_ok"`
	AnalyticsOK     bool    `json:"analytics_ok"`
	OrgName         string  `json:"org_name,omitempty"`
	Error           string  `json:"error,omitempty"`
}
