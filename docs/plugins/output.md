---
layout: default
parent: Plugin Catalog — All Terraform UI Features
title: "Browse Terraform Outputs and Their JSON"
id: output
key: o
category: navigation
default_enabled: true
seo:
  description: "View the terraform output values of the current workspace in a list filterable by name, with sensitive values hidden."
description: "Describes how the Outputs plugin shows the output values of the current workspace."
audience: "Users who need an output value of the current workspace, such as an endpoint or ID, without leaving tfui"
---

# Outputs

## Overview

Lists the current workspace's outputs with their values, filterable by name. Sensitive values show as `(sensitive)`. Outputs are fetched via `terraform output -json`.

## Screenshot

![Outputs]({{ site.baseurl }}/assets/demo/output.gif)

## Interactive (TUI)

Press `o` from the home menu. The plugin fetches outputs and displays them in a filterable list.

### Keybindings

| Key | Action | When |
|-----|--------|------|
| `j` / `k` / `↑` / `↓` | Navigate up/down | List |
| `g` / `G` | Jump to first/last | List |
| `/` | Filter outputs by name | List |
| `ctrl+r` | Refresh outputs | Outputs loaded or failed |
| `Esc` | Close filter; otherwise back to Home | Filter / List |
| `q` | Back to home | Always |

### Flow

```
Home ──o──→ Outputs (loading) ──→ Outputs (list)
                                     │
                                     ├── / → Filter by name
                                     ├── ctrl+r → Refresh
                                     └── q → Home
```

## Command Line (CLI)

```bash
tfui output -project ./infra
tfui output -project ./infra -json
tfui output -project ./infra -ci
```

Exit codes: see [CLI reference](../reference/cli-reference.md#exit-codes).

## Equivalence

| Goal | CLI | TUI |
|------|-----|-----|
| List all outputs | `tfui output -ci` | Press `o` |
| Get output as JSON | `tfui output -json` | N/A (TUI is visual) |
| Find one output | `tfui output -json \| jq '.<name>'` | `o` → `/` name |

## Related

- [State Browser](state.md) -- browse resources vs output values
- [Workspace](workspace.md) -- switch workspace to see different outputs
