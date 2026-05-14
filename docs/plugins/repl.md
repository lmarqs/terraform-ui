---
layout: plugin
title: Console
id: repl
key: t
category: operations
---

# Console (REPL)

Interactive terraform console session within the TUI.

## Interactive (TUI)

Press `t` from the home menu. The plugin launches an embedded `terraform console` session where you can evaluate expressions interactively.

### Keybindings

| Key | Action |
|-----|--------|
| `Enter` | Evaluate expression |
| `↑` / `↓` | Navigate history |
| `Esc` | Clear current input |
| `q` (when input empty) | Back to home |

### Features

- Expression history with up/down navigation
- Results displayed inline below each expression
- Error messages shown with context
- Uses `tea.ExecProcess` for proper terminal handoff to terraform console
