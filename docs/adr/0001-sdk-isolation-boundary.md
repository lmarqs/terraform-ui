---
layout: default
title: "ADR-0001: SDK isolation: plugins import only pkg/sdk"
grand_parent: Development
parent: Architecture
nav_order: 0001
seo:
  description: Architecture decision to enforce plugin isolation through a public SDK boundary
description: "Records why plugins import only pkg/sdk and never internal/."
audience: "Contributors adding a plugin or moving code across pkg/sdk"
---

# SDK isolation: plugins import only pkg/sdk

Amended by [ADR-0021](0021-plugins-as-use-cases.md): plugins also import BubbleTea, which owns their model and commands; `internal/` and cobra stay out of plugins.

Plugins depend exclusively on `pkg/sdk/`. They cannot import `internal/` packages. This boundary exists for two reinforcing reasons: (1) hexagonal architecture -- separating core contract from infrastructure keeps the system refactorable without breaking plugins, and (2) third-party plugin enablement -- external developers need a stable, well-defined surface to build against.

## Consequences

- Anything a plugin needs from the app must first be exposed in `pkg/sdk`.
- Go gates `internal/` at the module boundary, so once plugins become separate Go modules they physically cannot access internals. Today plugins share the module, so the compiler does not stop the import: the boundary is held by convention, checked by the plugin-edit hook (`.claude/settings.json`) and the `arch-checker` agent. No linter rule enforces it.

## Considered Options

### Single `pkg/sdk` module (chosen)

One import, one version, simple mental model for plugin authors. Sub-packages (`sdk/ui`, `sdk/frames`) organize internally without introducing independent versioning.

### Multi-module SDK

`pkg/sdk/service`, `pkg/sdk/events`, `pkg/sdk/ui` as separate Go modules. Rejected -- independent versioning only pays off if consumers pin different versions, and there's no realistic scenario for that. Coordination cost outweighs benefit at this project's scale.

### No boundary

Plugins import freely. Rejected outright -- defeats the entire plugin architecture; internal refactoring would break plugins.
