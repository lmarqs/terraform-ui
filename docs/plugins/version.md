---
layout: default
parent: Plugin Catalog — All Terraform UI Features
title: Version
id: version
key: "—"
category: utility
default_enabled: true
description: Display tfui version, platform info, and terraform provider selections
---

# Version

## Overview

Display tfui version, platform, and the resolved terraform binary version with provider selections. Useful for debugging environment issues and confirming which terraform is in use.

## Screenshot

![Version]({{ site.baseurl }}/assets/demo/version.gif)

## Interactive (TUI)

Type `:version` in command mode. The plugin is NavPush -- pressing `Esc` or `q` returns to the previous view.

### Keybindings

| Key | Action | Context |
|-----|--------|---------|
| `q` / `Esc` | Back to previous view | Always |

### Flow

```
:version → Version (display) → Esc → Previous view
```

## Command Line (CLI)

```bash
tfui version          # Text output
tfui version -json    # Structured JSON
```

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Always succeeds |

## Equivalence

| Goal | CLI | TUI |
|------|-----|-----|
| Show version info | `tfui version` | `:version` in command mode |
| Get version as JSON | `tfui version -json` | N/A (TUI is visual) |

## Configuration

```hcl
# tfui.hcl
plugin "version" {
  enabled = true
}
```

## Related

- [CLI Reference](../reference/cli-reference.md) -- full command documentation
