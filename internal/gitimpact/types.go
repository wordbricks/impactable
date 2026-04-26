package gitimpact

import (
	"encoding/json"
	"fmt"
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

// Tag represents a Git tag collected from GitHub.
type Tag struct {
	Name      string
	Sha       string
	CreatedAt time.Time
}

func (t Tag) MarshalJSON() ([]byte, error) {
	payload := struct {
		Name      string `json:"Name"`
		Sha       string `json:"Sha,omitempty"`
		CreatedAt string `json:"CreatedAt,omitempty"`
	}{
		Name: strings.TrimSpace(t.Name),
		Sha:  strings.TrimSpace(t.Sha),
	}
	if !t.CreatedAt.IsZero() {
		payload.CreatedAt = t.CreatedAt.UTC().Format(time.RFC3339)
	}
	return json.Marshal(payload)
}

func (t *Tag) UnmarshalJSON(payload []byte) error {
	var legacy string
	if err := json.Unmarshal(payload, &legacy); err == nil {
		parsed, ok := parseLegacyTag(legacy)
		if !ok {
			*t = Tag{Name: strings.TrimSpace(legacy)}
			return nil
		}
		*t = parsed
		return nil
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(payload, &raw); err != nil {
		return err
	}

	name, err := rawStringField(raw, "Name", "name")
	if err != nil {
		return fmt.Errorf("tag Name: %w", err)
	}
	sha, err := rawStringField(raw, "Sha", "SHA", "sha")
	if err != nil {
		return fmt.Errorf("tag Sha: %w", err)
	}
	createdAt, err := rawTimeField(raw, "CreatedAt", "createdAt", "created_at")
	if err != nil {
		return fmt.Errorf("tag CreatedAt: %w", err)
	}

	*t = Tag{
		Name:      strings.TrimSpace(name),
		Sha:       strings.TrimSpace(sha),
		CreatedAt: createdAt,
	}
	return nil
}

func parseLegacyTag(value string) (Tag, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return Tag{}, false
	}

	parts := strings.SplitN(trimmed, tagTimestampSeparator, 2)
	if len(parts) != 2 {
		return Tag{Name: trimmed}, false
	}

	name := strings.TrimSpace(parts[0])
	if name == "" {
		return Tag{}, false
	}

	createdAt, err := asTime(strings.TrimSpace(parts[1]))
	if err != nil {
		return Tag{Name: name}, false
	}
	return Tag{Name: name, CreatedAt: createdAt.UTC()}, true
}

func rawStringField(raw map[string]json.RawMessage, names ...string) (string, error) {
	value, ok := rawField(raw, names...)
	if !ok {
		return "", nil
	}
	if isJSONNull(value) {
		return "", nil
	}
	var decoded string
	if err := json.Unmarshal(value, &decoded); err == nil {
		return decoded, nil
	}
	var number json.Number
	if err := json.Unmarshal(value, &number); err == nil {
		return number.String(), nil
	}
	return "", fmt.Errorf("expected string")
}

func rawTimeField(raw map[string]json.RawMessage, names ...string) (time.Time, error) {
	value, ok := rawField(raw, names...)
	if !ok {
		return time.Time{}, nil
	}
	if isJSONNull(value) {
		return time.Time{}, nil
	}
	var decoded string
	if err := json.Unmarshal(value, &decoded); err != nil {
		return time.Time{}, fmt.Errorf("expected string")
	}
	if strings.TrimSpace(decoded) == "" {
		return time.Time{}, nil
	}
	return asTime(decoded)
}

func rawField(raw map[string]json.RawMessage, names ...string) (json.RawMessage, bool) {
	for _, name := range names {
		if value, ok := raw[name]; ok {
			return value, true
		}
	}
	return nil, false
}

func isJSONNull(value json.RawMessage) bool {
	return strings.EqualFold(strings.TrimSpace(string(value)), "null")
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
	PRNumber          int
	Score             float64
	Confidence        string
	PrimaryMetric     string
	BeforeValue       float64
	AfterValue        float64
	DeltaValue        float64
	BeforeWindowStart time.Time
	BeforeWindowEnd   time.Time
	AfterWindowStart  time.Time
	AfterWindowEnd    time.Time
	Reasoning         string
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
