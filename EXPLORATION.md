# Agent Mode: Exploration and Design

## Analysis

### Existing Architecture

terraform-ui uses a clean strategy pattern where the UI mode maps to a strategy function:

| Mode   | Strategy Function           | Behavior                      |
|--------|-----------------------------|-------------------------------|
| rich   | `_tfui_strategy_progress`   | 2-line UI (spinner + bar)     |
| simple | `_tfui_strategy_spinner`    | 1-line animated spinner       |
| plain  | `_tfui_strategy_silent`     | No UI, captures output        |

All strategies share the same interface: `(patterns, command) -> exit_code`. The orchestration layer (`_tfui_run`) resolves which strategy to use and delegates. The renderer layer (`_tfui_render_plan_tree`) parses plan JSON into a human-readable tree view on stdout.

Key insight: the strategy controls **how** the command runs (UI treatment), while the renderer controls **what** the user sees (output format). Adding agent mode requires a new strategy AND a new renderer.

### Design Decisions

1. **Agent as a strategy, not a format flag.** The existing pattern is `--mode <strategy>` mapping to behavior. Agent mode is a behavior: silent execution + structured JSON output. Adding `--output json` would break the clean mode/strategy mapping.

2. **Reuse `_tfui_strategy_silent` semantics.** The agent strategy is functionally identical to silent (no UI) but semantically distinct. Having a separate `_tfui_strategy_agent` function allows future divergence (e.g., streaming progress events as JSON) without changing the silent strategy.

3. **Risk assessment in jq.** The entire JSON transformation (action classification, risk assessment, summary aggregation) is a single jq filter. This keeps it pure, testable, and avoids shell string manipulation of JSON. The risk model is encoded as static arrays of resource type patterns.

4. **Risk model: action x resource type matrix.**

   | Action    | Critical Type   | High Risk Type | Medium Risk Type | Other    |
   |-----------|-----------------|----------------|------------------|----------|
   | delete    | critical        | critical       | high             | high     |
   | replace   | critical        | critical       | high             | high     |
   | update    | high            | high           | medium           | medium   |
   | create    | medium          | medium         | low              | low      |

   Rationale: deletes are always risky (data loss), and the resource type amplifies that risk. Creates are inherently safer but creating a database still deserves attention.

5. **`_tfui_run_sub` skip for silent modes.** Both `_tfui_strategy_silent` and `_tfui_strategy_agent` don't need the animation wrapper in `_tfui_run_sub`. Added early return to skip UI setup/teardown for these strategies.

### Output Schema

```json
{
  "has_changes": boolean,
  "summary": {
    "add": number,
    "change": number,
    "destroy": number,
    "replace": number
  },
  "changes": [
    {
      "action": "create" | "update" | "delete" | "replace",
      "address": string,
      "risk": "low" | "medium" | "high" | "critical"
    }
  ],
  "risk_level": "low" | "medium" | "high" | "critical",
  "destructive": boolean
}
```

Schema stability contract:
- Fields are never removed (only deprecated with new fields alongside)
- `changes` array preserves terraform's resource order
- `risk_level` is the maximum of all individual change risks
- `destructive` is true when any destroy or replace exists

### Resource Classification

Three tiers of resource types, targeting AWS/GCP/Azure:

- **Critical:** Databases, storage, encryption keys (data-bearing, often irreversible deletes)
- **High risk:** IAM, networking core, compute clusters (outage-causing, security-sensitive)
- **Medium risk:** Security groups, DNS, queues (connectivity-affecting, but recoverable)

Unknown resource types default to:
- `low` for creates
- `medium` for updates
- `high` for deletes/replaces

This errs on the side of caution for unknown resources being deleted.

### Edge Cases Handled

- Empty plan (no resource_changes key): `has_changes: false`
- Plan with only no-op/read actions: `has_changes: false`
- Plan with only data source reads: excluded from changes
- Replace actions (`["delete", "create"]`): counted in `summary.replace`, not add/destroy

## Viability Assessment

**Fully viable.** The implementation:
- Uses only bash + jq (no new dependencies)
- Fits cleanly into the existing strategy pattern
- Does not modify any existing public API signatures
- All 125 tests pass (95 existing + 30 new)
- Cross-platform: jq filter is portable, no bash-specific JSON handling

**Future extensibility:**
- The schema can grow (add `provider`, `module_path`, `before`/`after` values)
- Risk model can be externalized to a config file
- Agent strategy can evolve to emit streaming JSONL for real-time progress
- The `_tfui_render_plan_json` function can be reused by other features (policy engine, etc.)
