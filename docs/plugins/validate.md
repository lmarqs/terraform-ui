---
layout: default
parent: Plugin Catalog — All Terraform UI Features
title: "Terraform Validate with Grouped Diagnostics"
id: validate
key: v
category: operations
default_enabled: true
seo:
  description: "Run terraform validate and browse its errors and warnings, errors first, each with its file and line and terraform's detail."
description: "Describes how the Validate plugin runs terraform validate and shows its diagnostics."
audience: "Users checking configuration before a plan"
---

# Validate

## Overview

Run `terraform validate` and display diagnostics. Lists errors first, then warnings, each with its file and line. Expand a row to read terraform's detail.

## Screenshot

![Validate]({{ site.baseurl }}/assets/demo/validate.gif)

## Interactive (TUI)

Press `v` from the home menu. The plugin runs validation and shows errors/warnings in an expandable list.

### Keybindings

| Key | Action | When |
|-----|--------|------|
| `j` / `k` / `↑` / `↓` | Navigate up/down | List |
| `g` / `G` | Jump to first/last | List |
| `Enter` / `i` | Expand diagnostic detail | List |
| `ctrl+r` | Re-run validation | Validation finished or failed |
| `q` | Back to home | Always |

### Flow

```
Home ──v──→ Validate (loading) ──→ Validate (results)
                                      │
                                      ├── Enter → Expand diagnostic detail
                                      ├── ctrl+r → Re-run validation
                                      └── q → Home
```

## Command Line (CLI)

```bash
tfui validate -project ./infra
tfui validate -project ./infra -ci
tfui validate -project ./infra -json
```

Exit codes: see [CLI reference](../reference/cli-reference.md#exit-codes).

## Equivalence

| Goal | CLI | TUI |
|------|-----|-----|
| Validate configuration | `tfui validate -ci` | Press `v` |
| Get diagnostics as JSON | `tfui validate -json` | N/A (TUI is visual) |
| Re-validate after fix | `tfui validate -ci` | `v` → `ctrl+r` |

## Related

- [Init](init.md) -- run `terraform init` before validating
- [Plan](plan.md) -- plan runs implicit validation
