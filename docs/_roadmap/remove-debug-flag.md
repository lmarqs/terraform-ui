---
title: Remove --debug flag (replaced by --record)
status: planned
priority: low
created: 2026-05-16
effort: medium
tags: [cli, architecture, cleanup]
depends_on: []
---

## Summary

Remove the `--debug` flag, `internal/logging` package, and all `logging.Logger().Debug()` calls. The `--record` flag now provides a better debugging workflow (shareable frames + replayable tapes) than text-based debug logs.

## Scope

- Remove `--debug` persistent flag from `cmd/tfui/main.go`
- Remove `internal/logging/` package entirely
- Remove ~50 `logging.Logger().Debug(...)` calls across `internal/terraform/service.go`, `internal/terraform/state_ops.go`, `internal/ui/app.go`
- Remove `LogDir()` from config and `logger.dir` config override
- Remove `~/.tfui/logs/` convention
- Update docs: CLI reference, config reference

## Why

- `--record` captures what the user saw and did (replayable, shareable)
- Debug logs capture internal state transitions (not shareable, hard to interpret)
- Two observability mechanisms is one too many for a TUI tool
- Removing logging simplifies the codebase and removes the `~/.tfui` home directory dependency
