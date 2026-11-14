---
layout: default
title: "tfui CLI Output: stdout, stderr and Headless Mode"
parent: Reference
nav_order: 2
seo:
  description: "How tfui splits its output: the interactive TUI on stderr, structured output on stdout on exit, and -ci or CI=1 for headless runs that compose in pipes."
description: "Specifies what tfui reads from stdin and writes to stdout and stderr in each mode, with exit codes."
audience: "Scripts and agents calling tfui, and contributors changing output"
---

# CLI I/O Contract

**Every `tfui <command>` opens its plugin standalone, with the TUI on stderr, and writes its output to stdout on exit.
A headless run has no TUI and writes the output as soon as the plugin is done.**

This follows the fzf model: interactive UI on stderr, structured output on stdout, so a user gets full interactivity
and pipe composability at once.

## Modes

| Mode | TUI | stdout | Selected by |
|------|-----|--------|-------------|
| **Standalone** | Alt-screen on stderr; the user reviews and confirms | Plugin output on exit | Default: `tfui <command>` with a TTY on stderr |
| **Headless** | None, no keyboard input | Plugin output as soon as it is done | `-ci`, `CI=1`, an AI agent environment, or a stderr that is not a TTY |
| **Full TUI** | Alt-screen on stdout, all plugins | Nothing | `tfui` with no command |

### Mode resolution

```
if -ci OR CI=1:                    → headless
if CLAUDECODE or AI_AGENT set:     → headless (AI agent environment, see ADR-0022)
if stderr not TTY:                 → headless (nowhere to render)
otherwise:                         → standalone TUI
```

Agent environment variables match on non-empty. `CI` matches `1` exactly; `CI=true` does not match.

### What headless changes

- The macro driver runs the plugin in place of a terminal: no alt-screen, no keyboard input.
- Output is written when the plugin is ready or reaches a terminal state.
- The action verbs (`taint`, `untaint`, `import`) skip their confirmation: CLI arguments are declared intent, as in
  terraform's own prompt-free commands ([ADR-0022](../adr/0022-agent-environments-imply-headless.md)).
- `tfui apply` requires `-auto-approve`, as `terraform apply` does (see the table below).
- Exit codes are the same as in standalone mode.

## Rules

1. **stdout = data.** Tree views, JSON, resource lists, summaries. Never TUI rendering.
2. **stderr = TUI.** Alt-screen rendering in standalone mode, and status messages. Never data.
3. **`-json` controls format, not mode.** Both standalone and headless runs respect `-json` — it changes what's written to stdout, not whether the TUI appears.
4. **`-ci` controls mode, not format.** Disables TUI entirely. Output format determined by `-json` flag independently.
5. **A piped stdout keeps the TUI.** `tfui plan | jq` renders the TUI on stderr and sends the output to `jq` on exit.

## Full I/O Table

### Standalone mode

| Command | stdout (on exit) | stderr | Exit |
|---------|-----------------|--------|------|
| `tfui plan` | Tree view | TUI (alt-screen) | 0/2 |
| `tfui plan -json` | Plan JSON | TUI (alt-screen) | 0/2 |
| `tfui apply` | — | TUI; "Apply complete. Resources: …" on exit; a failed plan's terraform error | 0/1 |
| `tfui apply -json` | — | TUI; outcome message on exit | 0/1 |
| `tfui state` | Addresses (one/line) | TUI (alt-screen) | 0 |
| `tfui state -json` | Resource JSON array | TUI (alt-screen) | 0 |
| `tfui validate` | Diagnostics text | TUI (alt-screen) | 0/1 |
| `tfui validate -json` | Diagnostics JSON | TUI (alt-screen) | 0/1 |
| `tfui output` | key=value pairs | TUI (alt-screen) | 0 |
| `tfui output -json` | Outputs JSON | TUI (alt-screen) | 0 |
| `tfui init` | "Initialized successfully." | TUI (alt-screen) | 0/1 |
| `tfui version` | Version text | TUI (alt-screen) | 0 |
| `tfui version -json` | Version JSON | TUI (alt-screen) | 0 |

### Headless mode

| Command | stdout | stderr | Exit |
|---------|--------|--------|------|
| `tfui plan -ci` | Tree view | — | 0/2 |
| `tfui plan -ci -json` | Plan JSON | — | 0/2 |
| `tfui apply -ci -auto-approve` | — | "Apply complete. Resources: …" | 0 |
| `tfui apply -ci` (no `-auto-approve`) | — | "apply not allowed for non-interactive use" (mirrors terraform) | 1 |
| `tfui state -ci` | Addresses (one/line) | — | 0 |
| `tfui validate -ci` | Diagnostics text | — | 0/1 |
| `tfui validate -ci -json` | Diagnostics JSON | — | 0/1 |
| `tfui output -ci` | key=value pairs | — | 0 |
| `tfui output -ci -json` | Outputs JSON | — | 0 |
| `tfui init -ci` | "Initialized successfully." | — | 0/1 |
| `tfui version -ci` | Version text | — | 0 |
| `tfui taint addr -ci` | — | "✓ Tainted addr" | 0 |
| `tfui untaint addr -ci` | — | "✓ Untainted addr" | 0 |
| `tfui import addr id -ci` | — | "✓ Imported id as addr" | 0 |
| `tfui state rm\|mv\|taint\|untaint ... -ci` | — | — | 0 |

Apply writes nothing on stdout: the exit code is the machine signal, and the outcome summary goes to stderr, the
status channel.

### Headless failures

Every plugin command reports failure the same way: terraform's error message
on stderr, empty stdout, exit 1. A failed run never claims success.

| Command | stdout | stderr | Exit |
|---------|--------|--------|------|
| `tfui plan -ci` (terraform fails) | — | terraform error | 1 |
| `tfui validate -ci` (run itself fails) | — | terraform error | 1 |
| `tfui apply -ci -auto-approve` (fails) | — | terraform error | 1 |
| `tfui taint/untaint/import ... -ci` (fails) | — | terraform error | 1 |
| `tfui init -ci` (fails) | — | terraform error | 1 |

### Full TUI mode (`tfui` no command)

| Command | stdout | stderr | Exit |
|---------|--------|--------|------|
| `tfui` | (alt-screen) | — | 0/1 |
| `tfui -plan file` | (alt-screen) | — | 0/1 |
| `tfui -state file` | (alt-screen) | — | 0/1 |
| `tfui -macro tape -plan file` | Commands | — | 0/1 |

### Pre-seeded data on subcommands

`-plan` and `-state` work on any command. Pre-seeded state answers the state reads instead of terraform; pre-seeded plan data is read only under `-macro` ([CLI Reference](cli-reference.md)):

| Command | stdout (on exit) | stderr | Exit |
|---------|-----------------|--------|------|
| `tfui state -state file` | Addresses (from pre-seeded data) | TUI (alt-screen) | 0 |
| `tfui state -ci -state file` | Addresses (from pre-seeded data) | — | 0 |
| `tfui plan -plan file` | Tree view (terraform still plans) | TUI (alt-screen) | 0/2 |

### Imperative commands (direct execution, no TUI)

| Command | stdout | stderr | Exit |
|---------|--------|--------|------|
| `tfui workspace show` | Workspace name | — | 0/1 |
| `tfui workspace list` | Workspace names | — | 0/1 |
| `tfui workspace select name` | — | Status message | 0/1 |
| `tfui workspace new name` | — | Status message | 0/1 |
| `tfui workspace delete name` | — | Status message | 0/1 |
| `tfui force-unlock id` | — | Status/prompt | 0/1 |
| `tfui scaffold -yes` | HCL content | Status message | 0/1 |

## Exit Codes

| Code | Meaning | Scope |
|------|---------|-------|
| `0` | Success / no changes | All commands |
| `1` | Error / validation failure | All commands |
| `2` | Plan has changes, as terraform's `-detailed-exitcode` | `plan` command only |

A flag or argument error that stops the command before its plugin runs exits 1. A macro run's codes are in the
[macro language](macro-language.md#exit-codes).

## Plan tree view

`tfui plan` writes this to stdout unless `-json` is given:

```
+ aws_s3_bucket.logs
~ aws_iam_role.api
- aws_security_group.old
-/+ aws_instance.web

Plan: 1 to add, 2 to change, 1 to destroy.
Risk: high
```

Symbols: `+` create, `~` update, `-` delete, `-/+` replace, `<=` read. A replace counts as a change. `Risk:` appears
only when a change carries risk.

## Pipe Scenarios

### Standalone + pipe (fzf model)

```bash
# TUI renders on stderr, user reviews plan interactively.
# On quit, tree view flows to grep via stdout.
tfui plan | grep "aws_"

# TUI renders on stderr, user reviews plan interactively.
# On quit, JSON flows to jq via stdout.
tfui plan -json | jq '.resource_changes[].address'

# TUI renders on stderr for state browsing.
# On quit, addresses flow to wc.
tfui state | wc -l
```

In pipe scenarios, the TUI blocks the downstream process until the user quits. This is intentional — the user reviews interactively, then the result flows.

### Headless in scripts

```bash
# No TUI, immediate output:
CI=1 tfui plan -json | jq '.resource_changes[].address'

# Explicit -ci flag:
tfui validate -ci -json | jq '.valid'

# Auto-detected (stderr not TTY in most CI runners):
tfui plan -json > plan-output.json

# Inside an AI coding agent (CLAUDECODE / AI_AGENT set): bare invocations
# are already headless — no flags needed (ADR-0022):
tfui plan
tfui validate
```

### Macro recording

```bash
# Headless; stdout: the terraform commands the tape would run
tfui -macro deploy.tape -plan ./plan.json
```

## Decisions and Tradeoffs

### Why TUI on stderr?

The fzf model is proven: render UI on stderr (or /dev/tty), emit result on stdout. This means:
- `tfui plan | jq` works — TUI is interactive while pipe stays clean
- stdout is always machine-parseable (no ANSI escape sequences mixed in)
- The TUI is feedback/interaction, stdout is data — clean Unix separation

### Why block until user quits?

The user reviews the plan interactively (expand/collapse, filter, inspect attributes), then presses `q` to exit. Only then does the output flow to stdout. This is the same UX as `fzf`, `less`, or `git log` with a pager — review, then exit, then the terminal has the result.

### Why `-ci` instead of auto-detecting stdout pipe?

When stdout is piped (`tfui plan | jq`), we still show the TUI on stderr. The user wants both: interactive review AND piped output. Auto-disabling TUI on pipe would break this.

`-ci` is the explicit "I don't want any TUI at all" signal; the other triggers are in [Mode resolution](#mode-resolution). Runners that set `CI=true`, such as GitHub Actions, run headless because their stderr is not a TTY.

### Why agent environments force headless

Recorded in [ADR-0022](../adr/0022-agent-environments-imply-headless.md). [Terraform for AI Coding Agents](../features/ai-agents.md) is the agent-facing overview.

### Why full TUI (`tfui` no args) uses stdout?

The full multi-plugin TUI has no single "result" to emit — it's a dashboard. Rendering on stdout (alt-screen) is standard for full-screen TUI apps. Only standalone mode (single plugin invocation) uses the stderr+stdout split.
