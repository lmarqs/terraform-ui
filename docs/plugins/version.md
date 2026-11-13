---
layout: default
parent: Plugin Catalog — All Terraform UI Features
title: "tfui Version, Terraform Version and Providers"
id: version
key: "—"
category: utility
default_enabled: true
seo:
  description: "Show the tfui version and platform, the resolved terraform binary version and its provider selections, to confirm which terraform is in use."
description: "Describes what the Version plugin shows: tfui version, platform and provider selections."
audience: "Users filing a bug report, or checking which terraform binary tfui runs"
---

# Version

## Overview

Display tfui version, platform, and the resolved terraform binary version with provider selections. Useful for debugging environment issues and confirming which terraform is in use.

## Screenshot

![Version]({{ site.baseurl }}/assets/demo/version.gif)

## Interactive (TUI)

Type `:version` in command mode. `Esc` or `q` returns to where you were.

### Keybindings

| Key | Action | When |
|-----|--------|------|
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

Exit codes: see [CLI reference](../reference/cli-reference.md#exit-codes).

## Equivalence

| Goal | CLI | TUI |
|------|-----|-----|
| Show version info | `tfui version` | `:version` in command mode |
| Get version as JSON | `tfui version -json` | N/A (TUI is visual) |

## Related

- [CLI Reference](../reference/cli-reference.md) -- full command documentation
