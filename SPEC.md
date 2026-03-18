# Git Impact Analyzer - SPEC.md

## 1. Project Overview

A tool that analyzes Git history (PRs, feature branches, commits) and quantitatively measures the real **impact** each change had on the product.

It uses Velen CLI to access already connected data sources such as GitHub, databases, and analytics platforms, then traces the causal path from code changes to deployment to changes in user behavior.

---

## 2. Core Goals

| Goal | Description |
|------|-------------|
| **Impact scoring by PR** | Score how much each PR affected user metrics such as MAU, conversion rate, and error rate |
| **Feature-level analysis** | Group multiple PRs into a single feature and measure impact at the feature level |
| **Automatic correlation analysis** | Detect time-series correlations between when code changes occurred and when metrics changed |
| **Team insights** | Report which teams or developers produced the highest impact |

---

## 3. Architecture

```
┌─────────────────────────────────────────────────────┐
│                  Git Impact Analyzer                │
├─────────────────────────────────────────────────────┤
│                                                     │
│  ┌───────────┐  ┌──────────┐   ┌──────────────┐     │
│  │ Collector │─▶│ Linker   │─▶│ Impact Scorer│     │
│  └────┬──────┘  └────┬─────┘   └──────┬───────┘     │
│       │              │                │             │
│  Velen CLI       Velen CLI        Velen CLI         │
│  (GitHub)        (Database)       (Analytics)       │
│                                                     │
├─────────────────────────────────────────────────────┤
│                 Report Generator                    │
└─────────────────────────────────────────────────────┘
```

### 3.1 Components

**Collector** - Collect Git data
- Collect PR lists, commit logs, and branch information from the GitHub source connected to Velen
- Discover GitHub sources with `velen source list`, then fetch PR metadata with `velen query`

**Linker** - Link code changes ↔ deployments ↔ metrics
- Map PR merge time to deployment time, using sources such as a deployment table in the database
- Link deployment times to metric changes within the relevant time window

**Impact Scorer** - Calculate scores
- Compare metrics before and after deployment using before/after window analysis
- Compute an impact score by combining delta, statistical significance, and scope of influence

**Report Generator** - Output results
- Generate impact reports by PR, feature, and team
- Visualize results with time-series charts, ranking tables, and more

---

## 4. Data Sources and Velen CLI Usage

All external data access must go through Velen CLI. Direct credentials are not used.

### 4.1 Access Flow

```bash
# 1. Authenticate and verify org
velen auth whoami
velen org current

# 2. Discover available sources
velen source list
velen source show <source_key>

# 3. Query each source
velen query --source <source_key> --file ./queries/<query_name>.sql
```

### 4.2 Required Source Types

| Source Type | Purpose | Expected Data |
|-------------|---------|---------------|
| **GitHub** | PR / commit / branch metadata | PR title, author, merge time, changed files, labels |
| **Database (Warehouse)** | Deployment history, user data | deployments, user_events, feature_flags |
| **Analytics** | User behavior metrics | page_views, conversions, error_rates, session_duration |

### 4.3 Source Discovery Strategy

At project initialization, run `velen source list` to automatically discover available sources. Check whether each source supports `QUERY`, then map roles automatically based on provider type such as github, postgres, bigquery, or amplitude.

---

## 5. Impact Measurement Model

### 5.1 PR-Level Impact Score

```
impact_score = Σ(metric_delta × metric_weight × confidence)
```

- **metric_delta**: rate of change before and after deployment (for example, conversion rate +2.3%)
- **metric_weight**: business importance of the metric, configurable
- **confidence**: statistical confidence level, accounting for noise, simultaneous deployments, and more

### 5.2 Time Windows

| Parameter | Default | Description |
|-----------|---------|-------------|
| `before_window` | 7 days | Comparison period before deployment |
| `after_window` | 7 days | Observation period after deployment |
| `cooldown` | 24 hours | Stabilization wait time immediately after deployment |

### 5.3 Handling Confounding Deployments

If multiple PRs are deployed during the same time window:
- Attempt to separate impact areas based on the scope of changed files
- If separation is not possible, assign shared impact to the PR group and reduce confidence

---

## 6. Feature Grouping

Ways to group multiple PRs into a single feature:

1. **Label-based**: GitHub PR labels such as `feature/onboarding-v2`
2. **Branch prefix**: PR groups derived from `feature/*` branches
3. **Time + author clustering**: related PRs submitted by the same author within a short time window
4. **Manual mapping**: explicit PR → Feature mapping through a config file

---

## 7. Output Formats

### 7.1 PR Impact Report

```
PR #142 - "Payment Page Redesign"
  Author: @kim
  Merged: 2026-02-15, Deployed: 2026-02-15 14:30 KST
  Impact Score: 8.2 / 10
  ----------------------
  conversion_rate:  +3.1% (confidence: high)
  error_rate:       -0.4% (confidence: medium)
  avg_session_time: +12s  (confidence: low)
```

### 7.2 Feature Impact Report

```
Feature: "Onboarding v2" (PR #138, #140, #142, #145)
  Period: 2026-02-10 ~ 2026-02-20
  Combined Impact Score: 9.1 / 10
  Top Metric: new_user_retention +18%
```

### 7.3 Team Leaderboard

```
Rank  Team        Avg Impact  Top Feature
1     Growth      7.8         Onboarding v2
2     Platform    6.2         API Caching Improvement
3     Payments    5.9         Payment Page Redesign
```

---

## 8. Configuration

```yaml
# impact-analyzer.yaml

velen:
  org: "my-company"                    # Velen org slug
  sources:
    github: "github-main"              # GitHub source key
    warehouse: "prod-warehouse"        # data warehouse source key
    analytics: "amplitude-prod"        # analytics source key

metrics:
  - name: conversion_rate
    query_file: queries/conversion_rate.sql
    weight: 3.0
    direction: up                      # up = increase is good, down = decrease is good

  - name: error_rate
    query_file: queries/error_rate.sql
    weight: 2.0
    direction: down

  - name: p95_latency
    query_file: queries/latency.sql
    weight: 1.5
    direction: down

analysis:
  before_window_days: 7
  after_window_days: 7
  cooldown_hours: 24
  min_confidence: 0.6                  # below this, show as "low confidence" in reports

feature_grouping:
  strategies:
    - label_prefix                     # based on PR labels
    - branch_prefix                    # based on branch names
  custom_mappings_file: feature-map.yaml
```

---

## 9. CLI Interface

```bash
# Run full analysis
git-impact analyze --config impact-analyzer.yaml --since 2026-01-01

# Analyze a specific PR
git-impact analyze --pr 142

# Analyze at the feature level
git-impact analyze --feature "onboarding-v2"

# Generate reports
git-impact report --format markdown --output reports/
git-impact report --format html --output reports/

# Check source connectivity
git-impact check-sources
```

---

## 10. Tech Stack

| Area | Choice | Reason |
|------|--------|--------|
| Language | Go | Single-binary deployment, fast execution speed, good fit for CLI tools |
| TUI framework | [Bubble Tea](https://github.com/charmbracelet/bubbletea) | Elm-architecture-based TUI framework, ideal for interactive UIs |
| UI components | Bubbles (spinner, table, progress, etc.) | Official component library for Bubble Tea |
| Styling | Lip Gloss | Terminal styling library from the Charm ecosystem |
| CLI entrypoint | Cobra + Viper | Integrates subcommands, flags, and config files, and serves as the Bubble Tea entrypoint |
| Velen integration | Invoke `velen` CLI via `os/exec` | Run Velen CLI as a subprocess and parse query results |
| Data processing | SQL (Velen query) + local aggregation | Push heavy processing to the remote DB, aggregate locally in Go |
| Report output | Terminal: Bubble Tea TUI / Files: HTML (go-echarts) | Interactive terminal view plus static report files |
| Configuration | YAML (Viper) | Readable, supports comments, and Viper handles YAML natively |
| Testing | Go testing + testify | Standard-library-based with assertion helpers |

### 10.1 Planned Use of the Charm Ecosystem

Use Bubble Tea's Elm architecture (Model -> Update -> View) to build the interactive TUI.

```
charmbracelet/bubbletea  - main TUI framework (Model, Update, View loop)
charmbracelet/bubbles    - prebuilt components (table, spinner, progress, list, viewport)
charmbracelet/lipgloss   - terminal layout and styling (color, border, alignment)
charmbracelet/log        - structured terminal logging
```

**Primary TUI screens:**

| Screen | Components Used | Description |
|--------|-----------------|-------------|
| Source connection check | spinner + list | Show loading while discovering Velen sources, then display the result list |
| Analysis progress | progress bar | Show step-by-step progress for PR collection, metric lookup, and score calculation |
| PR impact table | table (sortable / filterable) | Display impact scores per PR in an interactive table |
| Feature detail view | viewport + table | Show detailed metrics and related PRs for the selected feature |
| Team leaderboard | table + bar chart (lipgloss) | Display team rankings visually in the terminal |

---

## 11. Implementation Phases

### Phase 1 - Foundation (MVP)
- [ ] Velen CLI integration wrapper (auth, source discovery, query execution)
- [ ] Collect PR lists from the GitHub source
- [ ] Before/after comparison for a single metric
- [ ] Single-PR impact score calculation
- [ ] Output results through the CLI

### Phase 2 - Expansion
- [ ] Support multiple metrics
- [ ] Feature grouping based on labels and branches
- [ ] Detect confounding deployments and adjust confidence
- [ ] Generate Markdown and HTML reports

### Phase 3 - Advanced
- [ ] Team leaderboard
- [ ] Time-series trend analysis (impact trend by feature)
- [ ] Custom metric plugin system
- [ ] CI/CD integration (automatic impact preview when a PR is merged)

---

## 12. Constraints and Considerations

- **Read-only**: Velen CLI only supports read-only access. All queries must use SELECT only.
- **Query cost**: Always specify a date range and LIMIT to avoid expensive large-scale queries.
- **Org context**: Before starting analysis, verify the correct org with `velen org current`.
- **Source availability**: Confirm in advance with `velen source show` that every source has `QUERY=yes`.
- **Statistical limitations**: Before/after comparisons show correlation, not causation. Reports should state this clearly.
- **Privacy**: Design queries so they do not collect personally identifiable information when analyzing user behavior data.
