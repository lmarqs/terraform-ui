---
layout: default
title: "ADR-0002: Dual runtime strategy: ExecService and MacroService"
grand_parent: Development
parent: Architecture
nav_order: 0002
description: Decision to support both live execution and recorded macro playback runtimes
---

# Dual runtime strategy: ExecService and MacroService

The system selects its execution mode once at startup -- either live execution (ExecService) or command recording (MacroService). Every call flows through a single adapter for the entire session. Plugins are unaware of which runtime they're operating under.

This is a system-wide invariant, not a per-call decision. Mixing execution modes within a session would be both confusing (user can't tell which calls are real) and dangerous (accidental mutation during dry-run). The core declares the port (`sdk.Service`); the runtime adapter fulfills it entirely. Implementation details like caching are internal to the adapter -- invisible to plugins.

## Considered Options

### Single composed service

Previously existed as `CompositeService`, eliminated. Rejected -- there is no call-level routing decision to make. The execution mode defines the whole system's behavior for UX and safety reasons. Composition adds indirection without purpose when the choice is binary and session-global.

### Decorator/wrapper pattern

One base service wrapped with recording. Rejected -- implies recording is layered on top of execution, but they are peers. Neither wraps the other. Recording replaces execution, it doesn't observe it.
