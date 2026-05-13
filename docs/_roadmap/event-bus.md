---
title: Event Bus (replace session keys)
status: planned
priority: high
created: 2026-05-12
effort: medium
tags: [architecture, extensibility]
depends_on: []
---

# Event Bus: Replace Session Keys

## Problem

`pkg/sdk/keys.go` defines static session key constants for inter-plugin communication. This creates coupling: adding a new communication channel requires touching a central file. Doesn't scale toward external plugin extensibility.

Currently, persistent cross-plugin state (e.g., "last plan result", "active workspace") lives in a stringly-typed key-value session. Plugins read/write via `session.Set(key, value)` and `GetTyped[T](session, key)`.

## Current Session Keys

- `plan.summary` — *PlanSummary from last plan
- `plan.file` — path to tfplan.out
- `plan.resource_count` — int
- `chdir.active` — relative chdir member path
- `chdir.active_abs` — absolute path
- `chdir.count` — number of members
- `workspace.active` — current workspace name
- `config.var_files` — resolved []string
- `config.vars` — resolved map[string]string
- `config.extra_args` — []string from --

## Proposed Design

Replace session keys with a typed event bus. Two kinds of state:

1. **Transient events** (already handled by `tea.Msg`): plan completed, workspace switched, chdir selected
2. **Persistent state** (needs the bus): "what was the last plan?", "what's the active workspace?"

Options:
- **A)** Typed store with publish/subscribe — plugins register interest in specific state changes
- **B)** Extend BubbleTea messages with "sticky" messages that persist in a store
- **C)** Plugin-exported state interfaces — each plugin exposes its state as a typed interface, others query it

## Decision

TBD — requires design session focused on external plugin extensibility.

## Success Criteria

- `pkg/sdk/keys.go` deleted
- No magic strings for inter-plugin communication
- External plugins can participate without modifying core
- Type safety at compile time
