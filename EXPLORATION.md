# Plan-over-Plan Diffing — Design Exploration

## Problem Statement

AI agents iterate: edit HCL, plan, check result, edit again. On each iteration,
re-reading the full plan wastes context. What agents need is: "what changed
between this plan and the last one?" — new creates, resolved destroys, risk trend.

## Storage Strategy

### Location: `.tfui/plans/` in the workspace

- Created alongside `.terraform/` (same ignore/cleanup conventions)
- One JSON summary per plan, named by ISO-8601 timestamp
- Path: `<working_dir>/.tfui/plans/<ISO8601>.json`

### What to persist (plan summary, not full plan)

```json
{
  "timestamp": "2024-01-15T10:30:00Z",
  "creates": ["aws_instance.web"],
  "updates": [],
  "destroys": ["aws_iam_role.old"],
  "replaces": []
}
```

This is tiny (just resource addresses by action type) — safe to keep many.

### Retention: keep last 10 summaries

- Ring buffer approach: after saving, delete oldest beyond 10
- Simple `ls -t | tail -n +11 | xargs rm` logic
- Configurable via `TFUI_HISTORY_LIMIT` env var (default: 10)

## Interface: `tfui diff` subcommand

Chosen over `--diff` flag because:
- Diff can be run independently (without re-planning)
- Cleaner separation of concerns
- Library gets `tfui_diff` public function for embedding

Also: `tfui plan --diff` runs plan then immediately shows the diff (convenience).

## Diff Algorithm

1. Load current summary (latest) and previous summary (second-latest)
2. Compute set differences per category:
   - `new_creates` = creates in current but not in previous
   - `resolved_creates` = creates in previous but not in current
   - Same for updates, destroys, replaces
3. Compute summary counts (before/after)
4. Compute risk trend:
   - Score = destroys * 3 + replaces * 2 + updates * 1
   - Compare previous score vs current score
   - "improving" / "worsening" / "unchanged"

## Output Formats

### JSON (stdout, for machines/agents)

```json
{
  "previous_plan": "2024-01-15T10:30:00Z",
  "current_plan": "2024-01-15T10:35:00Z",
  "delta": {
    "new_creates": ["aws_instance.worker"],
    "resolved_creates": [],
    "new_destroys": [],
    "resolved_destroys": ["aws_iam_role.old_role"],
    "new_updates": ["aws_security_group.allow_tls"],
    "resolved_updates": [],
    "new_replaces": [],
    "resolved_replaces": []
  },
  "summary": {
    "before": { "add": 1, "change": 0, "destroy": 1, "replace": 0 },
    "after": { "add": 2, "change": 1, "destroy": 0, "replace": 0 }
  },
  "risk_trend": "improving"
}
```

### Human-readable (fd3 or stderr, for terminals)

```
Plan diff (vs 5 min ago):
  ++ aws_instance.worker (new create)
  -- aws_iam_role.old_role (destroy resolved)
  ~+ aws_security_group.allow_tls (new update)
  Risk: improving (score 3 -> 1)
```

## Edge Cases

1. **First plan ever** — no previous summary exists; diff reports "no previous
   plan to compare" and exits cleanly (exit 0, empty delta)
2. **Targets/vars change** — still compare. The summary captures what terraform
   would do, regardless of how it got there. The diff shows what matters: "these
   resources are no longer at risk."
3. **Identical plans** — all delta arrays empty, risk "unchanged", clear message
4. **Working dir changes** — each working dir has its own `.tfui/plans/` so
   histories don't cross
5. **Concurrent runs** — timestamp-based naming avoids collisions (sub-second
   resolution with `date +%Y%m%dT%H%M%S`)

## Implementation Plan

1. Add `_tfui_diff_*` functions to `lib/tfui.sh`:
   - `_tfui_diff_save_summary` — extract and persist plan summary
   - `_tfui_diff_load_latest` — read most recent two summaries
   - `_tfui_diff_compute` — compute the delta JSON
   - `_tfui_diff_render_human` — format human-readable output
   - `_tfui_diff_cleanup` — prune old summaries
2. Add `tfui_diff` public function
3. Hook `_tfui_diff_save_summary` into `tfui_plan` (after rendering)
4. Add `diff` subcommand to `bin/tfui`
5. Add `--diff` flag to `tfui plan` CLI subcommand
6. Write BATS tests with synthetic plan JSON

## Viability Assessment

- **Pure bash**: all operations are file I/O + jq queries — no new deps
- **Cross-platform**: `date`, `mkdir`, `jq`, `sort` — all portable
- **Performance**: summaries are <1KB each; diffing two is instant
- **Risk**: minimal — additive feature, doesn't change existing plan/apply flow
- **Agent value**: high — reduces context window waste significantly
