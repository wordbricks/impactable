package gitimpact

import "time"

// Config captures the impact-analyzer.yaml schema.
type Config struct {
	Velen           VelenConfig           `mapstructure:"velen"`
	Analysis        AnalysisConfig        `mapstructure:"analysis"`
	FeatureGrouping FeatureGroupingConfig `mapstructure:"feature_grouping"`
}

type VelenConfig struct {
	Org     string       `mapstructure:"org"`
	Sources VelenSources `mapstructure:"sources"`
}

type VelenSources struct {
	GitHub    string `mapstructure:"github"`
	Analytics string `mapstructure:"analytics"`
}

type AnalysisConfig struct {
	BeforeWindowDays int `mapstructure:"before_window_days"`
	AfterWindowDays  int `mapstructure:"after_window_days"`
	CooldownHours    int `mapstructure:"cooldown_hours"`
}

type FeatureGroupingConfig struct {
	Strategies         []string `mapstructure:"strategies"`
	CustomMappingsFile string   `mapstructure:"custom_mappings_file"`
}

// AnalysisContext is the structured context passed into analysis runtime.
type AnalysisContext struct {
	WorkingDirectory string
	ConfigPath       string
	Since            *time.Time
	PRNumber         int
	Feature          string
}

type PR struct {
	Number      int
	Title       string
	Author      string
	MergedAt    time.Time
	Branch      string
	Labels      []string
	ChangedFile []string
}

type Deployment struct {
	PRNumber   int
	Marker     string
	Source     string
	DeployedAt time.Time
}

type FeatureGroup struct {
	Name      string
	PRNumbers []int
}

type ContributorStats struct {
	Author       string
	PRCount      int
	AverageScore float64
	TopPRNumber  int
}

type PRImpact struct {
	PRNumber   int
	Score      float64
	Confidence string
	Reasoning  string
}

type AnalysisResult struct {
	GeneratedAt   time.Time
	PRs           []PR
	Deployments   []Deployment
	FeatureGroups []FeatureGroup
	Contributors  []ContributorStats
	PRImpacts     []PRImpact
}
