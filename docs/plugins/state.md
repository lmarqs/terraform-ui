---
layout: default
parent: Plugin Catalog — All Terraform UI Features
title: Terraform State Browser — Interactive Explorer
id: state
key: s
seo:
  description: Browse, search, and inspect terraform state resources interactively. Taint, untaint, move, and delete with safety prompts instead of raw CLI.
description: "Describes how the State plugin browses state and taints, untaints, moves and removes resources."
audience: "Users inspecting state, or about to remove, move or taint a resource in it"
category: navigation
default_enabled: true
---

# State Browser

## Overview

The State Browser lists every resource in the current state as a filterable tree, shows any resource's attributes, and removes, moves, taints or untaints resources after a confirmation, one at a time or in a batch.

Terraform's state commands are one-at-a-time, confirmationless, and require exact address typing:

```bash
terraform state list                     # flat list, no hierarchy
terraform state show aws_instance.web    # one resource at a time
terraform state rm aws_instance.web      # no confirmation! irreversible!
```

Exploring state requires chaining `state list | grep | state show` repeatedly. Mutating state is dangerous with zero safety rails — one typo in `state rm` and the resource is gone. The State Browser adds:

- **Browse without committing** — see all resources, inspect any, without running N commands
- **Filter/search** — fzf fuzzy matching on resource addresses
- **Tree mode** — module hierarchy view (terraform has no grouped view)
- **Safe mutations** — confirmation before rm/mv (terraform provides none!)
- **Batch operations** — pin multiple, then act on all at once

## Screenshot

![State Browser]({{ site.baseurl }}/assets/demo/state-browse.gif)

## Interactive (TUI)

Press `s` from the home menu. The plugin loads the current terraform state.

### Keybindings

| Key | Action | When |
|-----|--------|------|
| `j` / `k` / `↑` / `↓` | Navigate up/down | List |
| `g` / `G` | Jump to first/last | List |
| `Enter` / `i` | Inspect resource detail | List |
| `/` | Enter filter mode | List |
| `Space` | Pin/unpin resource | List |
| `Ctrl+t` | Toggle tree/flat mode | List |
| `Ctrl+p` | Toggle pinned-only view | List |
| `Ctrl+u` | Clear all pins | List |
| `[` / `]` | Collapse/expand all (tree) | Tree mode |
| `←` / `→` | Horizontal pan | List & detail |
| `Ctrl+w` | Toggle line wrap | Detail |
| `d` | Delete from state | List (cursor item) |
| `m` | Move (rename address) | List (cursor item) |
| `t` | Taint → navigates to taint plugin | List (cursor item) |
| `T` | Untaint → navigates to untaint plugin | List (cursor item) |
| `n` | Import → navigates to import plugin | List (cursor item) |
| `e` | Edit in $EDITOR | List (cursor item) |
| `!` | Batch action palette | List (when pins > 0) |
| `ctrl+r` | Reload state | List, once loading finished or failed |
| `u` | Force-unlock | Error (locked) |
| `Esc` / `q` | Back / exit detail | Any |

### Flow

```
Home ──s──→ State (list)
               │
               ├── Enter → Detail (inspect) ──Esc──→ back to list
               ├── / → Filter (type to search) ──Esc──→ back to list
               ├── d → Confirm delete ──y──→ deleted, refresh
               ├── m → Enter new address ──Enter──→ Confirm move ──y──→ moved, refresh
               ├── t → Confirm taint ──y──→ tainted, refresh
               ├── Space → toggle pin
               ├── ! → Batch palette (d/m/t/T/n/e) → act on all pinned
               └── q → Home
```

## Command Line (CLI)

### State Mutations

```bash
# Remove resource from state (does NOT destroy infrastructure)
tfui state rm aws_instance.old -project ./infra

# Move/rename resource address in state
tfui state mv aws_instance.web aws_instance.main -project ./infra

# Mark resource for recreation on next apply
tfui state taint aws_instance.web -project ./infra

# Remove taint mark
tfui state untaint aws_instance.web -project ./infra
```

To import a resource, use [`tfui import`](import.md). To browse a state file instead of the live state, pass
`-state`; the [CLI reference](../reference/cli-reference.md) describes it.

Exit codes: see [CLI I/O contract](../reference/cli-io-contract.md#exit-codes).

## Equivalence

| Goal | CLI | TUI |
|------|-----|-----|
| List resources | `tfui state -ci` | Press `s` |
| Inspect resource | `terraform state show ADDR` | `s` → navigate → `Enter` |
| Remove from state | `tfui state rm ADDR` | `s` → navigate → `d` → `y` |
| Rename in state | `tfui state mv A B` | `s` → navigate → `m` → type B → `Enter` → `y` |
| Taint resource | `tfui state taint ADDR` | `s` → navigate → `t` → `y` |
| Untaint resource | `tfui state untaint ADDR` | `s` → navigate → `T` → `y` |
| Import resource | `tfui import ADDR ID` | `s` → navigate → `n` → `Enter` → type ID → `Enter` → `y` |
| Batch delete | `tfui state rm X Y` | `s` → pin multiple → `!` → `d` → `y` |

## Related

- [Workspace](workspace.md) -- switch workspace before browsing state
- [Context](context.md) -- switch project chdir
- [Plan](plan.md) -- see what would change after state mutations
