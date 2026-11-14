---
layout: default
title: CLI Reference — tfui Commands and Flags
parent: Reference
nav_order: 1
seo:
  description: Complete command-line reference for tfui. All commands, flags, environment variables, and exit codes.
description: "Lists every tfui command and flag, with the environment variables and exit codes that change its behavior."
audience: "Users scripting tfui, and contributors changing the CLI"
---

# CLI Reference

## Usage

```
tfui [flags]
tfui [command] [flags]
```

Each command opens its plugin standalone, with the TUI on stderr, or runs headless. The
[CLI I/O contract](cli-io-contract.md) says which mode a run takes, what each command writes to stdout and stderr,
and the exit codes.

## Commands

Risk, phantom and blast radius are TUI plugins (`R`, `P`, `B`) with no command of their own.

### `tfui` (no command)

Launches the full interactive TUI with all plugins, home screen, and inter-plugin navigation.

```bash
tfui                           # Full TUI in current directory
tfui -project ./infra         # Full TUI rooted at ./infra
tfui -plan ./plan.json        # Full TUI with a pre-seeded plan (see Pre-seeded data)
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
tfui plan -target=a -target=b        # Repeat the flag per address
tfui plan -lock=false               # Skip state locking for this run
tfui plan -lock-timeout=30s         # Retry a held lock for 30s
```

`-target` is repeatable and each value reaches terraform verbatim, so quoted
index keys work as typed: `-target='module.a["x y"]'`. There is no
comma-separated form — a comma is part of an address, not a separator.

### `tfui apply`

Run terraform apply. Opens the plan view, where `a` applies the reviewed plan.

```bash
tfui apply                          # TUI: plan view, then confirmation, then progress
tfui apply -auto-approve           # TUI: skips plan review + confirmation, shows progress
tfui apply -ci -auto-approve       # No TUI: apply immediately
tfui apply -json                    # TUI: JSON output on exit
tfui apply -target=aws_instance.web # Plan view scoped to this address, then apply that plan
tfui apply -target=a -target=b      # Repeat the flag per address
tfui apply -lock=false              # Skip state locking for this run
tfui apply -lock-timeout=30s        # Retry a held lock for 30s
```

`-lock` and `-lock-timeout` apply to the plan and the apply alike, and
override `defaults { lock }` / `defaults { lock_timeout }` from `tfui.hcl` for
that run only. Omitting them keeps whatever the config resolved.

Standalone `tfui apply` is the TUI's plan → apply flow without the header: it
runs a plan and opens the plan view with the full diff. `a` applies that saved
plan after a confirmation; the session ends when the user leaves apply, after
it finishes or by declining the confirmation. `-auto-approve` skips the review and the prompt, and
terraform plans and applies in one run.

A [headless](cli-io-contract.md#mode-resolution) apply requires `-auto-approve`,
as `terraform apply` does. With no TTY to
approve from, tfui reproduces terraform's own error, "apply not allowed for
non-interactive use" (exit 1), rather than applying unconfirmed. Under
`-macro` the guard does not apply, because nothing runs.

### `tfui state`

State browser and operations. Opens the state plugin TUI.

```bash
tfui state                          # TUI: browse resources interactively
tfui state -ci                     # No TUI: addresses to stdout
tfui state -json                    # TUI: JSON output on exit
tfui state rm <address>...          # No TUI: terraform state rm, per address
tfui state mv <source> <dest>       # No TUI: terraform state mv
tfui state taint <address>...       # No TUI: terraform taint, per address
tfui state untaint <address>...     # No TUI: terraform untaint, per address
```

The `rm`, `mv`, `taint` and `untaint` subcommands run at once, with no TUI.

### `tfui taint`, `tfui untaint`

Mark resources for replacement, or clear the mark. Opens the taint or untaint
plugin TUI, which asks for confirmation.

```bash
tfui taint <address>...            # TUI: confirm, then terraform taint
tfui taint aws_instance.web -ci    # No TUI: taints at once
tfui untaint <address>...          # TUI: confirm, then terraform untaint
```

Headless, they skip the confirmation ([what headless changes](cli-io-contract.md#what-headless-changes)).

### `tfui import`

Import existing infrastructure into state. Opens the import plugin TUI, which
asks for confirmation.

```bash
tfui import <address> <id>         # TUI: confirm, then terraform import
tfui import <address> <id> -ci     # No TUI: imports at once
```

Headless, it skips the confirmation ([what headless changes](cli-io-contract.md#what-headless-changes)).

### `tfui validate`

Run terraform validate. Opens the validate plugin TUI.

```bash
tfui validate                       # TUI: review diagnostics interactively
tfui validate -ci                  # No TUI: diagnostics to stdout
tfui validate -ci -json            # No TUI: JSON diagnostics to stdout
```

### `tfui output`

Show terraform outputs. Opens the output plugin TUI.

```bash
tfui output                         # TUI: browse outputs interactively
tfui output -ci                    # No TUI: key=value pairs to stdout
tfui output -ci -json              # No TUI: JSON to stdout
```

### `tfui init`

Run terraform init. Executes immediately by default (preserving terraform's native workflow).

```bash
tfui init                           # Run immediately with defaults
tfui init -upgrade                  # Run with -upgrade
tfui init -upgrade -backend=false   # Run with -upgrade -backend=false
tfui init -backend-config=path.hcl  # Run with -backend-config=path.hcl
tfui init -ci                       # Headless execution (no TUI)
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `-upgrade` | `false` | Upgrade modules and plugins |
| `-reconfigure` | `false` | Reconfigure backend |
| `-backend` | `true` | Configure backend (`-backend=false` to skip) |
| `-backend-config` | — | Backend configuration values (repeatable) |

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

`plan` and `apply` accept `-lock` and `-lock-timeout` too — see their sections above.

### `tfui force-unlock`

Remove a terraform state lock (imperative, no TUI).

```bash
tfui force-unlock <lock-id>          # interactive confirmation
tfui force-unlock -force <lock-id>  # skip confirmation (CI/scripts)
tfui force-unlock -chdir modules/vpc <lock-id>  # unlock a member's state
```

### `tfui scaffold`

Generate a `tfui.hcl` configuration file by detecting terraform project patterns.

```bash
tfui scaffold              # interactive wizard
tfui scaffold -yes        # non-interactive (accept defaults)
tfui scaffold -force      # overwrite existing
```

## Global Flags

Every command accepts these flags:

| Flag | Default | Description |
|------|---------|-------------|
| `-project` | `.` | Project root directory (where tfui.hcl lives) |
| `-terraform-bin` | `terraform` | Path to the terraform or tofu binary; `-config terraform.bin=X` takes precedence, and `tfui.hcl` `terraform { bin }` applies when neither is set |
| `-chdir` | | Select a member directory (validated in project mode) |
| `-config` | | Override config values (repeatable, `key=value`) |
| `-ci` | `false` | Run [headless](cli-io-contract.md#mode-resolution): no TUI, output as soon as the plugin is done |
| `-json` | `false` | JSON on stdout, for `plan`, `state`, `validate`, `output` and `version` |
| `-plan` | | Pre-seed plan data from a file or stdin |
| `-state` | | Pre-seed state data from a file or stdin |
| `-macro` | | Run a tape file (headless TUI recording) |
| `-record` | | Capture session frames to a directory |

`-ci` is one of the headless triggers; the others are in [Mode resolution](cli-io-contract.md#mode-resolution).

### Flag syntax

Every flag works with one dash or two: `-json` and `--json`, `-target=X` and `--target=X`. A multi-letter flag tfui
does not know is double-dashed too, so a terraform flag reaches cobra as `--name`; a single-letter flag passes
through unchanged.

These terraform flags take their value as the next argument when it is not joined with `=` (`-target X`):
`target`, `var`, `var-file`, `replace`, `out`, `parallelism`, `lock-timeout`, `chdir`, `workspace`, `input`,
`backend-config`, `plugin-dir`, `get`. These carry their value in the flag (`-lock=false`) and never take the next
argument: `json`, `lock`, `destroy`, `refresh-only`, `compact-warnings`, `upgrade`, `reconfigure`, `force-copy`,
`backend`.

Everything after `--` is passed through to terraform: under `-macro` it is appended to the recorded `terraform plan`
and `terraform apply` commands. A live run does not forward it.

### Pre-seeded data (`-plan`, `-state`)

`-plan` takes plan JSON, the output of `terraform show -json <planfile>`; a
binary plan file is rejected with a hint to convert it. `-state` takes the
output of `terraform show -json` or `terraform state pull`.

Pre-seeded state answers the state reads (resource list and inspect), and a
pre-seeded plan answers `tfui plan` and the plan view, instead of terraform, for
the whole session. `-json` output still comes from `terraform plan -json`.

```bash
tfui state -state ./state.json                 # standalone state TUI, browses pre-loaded state
terraform state pull | tfui state -state - -ci # addresses from piped state
tfui -plan ./plan.json -macro ./review.tape    # macro over a pre-seeded plan
```

A file is re-read when a mutation invalidates the data; stdin is read once.

### URI Resolution Rules

| Input | Resolved as |
|-------|-------------|
| `-` | stdin |
| `/absolute/path` | absolute local path |
| `./relative/path` | relative to CWD |
| `../parent/path` | relative to CWD |
| `file:///path` | local path (scheme stripped) |
| `bare-name` | `-plan`, `-state`: relative to CWD; `-macro`: **ERROR** — suggests `./bare-name` |

Constraint: only one flag can use `-` (stdin) per invocation.

### Macros (`-macro`, `-record`)

Macros are command generators, never executors. They record what terraform would run and output commands to stdout.

`-macro` works on both the root command and subcommands. On root, it drives the full TUI headlessly for multi-plugin recording. On subcommands, it drives the standalone plugin headlessly with the macro service (recording commands without execution).

```bash
tfui -macro deploy.tape -plan ./plan.json            # inspect commands
tfui -macro deploy.tape -plan ./plan.json | sh       # user opts in to execute
```

`-record` is orthogonal to `-macro`. It captures ANSI frames + timing metadata to a directory, and enables debug logging (written as `session.log` in the same directory). Combined with `-macro`, it enables deterministic GIF generation from tapes. Without `-macro`, it records interactive sessions and generates a replayable tape.

See [Macro Language](macro-language.md) for the DSL reference and the macro [exit codes](macro-language.md#exit-codes).

## Exit Codes

The exit codes are in the [CLI I/O contract](cli-io-contract.md#exit-codes); a macro run's are in the
[macro language](macro-language.md#exit-codes).
