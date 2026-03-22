package gitimpact

import (
	"strings"
	"time"
)

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
	LastWaitResponse string
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
	// Engine metadata
	Output        string
	Phase         Phase
	Iteration     int
	GeneratedAt   time.Time
	PRs           []PR
	Deployments   []Deployment
	FeatureGroups []FeatureGroup
	Contributors  []ContributorStats
	PRImpacts     []PRImpact
}

// Velen types

type WhoAmIResult struct {
	Email string `json:"email"`
	Org   string `json:"org"`
}

type OrgResult struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
	Org  string `json:"org"`
}

type Source struct {
	Key          string   `json:"key"`
	Name         string   `json:"name"`
	ProviderType string   `json:"provider_type"`
	Provider     string   `json:"provider"`
	Capabilities []string `json:"capabilities"`
	Query        any      `json:"query"`
	Status       string   `json:"status"`
}

func (s Source) SupportsQuery() bool {
	for _, capability := range s.Capabilities {
		if strings.EqualFold(strings.TrimSpace(capability), "QUERY") {
			return true
		}
	}
	switch typed := s.Query.(type) {
	case bool:
		return typed
	case string:
		switch strings.ToLower(strings.TrimSpace(typed)) {
		case "1", "true", "t", "yes", "y", "query", "supported":
			return true
		}
	}
	return false
}

func (s Source) SourceKey() string {
	if trimmed := strings.TrimSpace(s.Key); trimmed != "" {
		return trimmed
	}
	return strings.TrimSpace(s.Name)
}

func (s Source) ProviderLabel() string {
	if trimmed := strings.TrimSpace(s.ProviderType); trimmed != "" {
		return trimmed
	}
	return strings.TrimSpace(s.Provider)
}

type QueryResult struct {
	Columns  []string        `json:"columns"`
	Rows     [][]interface{} `json:"rows"`
	RowCount int             `json:"row_count"`
}

type VelenError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *VelenError) Error() string {
	if e == nil {
		return ""
	}
	if strings.TrimSpace(e.Code) == "" {
		return e.Message
	}
	if strings.TrimSpace(e.Message) == "" {
		return e.Code
	}
	return e.Code + ": " + e.Message
}

// Release represents a GitHub release.
type Release struct {
	Name        string
	TagName     string
	PublishedAt time.Time
}

// AmbiguousDeployment represents a deployment that could not be unambiguously inferred.
type AmbiguousDeployment struct {
	PRNumber int
	Options  []string
	Reason   string
}

// EngineRunMeta holds engine-level metadata about a run (used by tests and engine internals).
// These fields are populated by the engine and not part of the analysis content.
// We embed these directly in AnalysisResult for convenience.
