---
layout: default
parent: Plugins
title: Console
id: console
key: "~"
category: operations
default_enabled: true
description: Interactive terraform console REPL session within the TUI
---

## Overview

Interactive terraform console session within the TUI. Evaluate terraform expressions against the current state without leaving the application.

## Interactive (TUI)

Press `~` from the home menu. The plugin launches an embedded `terraform console` session where you can evaluate expressions interactively.

### Keybindings

| Key | Action | Context |
|-----|--------|---------|
| `Enter` | Evaluate expression | Input |
| `↑` / `↓` | Navigate history | Input |
| `Ctrl+U` | Clear current input | Input |
| `Esc` | Back to home | Always |
| `q` | Back to home (when input empty) | Empty input |

### Flow

```
Home ──~──→ Console (REPL)
               │
               ├── type expression → Enter → evaluate → show result
               ├── ↑/↓ → recall previous expressions
               └── Esc → Home
```

## Command Line (CLI)

```bash
tfui console --project ./infra
```

The CLI mode hands off to `terraform console` directly with proper terminal handling.

## Configuration

```hcl
# tfui.hcl
plugin "console" {
  enabled = true
}
```

## Screenshots

```
Console

> aws_instance.web.id
"i-0abc123def456"

> length(aws_instance.web.tags)
3

> formatdate("YYYY-MM-DD", timestamp())
"2026-05-16"

>
```

## Related

- [State Browser](state.md) -- browse resources visually instead of querying
- [Outputs](output.md) -- view output values without expressions
