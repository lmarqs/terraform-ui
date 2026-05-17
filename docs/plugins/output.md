---
layout: default
parent: Plugins
title: Outputs
id: output
key: o
category: navigation
description: View terraform output values for the current workspace
---

# Outputs

View terraform output values for the current workspace.

## Interactive (TUI)

Press `o` from the home menu. The plugin fetches outputs via `terraform output -json` and displays them in a filterable list.

### Keybindings

| Key | Action |
|-----|--------|
| `Enter` / `i` | Inspect output value (expanded JSON) |
| `/` | Filter outputs by name |
| `ctrl+r` | Refresh outputs |
| `q` | Back to home |

### Features

- Fuzzy filter across output names
- Inspect frame shows full JSON value with type information
- Sensitive outputs marked but values hidden
