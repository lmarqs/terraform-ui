# Environment Awareness — Exploration

## Problem Statement

The same terraform plan has vastly different risk profiles depending on environment.
Deleting 1 resource in dev is routine; in production it's an incident. Without
environment context, agents and operators cannot make informed escalation decisions.

## Signal Analysis

### 1. Environment Variable (`TFUI_ENVIRONMENT`)

- **Reliability**: Highest — explicit user/CI intent
- **Confidence**: high
- **Pros**: No ambiguity, no filesystem or terraform access needed
- **Cons**: Requires caller to set it; won't work if unset

### 2. Custom File (`.tfui-env`)

- **Reliability**: High — checked into repo or placed by CI
- **Confidence**: high
- **Pros**: Works without env vars, can be versioned per directory
- **Cons**: Requires adoption; may be stale if directory is reused

### 3. Terraform Workspace Name

- **Reliability**: High — explicit user choice via `terraform workspace select`
- **Confidence**: high
- **Pros**: Already part of terraform workflow; reflects intentional environment targeting
- **Cons**: Requires running `terraform workspace show` (subprocess cost);
  some teams use default workspace for everything
- **Decision**: Use workspace file (`.terraform/environment`) to avoid subprocess

### 4. Variable Values in Plan JSON

- **Reliability**: Medium — depends on project conventions
- **Confidence**: medium
- **Pros**: Uses data already captured during `tfui_plan`; no extra commands
- **Cons**: Variable naming varies (`env`, `environment`, `stage`, `deploy_env`);
  value might not map cleanly to classification

### 5. Directory Path Patterns

- **Reliability**: Low — heuristic, false positives possible
- **Confidence**: low
- **Pros**: Zero cost, always available
- **Cons**: "productive" contains "prod"; nested paths may be misleading;
  no standard for directory naming
- **Decision**: Use word-boundary matching (whole path segment)

## Detection Strategy

### Precedence (highest to lowest)

1. `TFUI_ENVIRONMENT` — explicit override always wins
2. `.tfui-env` file in working directory — repo-level declaration
3. Terraform workspace (`.terraform/environment` file) — avoids subprocess
4. Plan JSON variables (`.variables.environment.value`, `.variables.env.value`)
5. Working directory path segments

### Confidence Levels

| Signal | Confidence |
|--------|-----------|
| TFUI_ENVIRONMENT env var | high |
| .tfui-env file | high |
| Workspace name | high |
| Plan variables | medium |
| Directory path | low |

### Classification

Environments map to one of: `production`, `staging`, `development`, `unknown`.

Pattern matching is case-insensitive and uses word-boundary logic:
- Match against path segments (split on `/`) for directory signals
- Match against whole value for other signals
- Never substring-match inside arbitrary words

### Configuration

`.tfui-config.json` (optional, in working directory or repo root):

```json
{
  "environments": {
    "production": { "patterns": ["prod", "production", "prd"], "risk_multiplier": 3.0 },
    "staging": { "patterns": ["staging", "stg", "preprod", "uat"], "risk_multiplier": 1.5 },
    "development": { "patterns": ["dev", "development", "sandbox", "local"], "risk_multiplier": 0.5 }
  }
}
```

If no config file exists, built-in defaults are used (same as above).

## Risk Adjustment

Base risk levels from plan analysis:
- `low` — only creates
- `medium` — updates
- `high` — any deletes or replaces

Risk multiplier from environment:
- production: 3.0
- staging: 1.5
- development: 0.5
- unknown: 1.0

Adjusted risk is computed by applying the multiplier to a numeric score:
- low=1, medium=2, high=3
- adjusted = base_score * multiplier
- Map back: <=1.5 low, <=3.0 medium, <=4.5 high, >4.5 critical

## Output Format

The environment detection result is a JSON object (via jq):

```json
{
  "environment": {
    "detected": "production",
    "confidence": "high",
    "signals": ["env_var: TFUI_ENVIRONMENT=production"],
    "risk_multiplier": 3.0
  },
  "base_risk_level": "medium",
  "adjusted_risk_level": "critical"
}
```

## Implementation Approach

1. Add `_tfui_detect_environment` to `lib/tfui.sh` — pure bash, returns classification
2. Add `_tfui_env_risk_multiplier` — returns multiplier for detected environment
3. Add `_tfui_compute_base_risk` — analyzes plan JSON for base risk
4. Add `_tfui_compute_adjusted_risk` — combines base risk with environment multiplier
5. Add `tfui_env_report` — public API that outputs the JSON report
6. Tests: unit test each signal source independently using BATS

## Viability Assessment

**Viable.** All signals are available without breaking changes:
- No new external dependencies (jq already required)
- No new terraform commands needed (workspace from file, variables from plan JSON)
- Backward compatible — detection is opt-in, doesn't alter existing plan/apply flow
- Conservative default — returns "unknown" when signals are ambiguous
