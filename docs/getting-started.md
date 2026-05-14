---
layout: default
title: Getting Started
description: Quick tutorial to get up and running with terraform-ui
---

# Getting Started

## Installation

Pick one:

```bash
# Homebrew
brew install lmarqs/tap/tfui

# Go install
go install github.com/lmarqs/terraform-ui/cmd/tfui@latest

# Or download binary from GitHub Releases
```

Requires terraform (or tofu) on your `PATH`.

### For contributors

```bash
git clone https://github.com/lmarqs/terraform-ui.git
cd terraform-ui
mise install
mise run go:build
./dist/tfui version
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
tfui plan --project ./infra

# Plan with JSON output (terraform-compatible NDJSON)
tfui plan --project ./infra -json | jq .

# Apply (suppress spinner for CI)
tfui apply --project ./infra --ci

# Silent (no UI, just exit code)
tfui plan --project ./infra --ci
```

Exit code `2` means "changes detected" — useful for CI gates.

## Configuring a Monorepo

Create a `tfui.hcl` in your repository root:

```yaml
# tfui.hcl
projects:
  paths:
    - "envs/*"
    - "modules/*"
    - "services/*/terraform"
```

With this config, pressing `m` in the TUI shows a project picker. Each matched directory containing `.tf` files becomes a selectable project.

See [Configuration](configuration.md) for all options.
