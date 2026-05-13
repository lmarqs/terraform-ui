---
layout: default
title: terraform-ui
description: Interactive terminal UI for Terraform operations
---

# terraform-ui

A k9s-style interactive terminal UI for Terraform. Plan, analyze risk, inspect blast radius, and apply — all from a keyboard-driven TUI.

## Install

### Homebrew

```bash
brew install lmarqs/tap/tfui
```

### Go install

```bash
go install github.com/lmarqs/terraform-ui/cmd/tfui@latest
```

### Binary download

Download the latest release from [GitHub Releases](https://github.com/lmarqs/terraform-ui/releases). Extract the binary and place it on your `PATH`.

```bash
# Example: Linux amd64
curl -sL https://github.com/lmarqs/terraform-ui/releases/latest/download/tfui_linux_amd64.tar.gz | tar xz
sudo mv tfui /usr/local/bin/
```

## Quick Start

```bash
# Launch interactive TUI
tfui

# Or use non-interactive mode (backward compatible)
tfui plan --dir ./infra
tfui apply --dir ./infra 
```

## Home Screen

```
┌─────────────────────────────────────────────────────────┐
│  terraform-ui                        workspace: default  │
├─────────────────────────────────────────────────────────┤
│                                                         │
│   [p] Plan          [r] Risk Analysis                   │
│   [a] Apply         [b] Blast Radius                    │
│   [s] State         [w] Workspaces                      │
│   [m] Projects      [?] Help                            │
│                                                         │
│   dir: ./infra                                          │
│   terraform: v1.14.0                                    │
│                                                         │
├─────────────────────────────────────────────────────────┤
│  q quit  / search  ? help                               │
└─────────────────────────────────────────────────────────┘
```

## Features

- **Interactive Plan Review** — Navigate changes, expand attribute diffs, see risk badges
- **Risk Analysis** — Automatic classification of changes as critical/high/medium/low
- **Blast Radius** — Visualize affected modules and resource dependencies
- **Live Apply** — Per-resource progress tracking with real-time status
- **State Browser** — Navigate and inspect terraform state resources
- **Workspace Management** — List, switch, and manage workspaces
- **Monorepo Support** — Discover and select projects via `tfui.hcl`
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
