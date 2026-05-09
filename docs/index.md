---
layout: default
title: terraform-ui
description: Interactive terminal UI for Terraform operations
---

# terraform-ui

A k9s-style interactive terminal UI for Terraform. Plan, analyze risk, inspect blast radius, and apply — all from a keyboard-driven TUI.

## Quick Start

```bash
# Install
go install github.com/lmarqs/terraform-ui/cmd/tfui@latest

# Launch interactive TUI
tfui

# Or use non-interactive mode (backward compatible)
tfui plan --dir ./infra
tfui apply --dir ./infra --mode progress
```

## Features

- **Interactive Plan Review** — Navigate changes, expand attribute diffs, see risk badges
- **Risk Analysis** — Automatic classification of changes as critical/high/medium/low
- **Blast Radius** — Visualize affected modules and resource dependencies
- **Live Apply** — Per-resource progress tracking with real-time status
- **State Browser** — Navigate and inspect terraform state resources
- **Workspace Management** — List, switch, and manage workspaces
- **Monorepo Support** — Discover and select projects via `tfui.yaml`
- **Phantom Change Detection** — Identify no-op changes that terraform incorrectly reports

## How It Works

```
tfui                          → interactive TUI (default)
tfui --dir ./infra            → TUI scoped to directory
tfui plan --dir ./infra       → non-interactive plan
tfui apply --dir ./infra      → non-interactive apply
```

Bare `tfui` opens the full-screen TUI. Subcommands (`plan`, `apply`) run in non-interactive mode with animated terminal feedback — spinner, progress bar, and tree-view diff output.

## Navigation

| Key | Action |
|-----|--------|
| `p` | Run plan |
| `r` | Risk analysis |
| `b` | Blast radius |
| `a` | Apply |
| `s` | State browser |
| `w` | Workspaces |
| `m` | Projects (monorepo) |
| `/` | Search |
| `?` | Help |
| `q` | Quit / back |
| `↑↓` or `jk` | Navigate |
| `Enter` | Select |
| `Esc` | Back to home |
