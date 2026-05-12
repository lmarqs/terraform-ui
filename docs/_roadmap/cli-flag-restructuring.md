---
title: CLI Flag Restructuring (--format + --progress)
status: planned
priority: medium
created: 2026-05-11
effort: medium
tags: [cli, ux, breaking-change]
depends_on: [terraform-flags]
---

## Summary

The current `--mode` flag conflates two orthogonal concerns: output format and progress feedback. Split into `--format` and `--progress` for composability.

## Problem

`--mode agent` means "JSON output" (format concern). `--mode spinner` means "show spinner" (feedback concern). Users can't get JSON output WITH a spinner, or text output WITHOUT a timer.

## Proposal

```bash
# Current (conflated)
tfui plan --mode silent     # text, no animation
tfui plan --mode spinner    # text, spinner
tfui plan --mode progress   # text, spinner + elapsed (default)
tfui plan --mode agent      # JSON, no animation

# Proposed (composable)
tfui plan --format text --progress none       # replaces --mode silent
tfui plan --format text --progress spinner    # replaces --mode spinner
tfui plan --format text --progress timer      # replaces --mode progress (default)
tfui plan --format json --progress none       # replaces --mode agent
tfui plan --format json --progress timer      # NEW: JSON + elapsed time feedback on stderr
```

## Migration

1. Introduce `--format` and `--progress` as new flags
2. Keep `--mode` working (maps to new flags internally)
3. Log deprecation warning when `--mode` is used
4. Remove `--mode` in next major version

## Design

| Flag | Values | Default | Scope |
|------|--------|---------|-------|
| `--format` | `text`, `json` | `text` | What stdout looks like |
| `--progress` | `none`, `spinner`, `timer` | `timer` | Feedback on stderr during execution |

Key insight: format goes to stdout, progress goes to stderr. This enables:
```bash
tfui plan --format json --progress timer > plan.json  # JSON captured, timer visible
tfui plan --format json 2>/dev/null | jq .            # clean JSON pipe
```
