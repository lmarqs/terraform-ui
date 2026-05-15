---
title: Navigation stack supports multiple push levels
status: bug
priority: medium
created: 2026-05-15
effort: small
tags: [navigation, architecture, ux]
depends_on: []
---

## Summary

Cross-plugin `returnTo` is currently a single field — only one level of Push is supported. If plugin A pushes B which pushes C, the return to A is lost (overwritten). The navigation should use a stack so completing C returns to B, then completing B returns to A.

## Current behavior

`internal/ui/app.go` holds `returnTo sdk.Plugin` as a single value. `navigateTo()` overwrites it on each NavPush. Multi-level push chains lose earlier return destinations.

## Expected behavior

`returnTo` becomes a slice (stack). Each NavPush appends to the stack. `navigateBack()` pops the last entry. NavReplace clears the entire stack (lateral switch has no history).

## Why it hasn't broken yet

The only NavPush plugins today (chdir, workspaces) are leaf subtasks — they never push further. But this will break once deeper workflows are introduced.

## Reference

See ADR [0004-navigation-model](../adr/0004-navigation-model.md) for the intended design.
