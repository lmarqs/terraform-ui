---
layout: default
title: CLI UX Guidelines
nav_order: 14
description: UX design guidelines for the CLI interface
nav_exclude: true
---

# CLI UX Guidelines — terraform-ui

## 1. Design Principle

Every `tfui <command>` launches the actual plugin in a standalone TUI. The TUI renders on stderr, output goes to stdout on exit. `--ci` or `CI=1` disables the TUI for headless use.

This follows the fzf model: interactive UI on one fd, structured output on another.

## 2. Two Execution Modes

| Mode | TUI | stdout | Trigger |
|------|-----|--------|---------|
| **Standalone** | Alt-screen on stderr | Plugin output on exit | Default (stderr is TTY) |
| **CI** | None | Plugin output immediately | `--ci`, `CI=1`, stderr not TTY |

Mode resolution:
```go
if --ci OR CI=1:     → CI mode
if stderr not TTY:   → CI mode
otherwise:           → Standalone TUI
```

## 3. Output Channel Rules

| Channel | Content | Never |
|---------|---------|-------|
| **stdout** | Data output: tree view, JSON, resource lists, summaries (written on TUI exit) | TUI rendering, ANSI sequences |
| **stderr** | TUI rendering (alt-screen in standalone mode) | Data output |

### Critical invariants

- `-json` flag: changes output FORMAT (JSON vs human-readable), not mode
- `--ci` flag: changes execution MODE (headless vs TUI), not format
- Both flags are orthogonal: `tfui plan --ci -json` = headless + JSON output
- Piped stdout: TUI still renders on stderr (fzf model)

## 4. Flag Conventions

### Terraform compatibility
- All terraform flags work with single dash (`-json`, `-target`) or double dash (`--json`, `--target`)
- `normalizeArgs()` converts single-dash terraform flags to double-dash for cobra
- Unknown flags are left unchanged (future terraform flags don't break)

### Novel flags (tfui-only)
- Always double-dash: `--ci`, `--project`, `--macro`, `--plan`, `--state`, `--terraform-bin`, `--config`, `--chdir`
- Use names terraform hasn't claimed — no collision risk
- `--plan` and `--state` are available on ALL commands (pre-seed data, skip terraform execution)
- `--macro` is root-only (drives full multi-plugin TUI headlessly — doesn't map to a single subcommand)

### Passthrough (`--`)
- Everything after `--` is stored as ExtraArgs
- `splitPassthrough()` runs BEFORE `normalizeArgs()`
- ExtraArgs passed to Plan/Apply via sdk options
- ExecService does NOT forward ExtraArgs directly (uses typed API)

### Flag normalization sets

**Value flags** (consume next arg when no `=`):
`target`, `var`, `var-file`, `replace`, `out`, `parallelism`, `lock`, `lock-timeout`, `chdir`, `workspace`, `input`, `backend`, `backend-config`, `plugin-dir`, `get`

**Bool flags** (never consume next arg):
`json`, `destroy`, `refresh-only`, `compact-warnings`, `upgrade`, `reconfigure`, `force-copy`

## 5. Exit Codes

| Code | Meaning | Commands |
|------|---------|----------|
| `0` | Success (no changes, or operation completed) | All |
| `1` | Error / validation failure | All |
| `2` | Changes present | `plan` only |

Rules:
- Exit 2 must ONLY come from plan plugin's `ExitCode()` method
- cobra errors → exit 1 (handled by `SilenceUsage: true` + `os.Exit(1)`)
- Macro errors → exit code from `macro.RunError.Code`

## 6. Binary Resolution Priority

```
--terraform-bin flag  >  --config terraform.bin=X  >  tfui.hcl terraform { bin }  >  "terraform"
```

## 7. Pipe Ergonomics

### Standalone TUI + pipe (fzf model)

```bash
# User reviews plan in TUI, quits, then output flows to grep:
tfui plan | grep "aws_"

# User reviews plan in TUI, quits, then JSON flows to jq:
tfui plan -json | jq '.changes[].address'

# User browses state in TUI, quits, then addresses flow to wc:
tfui state | wc -l
```

### CI mode in scripts

```bash
# No TUI, immediate output:
CI=1 tfui plan -json | jq '.summary'
tfui validate --ci -json | jq '.valid'

# Auto-detected (stderr not TTY):
tfui plan -json > plan-output.json
```

### Novel command chains (pure stdin filters, no TUI)

```bash
tfui show -json tfplan.out | tfui risk          # plan JSON → risk report
tfui show -json tfplan.out | tfui risk --json | jq '.high_risk[]'
```

## 8. Tree View Format

Default human-readable output for `tfui plan`:

```
+ aws_s3_bucket.logs
~ aws_iam_role.api
- aws_security_group.old
-/+ aws_instance.web

Plan: 1 to add, 1 to change, 1 to destroy.
Risk: high
```

Symbols: `+` create, `~` update, `-` delete, `-/+` replace, `<=` read

## 9. Error Message Formatting

- Errors go to stderr via cobra's error handling or `fmt.Errorf`
- Error messages start with lowercase (Go convention)
- No "Press X to Y" patterns in CLI output
- Wrap errors with context: `fmt.Errorf("plan failed: %w", err)`
- Hints use `\n\nhint:` suffix for actionable guidance

## 10. `--ci` Mode

Purpose: disable TUI entirely for headless/scripted use.

Detection:
1. `--ci` flag on the command
2. `CI=1` environment variable
3. stderr is not a TTY (auto-detected)

Effects:
- No TUI rendering (no alt-screen, no keyboard input)
- Plugin runs headlessly via macro driver
- Output written directly to stdout when plugin reaches Ready state
- Exit codes unchanged

## 11. Novel Commands

Commands with no terraform equivalent (`risk`, `phantom`, `blast-radius`):
- Read plan JSON from stdin (terraform-compatible input)
- Default output: human-readable report on stdout
- `--json` output: our schema (not terraform's) on stdout
- No TUI (pure stdin→stdout filter)
- Exit 0 on success, 1 on error

## 12. Imperative Commands (no TUI)

Workspace operations and force-unlock are direct CLI commands — they execute immediately without TUI because they are one-shot imperative actions:

```bash
tfui workspace select prod     # switches, prints status to stderr
tfui workspace new staging     # creates, prints status to stderr
tfui force-unlock abc123       # unlocks, prints status to stderr
```
