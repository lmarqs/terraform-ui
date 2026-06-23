---
layout: default
title: Install tfui — Terraform TUI Quick Start
parent: Guides — Getting Started with Terraform UI
nav_order: 1
description: Install terraform-ui via Homebrew, go install, or binary download. Review your first terraform plan in under 60 seconds.
---

# Getting Started

## Installation

Pick one:

```bash
# Homebrew
brew install lmarqs/tap/tfui

# Go install
go install github.com/lmarqs/terraform-ui/cmd/tfui@latest

# mise
mise use github:lmarqs/terraform-ui

# Or download binary from GitHub Releases
```

Requires terraform (or tofu) on your `PATH`.

### For contributors

```bash
git clone https://github.com/lmarqs/terraform-ui.git
cd terraform-ui
mise install
mise run build
./dist/tfui_darwin_amd64_v1/tfui version
./dist/tfui_darwin_arm64_v8.0/tfui version
./dist/tfui_linux_amd64_v1/tfui version
./dist/tfui_linux_arm64_v8.0/tfui version
```

## First Run

Navigate to any directory containing `.tf` files and run:

```bash
tfui
```

This opens the interactive TUI. You'll see the home screen with keybindings for each operation.

## Running a Plan Interactively

1. Press `p` from the home screen
2. terraform-ui runs `terraform plan` and parses the output
3. Results appear as a navigable tree — expand resources with `Enter`
4. Risk badges show next to each change (critical/high/medium/low)
5. Press `q` to return to the home screen

## Non-Interactive Mode for CI

Use subcommands for scripts and CI pipelines:

```bash
# Plan with progress bar
tfui plan -project ./infra

# Plan with JSON output (terraform-compatible NDJSON)
tfui plan -project ./infra -json | jq .

# Apply (suppress spinner for CI)
tfui apply -project ./infra -ci

# Silent (no UI, just exit code)
tfui plan -project ./infra -ci
```

Exit code `2` means "changes detected" — useful for CI gates.

## Configuring a Monorepo

Create a `tfui.hcl` in your repository root:

```hcl
member "envs/prod" {}
member "envs/staging" {}
member "modules/vpc" {}
member "services/api/terraform" {}
```

With this config, pressing `C` in the TUI opens the context manager where you can select a chdir member. Each directory listed must contain `.tf` files.

See [Configuration](configuration.md) for all options.

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

## Features

- **Interactive Plan Review** — Navigate changes, expand attribute diffs, see risk badges
- **[Risk Analysis](../features/risk-analysis.md)** — Automatic classification of changes as critical/high/medium/low
- **[Blast Radius](../features/blast-radius.md)** — Visualize affected modules and resource dependencies
- **Live Apply** — Per-resource progress tracking with real-time status
- **State Browser** — Navigate and inspect terraform state resources
- **Workspace Management** — List, switch, and manage workspaces
- **Monorepo Support** — Discover and select chdir members via `tfui.hcl`
- **[Phantom Change Detection](../features/phantom-changes.md)** — Identify no-op changes that terraform incorrectly reports
