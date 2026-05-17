---
layout: default
title: "ADR-0012: Broadcast result messages to all plugins"
grand_parent: Development
parent: Architecture
nav_order: 0012
description: Decision to broadcast terraform operation results to all plugins for reactive updates
---

# Broadcast result messages to all plugins

Plugin async operations (plan, state list, apply) produce result messages that must reach their owning plugin regardless of which plugin the user is currently viewing. BubbleTea's default routing delivers all messages to the single active Model -- which in our app means only the active plugin sees results. If a user navigates away during a 60-second plan, the result is silently dropped and the plugin stays stuck in Loading forever.

We broadcast all non-event, non-tick messages to every plugin. Each plugin's `Update()` type-switch naturally ignores message types it doesn't own. Timer ticks are excluded from broadcast to prevent exponential growth when multiple timers run simultaneously.

## Consequences

- **Event handlers must not start async operations.** Handler return values produce commands whose results broadcast to all plugins. This is safe but wasteful -- and the handler runs while the plugin may not be active, so the operation's context (cursor position, filter state) may be stale. Handlers should only mutate local state (reset, mark stale). Async work belongs in `Activate()` or `Refresh()`.
- **Timer ticks route only to the active plugin.** `TimerTickMsg` carries no plugin identity. Inactive plugins resume their tick chain via `Activate()` when re-entered.
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
