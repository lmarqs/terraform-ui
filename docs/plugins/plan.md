---
layout: default
parent: Plugin Catalog — All Terraform UI Features
title: Terraform Plan Viewer — Tree Navigation
id: plan
key: p
seo:
  description: Navigate terraform plan changes as an expandable tree with inline risk badges, attribute diffs, and targeted apply.
description: "Describes how the Plan plugin shows a plan as a tree with risk badges, attribute diffs and pins."
audience: "Users reviewing a plan before they apply it, and contributors changing plan"
category: operations
default_enabled: true
---

# Plan Review

## Overview

Plan Review runs `terraform plan` and shows the changes as a tree you can filter, expand and pin, with a risk badge on each change.

Running `terraform plan` on a module with 30+ resources produces a wall of text. You must read every line to find the one dangerous delete among 29 creates. There's no prioritization, no risk signal, no way to focus on what matters. The Plan screen adds:

- **Risk badges** surface dangerous changes immediately (no scanning required)
- **Expand/collapse** lets you focus on relevant diffs (attention management)
- **Phantom detection** separates real changes from computed-attribute noise
- **Pin resources** to scope the next plan, and the apply after it, to them

## Screenshot

![Plan Review]({{ site.baseurl }}/assets/demo/plan-review.gif)

## Interactive (TUI)

Press `p` from the home menu. The plugin immediately runs `terraform plan` against the current context.

### Keybindings

| Key | Action | When |
|-----|--------|------|
| `j` / `k` / `↑` / `↓` | Navigate up/down | List |
| `g` / `G` | Jump to first/last change | List |
| `Enter` / `i` | Expand/collapse a module, or inspect a change's attribute diffs | List |
| `/` | Filter (fuzzy search) | List |
| `Space` | Pin/unpin selected resource | List & detail |
| `Ctrl+p` | Toggle pinned-only view | List |
| `Ctrl+u` | Clear all pins | List |
| `Ctrl+t` | Toggle tree/flat view | List |
| `[` / `]` | Collapse/expand all (tree) | Tree mode |
| `←` / `→` | Horizontal pan | List & detail |
| `Ctrl+w` | Toggle line wrap | List & detail |
| `a` | Apply (replans if pins changed, then confirms) | List |
| `A` | Auto-approve (apply immediately, no confirmation) | List |
| `t` | Taint -- navigates to taint plugin | List |
| `T` | Untaint -- navigates to untaint plugin | List |
| `e` | Edit in $EDITOR | List & detail |
| `!` | Batch action palette (apply, auto-apply, taint, untaint) | List (when pins > 0) |
| `l` | Show the terraform log | After a plan ran |
| `ctrl+r` | Re-run plan (refresh) | Plan finished or failed |
| `u` | Force-unlock | Locked error |
| `Esc` / `q` | Back to home | Always |

### Flow

```
Home ──p──→ Plan (loading) ──→ Plan (results)
                                  │
                                  ├── Enter → expand diffs
                                  ├── Space → pin for apply
                                  ├── a → Apply (pinned resources or all)
                                  └── q → Home
```

## Command Line (CLI)

```bash
# Default: spinner with elapsed time
tfui plan -project ./infra

# Headless: tree-view text output (no animation)
tfui plan -project ./infra -ci

# JSON: terraform's own plan document
tfui plan -project ./infra -json

# Targeted: plan only specific resources
tfui plan -project ./infra -target aws_instance.web -target aws_s3_bucket.old

# With chdir
tfui plan -project ./infra -chdir modules/networking
```

### Output Examples

**Headless:**
```
+ aws_instance.web
~ aws_security_group.main
- aws_s3_bucket.old
-/+ aws_db_instance.primary

Plan: 1 to add, 2 to change, 1 to destroy.
Risk: critical
```

Replacements count under "change". The `Risk:` line shows the highest risk among the changes and is left out when no change carries risk.

**`-json`:** the output is terraform's own `terraform show -json` document, passed through unchanged. Risk and phantom analysis are not in it.

Exit codes: see [CLI I/O contract](../reference/cli-io-contract.md#exit-codes). Plan exits `2` when the plan has changes.

## Equivalence

| Goal | CLI | TUI |
|------|-----|-----|
| See what will change | `tfui plan -ci` | Press `p` |
| Get plan as JSON | `tfui plan -json` | N/A (TUI is visual) |
| Plan specific resources | `tfui plan -target X` | Press `p`, pin resources |
| Plan then apply | `tfui apply` | `p` → review → `a` → `y` |

## Configuration

```hcl
# tfui.hcl (project root)
defaults {
  plugin "plan" {
    targets = ["module.networking", "aws_instance.web"]
  }
}
```

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `targets` | list | `[]` | Default resource targets for plan |

`targets` is a project default, used only when the run carries no explicit
selection. An explicit `-target` flag or a TUI pin **replaces** it. Declare the
same list under `plugin "apply"` to keep a scoped apply in step with the scoped
plan. To turn the plugin off, see
[Enabling/Disabling Plugins](index.md#enablingdisabling-plugins).

## Related

- [Apply](apply.md) -- execute the planned changes
- [Risk Analysis](risk.md) -- risk breakdown by severity
- [Blast Radius](blastradius.md) -- module-grouped impact view
