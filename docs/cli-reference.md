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

## Design Principle

tfui is a superset of terraform. All terraform flags work identically. Additions use names terraform hasn't claimed. See [CLI I/O Contract](cli-io-contract.md) for the full stdin/stdout/stderr specification.

## Commands

### `tfui` (no command)

Launches the interactive TUI. This is the default when no subcommand is given.

```bash
tfui                           # TUI in current directory
tfui --project ./infra         # TUI scoped to specific directory
tfui --plan ./tfplan.out       # TUI with pre-computed plan (binary)
tfui --state ./terraform.tfstate  # TUI with pre-loaded state
```

### `tfui plan`

Run terraform plan. Produces a tree view on stdout and saves the binary plan file.

```bash
tfui plan
tfui plan -out=tfplan.out
tfui plan -target=aws_instance.web
tfui plan -json                     # NDJSON events (terraform-compatible)
tfui plan --ci                      # suppress spinner
```

| stdout | stderr | Exit |
|--------|--------|------|
| Tree view (default) or NDJSON (`-json`) | Spinner (if TTY, unless `--ci`) | 0/1/2 |

### `tfui apply`

Run terraform apply.

```bash
tfui apply tfplan.out
tfui apply -json tfplan.out         # NDJSON events (terraform-compatible)
tfui apply --ci
```

| stdout | stderr | Exit |
|--------|--------|------|
| Apply summary (default) or NDJSON (`-json`) | Spinner (if TTY, unless `--ci`) | 0/1 |

### `tfui state`

State management operations.

```bash
tfui state rm <address>             # remove from state
tfui state mv <source> <dest>       # rename in state
tfui state import <address> <id>    # import existing resource
tfui state taint <address>          # mark for recreation
tfui state untaint <address>        # remove taint mark
```

### `tfui validate`

Run terraform validate.

```bash
tfui validate                       # enriched diagnostics
tfui validate -json                 # JSON diagnostics (terraform-compatible)
```

### `tfui output`

Show terraform outputs.

```bash
tfui output                         # human-readable
tfui output -json                   # JSON (terraform-compatible)
tfui output <name>                  # single value
```

### `tfui scaffold`

Generate a `tfui.hcl` configuration file by detecting terraform project patterns.

```bash
tfui scaffold              # interactive
tfui scaffold --yes        # non-interactive (accept defaults)
tfui scaffold --force      # overwrite existing
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
| `--chdir` | | Select chdir member (validated in project mode) |
| `--config` | | Override config values (repeatable, `key=value`) |
| `--debug` | `false` | Enable debug logging to `~/.tfui/logs/` |

## Additive Flags (tfui-only)

| Flag | Default | Description |
|------|---------|-------------|
| `--ci` | `false` | Suppress stderr spinner/progress |

## TUI Flags

| Flag | Description |
|------|-------------|
| `--plan` | Load binary plan file into TUI (review and apply) |
| `--state` | Load state file into TUI (view and mutate via `-state=`) |
| `--macro` | Run tape file (requires `--plan` or `--state`) |

### `--plan` behavior

Accepts binary plan files (output of `terraform plan -out=`):

```bash
tfui --plan ./tfplan.out            # review AND apply
terraform show -json tfplan.out | tfui --plan -   # stdin: view-only (can't apply)
```

- `Plan()` → `terraform show -json <file>` for display
- `Apply()` → `terraform apply <file>` directly
- Stdin source → cached, view-only, non-refreshable

### `--state` behavior

Accepts state files:

```bash
tfui --state ./terraform.tfstate    # view and mutate
terraform state pull | tfui --state -   # stdin: view-only
```

- `StateList()`/`Show()` → re-read file on each call
- Mutations → `terraform state rm -state=<file>` (delegates with `-state=` flag)
- Refresh → re-reads file from disk (catches external changes)

### URI Resolution Rules

| Input | Resolved as |
|-------|-------------|
| `-` | stdin |
| `/absolute/path` | absolute local path |
| `./relative/path` | relative to CWD |
| `../parent/path` | relative to CWD |
| `file:///path` | local path (scheme stripped) |
| `bare-name` | **ERROR** — suggests `./bare-name` |

Constraint: only one flag can use `-` (stdin) per invocation.

## Macro Mode (`--macro`)

Macros are command generators, never executors. They record what terraform would run and output commands to stdout:

```bash
tfui --macro deploy.tape --plan ./tfplan.out            # inspect commands
tfui --macro deploy.tape --plan ./tfplan.out | sh       # user opts in to execute
```

See [Macro Language](macro-language.md) for the DSL reference.

### Macro Exit Codes

| Code | Meaning |
|------|---------|
| 0 | All assertions passed |
| 1 | Assertion failure |
| 2 | Syntax error in tape file |
| 3 | Timeout waiting for condition |

## Exit Codes (CLI)

| Code | Meaning |
|------|---------|
| 0 | Success (no changes for plan, or apply succeeded) |
| 1 | Error |
| 2 | Plan has changes (terraform-compatible) |
