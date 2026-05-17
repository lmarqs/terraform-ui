---
layout: default
title: "ADR-0001: SDK isolation: plugins import only pkg/sdk"
grand_parent: Development
parent: Architecture
nav_order: 0001
description: Architecture decision to enforce plugin isolation through a public SDK boundary
---

# SDK isolation: plugins import only pkg/sdk

Plugins depend exclusively on `pkg/sdk/`. They cannot import `internal/` packages. This boundary exists for two reinforcing reasons: (1) hexagonal architecture -- separating core contract from infrastructure keeps the system refactorable without breaking plugins, and (2) third-party plugin enablement -- external developers need a stable, well-defined surface to build against.

Go enforces this naturally: `internal/` is compiler-gated at the module boundary, and once plugins become separate Go modules they physically cannot access internals. Today (same module), the boundary is enforced by linting and convention.

## Considered Options

### Single `pkg/sdk` module (chosen)

One import, one version, simple mental model for plugin authors. Sub-packages (`sdk/ui`, `sdk/frames`) organize internally without introducing independent versioning.

### Multi-module SDK

`pkg/sdk/service`, `pkg/sdk/events`, `pkg/sdk/ui` as separate Go modules. Rejected -- independent versioning only pays off if consumers pin different versions, and there's no realistic scenario for that. Coordination cost outweighs benefit at this project's scale (~28 top-level SDK files).

### No boundary

Plugins import freely. Rejected outright -- defeats the entire plugin architecture; internal refactoring would break plugins.
