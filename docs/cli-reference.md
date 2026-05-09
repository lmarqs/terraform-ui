---
layout: default
title: CLI Reference
description: Complete command-line reference for terraform-ui
---

# CLI Reference

## Usage

```
tfui [flags]
tfui [command] [flags]
```

## Commands

### `tfui` (no command)

Launches the interactive TUI. This is the default when no subcommand is given.

```bash
tfui                    # TUI in current directory
tfui --dir ./infra      # TUI scoped to specific directory
```

### `tfui plan`

Run terraform plan with animated terminal feedback.

```bash
tfui plan --dir ./infra
tfui plan --dir ./infra --mode agent | jq .
tfui plan --dir ./infra --target aws_instance.web
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--dir` | `.` | Working directory |
| `--mode` | `progress` | UI mode: `silent`, `spinner`, `progress`, `agent` |
| `--target` | | Resource target (repeatable) |
| `--terraform-bin` | `terraform` | Path to terraform binary |

### `tfui apply`

Run terraform apply with animated terminal feedback.

```bash
tfui apply --dir ./infra
tfui apply --dir ./infra --mode spinner
```

**Flags:** Same as `plan`.

### `tfui version`

Print the version.

```bash
tfui version
```

## Modes

| Mode | Description |
|------|-------------|
| `silent` | No UI, plain output |
| `spinner` | One-line animated spinner |
| `progress` | Two-line: spinner + progress bar |
| `agent` | Structured JSON output for automation |

## Environment Variables

terraform-ui respects standard terraform environment variables:

- `TF_CLI_ARGS_plan` — Extra arguments for terraform plan
- `TF_CLI_ARGS_apply` — Extra arguments for terraform apply
- `TF_WORKSPACE` — Override workspace selection

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success (no changes for plan, or apply succeeded) |
| 1 | Error (terraform failed, invalid config, etc.) |
| 2 | Plan has changes (useful for CI: changes detected) |
