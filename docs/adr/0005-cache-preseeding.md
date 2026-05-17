---
layout: default
title: "ADR-0005: Cache pre-seeding via --plan and --state flags"
grand_parent: Development
parent: Architecture
nav_order: 0005
---

# Cache pre-seeding via --plan and --state flags

When the user provides `--plan` or `--state` at startup, the data is loaded into the global state (ServiceCache) before the service is created. The remaining lifecycle is identical — plugins are unaware of whether data came from pre-seeding or a live terraform call.

This is not a "read-only mode." There is no read-only mode. Pre-seeding is simply how initial state arrives. If a mutation is attempted and there's no terraform binary or the cache has no plan file to apply, it fails the same way any failed terraform call would. Plugins should never branch on "was this pre-seeded?" — they read from the single source of truth and behave identically regardless of origin.

Macro mode (MacroService) is a separate, orthogonal concept — it records commands instead of executing them. It is not "read-only" either; it's a different runtime strategy (see ADR-0002).

## Consequences

Any documentation or code referencing "read-only mode" as a distinct concept should be cleaned up. The concepts are:
- **Pre-seeding**: `--plan`/`--state` populate global state at startup
- **Macro mode**: MacroService records commands (ADR-0002)
- Neither is "read-only mode"
