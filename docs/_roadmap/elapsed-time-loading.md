---
title: Elapsed Time in Loading States
status: planned
priority: medium
created: 2026-05-14
effort: small
tags: [ux, consistency, loading]
depends_on: []
---

## Summary

Add elapsed time display to all plugin loading states using the shared `pkg/sdk/ui.Timer` component. Currently only the apply plugin shows elapsed time — all others show static text.

## Problem

Inconsistent loading UX across plugins. Users have no sense of how long an operation has been running. Long operations (plan on large infra, state refresh) feel unresponsive without a time indicator.

## Plugins to Update

| Plugin | Current Loading View | After |
|--------|---------------------|-------|
| plan | `Running terraform plan...` | `Running terraform plan... 5s` |
| state | `Loading terraform state...` | `Loading terraform state... 3s` |
| validate | `Running terraform validate...` | `Running terraform validate... 2s` |
| output | `Loading terraform outputs...` | `Loading terraform outputs... 1s` |
| workspace | `Loading workspaces...` | `Loading workspaces... 1s` |
| forceunlock | `Force-unlocking {id}...` | `Force-unlocking {id}... 2s` |
| apply | Already has elapsed time | Migrate to shared Timer |

## Implementation

The shared `Timer` component already exists at `pkg/sdk/ui/timer.go`. Each plugin needs:

1. Add `timer ui.Timer` field to plugin struct
2. Call `e.timer.Start()` when transitioning to `StatusLoading` (return tick cmd)
3. Handle `ui.TimerTickMsg` in `Update()` — call `e.timer.Tick()` (return next tick cmd)
4. Call `e.timer.Stop()` when leaving `StatusLoading`
5. Render `e.timer.FormatElapsed()` in the loading view line

Per plugin: ~10 lines of change. No new dependencies.

## Style Decision

Use uniform `StyleFaintItalic` for all loading messages (no `>>>` spinner prefix). The elapsed time alone provides progress feedback.

## Verification

- Update golden tests for loading state views
- Add/update macro tape that waits for elapsed time to appear (e.g., `wait view 1s`)
