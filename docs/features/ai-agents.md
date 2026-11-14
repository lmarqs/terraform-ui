---
layout: default
title: Terraform for AI Coding Agents
parent: Features — Terraform Plan Analysis Tools
nav_order: 4
seo:
  description: Run terraform through AI coding agents. Automatic headless mode, plan output in a fraction of the tokens, truthful errors, and meaningful exit codes — zero configuration.
description: "Explains how tfui behaves under an AI coding agent: headless detection, compact output, errors and exit codes."
audience: "Users running terraform through an agent, and contributors changing headless output"
---

## Overview

AI coding agents (any harness that sets `CLAUDECODE` or `AI_AGENT`) run `tfui` commands as if they were terraform — and get output designed for a context window instead of a scrollback buffer. CI runs with `CI=1`, and any run whose stderr is not a terminal, get the same output.

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

{% assign ba = site.data.before_after -%}
Measured on the landing page's demo plan ({{ ba.changes }} changes: {{ ba.add }} to add, {{ ba.change }} to change, {{ ba.destroy }} to destroy; Terraform v1.14.9, identical configuration for both commands). Tokens are the input tokens Claude ({{ ba.token_model }}) counts for each output.

| Command | Tokens | Lines | Bytes |
|---|---|---|---|
| `terraform plan -no-color` | {{ ba.before_tokens }} | {{ ba.before_lines }} | {{ ba.before_bytes }} |
| `tfui plan` (headless) | {{ ba.agent_tokens }} | {{ ba.agent_lines }} | {{ ba.agent_bytes }} |

**{{ ba.before_tokens | divided_by: ba.agent_tokens }}× fewer tokens, {{ ba.before_lines | divided_by: ba.agent_lines }}× fewer lines**, and the agent still gets every change address, the add/change/destroy summary, and a risk verdict. Output scales with one line per resource, not with attribute count, so the gap widens as plans grow. `mise run demo:before-after` regenerates these numbers.

Success runs emit zero bytes on stderr. Nothing in the output needs post-processing: no ANSI escapes, no spinners, no progress noise.

Need full detail for a specific decision? The agent can ask for it explicitly with `-json`:

```bash
tfui plan -json     # machine-readable plan JSON on stdout
tfui validate -json # structured diagnostics
```

## How detection works

Headless mode resolves before anything renders. Claude Code sets `CLAUDECODE` and `AI_AGENT` in every shell it
spawns, so both bare `tfui <command>` invocations and PTY-attached ones run headless. Other agents are covered by
whichever signal they provide: `AI_AGENT`, `CI=1`, or a non-TTY stderr. The full order is the contract's
[mode resolution](../reference/cli-io-contract.md#mode-resolution), and why agents are detected from the environment
rather than a flag is in [ADR-0022](../adr/0022-agent-environments-imply-headless.md).

## The failure contract

The most expensive output an agent can receive is a lie, so a failed run never claims success: it follows the
contract's [failure rows](../reference/cli-io-contract.md#headless-failures).

```console
$ tfui plan        # inside an agent session, broken configuration
$ echo $?
1
```

stderr (tfui's one-line prefix, then terraform's diagnostics verbatim):

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

`tfui plan` exits 2 when the plan has changes, as terraform's `-detailed-exitcode` does, so an agent can run it,
branch on the exit code alone, and read the output only when it needs the detail. Every code is in the
[contract](../reference/cli-io-contract.md#exit-codes).

## Every verb, one contract

All commands follow the same rules headlessly, including the state-mutating ones, which skip their interactive confirmation as `terraform taint` does:

```console
$ tfui taint local_file.alpha
✓ Tainted local_file.alpha        # stderr; exit 0
$ tfui apply -auto-approve
Apply complete. Resources: 1 added, 0 changed, 0 destroyed.   # stderr; exit 0
```

What each command writes headless is in the contract's
[headless table](../reference/cli-io-contract.md#headless-mode).

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

