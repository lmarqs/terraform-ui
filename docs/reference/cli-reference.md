---
layout: default
title: CLI Reference
parent: Reference
nav_order: 1
description: Complete command-line reference for terraform-ui
---

# CLI Reference

## Usage

```
tfui [flags]
tfui [command] [flags]
```

## Execution Model

Every `tfui <command>` launches the plugin in a standalone TUI (rendered on stderr). On exit, structured output goes to stdout. Use `-ci` or `CI=1` for headless mode.

```
tfui plan           → Standalone TUI on stderr, tree view to stdout on exit
tfui plan -ci      → No TUI, tree view to stdout immediately
tfui plan -json     → Standalone TUI on stderr, JSON to stdout on exit
tfui                → Full multi-plugin TUI (alt-screen on stdout, no output)
```

See [CLI I/O Contract](cli-io-contract.md) for the full stdin/stdout/stderr specification.

## Commands

### `tfui` (no command)

Launches the full interactive TUI with all plugins, home screen, and inter-plugin navigation.

```bash
tfui                           # Full TUI in current directory
tfui -project ./infra         # Full TUI scoped to specific directory
tfui -plan ./tfplan.out       # Full TUI with pre-computed plan
tfui -state ./terraform.tfstate  # Full TUI with pre-loaded state
```

### `tfui plan`

Run terraform plan. Opens the plan plugin TUI for interactive review.

```bash
tfui plan                           # TUI: review plan interactively, tree view on exit
tfui plan -json                     # TUI: review plan, JSON output on exit
tfui plan -ci                      # No TUI: tree view to stdout immediately
tfui plan -ci -json                # No TUI: JSON to stdout immediately
tfui plan -target=aws_instance.web  # Targeted plan
tfui plan -out=tfplan.out           # Save binary plan file
```

| Mode | stdout (on exit) | stderr | Exit |
|------|-----------------|--------|------|
| Standalone | Tree view or JSON | TUI (alt-screen) | 0/2 |
| CI | Tree view or JSON | — | 0/2 |

Exit code 2 = changes present (terraform-compatible).

### `tfui apply`

Run terraform apply. Opens the apply plugin TUI.

```bash
tfui apply                          # TUI: shows confirmation, then progress
tfui apply -auto-approve           # TUI: skips confirmation, shows progress
tfui apply -ci                     # No TUI: apply immediately
tfui apply -json                    # TUI: JSON output on exit
tfui apply -target=aws_instance.web # Targeted apply
```

| Mode | stdout (on exit) | stderr | Exit |
|------|-----------------|--------|------|
| Standalone | "Apply complete." or JSON | TUI (alt-screen) | 0/1 |
| CI | "Apply complete." or JSON | — | 0/1 |

### `tfui state`

State browser and operations. Opens the state plugin TUI.

```bash
tfui state                          # TUI: browse resources interactively
tfui state -ci                     # No TUI: addresses to stdout
tfui state -json                    # TUI: JSON output on exit
```

| Mode | stdout (on exit) | stderr | Exit |
|------|-----------------|--------|------|
| Standalone | Addresses (one/line) or JSON | TUI (alt-screen) | 0 |
| CI | Addresses (one/line) or JSON | — | 0 |

### `tfui validate`

Run terraform validate. Opens the validate plugin TUI.

```bash
tfui validate                       # TUI: review diagnostics interactively
tfui validate -ci                  # No TUI: diagnostics to stdout
tfui validate -ci -json            # No TUI: JSON diagnostics to stdout
```

| Mode | stdout (on exit) | stderr | Exit |
|------|-----------------|--------|------|
| Standalone | Diagnostics text or JSON | TUI (alt-screen) | 0/1 |
| CI | Diagnostics text or JSON | — | 0/1 |

Exit code 1 = validation errors present.

### `tfui output`

Show terraform outputs. Opens the output plugin TUI.

```bash
tfui output                         # TUI: browse outputs interactively
tfui output -ci                    # No TUI: key=value pairs to stdout
tfui output -ci -json              # No TUI: JSON to stdout
```

### `tfui init`

Run terraform init. Opens the init plugin TUI (form + progress).

```bash
tfui init                           # TUI: form for options, shows progress
tfui init -ci                      # No TUI: runs init immediately
```

### `tfui version`

Show version information. Opens the version plugin TUI.

```bash
tfui version                        # TUI: shows version info
tfui version -ci                   # No TUI: version text to stdout
tfui version -ci -json             # No TUI: version JSON to stdout
```

### `tfui workspace`

Workspace management operations (imperative, no TUI).

```bash
tfui workspace show                    # print current workspace name
tfui workspace list                    # list all workspaces
tfui workspace select <name>           # switch to workspace
tfui workspace new <name>              # create and switch to workspace
tfui workspace new <name> -lock=false  # create without locking state
tfui workspace delete <name>           # delete workspace
tfui workspace delete <name> -force    # delete non-empty workspace
```

| Flag | Applies to | Description |
|------|-----------|-------------|
| `-lock` | `new`, `delete` | Lock state during operation (default: true) |
| `-lock-timeout` | `new`, `delete` | Duration to wait for a state lock |
| `-force` | `delete` | Force deletion of a non-empty workspace |

### `tfui force-unlock`

Remove a terraform state lock (imperative, no TUI).

```bash
tfui force-unlock <lock-id>          # interactive confirmation
tfui force-unlock -force <lock-id>  # skip confirmation (CI/scripts)
```

### `tfui scaffold`

Generate a `tfui.hcl` configuration file by detecting terraform project patterns.

```bash
tfui scaffold              # interactive wizard
tfui scaffold -yes        # non-interactive (accept defaults)
tfui scaffold -force      # overwrite existing
```

## Global Flags

Available on all commands:

| Flag | Default | Description |
|------|---------|-------------|
| `-project` | `.` | Project root directory (where tfui.hcl lives) |
| `-terraform-bin` | `terraform` | Path to terraform/tofu/terragrunt binary |
| `-chdir` | | Select chdir member (validated in project mode) |
| `-config` | | Override config values (repeatable, `key=value`) |

## Mode Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-ci` | `false` | Disable TUI, output directly to stdout |
| `-json` | `false` | Output in JSON format |

`-ci` is also triggered by `CI=1` environment variable or stderr not being a TTY.

## Data Flags (available on all commands)

| Flag | Description |
|------|-------------|
| `-plan` | Pre-seed plan data from file or stdin |
| `-state` | Pre-seed state data from file or stdin |

These flags work on any command. When provided, the plugin reads from the pre-seeded cache instead of executing terraform:

```bash
tfui -plan ./tfplan.out            # full TUI with pre-seeded plan
tfui plan -plan ./tfplan.out       # standalone plan TUI, reviews existing data
tfui state -state ./state.json     # standalone state TUI, browses pre-loaded state
tfui plan -ci -plan ./tfplan.out  # CI mode, outputs tree from pre-seeded plan
```

## Macro Flag

| Flag | Description |
|------|-------------|
| `-macro` | Run tape file (headless TUI recording) |
| `-record` | Capture session frames to directory |

`-macro` is available on all commands. On the root command it drives the full multi-plugin TUI headlessly; on subcommands it drives the standalone plugin headlessly and outputs recorded commands to stdout.

`-record` is orthogonal to `-macro`. It captures ANSI frames + timing metadata to a directory, and enables debug logging (written as `debug-*.log` in the same directory). Combined with `-macro`, it enables deterministic GIF generation from tapes. Without `-macro`, it records interactive sessions and generates a replayable tape.

### `-plan` behavior

Accepts binary plan files (output of `terraform plan -out=`):

```bash
tfui -plan ./tfplan.out            # review AND apply
terraform show -json tfplan.out | tfui -plan -   # stdin: view-only (can't apply)
```

### `-state` behavior

Accepts state files:

```bash
tfui -state ./terraform.tfstate    # view and mutate
terraform state pull | tfui -state -   # stdin: view-only
```

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

## Macro Mode (`-macro`)

Macros are command generators, never executors. They record what terraform would run and output commands to stdout.

`-macro` works on both the root command and subcommands. On root, it drives the full TUI headlessly for multi-plugin recording. On subcommands, it drives the standalone plugin headlessly with the macro service (recording commands without execution).

```bash
tfui -macro deploy.tape -plan ./tfplan.out            # inspect commands
tfui -macro deploy.tape -plan ./tfplan.out | sh       # user opts in to execute
```

See [Macro Language](macro-language.md) for the DSL reference.

### Macro Exit Codes

| Code | Meaning |
|------|---------|
| 0 | All assertions passed |
| 1 | Assertion failure |
| 2 | Syntax error in tape file |
| 3 | Timeout waiting for condition |

## Exit Codes

| Code | Meaning | Scope |
|------|---------|-------|
| 0 | Success (no changes for plan, or operation completed) | All |
| 1 | Error / validation failure | All |
| 2 | Plan has changes (terraform-compatible) | `plan` only |
