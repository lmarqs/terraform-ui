---
layout: default
title: "ADR-0012: Broadcast result messages to all plugins"
grand_parent: Development
parent: Architecture
nav_order: 0012
seo:
  description: Decision to broadcast terraform operation results to all plugins for reactive updates
description: "Records why the result of a terraform operation is broadcast to every plugin."
audience: "Contributors adding an operation or reacting to one"
---

# Broadcast result messages to all plugins

We broadcast every message except events and plugin-local messages (timer ticks, stream lines) to every plugin. Each plugin's `Update()` type-switch naturally ignores message types it doesn't own.

Plugin async operations (plan, state list, apply) produce result messages that must reach their owning plugin regardless of which plugin the user is currently viewing. BubbleTea's default routing delivers all messages to the single active Model -- which in our app means only the active plugin sees results. If a user navigates away during a 60-second plan, the result is silently dropped and the plugin stays stuck in Loading forever.

## Consequences

- **Event handlers must not start async operations.** Handler return values produce commands whose results broadcast to all plugins. This is safe but wasteful -- and the handler runs while the plugin may not be active, so the operation's context (cursor position, filter state) may be stale. Handlers should only mutate local state (reset, mark stale). Async work belongs in `Activate()` or `Refresh()`.
- **Plugin-local messages route only to the active plugin.** Messages that originate from an async channel or timer owned by a single plugin have no meaning to other plugins. Current instances: `TimerTickMsg`, `StreamLineMsg`, `StreamDoneMsg`. Broadcasting timer ticks would grow them exponentially when several timers run at once. Broadcasting stream messages would invoke handlers on plugins whose channels are nil (unactivated), which deadlocks the macro driver: it processes batch commands sequentially (each `Cmd` blocks until it returns a `Msg`), so a `Cmd` reading from a nil or unwritten channel blocks the entire event loop. The real BubbleTea runtime launches batch commands as concurrent goroutines and does not deadlock; this routing rule is the guard for the macro driver.
- **Plugins must tolerate receiving messages they didn't request.** The default `Update()` switch must fall through cleanly for unrecognized types (already true by convention, now load-bearing).
- **Stale flag pattern for invalidation.** When a plugin has visible results (`StatusDone`) and receives an invalidation event, it preserves results and sets a `stale` flag. `Activate()` re-runs the operation on next entry; `ctrl+r` works for immediate refresh.

## Considered Options

### Scoped/tagged messages

Wrap each command's output with the originating plugin ID, route by tag. Structurally cleaner but adds wrapping ceremony at every async call site for zero user-visible benefit.

### Route only to active plugin (BubbleTea default)

The original behavior. Fails when users navigate freely during long operations.

### Actor model (per-plugin goroutine + channel)

Each plugin gets lifecycle-managed concurrency. Correct, but a full rewrite of the BubbleTea integration. Disproportionate to the problem.

Broadcast was chosen because it's minimal (one routing change in `app.Update()`), zero-ceremony for plugin authors, and the type-switch isolation already existed.
