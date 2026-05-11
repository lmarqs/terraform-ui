---
title: Universal Source Abstraction
status: completed
priority: high
created: 2026-05-11
effort: large
tags: [source, architecture]
depends_on: []
---

## Summary

URI-based I/O layer for loading external data (plan, state, macros). All external inputs resolve through a Provider + Resolver pipeline.

## Design

```
Consumer (LoadPlan, LoadState, tape parser)
    ↓
Resolver (URI dispatch, strict rules)
    ↓
Provider (LocalProvider, StdinProvider)
```

**URI rules (no heuristics):**
- `-` → stdin
- `/path` → absolute local
- `./path` or `../path` → relative to CWD
- `scheme://...` → dispatches to registered provider
- Anything else → error with actionable suggestion

## Delivered

- `internal/source/` package (source.go, local.go, stdin.go, loader.go)
- `--plan` and `--state` CLI flags
- `--scope` flag for non-interactive scope selection
- `StaticService` for read-only mode
- Raw tfstate format support (auto-detect)
- Graceful non-TTY degradation (text output)
- TTY detection via `/dev/tty`
- 100+ adversarial tests
