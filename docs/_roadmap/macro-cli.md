---
title: Macro CLI Integration
status: planned
priority: high
created: 2026-05-11
effort: medium
tags: [macro, cli, testing]
depends_on: [source-abstraction]
---

## Summary

Wire the macro engine (Driver + tape DSL) to a `--macro` CLI flag for automated TUI testing and CI visual regression.

## Problem

The macro engine exists (`internal/macro/`) but nothing invokes it from the CLI. Users can't run tape scripts without writing Go code.

## Design

```bash
tfui --macro ./scripts/verify-plan.tape
tfui --plan ./plan.json --macro ./scripts/check.tape
cat script.tape | tfui --macro -
```

The `--macro` flag:
1. Resolves the URI via source.Resolver
2. Parses bytes via `macro.ParseTape()`
3. Creates the App + Driver (no `tea.Program`, no TTY needed)
4. Executes commands sequentially
5. Exits with code 0/1/2/3

**Architecture issue to fix first:** `WaitUntil` in the driver is a busy-wait that can never resolve (single-threaded model). Must redesign to process pending `tea.Cmd` results during the wait loop.

## Open Questions

- `--width` / `--height` flags for controlling terminal dimensions in macro mode?
- Golden file comparison built-in (`assert screenshot ./golden/expected.txt`)?
- Should `--macro` imply no TTY needed (always non-interactive)?

## Tasks

- [ ] Fix `WaitUntil` design (process commands during wait)
- [ ] Add `--macro` flag to root command
- [ ] Wire: resolve URI → parse tape → create Driver → execute
- [ ] Exit codes: 0=pass, 1=assertion fail, 2=syntax error, 3=timeout
- [ ] Add `--width`/`--height` flags
- [ ] Integration test: run macro against fixture plan
