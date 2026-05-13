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
tfui --project ./infra  # TUI scoped to specific directory
```

### `tfui plan`

Run terraform plan with animated terminal feedback.

```bash
tfui plan --project ./infra
tfui plan --project ./infra --output json | jq .
tfui plan --project ./infra --target aws_instance.web
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--ci` | `false` | Suppress spinner (CI-friendly) |
| `--output` | `text` | Output format: `text`, `json` |
| `--target` | | Resource target (repeatable) |

### `tfui apply`

Run terraform apply with animated terminal feedback.

```bash
tfui apply --project ./infra
tfui apply --project ./infra --ci
```

**Flags:** Same as `plan`.

### `tfui init`

Generate a `tfui.hcl` configuration file by detecting terraform project patterns.

```bash
tfui init
```

### `tfui version`

Print the version.

```bash
tfui version
```

## Global Flags

Available on all commands:

| Flag | Default | Description |
|------|---------|-------------|
| `--project` | `.` | Project root directory (where tfui.hcl lives) |
| `--terraform-bin` | `terraform` | Path to terraform/tofu/terragrunt binary |
| `--config` | | Override config values (repeatable, `key=value`) |
| `--debug` | `false` | Enable debug logging to `~/.tfui/logs/` |

## Read-Only Mode (`--plan`, `--state`)

Load pre-computed plan/state data without running terraform. Opens the TUI in read-only mode where mutating operations are disabled.

```bash
# Local files (explicit path required: ./ or / prefix)
tfui --plan ./plan.json
tfui --state ./terraform.tfstate
tfui --plan ./plan.json --state ./state.json

# Stdin (pipe from terraform or curl)
terraform show -json tfplan.out | tfui --plan -
terraform state pull | tfui --state -

# Absolute paths
tfui --plan /ci/artifacts/plan.json

# file:// scheme
tfui --plan file:///absolute/path/plan.json
```

| Flag | Description |
|------|-------------|
| `--plan` | Load plan from URI |
| `--state` | Load state from URI |

### URI Resolution Rules

URIs must be explicit — no bare filenames:

| Input | Resolved as |
|-------|-------------|
| `-` | stdin |
| `/absolute/path.json` | absolute local path |
| `./relative/path.json` | relative to CWD |
| `../parent/path.json` | relative to CWD |
| `file:///path.json` | local path (scheme stripped) |
| `s3://bucket/key.json` | S3 (requires provider, future) |
| `https://host/path.json` | HTTP (requires provider, future) |
| `plan.json` | **ERROR** — ambiguous, suggests `./plan.json` |

### Constraints

- Only one flag can use `-` (stdin) per invocation
- Plan files must be JSON format (output of `terraform show -json <planfile>`)
- Binary `.tfplan` files are not supported directly — convert first with `terraform show -json`
- State files must be JSON format (`.tfstate` or output of `terraform state pull`)

### Read-Only Behavior

When `--plan` or `--state` is provided (interactive TUI mode):

- Header displays `[read-only]` badge
- Workspace reported as `"readonly"`
- Actions show the equivalent terraform command as an error message
- Risk classification and phantom detection still run on loaded plan data

When combined with `--macro`, all operations are recorded and printed to stdout (see [Macro Mode](#macro-mode-macro)).

## Macro Mode (`--macro`)

Run automated TUI interactions from a tape file. Mutating terraform operations triggered during playback are printed to stdout. See [Macro Language](macro-language.md) for the full DSL reference.

```bash
# From file
tfui --plan ./plan.json --macro ./scripts/verify-plan.tape

# From stdin
echo "wait ready; key p; assert view aws_instance" | tfui --plan ./plan.json --macro -

# Pipe commands to shell (apply via macro)
tfui --plan ./plan.json --state ./state.json --macro ./deploy.tape | sh

# Use tofu binary
tfui --plan ./plan.json --state ./state.json --macro ./deploy.tape --terraform-bin tofu | sh
```

| Flag | Description |
|------|-------------|
| `--macro` | Run tape file (path or `-` for stdin). Requires `--plan` or `--state`. |

### Stdout Output

Every terraform operation triggered during macro playback is printed to stdout after completion:

```bash
$ tfui --plan ./plan.json --state ./state.json --macro ./taint.tape
terraform workspace show
terraform state list
terraform taint aws_instance.web

$ tfui --plan ./plan.json --state ./state.json --macro ./apply.tape
terraform workspace show
terraform plan
terraform apply -target=aws_instance.web
```

### Exit Codes (macro mode)

| Code | Meaning |
|------|---------|
| 0 | All assertions passed |
| 1 | Assertion failure |
| 2 | Syntax error in tape file |
| 3 | Timeout waiting for condition |

## Environment Variables

terraform-ui respects standard terraform environment variables:

- `TF_CLI_ARGS_plan` — Extra arguments for terraform plan
- `TF_CLI_ARGS_apply` — Extra arguments for terraform apply
- `TF_WORKSPACE` — Override workspace selection

## Exit Codes (TUI/plan/apply)

| Code | Meaning |
|------|---------|
| 0 | Success (no changes for plan, or apply succeeded) |
| 1 | Error (terraform failed, invalid config, etc.) |
| 2 | Plan has changes (useful for CI: changes detected) |
