---
layout: plugin
title: Apply
id: apply
key: a
description: Apply terraform changes with confirmation, targeting, and elapsed time tracking
category: operations
default_enabled: true
---

## Why This Screen Exists

Terraform's `apply` prompt says "Do you want to perform these actions?" with no context — the same prompt whether you're creating 1 resource or deleting 50. You type "yes" based on faith, not understanding.

The Apply screen adds:

- **Context-aware confirmation** — shows resource count and whether targeting is active
- **Elapsed time tracking** — long applies (10+ minutes) need progress feedback
- **Pin-scoped execution** — apply only pinned resources without typing `--target` addresses
- **Error recovery** — retry from same session without re-running the full command

## Interactive (TUI)

### Entry Points

- **From Plan:** Press `a` after reviewing changes. If resources are pinned, apply targets only those.
- **From Home:** Press `a` directly. Shows idle state until you confirm.

### Keybindings

| Key | Action | When |
|-----|--------|------|
| `Enter` | Start apply (shows confirmation) | Idle state |
| `y` / `Enter` | Confirm and execute | Confirming state |
| `n` / `Esc` | Cancel | Confirming state |
| `r` | Retry after failure | Error state |
| `Esc` / `q` | Back to home | Any state |

### Flow

```
Plan ──a──→ Apply (confirming)
               │
               ├── y → Apply (running) → Apply (success)
               │                       → Apply (error) ──r──→ retry
               └── n → Apply (idle)
```

### Screenshots

**Confirmation (targeted):**
```
Apply

Are you sure you want to apply these changes?
Targeting 3 resource(s).

[y]es / [n]o
```

**Running:**
```
Apply

>>> Applying changes... 1m23s
```

**Success:**
```
Apply

Apply complete! Resources are up-to-date.
Duration: 2m45s
```

## Command Line (CLI)

```bash
# Default: plan first, then apply (with progress)
tfui plan --project ./infra
tfui apply --project ./infra

# Silent: no animation
tfui apply --project ./infra --ci

# NDJSON events (terraform-compatible)
tfui apply --project ./infra -json

# Targeted: apply only specific resources
tfui plan --project ./infra --target aws_instance.web
tfui apply --project ./infra

# With chdir (monorepo)
tfui apply --project ./infra --chdir modules/networking
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

## Equivalence

| Goal | CLI | TUI |
|------|-----|-----|
| Apply all changes | `tfui plan && tfui apply` | `p` → `a` → `y` |
| Apply specific resources | `tfui plan --target X && tfui apply` | `p` → pin X → `a` → `y` |
| Check apply result | Exit code + stdout | Success/error screen |

## How Targeting Works

**CLI:** Pass `--target` to the `plan` command. The saved plan file already contains only targeted changes. Apply then applies that plan.

**TUI:** Pin resources with `Space` in the Plan view. When you press `a`, tfui re-plans with only pinned resources as targets, then applies that targeted plan.

**Key insight:** You don't pass `--target` to `apply`. Targeting happens at plan time — apply always executes the saved plan file exactly.

## Configuration

```yaml
# tfui.hcl
plugins:
  apply:
    enabled: true
    targets:
      - "module.networking"
```

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enabled` | bool | `true` | Enable/disable the plugin |
| `targets` | list | `[]` | Default resource targets (used when no pins active) |

## Related

- [Plan](plan.md) -- review changes before applying
- [Risk Analysis](risk.md) -- assess risk before applying
