---
title: --target Pre-Pins in TUI Mode
status: planned
priority: medium
created: 2026-05-11
effort: small
tags: [ux, cli, pins]
depends_on: []
---

## Summary

When launching the TUI with `--target` flags, pre-pin those resources so the user starts with a selection already active.

## Problem

Currently `--target` only affects CLI subcommands (`tfui plan --target X`). If you launch the interactive TUI with `--target`, it's ignored. Users who know what they want to target can't carry that intent into the TUI.

## Proposal

```bash
# Launch TUI with aws_instance.web pre-pinned
tfui --target aws_instance.web --target aws_s3_bucket.old
```

When TUI opens:
- Pinned set already contains the targeted addresses
- Plan view shows them with `*` marker
- Apply will scope to pinned (as usual)
- User can add/remove pins interactively

This composes CLI precision with TUI flexibility — start with a selection, refine visually.

## Implementation

1. Parse `--target` on the root command (not just plan/apply subcommands)
2. In TUI startup, write targets to session as initial pins:
   ```go
   pins := sdk.NewPinService(session)
   for _, t := range cfg.Targets {
       pins.Toggle(t)
   }
   ```
3. Plugins that read pins will see them immediately on first render

## Edge Cases

- Target address doesn't exist in state/plan: pin is set but not visually confirmed until data loads
- Combining `--target` with `--plan` (read-only): pins are visual only (no apply possible)
