package gitimpact

import (
	"encoding/json"
	"strings"
	"time"
)

// Config captures the impact-analyzer.yaml schema.
type Config struct {
	OneQuery        OneQueryConfig        `mapstructure:"onequery"`
	Analysis        AnalysisConfig        `mapstructure:"analysis"`
	FeatureGrouping FeatureGroupingConfig `mapstructure:"feature_grouping"`
}

type OneQueryConfig struct {
	Org     string          `mapstructure:"org"`
	Sources OneQuerySources `mapstructure:"sources"`
}

type OneQuerySources struct {
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

// OneQuery types

type WhoAmIResult struct {
	Email string `json:"email"`
	Org   string `json:"org"`
}

func (r *WhoAmIResult) UnmarshalJSON(payload []byte) error {
	var raw struct {
		Email        string `json:"email"`
		Org          string `json:"org"`
		EffectiveOrg string `json:"effectiveOrg"`
		User         struct {
			Email string `json:"email"`
		} `json:"user"`
	}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return err
	}
	r.Email = strings.TrimSpace(raw.Email)
	if r.Email == "" {
		r.Email = strings.TrimSpace(raw.User.Email)
	}
	r.Org = strings.TrimSpace(raw.Org)
	if r.Org == "" {
		r.Org = strings.TrimSpace(raw.EffectiveOrg)
	}
	return nil
}

type OrgResult struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
	Org  string `json:"org"`
}

type Source struct {
	Key            string   `json:"key"`
	SourceKeyValue string   `json:"sourceKey"`
	Name           string   `json:"name"`
	DisplayName    string   `json:"displayName"`
	ProviderType   string   `json:"provider_type"`
	Provider       string   `json:"provider"`
	Capabilities   []string `json:"capabilities"`
	Query          any      `json:"query"`
	Queryable      bool     `json:"queryable"`
	Status         string   `json:"status"`
}

func (s Source) SourceKey() string {
	if trimmed := strings.TrimSpace(s.Key); trimmed != "" {
		return trimmed
	}
	if trimmed := strings.TrimSpace(s.SourceKeyValue); trimmed != "" {
		return trimmed
	}
	if trimmed := strings.TrimSpace(s.Name); trimmed != "" {
		return trimmed
	}
	return strings.TrimSpace(s.DisplayName)
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

func (r *QueryResult) UnmarshalJSON(payload []byte) error {
	var raw struct {
		Columns       []json.RawMessage `json:"columns"`
		Rows          []json.RawMessage `json:"rows"`
		RowCount      *int              `json:"row_count"`
		RowCountCamel *int              `json:"rowCount"`
	}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return err
	}

	columns := make([]string, 0, len(raw.Columns))
	for _, item := range raw.Columns {
		var column string
		if err := json.Unmarshal(item, &column); err == nil {
			columns = append(columns, column)
			continue
		}

		var objectColumn struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(item, &objectColumn); err != nil {
			return err
		}
		columns = append(columns, objectColumn.Name)
	}

	rows := make([][]interface{}, 0, len(raw.Rows))
	for _, item := range raw.Rows {
		var row []interface{}
		if err := json.Unmarshal(item, &row); err == nil {
			rows = append(rows, row)
			continue
		}

		var objectRow struct {
			Values []interface{} `json:"values"`
		}
		if err := json.Unmarshal(item, &objectRow); err != nil {
			return err
		}
		rows = append(rows, objectRow.Values)
	}

	rowCount := len(rows)
	if raw.RowCount != nil {
		rowCount = *raw.RowCount
	} else if raw.RowCountCamel != nil {
		rowCount = *raw.RowCountCamel
	}

	r.Columns = columns
	r.Rows = rows
	r.RowCount = rowCount
	return nil
}

type OneQueryError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Detail  string `json:"detail"`
	Title   string `json:"title"`
}

func (e *OneQueryError) Error() string {
	if e == nil {
		return ""
	}
	message := e.Message
	if strings.TrimSpace(message) == "" {
		message = e.Detail
	}
	if strings.TrimSpace(message) == "" {
		message = e.Title
	}
	if strings.TrimSpace(e.Code) == "" {
		return message
	}
	if strings.TrimSpace(message) == "" {
		return e.Code
	}
	return e.Code + ": " + message
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
