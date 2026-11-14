---
layout: default
parent: Plugin Catalog — All Terraform UI Features
title: "Terraform Init with a Form for Its Options"
id: init
key: i
seo:
  description: "Run terraform init from a terminal UI form with options such as upgrade, reconfigure and backend-config, and watch its progress in real time."
description: "Describes how the Init plugin runs terraform init with its options in a form."
audience: "Users initializing a project, or upgrading or reconfiguring one after a change"
category: operations
default_enabled: true
---

# Init

## Overview

The Init plugin provides a TUI interface for `terraform init`. It presents a form for the init options and shows real-time progress. The form starts from its defaults each time you open it.

## Screenshot

![Init]({{ site.baseurl }}/assets/demo/init.gif)

## Interactive (TUI)

Press `i` from the home menu to open the Init plugin.

### Keybindings

| Key | Action | When |
|-----|--------|------|
| `j` / `k` / `↑` / `↓` | Move between fields | Form |
| `Space` | Toggle the focused checkbox, or edit the focused text field | Form |
| `Enter` | Run init | Form |
| `Esc` | Cancel and return home | Always |

### Form Options

| Field | Flag | Default |
|-------|------|---------|
| upgrade | `-upgrade` | off |
| reconfigure | `-reconfigure` | off |
| backend | `-backend` | on |
| get | `-get` | on |
| force-copy | `-force-copy` | off |
| from-module | `-from-module=...` | (empty) |
| plugin-dir | `-plugin-dir=...` | (empty) |
| backend-config | `-backend-config=...` | (empty) |

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

`tfui init` with no init flag opens the form. Any of `-upgrade`, `-reconfigure`, `-backend` or `-backend-config`, or a [headless run](../reference/cli-io-contract.md#mode-resolution), starts init at once.

Exit codes: see [CLI I/O contract](../reference/cli-io-contract.md#exit-codes).

## Equivalence

| Goal | CLI | TUI |
|------|-----|-----|
| Initialize terraform | `tfui init` | Press `i` → `Enter` |
| Init with upgrade | `tfui init -upgrade` | `i` → `Space` on upgrade → `Enter` |
| Reconfigure backend | `tfui init -reconfigure` | `i` → `Space` on reconfigure → `Enter` |

## Related

- [Scaffold](../guides/scaffold.md) -- generate `tfui.hcl` config
- [Validate](validate.md) -- validate configuration after init
