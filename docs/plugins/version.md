---
layout: default
parent: Plugins
title: Version
id: version
key:
category: utility
default_enabled: true
description: Display tfui version, platform info, and terraform provider selections
---

## Overview

Display tfui version, platform, and the resolved terraform binary version with provider selections. Useful for debugging environment issues and confirming which terraform is in use.

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

| Code | Meaning |
|------|---------|
| 0 | Always succeeds |

## Configuration

```hcl
# tfui.hcl
plugin "version" {
  enabled = true
}
```

## Screenshots

```
tfui v0.1.0
on linux_amd64

terraform v1.14.9
on linux_amd64
+ provider registry.terraform.io/hashicorp/aws v5.0.0
+ provider registry.terraform.io/hashicorp/local v2.8.0
```

## Related

- [CLI Reference](../reference/cli-reference.md) -- full command documentation
