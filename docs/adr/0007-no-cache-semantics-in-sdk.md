---
layout: default
title: "ADR-0007: No cache semantics in the SDK contract"
grand_parent: Development
parent: Architecture
nav_order: 0007
---

# No cache semantics in the SDK contract

The `Service` interface exposes no cache-related methods. Plugins request fresh data via `StateList(ctx, sdk.SkipCache())` — a functional option that says "I want fresh data" without revealing that a cache exists. There is no `InvalidateState()`, `RefreshCache()`, or similar on the interface.

Caching is an infrastructure concern that belongs in the adapter (ExecService), not the port (sdk.Service). Plugin authors shouldn't know or care how data is stored between calls. The SDK is a thin layer helping users operate terraform — terraform itself is the source of truth.

## Considered options

- **`InvalidateState()` on the Service interface** — rejected. Exposes "there is a cache" as a concept plugins must understand. Leaks adapter internals into the core contract.
- **Separate `CacheControl` interface** — rejected. Same problem with extra indirection. Plugins still learn about caching.
- **Variadic functional options on existing methods** (chosen) — `SkipCache()` is one option among potentially others. The method signature stays clean; the hint is optional. Internally, the adapter decides what "skip cache" means.
