---
layout: default
title: "ADR-0003: Typed event bus for inter-plugin communication"
grand_parent: Development
parent: Architecture
nav_order: 0003
seo:
  description: Decision to use typed events instead of stringly-typed state sharing between plugins
description: "Records why plugins talk through typed events instead of shared state."
audience: "Contributors adding an event or wiring plugins together"
---

# Typed event bus for inter-plugin communication

Amended by [ADR-0018](0018-atomic-context-replacement.md): pins moved from a shared PinService into the atomic Context, and a pin change is a `ContextChangedEvent` instead of a `PinsChangedEvent`.

Plugins communicate through statically-typed events dispatched by the app. Each event is a struct in `pkg/sdk/events.go`; plugins declare interest by implementing a handler interface (e.g., `ContextChangedHandler`). The app mediates all dispatch -- plugins never reference each other.

The events model the user's terraform working memory: "I'm in a different directory," "I have a plan to review," "something is locked." The TUI externalizes mental state that terraform's CLI forces users to carry manually (clipboard, recall). Events are the transitions of that mental model.

Typed events serve as the **discovery mechanism** for plugin developers. A new plugin author reads `events.go` and immediately knows every state transition they can react to. Implementing a handler interface is a compile-time declaration of interest -- gaps are visible, not hidden.

## Consequences

- Events and shared state both exist, with separate jobs:
  - **Events** = "something happened, decide what to do." Reactive, discoverable, compile-time safe.
  - **Shared state** (the Context, ServiceCache) = "what's true right now, query when you need it." Read on demand.

  Events notify; shared state answers queries. Pins show the two working together: they are part of the Context (queryable), and a `ContextChangedEvent` for which `OnlyPinsChanged()` is true tells plugins their view is stale.
- Adding a new event requires: event struct, handler interface, bus field, dispatch case, bus registration. This is verbose but deliberate -- each event is a conscious extension of the plugin contract, not something that happens accidentally.

## Considered Options

### String-keyed pub/sub

`bus.Emit("workspace.changed", data)`. Rejected -- not discoverable, not type-safe, refactoring breaks silently. Plugin authors can't grep for what's available.

### Direct plugin references

Plugin A calls plugin B. Rejected -- violates the hexagonal boundary. Adding or removing a plugin would require editing other plugins.

### Shared observable state only

No events, plugins poll or diff. Rejected -- reinvents events poorly. "When do I re-read?" becomes the new problem, and you'd need notification anyway.
