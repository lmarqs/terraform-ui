---
layout: default
title: "ADR-0022: Agent environments imply headless mode"
grand_parent: Development
parent: Architecture
nav_order: 0022
description: Decision to auto-detect AI agent environments via env vars and resolve them to headless mode
---

# Agent environments imply headless mode

`tfui <verb>` resolves to headless mode when the process runs inside an AI coding agent, detected through environment variables: a non-empty `CLAUDECODE` (set by Claude Code) or a non-empty `AI_AGENT`. The effect is identical to `CI=1`.

## The problem

Mode resolution used three signals: `--ci`, `CI=1`, and stderr TTY detection. Coding agents break the third: they may attach a PTY to the commands they spawn, so `tfui plan` launches the alt-screen TUI on stderr — into a session no human can drive. The agent's transcript fills with ANSI escape noise and the process blocks until killed.

## Why environment variables, not a flag

The explicit spelling of agent mode already exists: `-ci` selects the mode and `-json` selects the format. A `--agent` flag would be a second name for `-ci` — a redundant axis with no distinct semantics, exactly the kind of surface growth ADR-0017 exists to prevent. The gap is *implicit* invocation only: an agent (or a user prompting one) running bare `tfui plan` should get useful output without knowing tfui's flag surface. Environment detection follows the `CI=1` precedent: the runtime context, not the argument list, declares that no human is present.

## Semantics

- `CLAUDECODE` and `AI_AGENT` match on **non-empty**, not `== "1"`. Agents set truthy strings of their choosing; exact-match would silently miss them.
- `CI` keeps its historical exact-match (`== "1"`). Known gap: GitHub Actions sets `CI=true`, which this check misses — masked in practice by the non-TTY-stderr fallback. Changing it is out of scope here.
- Resolution order: `--ci` flag, then `CI=1`, then agent vars, then `!isStderrTTY()`. The rule lives in `resolveSilentStderrFrom` (cmd/tfui/session.go), pure and injected for testing — under `go test` stderr is never a TTY, so the interactive branch is only reachable with an injected TTY probe.

## Action verbs: declared intent skips the confirm

Headless mode also resolves the confirm prelude of the action verbs (taint, untaint, import). Their interactive confirmation cannot be answered without a TUI, so a headless invocation would block until the driver timeout. The cmd layer sets `Input.AutoConfirm` when the session is headless (or macro-driven), and the plugin starts the run directly. This mirrors terraform itself — `terraform taint/untaint/import` never prompt — and reuses ADR-0017's resolution: CLI arguments in a non-interactive context are declared intent. It is an activation-path distinction, not a flag.

## Testing note

The TTY-override branch (agent var set while stderr *is* a TTY) is covered by the pure-function unit test, not integration tests — spawning a real PTY would add a dependency (`creack/pty`) for one assertion. Integration tests assert output correctness under `CLAUDECODE=1` with pipes.
