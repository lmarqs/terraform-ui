---
layout: default
parent: Plugins
title: Init
id: init
key: i
description: Run terraform init with form-based options
category: operations
default_enabled: true
---

# Init

## Overview

The Init plugin provides a TUI interface for `terraform init`. It presents a form for common init options (upgrade, migrate-state, reconfigure, backend-config) and shows real-time progress. Form values are preserved across runs within a session for convenient re-initialization.

## Screenshot

![Init]({{ site.baseurl }}/assets/demo/init.gif)

## Interactive (TUI)

Press `i` from the home menu to open the Init plugin.

### Keybindings

| Key | Action | Context |
|-----|--------|---------|
| `Enter` | Submit form / confirm | Form |
| `Tab` | Next field | Form |
| `Shift+Tab` | Previous field | Form |
| `Esc` | Cancel and return home | Always |

### Form Options

| Field | Flag | Default |
|-------|------|---------|
| Upgrade | `-upgrade` | false |
| Migrate State | `-migrate-state` | false |
| Reconfigure | `-reconfigure` | false |
| Backend Config | `-backend-config=...` | (empty) |

### Flow

```
Home ──i──→ Init (form) ──Enter──→ Running (spinner + timer) ──→ Done/Error
                │
                └── Esc → Home
```

## Command Line (CLI)

```bash
tfui init -project ./infra
tfui init -project ./infra -upgrade
```

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Init succeeded |
| 1 | Init failed |

## Equivalence

| Goal | CLI | TUI |
|------|-----|-----|
| Initialize terraform | `tfui init` | Press `i` → `Enter` |
| Init with upgrade | `tfui init -upgrade` | `i` → check Upgrade → `Enter` |
| Reconfigure backend | `tfui init -reconfigure` | `i` → check Reconfigure → `Enter` |

## Configuration

```hcl
# tfui.hcl
plugin "init" {
  enabled = true
}
```

## Related

- [Scaffold](../guides/scaffold.md) -- generate `tfui.hcl` config
- [Validate](validate.md) -- validate configuration after init
