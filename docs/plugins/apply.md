---
layout: default
parent: Plugin Catalog — All Terraform UI Features
title: "Terraform Apply with Context and Pinned Targets"
id: apply
key: a
seo:
  description: "Apply Terraform changes from a terminal UI: review the plan, confirm, apply only pinned resources, and track elapsed time."
description: "Describes how the Apply plugin applies changes: confirmation, pinned targets and its settings."
audience: "Users applying changes after reviewing a plan, and contributors changing apply"
category: operations
default_enabled: true
---

# Apply

## Overview

Apply runs `terraform apply` after you review the plan and confirm with `y`. It applies only your pinned resources when you have pins, and tracks elapsed time.

Terraform's `apply` prompt says "Do you want to perform these actions?" with no context — the same prompt whether you're creating 1 resource or deleting 50. You type "yes" based on faith, not understanding. The Apply screen adds:

- **Confirmation after review** — `tfui apply` and the Plan screen show the plan before Apply asks to confirm
- **Elapsed time tracking** — long applies (10+ minutes) need progress feedback
- **Pin-scoped execution** — apply only pinned resources without typing `-target` addresses
- **Error recovery** — retry a failed one-shot apply from the same session; a failed plan-file apply goes back to plan for a fresh, reviewed plan

## Screenshot

![Apply]({{ site.baseurl }}/assets/demo/apply.gif)

## Targeting

**CLI:** Pass `-target` to `tfui apply`. `tfui apply -target=X` opens the plan view with those targets, so the saved plan holds only the targeted changes and `a` applies that plan. Without `-target`, that plan takes the `targets` set in `plugin "apply"`. `tfui apply -auto-approve -target=X` has no saved plan: terraform plans and applies in one shot, with the targets on that command. Apply never reads a plan an earlier `tfui plan` saved, so a `-target` given only to `tfui plan` does not reach the apply: `tfui plan -target X && tfui apply` plans every pending change.

**TUI:** Pin resources with `Space` in the Plan view. When you press `a`, tfui re-plans with only pinned resources as targets, then applies that targeted plan.

## Interactive (TUI)

Press `a` in Plan after reviewing changes; if resources are pinned, apply targets only those. Pressing `a` on Home opens Apply with no reviewed plan, so it stays idle until Plan hands it one.

### Keybindings

| Key | Action | When |
|-----|--------|------|
| `Enter` | Ask for confirmation again after `n` | Idle, with a reviewed plan |
| `y` / `Y` / `Enter` | Confirm and execute | Confirming |
| `n` / `N` / `Esc` | Cancel and return to Plan | Confirming |
| `l` | Show the terraform log | Done, Error |
| `ctrl+r` | Retry | Error, after `tfui apply -auto-approve` |
| `ctrl+r` | Back to Plan | Done |
| `Esc` | Back | Done, Error |
| `q` | Back to home | Always |

### Flow

```
Plan ──a──→ Apply (confirming)
Plan ──a──→ (pins changed: Plan re-plans) ──→ Apply (confirming)
Plan ──A──→ Apply (auto-approve: skip confirmation)
               │
               ├── y → Apply (running) → Apply (success)
               │                       → Apply (error) ──Esc──→ Plan, re-plan
               └── n/Esc → return to plan
```

## Command Line (CLI)

```bash
# Default: review the plan, then confirm the apply
tfui apply -project ./infra

# Auto-approve: skip review and confirmation (required for non-interactive apply)
tfui apply -project ./infra -auto-approve

# Silent: no animation. Mirrors terraform — non-interactive apply without a
# plan file requires -auto-approve, else "Apply not allowed for non-interactive
# use" (exit 1), exactly as `terraform apply` behaves with no TTY.
tfui apply -project ./infra -ci -auto-approve

# Targeted: apply only specific resources
tfui apply -project ./infra -target aws_instance.web

# With chdir
tfui apply -project ./infra -chdir modules/networking
```

Without `-auto-approve`, `tfui apply` opens Plan first with any `-target` you passed; press `a` there to hand the reviewed plan to Apply.

### Output Examples

Apply writes nothing to stdout. The outcome line goes to stderr, and the exit code is the signal.

**Silent mode:**
```
Apply complete. Resources: 1 added, 0 changed, 0 destroyed.
```

The counts are terraform's own tally, so an apply can be reconciled against the
plan that was approved. A run that fails partway prints no tally — terraform
does not emit one — so the outcome reports what did finish instead:

```
Error: creating local_file: ...
Applied before the failure: 1 added, 0 changed, 0 destroyed.
```

Exit codes: see [CLI reference](../reference/cli-reference.md#exit-codes).

## Equivalence

| Goal | CLI | TUI |
|------|-----|-----|
| Apply all changes | `tfui apply` | `p` → `a` → `y` |
| Apply specific resources | `tfui apply -target X` | `p` → pin X → `a` → `y` |
| Check apply result | Exit code + stderr outcome line | Success/error screen |

## Configuration

```hcl
# tfui.hcl (project root)
defaults {
  plugin "apply" {
    targets = ["module.networking"]
  }
}
```

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `targets` | list | `[]` | Default resource targets for `tfui apply`: the plan it opens, or the one-shot run with `-auto-approve` |

An explicit `-target` replaces the list rather than adding to it, the same way it
does for `plugin "plan"`. To turn the plugin off, see
[Enabling/Disabling Plugins](index.md#enablingdisabling-plugins).

## Related

- [Plan](plan.md) -- review changes before applying
- [Risk Analysis](risk.md) -- assess risk before applying
