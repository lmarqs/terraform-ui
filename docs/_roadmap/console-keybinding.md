---
title: Console Keybinding Reassignment
status: planned
priority: medium
created: 2026-05-15
effort: small
tags: [ux, keybinding]
depends_on: []
---

## Summary

Move the console (REPL) plugin from the `t` global keybinding to `:console` command-only access (or `~` as an alternative). Frees `t` for consistent contextual taint across all plugins.

## Problem

`t` is currently dual-purpose:
- **Global plugin switch**: `t` navigates to the console/REPL plugin
- **Contextual verb**: `t` means "taint" inside the state plugin

This conflict means:
1. `t` in state = taint (correct, matches terraform verb)
2. `t` from home or other plugins = console (unrelated to terraform's `t`)
3. Adding `t` to plan plugin (for taint) would conflict with the global `t`

The console is a niche power-user feature. Taint is a core terraform workflow verb.

## Design

### Before

| Key | Context | Target |
|-----|---------|--------|
| `t` | Global | Console plugin |
| `t` | In state | Taint (inline action) |

### After

| Key | Context | Target |
|-----|---------|--------|
| `t` | In state | Navigate to taint plugin |
| `t` | In plan | Navigate to taint plugin |
| `~` | Global | Console plugin (optional) |
| `:console` | Command mode | Console plugin |

### Rationale

- `~` visually suggests a terminal prompt — fitting for a REPL
- Console has no time-critical workflow need; `:console` is fast enough
- `t` as a consistent contextual verb ("taint the thing I'm looking at") aligns with the keybinding philosophy: bare lowercase = terraform mutation on cursor resource
- No other plugin uses `~`

### Alternative: No Global Key

Console could be command-only (`:console`). It's a debugging/exploration tool, not a primary workflow. Users who want it can type 8 characters.

## Migration

1. Change console plugin registration: remove `Keybinding: "t"` or change to `"~"`
2. Update hint bar in home/menu view
3. Update `docs/tui-ux.md` keybinding map
