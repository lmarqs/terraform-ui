---
layout: default
title: CLI I/O Contract
parent: Reference
nav_order: 2
description: How tfui handles stdin, stdout, stderr across TUI and CI modes
---

# CLI I/O Contract

## Design Principle

**Every `tfui <command>` launches a standalone TUI on stderr. Output goes to stdout on exit. `--ci` or `CI=1` disables the TUI for headless use.**

This follows the fzf model: interactive UI on stderr (or /dev/tty), structured output on stdout. Users get full interactivity AND pipe composability simultaneously.

### Two Modes

```
Mode 1: Standalone TUI (default)
  TUI renders on stderr (alt-screen)
  User interacts, reviews, confirms
  On exit: structured output → stdout
  Pipe-friendly: tfui plan | jq works (TUI on stderr, JSON to jq after quit)

Mode 2: CI (headless)
  No TUI, no interactivity
  Output goes directly to stdout
  Triggered by: --ci flag, CI=1 env var, or stderr not a TTY
```

### Mode Resolution

```
if --ci OR CI=1:     → CI mode
if stderr not TTY:   → CI mode (nowhere to render)
otherwise:           → Standalone TUI mode
```

### Rules

1. **stdout = data.** Tree views, JSON, resource lists, summaries. Never TUI rendering.
2. **stderr = TUI.** Alt-screen rendering in standalone mode. Never data.
3. **`-json` controls format, not mode.** Both TUI and CI modes respect `-json` — it changes what's written to stdout, not whether the TUI appears.
4. **`--ci` controls mode, not format.** Disables TUI entirely. Output format determined by `-json` flag independently.
5. **Plugins produce output.** Each plugin implements `Outputter` — the same code runs in both modes.

## Full I/O Table

### Standalone TUI mode (default when stderr is TTY)

| Command | stdout (on exit) | stderr | Exit |
|---------|-----------------|--------|------|
| `tfui plan` | Tree view | TUI (alt-screen) | 0/2 |
| `tfui plan -json` | Plan JSON | TUI (alt-screen) | 0/2 |
| `tfui apply` | "Apply complete." | TUI (alt-screen) | 0/1 |
| `tfui apply -json` | `{"status":"complete"}` | TUI (alt-screen) | 0/1 |
| `tfui state` | Addresses (one/line) | TUI (alt-screen) | 0 |
| `tfui state -json` | Resource JSON array | TUI (alt-screen) | 0 |
| `tfui validate` | Diagnostics text | TUI (alt-screen) | 0/1 |
| `tfui validate -json` | Diagnostics JSON | TUI (alt-screen) | 0/1 |
| `tfui output` | key=value pairs | TUI (alt-screen) | 0 |
| `tfui output -json` | Outputs JSON | TUI (alt-screen) | 0 |
| `tfui init` | "Initialized successfully." | TUI (alt-screen) | 0/1 |
| `tfui version` | Version text | TUI (alt-screen) | 0 |
| `tfui version -json` | Version JSON | TUI (alt-screen) | 0 |

### CI mode (`--ci`, `CI=1`, or stderr not TTY)

| Command | stdout | stderr | Exit |
|---------|--------|--------|------|
| `tfui plan --ci` | Tree view | — | 0/2 |
| `tfui plan --ci -json` | Plan JSON | — | 0/2 |
| `tfui apply --ci` | "Apply complete." | — | 0/1 |
| `tfui apply --ci -json` | `{"status":"complete"}` | — | 0/1 |
| `tfui state --ci` | Addresses (one/line) | — | 0 |
| `tfui validate --ci` | Diagnostics text | — | 0/1 |
| `tfui validate --ci -json` | Diagnostics JSON | — | 0/1 |
| `tfui output --ci` | key=value pairs | — | 0 |
| `tfui output --ci -json` | Outputs JSON | — | 0 |
| `tfui init --ci` | "Initialized successfully." | — | 0/1 |
| `tfui version --ci` | Version text | — | 0 |

### Full TUI mode (`tfui` no command)

| Command | stdout | stderr | Exit |
|---------|--------|--------|------|
| `tfui` | (alt-screen) | — | 0/1 |
| `tfui --plan file` | (alt-screen) | — | 0/1 |
| `tfui --state file` | (alt-screen) | — | 0/1 |
| `tfui --macro tape --plan file` | Commands | — | 0/1 |

### Pre-seeded data on subcommands

`--plan` and `--state` work on any command. The plugin reads from cache instead of executing terraform:

| Command | stdout (on exit) | stderr | Exit |
|---------|-----------------|--------|------|
| `tfui plan --plan file` | Tree view (from pre-seeded data) | TUI (alt-screen) | 0/2 |
| `tfui state --state file` | Addresses (from pre-seeded data) | TUI (alt-screen) | 0 |
| `tfui plan --ci --plan file` | Tree view (from pre-seeded data) | — | 0/2 |

### Imperative commands (direct execution, no TUI)

| Command | stdout | stderr | Exit |
|---------|--------|--------|------|
| `tfui workspace show` | Workspace name | — | 0/1 |
| `tfui workspace list` | Workspace names | — | 0/1 |
| `tfui workspace select name` | — | Status message | 0/1 |
| `tfui workspace new name` | — | Status message | 0/1 |
| `tfui workspace delete name` | — | Status message | 0/1 |
| `tfui force-unlock id` | — | Status/prompt | 0/1 |
| `tfui scaffold --yes` | HCL content | Status message | 0/1 |

### Additive flags

| Flag | Effect | Scope |
|------|--------|-------|
| `--ci` | Disable TUI, direct output | All plugin commands |
| `-json` | JSON output format | plan, apply, validate, output, version |
| `--project dir` | Set project root | All commands |
| `--terraform-bin path` | Override binary | All commands |
| `--chdir member` | Select chdir member | All commands |
| `--config key=val` | Override config | All commands |

## Exit Codes

| Code | Meaning | Scope |
|------|---------|-------|
| `0` | Success / no changes | All commands |
| `1` | Error / validation failure | All commands |
| `2` | Plan has changes | `plan` command only |

## Pipe Scenarios

### Standalone TUI + pipe (fzf model)

```bash
# TUI renders on stderr, user reviews plan interactively.
# On quit, tree view flows to grep via stdout.
tfui plan | grep "aws_"

# TUI renders on stderr, user reviews plan interactively.
# On quit, JSON flows to jq via stdout.
tfui plan -json | jq '.changes[].address'

# TUI renders on stderr for state browsing.
# On quit, addresses flow to wc.
tfui state | wc -l
```

In pipe scenarios, the TUI blocks the downstream process until the user quits. This is intentional — the user reviews interactively, then the result flows.

### CI mode in scripts

```bash
# No TUI, immediate output:
CI=1 tfui plan -json | jq '.summary'

# Explicit --ci flag:
tfui validate --ci -json | jq '.valid'

# Auto-detected (stderr not TTY in most CI runners):
tfui plan -json > plan-output.json
```

### Novel command chains

```bash
# Plan → risk analysis (stdin filter, no TUI)
tfui show -json tfplan.out | tfui risk
# stdout: risk report

# Macro recording (headless)
tfui --macro deploy.tape --plan ./tfplan.out
# stdout: terraform commands
```

## Decisions and Tradeoffs

### Why TUI on stderr?

The fzf model is proven: render UI on stderr (or /dev/tty), emit result on stdout. This means:
- `tfui plan | jq` works — TUI is interactive while pipe stays clean
- stdout is always machine-parseable (no ANSI escape sequences mixed in)
- The TUI is feedback/interaction, stdout is data — clean Unix separation

### Why block until user quits?

The user reviews the plan interactively (expand/collapse, filter, inspect attributes), then presses `q` to exit. Only then does the output flow to stdout. This is the same UX as `fzf`, `less`, or `git log` with a pager — review, then exit, then the terminal has the result.

### Why `--ci` instead of auto-detecting stdout pipe?

When stdout is piped (`tfui plan | jq`), we still show the TUI on stderr. The user wants both: interactive review AND piped output. Auto-disabling TUI on pipe would break this.

`--ci` is the explicit "I don't want any TUI at all" signal. It's also auto-detected via `CI=1` env var (set by GitHub Actions, GitLab CI, Jenkins, etc.) and when stderr is not a TTY.

### Why full TUI (`tfui` no args) uses stdout?

The full multi-plugin TUI has no single "result" to emit — it's a dashboard. Rendering on stdout (alt-screen) is standard for full-screen TUI apps. Only standalone mode (single plugin invocation) uses the stderr+stdout split.

### Why workspace/force-unlock stay as direct CLI?

These are imperative one-shot operations (select, create, delete, unlock). They don't benefit from a TUI — the operation is the entire interaction. They complete immediately and print a status message.
