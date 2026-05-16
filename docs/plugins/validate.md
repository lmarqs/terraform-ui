---
layout: default
parent: Plugins
title: Validate
id: validate
key: v
category: operations
---

# Validate

Run `terraform validate` and display diagnostics.

## Interactive (TUI)

Press `v` from the home menu. The plugin runs validation and shows errors/warnings in an expandable list.

### Keybindings

| Key | Action |
|-----|--------|
| `Enter` | Expand diagnostic detail |
| `e` | Open file at error location in $EDITOR |
| `ctrl+r` | Re-run validation |
| `q` | Back to home |

### Features

- Diagnostics grouped by severity (error, warning)
- Expandable detail shows source location and suggestion
- Editor integration jumps to exact file:line of the diagnostic
