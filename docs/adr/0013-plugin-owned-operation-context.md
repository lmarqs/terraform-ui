---
layout: default
title: "ADR-0013: Plugins own their operation context"
parent: Architecture
nav_order: 0013
---

# Plugins own their operation context

Each plugin creates and cancels its own `context.WithCancel` for terraform operations. The app signals cancellation via `sdk.Cancellable` on navigation away, but never holds a reference to the context itself. This prevents orphaned terraform subprocesses from cascading lock errors when users navigate freely during long operations.

## Constraints this creates

- **Plugins that start async terraform operations must implement `sdk.Cancellable`.** The compliance suite (Rules 5-7) enforces this at test time.
- **Every async operation must cancel its predecessor.** Before creating a new context, the plugin calls `Cancel()` to kill any in-flight process from a prior activation or refresh.
- **The app respects `sdk.Busy` as a cancel guard.** If a plugin reports `Busy()` (holding a terraform state lock during apply), the app skips cancellation on navigation to avoid leaving a stale lock.
- **`Cancel()` is idempotent.** Calling it with no in-flight operation is a no-op (nil check on `cancelFn`). No sync primitives needed — BubbleTea's Update loop is single-threaded.

## Considered options

- **App-owned context via `sdk.Context`** — The app creates a cancellable context at Init/Activate and passes it through to the plugin. Simpler API, but couples operation lifetime to navigation state. Breaks when a plugin runs sequential operations (workspace: list then current) or when one operation should survive while another starts. Also violates the SDK isolation boundary — the app would need to know which operations are cancellable.
- **No explicit cancellation (rely on terraform lock timeout)** — Zero code change. Fails in practice: orphaned plan processes re-lock state faster than timeouts expire, producing the cascading lock errors observed in real sessions.
- **Actor model (per-plugin goroutine + channel)** — Each plugin gets lifecycle-managed concurrency. Correct, but a full rewrite of the BubbleTea integration. Disproportionate to the problem — one `cancelFn` field per plugin achieves the same subprocess termination.
