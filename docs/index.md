---
layout: default
title: Home
nav_order: 1
description: Interactive terminal UI for Terraform operations
permalink: /
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

# Or use non-interactive mode
tfui plan --project ./infra
tfui apply --project ./infra
```

## Home Screen

```
┌─────────────────────────────────────────────────────────┐
│  terraform-ui                        workspace: default │
├─────────────────────────────────────────────────────────┤
│                                                         │
│   [p] Plan          [R] Risk Analysis                   │
│   [s] State         [P] Phantom Changes                 │
│   [w] Workspaces    [B] Blast Radius                    │
│   [o] Outputs       [~] Console                         │
│   [v] Validate      [i] Init                            │
│   [C] Context                                           │
│                                                         │
├─────────────────────────────────────────────────────────┤
│  q quit  / filter  : command                            │
└─────────────────────────────────────────────────────────┘
```

## Features

- **Interactive Plan Review** — Navigate changes, expand attribute diffs, see risk badges
- **Risk Analysis** — Automatic classification of changes as critical/high/medium/low
- **Blast Radius** — Visualize affected modules and resource dependencies
- **Live Apply** — Per-resource progress tracking with real-time status
- **State Browser** — Navigate and inspect terraform state resources
- **Workspace Management** — List, switch, and manage workspaces
- **Monorepo Support** — Discover and select chdir members via `tfui.hcl`
- **Phantom Change Detection** — Identify no-op changes that terraform incorrectly reports

## How It Works

```
tfui                              → interactive TUI (default)
tfui --project ./infra            → TUI scoped to project directory
tfui plan --project ./infra       → non-interactive plan
tfui apply --project ./infra      → non-interactive apply
```

Bare `tfui` opens the full-screen TUI. Subcommands (`plan`, `apply`) run in non-interactive mode with animated terminal feedback — spinner, progress bar, and tree-view diff output.

## Navigation

| Key | Action |
|-----|--------|
| `p` | Plan |
| `a` | Apply |
| `s` | State browser |
| `w` | Workspaces |
| `o` | Outputs |
| `v` | Validate |
| `~` | Console (REPL) |
| `i` | Init |
| `R` | Risk analysis |
| `P` | Phantom changes |
| `B` | Blast radius |
| `C` | Context (project/chdir/workspace) |
| `/` | Filter |
| `:` | Command mode (`:q` quit — guarded during ops, `:q!` force quit) |
| `q` | Quit / back |
| `↑↓` or `jk` | Navigate |
| `Enter` | Inspect / expand |
| `Space` | Pin (target for plan/apply) |

## Documentation

- [Getting Started](getting-started.md) — Installation and first run
- [Configuration](configuration.md) — `tfui.hcl` reference
- [CLI Reference](cli-reference.md) — All commands and flags
- [Architecture](architecture.md) — Internal design
- [Plugins](plugins/) — Plugin catalog and docs
- [Macro Language](macro-language.md) — Tape DSL for automated testing
- [Testing](testing.md) — Test strategy and patterns
- [CLI I/O Contract](cli-io-contract.md) — stdin/stdout/stderr specification
- [TUI UX Spec](tui-ux.md) — Layout, navigation, and interaction patterns
- [CLI UX Spec](cli-ux.md) — Non-interactive command design
- [Roadmap](roadmap.md) — Planned features and initiatives

### Feature Deep Dives

- [Risk Analysis](risk-analysis.md) — Risk classification methodology
- [Phantom Changes](phantom-changes.md) — No-op change detection
- [Blast Radius](blast-radius.md) — Module impact visualization
- [Demo](demo.md) — Animated demos of key workflows
- [Architecture Decision Records](adr/) — Design decisions and rationale
