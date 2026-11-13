---
layout: default
title: "ADR-0005: Cache pre-seeding via -plan and -state flags"
grand_parent: Development
parent: Architecture
nav_order: 0005
seo:
  description: Decision to allow pre-seeding the service cache from local files or stdin
description: "Records why the -plan and -state flags pre-seed the service cache from files or stdin."
audience: "Contributors changing input flags or the service cache"
---

# Cache pre-seeding via -plan and -state flags

When the user provides `-plan` or `-state` at startup, the data is loaded into the global state (ServiceCache) before the service is created. The remaining lifecycle is identical -- plugins are unaware of whether data came from pre-seeding or a live terraform call.

Pre-seeding is simply how initial state arrives. Plugins should never branch on "was this pre-seeded?" -- they read from the single source of truth and behave identically regardless of origin.

Macro mode (MacroService) is a separate, orthogonal concept -- it records commands instead of executing them. It is not "read-only" either; it's a different runtime strategy (see ADR-0002).

## Consequences

- If a mutation is attempted and there's no terraform binary or the cache has no plan file to apply, it fails the same way any failed terraform call would.
- Any documentation or code referencing "read-only mode" as a distinct concept is wrong and gets cleaned up.

## Considered Options

### A read-only mode

Rejected. There is no read-only mode: a mode is a branch every plugin would take on where its data came from, while pre-seeding is only how initial state arrives.
