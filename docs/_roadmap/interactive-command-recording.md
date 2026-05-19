---
title: Interactive Command Recording (dry-run mode)
status: planned
priority: medium
created: 2026-05-14
effort: small
tags: [cli, ux, macro, architecture]
depends_on: []
---

## Summary

Allow users to run the full interactive TUI without executing mutations, printing recorded commands to stdout on exit. This is the keyboard-input counterpart to `-macro` (tape-input).

## Context Update (2026-05-16)

`-record <dir>` has been implemented for frame capture (ANSI frames + timing + tape generation). This is orthogonal — `-record` captures what was SEEN, while this feature captures what would be EXECUTED.

The remaining question: how to activate "record commands without executing" in interactive mode. Current macro service already does this for `-macro` (headless) — the gap is wiring it for the interactive TUI path.

## Recommendation

Use `-dry-run` flag to switch the interactive TUI backend from `ExecService` to `MacroService`. Combine with existing `-record` for a full capture:

```bash
tfui -dry-run                      # interactive TUI, commands to stdout on exit
tfui -dry-run | sh                 # review then execute
tfui -dry-run -record ./session/  # full capture: frames + tape + commands
```

## Implementation

~5 lines in `session.go`: when `-dry-run` is set in interactive mode, use `Recording` backend instead of `Exec`. The rest already works.
