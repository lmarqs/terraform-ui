---
title: Architecture Debt Cleanup
status: done
priority: medium
created: 2026-05-11
effort: medium
tags: [debt, refactor, testing]
depends_on: []
---

## Summary

Technical debt identified during the architect review of the source/macro implementation. These are correctness and robustness issues, not features.

## Need

Several implementation shortcuts were taken during the initial build that will cause problems as the codebase grows:

1. **Macro `WaitUntil` is broken** — it can never succeed for non-immediate predicates (single-threaded busy-wait with nothing changing the model). Any future macro usage beyond trivial assertions will hit this.
2. **Dependency inversion** — `internal/source` imports `internal/terraform`. The I/O layer depends on domain logic, preventing reuse or extraction.
3. **Zero test coverage on raw state parser** — `parseRawState` handles a complex format (nested modules, index keys, provider cleanup) with no tests. Bugs will be invisible.
4. **Concurrency bug** — `StdinProvider.consumed` flag has no synchronization.
5. **Platform portability** — TTY detection opens `/dev/tty` directly (fails in some container environments).

## Expected UX

No user-visible changes. These are internal quality improvements that prevent future bugs and enable future features (especially macro-cli which depends on a working `WaitUntil`).

## Advantages

- Unblocks macro-cli (WaitUntil fix is a prerequisite)
- Prevents state loading bugs in production (raw parser tests)
- Makes the codebase portable to more environments (TTY fix)
- Cleaner architecture for future provider additions (dependency fix)

## Tasks

- [x] Fix `WaitUntil`: process pending `tea.Cmd` results during wait loop
- [x] Move `LoadPlan`/`LoadState` parsing logic to `internal/terraform`, keep `internal/source` as pure byte resolution
- [x] Add table-driven tests for `parseRawState`: flat state, count/for_each, nested modules, data sources, empty state
- [x] Replace `StdinProvider.consumed` bool with `sync.Once`
- [x] Replace `hasTTY()` with `go-isatty` for cross-platform TTY detection
- [x] Fix `buildAddress` string key quoting (`%q` → terraform-style literal quotes)
- [x] Promote `ErrReadOnly` to `pkg/sdk` (plugins should be able to check it)
