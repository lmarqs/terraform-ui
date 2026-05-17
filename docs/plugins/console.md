---
layout: default
parent: Plugins
title: Console
id: console
key: "~"
category: operations
description: Interactive terraform console REPL session within the TUI
---

# Console (REPL)

Interactive terraform console session within the TUI.

## Interactive (TUI)

Press `~` from the home menu. The plugin launches an embedded `terraform console` session where you can evaluate expressions interactively.

### Keybindings

| Key | Action |
|-----|--------|
| `Enter` | Evaluate expression |
| `↑` / `↓` | Navigate history |
| `Ctrl+U` | Clear current input |
| `q` (when input empty) | Back to home |
| `Esc` | Back to home |

### Features

- Expression history with up/down navigation
- Results displayed inline below each expression
- Error messages shown with context
- Full keyboard capture (all printable characters available for expressions)
