---
layout: plugin
title: Plan Review
id: plan
key: p
description: Review terraform plan changes with risk classification and expandable attribute diffs
category: operations
default_enabled: true
---

## Why This Screen Exists

Running `terraform plan` on a module with 30+ resources produces a wall of text. You must read every line to find the one dangerous delete among 29 creates. There's no prioritization, no risk signal, no way to focus on what matters.

The Plan screen transforms plan output from a data dump into a decision-support tool:

- **Risk badges** surface dangerous changes immediately (no scanning required)
- **Expand/collapse** lets you focus on relevant diffs (attention management)
- **Phantom detection** separates real changes from computed-attribute noise
- **Pin resources** to build a selective apply target list visually

## Interactive (TUI)

Press `p` from the home menu. The plugin immediately runs `terraform plan` against the current scope.

### Keybindings

| Key | Action |
|-----|--------|
| `j` / `k` / `↑` / `↓` | Navigate up/down |
| `g` / `G` | Jump to first/last change |
| `Enter` / `i` | Expand/collapse attribute diffs |
| `Space` | Pin/unpin selected resource |
| `a` | Apply (confirms, then applies pinned or all) |
| `r` | Re-run plan (refresh) |
| `u` | Force-unlock (when state is locked) |
| `Esc` / `q` | Back to home |

### Flow

```
Home ──p──→ Plan (loading) ──→ Plan (results)
                                  │
                                  ├── Enter → expand diffs
                                  ├── Space → pin for apply
                                  ├── a → Apply (pinned targets or all)
                                  └── q → Home
```

### Screenshot

```
Plan Review

 > + aws_instance.web                          [low]
   ~ aws_security_group.main                   [medium]
 * - aws_s3_bucket.old                         [HIGH]
   -/+ aws_db_instance.primary                 [CRITICAL]

Plan: 1 to add, 1 to change, 1 to destroy, 1 to replace
Overall risk: CRITICAL

Enter expand  Space pin  a apply  r refresh  q back
```

## Command Line (CLI)

```bash
# Default: spinner with elapsed time
tfui plan --project ./infra

# Silent: tree-view text output (no animation)
tfui plan --project ./infra --mode silent

# Agent: structured JSON for automation/AI
tfui plan --project ./infra --mode agent

# Targeted: plan only specific resources
tfui plan --project ./infra --target aws_instance.web --target aws_s3_bucket.old

# With scope (monorepo)
tfui plan --project ./infra --scope modules/networking
```

### Output Examples

**Silent mode:**
```
  + aws_instance.web
  ~ aws_security_group.main
  - aws_s3_bucket.old
  -/+ aws_db_instance.primary

Plan: 1 to add, 1 to change, 1 to destroy, 1 to replace.
Risk: CRITICAL
```

**Agent mode (JSON):**
```json
{
  "changes": [
    {"address": "aws_instance.web", "action": "create", "risk": "low", "phantom": false},
    {"address": "aws_security_group.main", "action": "update", "risk": "medium", "phantom": false},
    {"address": "aws_s3_bucket.old", "action": "delete", "risk": "critical", "phantom": false}
  ],
  "summary": {"add": 1, "change": 1, "destroy": 1},
  "risk": "critical",
  "phantom_changes": 0,
  "phantom_resources": []
}
```

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Plan succeeded, no changes |
| 1 | Error (terraform failed, invalid config) |
| 2 | Plan succeeded, changes detected |

## Equivalence

| Goal | CLI | TUI |
|------|-----|-----|
| See what will change | `tfui plan --mode silent` | Press `p` |
| Get plan as JSON | `tfui plan --mode agent` | N/A (TUI is visual) |
| Plan specific resources | `tfui plan --target X` | Press `p`, pin resources |
| Plan then apply | `tfui plan && tfui apply` | `p` → review → `a` → `y` |

## Configuration

```yaml
# tfui.yaml
plugins:
  plan:
    enabled: true
    targets:
      - "module.networking"
      - "aws_instance.web"
```

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enabled` | bool | `true` | Enable/disable the plugin |
| `targets` | list | `[]` | Default resource targets for plan |

## Related

- [Apply](apply.md) -- execute the planned changes
- [Risk Analysis](risk.md) -- risk breakdown by severity
- [Blast Radius](blastradius.md) -- module-grouped impact view
