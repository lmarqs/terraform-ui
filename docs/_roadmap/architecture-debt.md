---
title: Architecture Debt (from review)
status: planned
priority: medium
created: 2026-05-11
effort: medium
tags: [debt, refactor, testing]
depends_on: []
---

## Summary

Technical debt identified during the architect review of the source/macro implementation.

## Tasks

- [ ] **WaitUntil redesign** — Driver is single-threaded, busy-wait can never resolve. Process pending `tea.Cmd` results during the wait loop.
- [ ] **Dependency inversion** — `internal/source` imports `internal/terraform` (I/O depends on domain). Move LoadPlan/LoadState to `internal/terraform`, keep source as pure byte resolution.
- [ ] **Raw state tests** — `parseRawState` has zero test coverage. Add table-driven tests covering: flat state, multi-instance (count/for_each), nested modules, data sources filtered.
- [ ] **Promote ErrReadOnly** — Move from `internal/terraform` to `pkg/sdk` so plugins can detect read-only mode.
- [ ] **StdinProvider race** — `.consumed` bool has no mutex. Use `sync.Once`.
- [ ] **hasTTY portability** — Replace `/dev/tty` open with `term.IsTerminal()` for container environments.
- [ ] **buildAddress quoting** — Uses Go `%q` (backslash escaping) instead of terraform's literal quotes for string keys.
