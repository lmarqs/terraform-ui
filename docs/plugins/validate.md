---
layout: default
parent: Plugins
title: Validate
id: validate
key: v
category: operations
default_enabled: true
description: Run terraform validate and display configuration diagnostics
---

## Overview

Run `terraform validate` and display diagnostics. The plugin groups errors and warnings by severity, with expandable source locations and suggestions for fixing each issue.

## Interactive (TUI)

Press `v` from the home menu. The plugin runs validation and shows errors/warnings in an expandable list.

### Keybindings

| Key | Action | Context |
|-----|--------|---------|
| `Enter` | Expand diagnostic detail | List |
| `e` | Open file at error location in $EDITOR | List |
| `ctrl+r` | Re-run validation | Always |
| `q` | Back to home | Always |

### Flow

```
Home ──v──→ Validate (loading) ──→ Validate (results)
                                      │
                                      ├── Enter → Expand diagnostic detail
                                      ├── e → Open $EDITOR at source location
                                      ├── ctrl+r → Re-run validation
                                      └── q → Home
```

## Command Line (CLI)

```bash
tfui validate --project ./infra
tfui validate --project ./infra --ci
tfui validate --project ./infra -json
```

| Code | Meaning |
|------|---------|
| 0 | Configuration valid |
| 1 | Validation errors found |

## Configuration

```hcl
# tfui.hcl
plugin "validate" {
  enabled = true
}
```

## Screenshots

```
Validate

✓ Configuration is valid  (0 errors, 2 warnings)

  ⚠ Warning: Argument is deprecated
    on modules/vpc/main.tf line 12
    The "instance_tenancy" argument is deprecated. Use "default_instance_tenancy" instead.

  ⚠ Warning: Version constraint not specific enough
    on providers.tf line 3

Enter detail  e edit  ctrl+r rerun  q back
```

## Related

- [Init](init.md) -- run `terraform init` before validating
- [Plan](plan.md) -- plan runs implicit validation
