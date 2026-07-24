---
layout: default
title: Terraform for AI Coding Agents
parent: Features — Terraform Plan Analysis Tools
nav_order: 4
description: Run terraform through AI coding agents like Claude Code. Automatic headless mode, 38x smaller plan output, truthful errors, and meaningful exit codes — zero configuration.
---

## Overview

AI coding agents (Claude Code, and any harness that sets `AI_AGENT` or `CI`) run `tfui` commands as if they were terraform — and get output designed for a context window instead of a scrollback buffer.

```
agent runs:  tfui plan
agent sees:  + local_file.f00
             + local_file.f01
             ...
             Plan: 50 to add, 0 to change, 0 to destroy.
             Risk: low
exit code:   2 (changes present)
```

No flags. No configuration. No MCP server. The agent's environment is detected automatically, the TUI never launches, and every command follows one contract: **data on stdout, status on stderr, meaningful exit codes, and a failed run never claims success.**

## Why this matters for agents

An agent reading raw `terraform plan` output pays for every line of it — attribute diffs, `(known after apply)` placeholders, provider noise — in context-window space. Worse, an interactive tool that detects a PTY will launch a full-screen UI the agent cannot drive: the process blocks and the agent sees ANSI escape soup.

tfui addresses both:

- **Concise by construction.** One line per change, a plan summary, and a [risk classification](risk-analysis.md) — the same tree the TUI renders, minus the TUI.
- **Headless by detection.** Agent environments are recognized before any rendering decision is made, so `tfui plan` inside an agent session can never hang on an alt-screen.

## Benchmark: context-window footprint

Measured on a 50-resource plan (all creates, `local_file` resources, Terraform v1.14.9, identical configuration for both commands):

| Command | Lines | Bytes |
|---|---|---|
| `terraform plan -no-color` | 763 | 34,426 |
| `tfui plan` (headless) | 53 | 905 |

**38× fewer bytes, 14× fewer lines** — and the agent still gets every change address, the add/change/destroy summary, and a risk verdict. Output scales with one line per resource, not with attribute count, so the gap widens as plans grow.

Success runs emit zero bytes on stderr. Nothing in the output needs post-processing: no ANSI escapes, no spinners, no progress noise.

Need full detail for a specific decision? The agent can ask for it explicitly with `-json`:

```bash
tfui plan -json     # machine-readable plan JSON on stdout
tfui validate -json # structured diagnostics
```

## How detection works

Headless mode resolves before anything renders:

```
if -ci flag or CI=1:               → headless
if CLAUDECODE or AI_AGENT set:     → headless (agent environment)
if stderr is not a TTY:            → headless
otherwise:                         → interactive TUI
```

Claude Code sets `CLAUDECODE` and `AI_AGENT` in every shell it spawns, so both bare `tfui <command>` invocations and PTY-attached ones resolve to headless automatically. Other agents are covered by whichever signal they provide: `AI_AGENT`, `CI=1`, or a non-TTY stderr. The full rationale is in [ADR-0022](../adr/0022-agent-environments-imply-headless.md).

There is no `--agent` flag — the explicit spelling already exists (`-ci` for mode, `-json` for format), and detection makes it unnecessary.

## The failure contract

The most expensive output an agent can receive is a lie. tfui's headless contract is: **terraform's error on stderr, empty stdout, exit 1 — always.**

```console
$ tfui plan        # inside an agent session, broken configuration
$ echo $?
1
```

stderr (terraform's own diagnostics, verbatim):

```
running terraform plan: exit status 1

Error: Reference to undeclared local value

  on main.tf line 15, in resource "local_file" "broken":
  15:   content  = local.nonexistent

A local value with the name "nonexistent" has not been declared.
```

`tfui validate` reports diagnostics as data (stdout) and exits 1 when the configuration is invalid:

```console
$ tfui validate
✗ Reference to undeclared local value (main.tf:15)
  A local value with the name "nonexistent" has not been declared.
$ echo $?
1
```

## Exit codes the agent can branch on

| Code | Meaning |
|---|---|
| `0` | Success — or no changes, for `plan` |
| `1` | Error / invalid configuration |
| `2` | Plan has changes (terraform `-detailed-exitcode` convention) |

An agent can run `tfui plan`, branch on the exit code alone, and only read output when it needs the detail.

## Every verb, one contract

All commands follow the same rules headlessly — including the state-mutating ones, which skip their interactive confirmation exactly as `terraform taint` does:

```console
$ tfui taint local_file.alpha
✓ Tainted local_file.alpha        # stderr; exit 0
$ tfui apply -auto-approve
Apply complete.                    # stderr; exit 0
```

| Command | stdout | stderr | Exit |
|---|---|---|---|
| `tfui plan` | tree view or JSON | — | 0/2 |
| `tfui validate` | diagnostics | — | 0/1 |
| `tfui state` | one address per line | — | 0 |
| `tfui output` | key=value or JSON | — | 0 |
| `tfui apply -auto-approve` | — | outcome summary | 0/1 |
| `tfui taint` / `untaint` / `import` | — | ✓ confirmation | 0/1 |
| any command, terraform fails | — | terraform's error | 1 |

The complete specification is the [CLI I/O Contract](../reference/cli-io-contract.md).

## Teach your agent to use it

Add one block to your project's `CLAUDE.md` (or `AGENTS.md`):

```markdown
## Terraform

Use `tfui` instead of raw `terraform` for read/review operations:

- `tfui plan` — one line per change + summary + risk level (exit 2 = changes present)
- `tfui validate` — compact diagnostics (exit 1 = invalid)
- `tfui state` — resource addresses, one per line
- `tfui output` — key=value pairs

Failures put terraform's error on stderr and exit 1. Add `-json` for
machine-readable output.
```

The same binary serves the humans on the team as a full [interactive TUI](../guides/getting-started.md) — one tool, reviewed by people, driven by agents.

## See also

- [CLI I/O Contract](../reference/cli-io-contract.md) — the full stdin/stdout/stderr specification
- [CLI Reference](../reference/cli-reference.md) — all commands and flags
- [Risk Analysis](risk-analysis.md) — the classification behind the `Risk:` line
- [ADR-0022](../adr/0022-agent-environments-imply-headless.md) — why agent environments imply headless
