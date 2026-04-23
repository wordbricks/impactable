# Git Impact Analyzer - SPEC.md

## 1. Project Overview

A tool that analyzes Git history (PRs, feature branches, commits) and quantitatively measures the real **impact** each change had on the product.

It uses OneQuery CLI to access already connected data sources such as GitHub and analytics platforms, then traces the causal path from code changes to deployment to changes in user behavior.

The analysis pipeline is driven by a single WTL (WhatTheLoop) Agent that progresses through ordered phases. Each phase is one WTL Turn. The Agent handles all OneQuery CLI interactions directly — source discovery, query writing, and execution.

**Scope**: Monorepo only.
**Agent runtime**: Codex App Server.

---

## 2. Core Goals

| Goal | Description |
|------|-------------|
| **Impact scoring by PR** | Score how much each PR affected user metrics such as MAU, conversion rate, and error rate |
| **Feature-level analysis** | Group multiple PRs into a single feature and measure impact at the feature level |
| **Automatic correlation analysis** | Detect time-series correlations between when code changes occurred and when metrics changed |
| **Contributor insights** | Report which individual contributors produced the highest impact |

---

## 3. Architecture

```
┌─────────────────────────────────────────────────────┐
│                  Git Impact Analyzer                │
├─────────────────────────────────────────────────────┤
│                                                     │
│              WTL Agent (single run)                 │
│                                                     │
│  ┌──────────────┐  ┌──────────┐  ┌──────────────┐  │
│  │ Source Check │  │ Collect  │  │    Linker    │  │
│  │  (pre-Turn)  │─▶│ (Turn 1) │─▶│   (Turn 2)   │  │
│  └──────────────┘  └──────────┘  └──────┬───────┘  │
│                                          │          │
│                    ┌─────────────────────┘          │
│                    ▼                                │
│             ┌──────────────┐   ┌──────────────┐    │
│             │ Impact Scorer│─▶ │    Report    │    │
│             │   (Turn 3)   │   │   (Turn 4)   │    │
│             └──────────────┘   └──────────────┘    │
│                                                     │
│  All OneQuery CLI calls made directly by the Agent  │
│  (GitHub + Analytics sources)                       │
│                                                     │
├─────────────────────────────────────────────────────┤
│         Observer → Bubble Tea Msg bridge            │
└─────────────────────────────────────────────────────┘
```

### 3.1 WTL Loop Structure

The entire analysis is a single WTL Run. Source check runs before Turn 1. Phases progress in order.

| Step | Phase | Goal | Directive on success |
|------|-------|------|----------------------|
| pre | Source Check | Verify GitHub and Analytics sources are available | proceed to Turn 1 |
| 1 | Collect | Fetch PR list, commits, tags, releases from GitHub | `advance_phase` |
| 2 | Link | Infer deployment times from GitHub tags/releases and map to PRs | `advance_phase` |
| 3 | Score | Explore analytics schema, query metrics, calculate impact scores | `advance_phase` |
| 4 | Report | Render interactive TUI with results | `complete` |

**Policy (PhasedDeliveryPolicy):**

| Condition | Directive |
|-----------|-----------|
| Phase goal achieved | `advance_phase` or `complete` |
| Turn failed (recoverable) | `retry` |
| Ambiguous situation (source not found, no data, uncertain deployment mapping) | `wait` → ask user via terminal prompt |
| Otherwise | `continue` |

**`wait` handling:**
When the Agent issues a `wait` directive, the TUI pauses and the terminal prompt takes over. The user types their response and presses Enter. The input is delivered back to the Agent and the run resumes.

**Observer → TUI bridge:**

Observer lifecycle events are converted to Bubble Tea `Msg` values and drive real-time TUI updates.

| Event | Bubble Tea Msg |
|-------|----------------|
| `TurnStarted` | Update phase progress indicator |
| `PhaseAdvanced` | Advance phase in progress bar |
| `WaitEntered` | Pause TUI, show terminal prompt |
| `WaitResolved` | Resume TUI |
| `RunCompleted` | Switch to interactive results view |
| `RunExhausted` | Display error state |

### 3.2 Components

**Source Check (pre-Turn 1)**
- Run automatically at the start of every `analyze` invocation
- Agent runs `onequery --org <org> auth whoami`, `onequery --org <org> org current`, `onequery --org <org> source list`
- Verifies GitHub and Analytics sources exist
- If a required source is missing, issue `wait` and ask the user to confirm before proceeding
- Also available as standalone `git-impact check-sources` command

**Collector (Turn 1)** - Collect Git data
- Agent queries GitHub source via OneQuery for PR list, commit logs, branches, tags, and releases
- Agent writes queries directly based on the analysis context (date range, PR number, feature name) passed from CLI args
- Monorepo: all data comes from a single repo

**Linker (Turn 2)** - Infer deployments and link to PRs
- Agent infers deployment times from GitHub data in this priority order:
  1. **GitHub Releases** — release publish time is treated as deployment time
  2. **Tags** — tags matching version patterns (e.g. `v*`, `release-*`) are treated as deployment markers
  3. **PR merge time** — used as fallback if no release or tag is found near the merge
- If the mapping is ambiguous, issue `wait` and ask the user to confirm
- Agent also proposes feature groupings based on label, branch, and author patterns
  - If groupings are uncertain, issue `wait` and ask the user to confirm
  - Confirmed groupings are saved to `feature-map.yaml`

**Impact Scorer (Turn 3)** - Calculate scores
- Agent explores the Analytics source schema via OneQuery to discover available metrics
- Agent writes and executes queries directly, using inferred deployment times as before/after boundaries
- Agent adjusts time windows from config defaults based on context (data density, deployment clustering, etc.)
- Agent judges confidence based on: data point count, simultaneous deployments, metric volatility, and other signals
- Agent explains its confidence reasoning in natural language alongside each score

**Report Generator (Turn 4)** - Output results
- Switch TUI to interactive results mode: sortable PR table with drill-down into feature and contributor views
- User can save results from within the TUI (e.g. `s` → markdown, `e` → HTML export)

---

## 4. Data Sources and OneQuery CLI Usage

All external data access must go through OneQuery CLI. The Agent handles all OneQuery CLI interactions directly — it writes queries, executes them, and interprets results.

### 4.1 Access Flow

```bash
# 1. Authenticate and verify org
onequery --org <org> auth whoami
onequery --org <org> org current

# 2. Discover available sources
onequery --org <org> source list
onequery --org <org> source show <source_key>

# 3. Query — written and executed directly by the Agent
onequery --org <org> query exec --source <source_key> --sql "<agent-generated SQL>"
```

### 4.2 Required Source Types

| Source Type | Purpose | Expected Data |
|-------------|---------|---------------|
| **GitHub** | PR / commit / branch / tag / release metadata | PR title, author, merge time, changed files, labels, tags, releases |
| **Analytics** | User behavior metrics | schema explored by Agent at runtime |

> **No warehouse source is required.** Deployment times are inferred from GitHub tags and releases by the Agent.

### 4.3 Source Discovery Strategy

At the start of every `analyze` run, the Agent:
1. Runs `onequery --org <org> source list` to find available sources
2. Identifies GitHub and Analytics sources by provider type
3. Confirms each required provider has a matching source
4. If a required source type is missing, issues `wait` and asks the user

---

## 5. Impact Measurement Model

### 5.1 PR-Level Impact Score

The Agent produces an impact score for each PR based on its holistic judgment of available analytics data.

```
impact_score = Agent judgment over available metrics,
               weighted by perceived business importance,
               adjusted by assessed confidence
```

The Agent:
- Discovers available metrics by exploring the Analytics source schema
- Determines which metrics are meaningful for each PR based on the nature of the change
- Assesses confidence based on data quality, simultaneity of deployments, and metric volatility
- Explains its reasoning in natural language in the report

### 5.2 Deployment Inference from GitHub

The Agent infers deployment times from the monorepo's GitHub data:

1. **GitHub Releases** — release publish time is treated as deployment time
2. **Tags** — tags matching version patterns (e.g. `v*`, `release-*`) are treated as deployment markers
3. **PR merge time** — used as fallback if no release or tag is found near the merge

If the inference is ambiguous, the Agent issues `wait` and asks the user to confirm.

### 5.3 Time Windows

Default values used as starting points. The Agent may adjust them based on context.

| Parameter | Default | Description |
|-----------|---------|-------------|
| `before_window` | 7 days | Comparison period before deployment |
| `after_window` | 7 days | Observation period after deployment |
| `cooldown` | 24 hours | Stabilization wait time immediately after deployment |

### 5.4 Handling Confounding Deployments

If multiple PRs are deployed during the same time window, the Agent:
- Attempts to separate impact areas based on the scope of changed files
- Factors the overlap into its confidence assessment
- Explains the confounding situation in natural language in the report

---

## 6. Feature Grouping

The Agent proposes feature groupings automatically based on:

1. **Label-based**: GitHub PR labels such as `feature/onboarding-v2`
2. **Branch prefix**: PR groups derived from `feature/*` branches
3. **Time + author clustering**: related PRs submitted by the same author within a short time window
4. **Manual mapping**: explicit PR → Feature mapping in `feature-map.yaml`

When groupings are uncertain, the Agent issues `wait`, presents the proposed groupings to the user, and saves confirmed groupings to `feature-map.yaml`. This file is managed by the Agent, not manually authored.

---

## 7. CLI Interface

```bash
# Run full analysis (source check runs automatically before Turn 1)
git-impact analyze --config impact-analyzer.yaml --since 2026-01-01

# Analyze a specific PR
git-impact analyze --pr 142

# Analyze at the feature level
git-impact analyze --feature "onboarding-v2"

# Check source connectivity (also runs automatically at analyze start)
git-impact check-sources
```

CLI arguments are parsed into a structured context object and passed to the Agent as part of its initial prompt.

### 7.1 TUI — Analysis Progress View

During analysis, the TUI shows real-time phase progress driven by WTL Observer events:

```
Git Impact Analyzer
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
[■■■■■■■■░░░░░░░░]  Turn 2/4 — Linking deployments

✓ Sources   GitHub + Analytics connected
✓ Collect   47 PRs, 3 releases, 12 tags fetched
→ Link      Inferring deployment times from GitHub releases...
  Score     Waiting
  Report    Waiting
```

When `wait` is triggered, the TUI pauses and a terminal prompt appears:

```
[?] Could not determine deployment for PR #142. Was this PR deployed via
    release v2.4.1 (2026-02-15)? (y/n/skip):
```

### 7.2 TUI — Interactive Results View (after completion)

After analysis completes, the TUI switches to an interactive mode:

```
PR Impact Results  (↑↓ navigate, Enter to expand, Tab switch view, s save, q quit)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  #   PR Title                    Author   Score   Confidence
▶ 142 Payment Page Redesign       @kim     8.2     high
  140 API Caching Improvement     @lee     6.5     medium
  138 Onboarding v2 - Step 1      @park    9.1     high
```

- `Enter` → PR detail view with per-metric breakdown and Agent reasoning
- `Tab` → switch between PR view, Feature view, Contributor leaderboard
- `s` → save report (markdown or HTML)
- `q` → quit

### 7.3 PR Detail View

```
PR #142 - "Payment Page Redesign"
  Author: @kim
  Merged: 2026-02-15, Deployed: 2026-02-15 14:30 (via release v2.4.1)
  Impact Score: 8.2 / 10
  ─────────────────────────────────────────────
  conversion_rate:  +3.1% (confidence: high)
  error_rate:       -0.4% (confidence: medium)
  avg_session_time: +12s  (confidence: low)

  Agent reasoning:
  "High confidence on conversion_rate due to clean 7-day window with no
  other deployments. Error rate confidence reduced due to low baseline
  volume. Session time signal is noisy."
```

### 7.4 Feature Impact View

```
Feature: "Onboarding v2" (PR #138, #140, #142, #145)
  Period: 2026-02-10 ~ 2026-02-20
  Combined Impact Score: 9.1 / 10
  Top Metric: new_user_retention +18%
```

### 7.5 Contributor Leaderboard

```
Rank  Author   PRs   Avg Impact  Top PR
1     @park    12    8.4         Onboarding v2 - Step 1
2     @kim     8     7.9         Payment Page Redesign
3     @lee     15    6.2         API Caching Improvement
```

---

## 8. Configuration

```yaml
# impact-analyzer.yaml

onequery:
  org: "my-company"                    # OneQuery org slug
  sources:
    github: "github-main"              # GitHub source key
    analytics: "amplitude-prod"        # analytics source key

analysis:
  before_window_days: 7                # default; Agent may adjust based on context
  after_window_days: 7                 # default; Agent may adjust based on context
  cooldown_hours: 24                   # default; Agent may adjust based on context

feature_grouping:
  strategies:
    - label_prefix                     # based on PR labels
    - branch_prefix                    # based on branch names
  custom_mappings_file: feature-map.yaml  # generated and managed by the Agent
```

---

## 9. Tech Stack

| Area | Choice | Reason |
|------|--------|--------|
| Language | Go | Single-binary deployment, fast execution speed, good fit for CLI tools |
| Agent loop | WTL (WhatTheLoop) | Shared loop interface for phased agent execution |
| Agent runtime | Codex App Server | LLM runtime powering the Agent |
| TUI framework | [Bubble Tea](https://github.com/charmbracelet/bubbletea) | Elm-architecture-based TUI framework, ideal for interactive UIs |
| UI components | Bubbles (spinner, table, progress, etc.) | Official component library for Bubble Tea |
| Styling | Lip Gloss | Terminal styling library from the Charm ecosystem |
| CLI entrypoint | Cobra + Viper | Integrates subcommands, flags, and config files |
| OneQuery integration | Invoke `onequery` CLI via `os/exec` | Agent runs OneQuery as a subprocess and parses results |
| Data processing | Agent-generated SQL via OneQuery | Agent writes and executes all queries at runtime |
| Report output | Terminal: interactive Bubble Tea TUI / Files: markdown + HTML | Interactive terminal view; file export triggered from TUI |
| Configuration | YAML (Viper) | Readable, supports comments, and Viper handles YAML natively |
| Testing | Go testing + testify | Standard-library-based with assertion helpers |

### 9.1 Planned Use of the Charm Ecosystem

```
charmbracelet/bubbletea  - main TUI framework (Model, Update, View loop)
charmbracelet/bubbles    - prebuilt components (table, spinner, progress, list, viewport)
charmbracelet/lipgloss   - terminal layout and styling (color, border, alignment)
charmbracelet/log        - structured terminal logging
```

**Primary TUI screens:**

| Screen | Components Used | Description |
|--------|-----------------|-------------|
| Source connection check | spinner + list | Auto-runs before Turn 1; shows source discovery results |
| Analysis progress | progress bar + phase list | Real-time phase progress driven by WTL Observer events |
| Wait prompt | terminal stdin | TUI pauses; user answers Agent question via terminal prompt |
| PR impact table | table (sortable / filterable) | Interactive table of impact scores per PR |
| PR detail view | viewport + table | Per-metric breakdown with Agent confidence reasoning |
| Feature detail view | viewport + table | Detailed metrics and related PRs for the selected feature |
| Contributor leaderboard | table + bar chart (lipgloss) | Individual contributor rankings by avg impact score |

### 9.2 WTL Integration

The WTL engine runs within the git-impact process. The Observer bridges WTL lifecycle events to Bubble Tea messages.

```
WTL Engine
  └── emits lifecycle events
        └── WTL Observer
              └── sends Bubble Tea Msg
                    └── Bubble Tea Update()
                          └── re-renders View()
```

---

## 10. Implementation Phases

### Phase 1 - Foundation (MVP)
- [ ] OneQuery CLI wrapper (auth, source discovery, query execution via `os/exec`)
- [x] Codex app-server phase-agent runtime (single thread, one WTL turn per phase)
- [x] WTL engine and PhasedDeliveryPolicy implementation
- [ ] WTL Observer → Bubble Tea Msg bridge
- [x] CLI arg → structured context → Agent initial prompt
- [ ] Source check (pre-Turn): verify GitHub + Analytics sources
- [ ] Collect PRs, tags, releases from GitHub source (Turn 1)
- [ ] Infer deployment times from GitHub releases and tags (Turn 2)
- [ ] Analytics schema exploration + single-metric query (Turn 3)
- [ ] Agent-judged impact score for a single PR
- [ ] Basic interactive TUI: phase progress + PR table (Turn 4)

### Phase 2 - Expansion
- [ ] Multi-metric analysis with Agent-determined weighting
- [ ] Feature grouping proposal → `wait` → user confirmation → `feature-map.yaml`
- [ ] Agent confidence reasoning displayed in PR detail view
- [ ] `wait` directive flow for ambiguous deployment and source situations
- [ ] TUI in-app export (markdown, HTML)

### Phase 3 - Advanced
- [ ] Contributor leaderboard
- [ ] Time-series trend analysis (impact trend by feature)
- [ ] CI/CD integration (automatic impact preview when a PR is merged)

---

## 11. Constraints and Considerations

- **Monorepo only**: Analysis targets a single repository. Multi-repo support is out of scope.
- **Read-only**: OneQuery CLI only supports read-only access. All queries must use SELECT only.
- **Agent-generated queries**: The Agent writes all SQL at runtime. No pre-written query files are used.
- **Query cost**: Agent must always specify a date range and LIMIT to avoid expensive large-scale queries.
- **Org context**: Before starting analysis, verify the correct org with `onequery --org <org> org current`.
- **Source availability**: Confirmed at the start of every `analyze` run. Missing sources trigger `wait`.
- **No warehouse dependency**: Deployment times are inferred from GitHub data only.
- **Agent judgment**: Metric selection, confidence scores, deployment inference, and time window adjustments are all produced by the Agent. Reports must make this clear.
- **Correlation not causation**: Before/after comparisons show correlation, not causation. Reports should state this clearly.
- **Privacy**: Agent must not query or surface personally identifiable information from user behavior data.
