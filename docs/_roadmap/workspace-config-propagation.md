---
title: Runtime Workspace Config Propagation
status: planned
priority: medium
created: 2026-05-12
effort: small
tags: [feature, config]
depends_on: [event-bus]
---

# Runtime Workspace Config Propagation

## Problem

When the user switches workspace via the workspaces plugin, `SessionKeyWorkspace` is updated. The resolved config (var-files, vars, plugin settings) should update accordingly. Currently, `Resolve()` runs once at startup in `PersistentPreRun` and seeds the session. Runtime workspace changes don't trigger re-resolution.

## Current State

- `BuildPlanOptions()`/`BuildApplyOptions()` read from session at call time (correct)
- Session is seeded with resolved values at startup (correct for initial workspace)
- Workspace plugin sets `SessionKeyWorkspace` on switch (correct)
- **Missing**: nothing re-runs `Resolve()` and updates `SessionKeyVarFiles`/`SessionKeyVars` when workspace changes

## Solution

The app layer needs to observe `SessionKeyWorkspace` changes and re-resolve:

1. After workspace switch succeeds, re-run `config.Resolve(root, child, newWorkspace)`
2. Update `SessionKeyVarFiles` and `SessionKeyVars` with new resolved values
3. Next plan/apply automatically picks up new values via `BuildPlanOptions()`

Depends on event bus for clean implementation (observe workspace state change).

## Workaround (current)

Restart tfui to pick up workspace-specific config. Or use `--workspace` flag at launch.
