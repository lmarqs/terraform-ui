---
layout: default
parent: Plugin Catalog — All Terraform UI Features
title: "Terraform Console Inside a Terminal UI"
id: console
key: "~"
category: operations
default_enabled: true
seo:
  description: "Evaluate Terraform expressions with terraform console inside tfui, against the current chdir, without leaving the application."
description: "Describes the Console plugin, which evaluates terraform console expressions inside the TUI."
audience: "Users who want to check an expression's value against their state without leaving tfui"
---

# Console

## Overview

Evaluate terraform expressions inside the TUI. Each expression you enter runs in its own `terraform console` call against the current chdir, so each `Enter` pays terraform's startup time.

## Screenshot

![Console]({{ site.baseurl }}/assets/demo/console.gif)

## Interactive (TUI)

Press `~` from the home menu, type an expression and press `Enter`.

### Keybindings

| Key | Action | When |
|-----|--------|------|
| `Enter` | Evaluate expression | Input |
| `↑` / `↓` | Navigate history | Input |
| `Ctrl+U` | Clear current input | Input |
| `Esc` | Back to home | Not while evaluating |
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

Console has no tfui command. Run `terraform console` directly.

## Equivalence

| Goal | CLI | TUI |
|------|-----|-----|
| Evaluate expression | `terraform console` | `~` → type → `Enter` |

## Related

- [State Browser](state.md) -- browse resources visually instead of querying
- [Outputs](output.md) -- view output values without expressions
