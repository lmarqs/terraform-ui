---
title: Auto-Approve (A key + --auto-approve flag)
status: planned
priority: medium
created: 2026-05-11
effort: medium
tags: [ux, cli, apply]
depends_on: [terraform-flags]
---

## Summary

Add `A` (capital) keybinding for auto-approve in TUI, and `--auto-approve` CLI flag for non-interactive apply without saved plan file.

## Problem

Currently all applies require:
1. Run plan (creates plan file)
2. Run apply (reads plan file, confirms)

There's no way to do a quick "just apply it" for confident users or CI pipelines.

## Proposal

### TUI: `A` keybinding

From Plan view:
- `a` = apply with confirmation (current behavior, safe)
- `A` = apply immediately, no confirmation (dangerous, explicit)

Both respect pins (targeted if pins exist, all if no pins).

Follows CLAUDE.md convention: capital letter = dangerous/power variant.

### CLI: `--auto-approve`

```bash
# Plan + apply in one step, no confirmation, no saved plan
tfui apply --auto-approve --project ./infra

# With targets
tfui apply --auto-approve --target aws_instance.web

# With vars
tfui apply --auto-approve --var-file prod.tfvars
```

Unlike the two-step flow, `--auto-approve` calls terraform apply directly (no intermediate plan file). Equivalent to `terraform apply -auto-approve`.

## Design

### TUI flow for `A`:

```
Plan view → user presses A
  → If pins: re-plan with --target (pinned only)
  → If no pins: use full plan
  → Skip confirmation prompt
  → Immediately execute apply
  → Show progress (elapsed time)
  → Show result (success/error)
```

### CLI flow for `--auto-approve`:

```
tfui apply --auto-approve
  → Run terraform apply -auto-approve (no plan file needed)
  → Pass --target, --var, --var-file if provided
  → Output result per --format flag
```

### Safety

- TUI `A` key requires the Plan view to be active (you've seen what will happen)
- CLI `--auto-approve` is explicit opt-in
- Both log a warning: "Auto-approve: applying without confirmation"
- Neither is available in read-only mode (returns CommandErr)
