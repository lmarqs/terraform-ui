---
layout: default
parent: Plugins
title: Outputs
id: output
key: o
category: navigation
default_enabled: true
description: View terraform output values for the current workspace
---

## Overview

View terraform output values for the current workspace. Outputs are fetched via `terraform output -json` and displayed in a filterable list with expandable JSON detail for complex values.

## Interactive (TUI)

Press `o` from the home menu. The plugin fetches outputs and displays them in a filterable list.

### Keybindings

| Key | Action | Context |
|-----|--------|---------|
| `Enter` / `i` | Inspect output value (expanded JSON) | List |
| `/` | Filter outputs by name | List |
| `ctrl+r` | Refresh outputs | Always |
| `Esc` | Back from detail / close filter | Detail / Filter |
| `q` | Back to home | Always |

### Flow

```
Home ──o──→ Outputs (loading) ──→ Outputs (list)
                                     │
                                     ├── Enter → Inspect value (detail)
                                     ├── / → Filter by name
                                     ├── ctrl+r → Refresh
                                     └── q → Home
```

## Command Line (CLI)

```bash
tfui output --project ./infra
tfui output --project ./infra -json
tfui output --project ./infra --ci
```

| Code | Meaning |
|------|---------|
| 0 | Outputs fetched successfully |
| 1 | Error fetching outputs |

## Configuration

```hcl
# tfui.hcl
plugin "output" {
  enabled = true
}
```

## Screenshots

```
Outputs                                          [5 values]

 > vpc_id          string   "vpc-0abc123def456"
   subnet_ids      list     (3 elements)
   db_endpoint     string   "db.example.com:5432"
   api_url         string   "https://api.example.com"
   config          object   (sensitive)

Enter inspect  / filter  ctrl+r refresh  q back
```

## Related

- [State Browser](state.md) -- browse resources vs output values
- [Workspace](workspace.md) -- switch workspace to see different outputs
