---
layout: default
parent: Plugin Catalog — All Terraform UI Features
title: Apply
id: apply
key: a
description: Apply terraform changes with confirmation, targeting, and elapsed time tracking
category: operations
default_enabled: true
---

# Apply

## Overview

Terraform's `apply` prompt says "Do you want to perform these actions?" with no context — the same prompt whether you're creating 1 resource or deleting 50. You type "yes" based on faith, not understanding.

The Apply screen adds:

- **Context-aware confirmation** — shows resource count and whether targeting is active
- **Elapsed time tracking** — long applies (10+ minutes) need progress feedback
- **Pin-scoped execution** — apply only pinned resources without typing `-target` addresses
- **Error recovery** — retry from same session without re-running the full command

## Screenshot

![Apply]({{ site.baseurl }}/assets/demo/apply.gif)

## Interactive (TUI)

### Entry Points

- **From Plan:** Press `a` after reviewing changes. If resources are pinned, apply targets only those.
- **From Home:** Press `a` directly. Shows idle state until you confirm.

### Keybindings

| Key | Action | Context |
|-----|--------|---------|
| `Enter` | Start apply (shows confirmation) | Idle |
| `y` / `Y` | Confirm and execute | Confirming |
| `n` / `Esc` | Cancel | Confirming |
| `r` | Retry after failure | Error |
| `Esc` / `q` | Back to home | Always |

Confirmation requires an explicit `y`/`Y` — `Enter` does **not** confirm. This
matches the project-wide confirmation convention (`ConfirmFrame` / `InputConfirm`)
and prevents the `Enter` that launched `tfui apply` from leaking into the TUI and
auto-confirming a destructive apply.

### Flow

```
Plan ──a──→ Apply (no targets → confirming)
Plan ──a──→ Apply (with targets → replanning → confirming)
Plan ──A──→ Apply (auto-approve: skip confirmation)
               │
               ├── y → Apply (running) → Apply (success)
               │                       → Apply (error) ──r──→ retry
               └── n/Esc → DeactivateMsg → return to plan
```

### Replan for Targeted Apply

When pinned resources exist, apply does NOT use the saved plan file directly (terraform constraint). Instead it replans with `-target` flags to produce a targeted plan, shows it for review, then applies that plan file. This ensures the user always reviews exactly what will be applied.

## Command Line (CLI)

```bash
# Default: plan first, then apply (with progress)
tfui plan -project ./infra
tfui apply -project ./infra

# Auto-approve: skip confirmation (required for non-interactive apply)
tfui apply -project ./infra -auto-approve

# Silent: no animation. Mirrors terraform — non-interactive apply without a
# plan file requires -auto-approve, else "Apply not allowed for non-interactive
# use" (exit 1), exactly as `terraform apply` behaves with no TTY.
tfui apply -project ./infra -ci -auto-approve

# NDJSON events (terraform-compatible)
tfui apply -project ./infra -json

# Targeted: apply only specific resources
tfui plan -project ./infra -target aws_instance.web
tfui apply -project ./infra

# With chdir (monorepo)
tfui apply -project ./infra -chdir modules/networking
```

### Output Examples

**Silent mode:**
```
Apply complete.
```

**Agent mode (JSON):**
```json
{
  "status": "complete"
}
```

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Apply succeeded |
| 1 | Apply failed (terraform error, no plan file, etc.) |

### Targeting

**CLI:** Pass `-target` to the `plan` command. The saved plan file already contains only targeted changes. Apply then applies that plan.

**TUI:** Pin resources with `Space` in the Plan view. When you press `a`, tfui re-plans with only pinned resources as targets, then applies that targeted plan.

**Key insight:** You don't pass `-target` to `apply`. Targeting happens at plan time — apply always executes the saved plan file exactly.

## Equivalence

| Goal | CLI | TUI |
|------|-----|-----|
| Apply all changes | `tfui plan && tfui apply` | `p` → `a` → `y` |
| Apply specific resources | `tfui plan -target X && tfui apply` | `p` → pin X → `a` → `y` |
| Check apply result | Exit code + stdout | Success/error screen |

## Configuration

```hcl
# tfui.hcl
plugin "apply" {
  enabled = true
  targets = ["module.networking"]
}
```

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enabled` | bool | `true` | Enable/disable the plugin |
| `targets` | list | `[]` | Default resource targets (used when no pins active) |

## Related

- [Plan](plan.md) -- review changes before applying
- [Risk Analysis](risk.md) -- assess risk before applying
