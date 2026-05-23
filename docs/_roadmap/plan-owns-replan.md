---
title: Plan Owns All Replanning
status: planned
priority: critical
created: 2026-05-22
effort: medium
tags: [plugins, plan, apply, data-flow]
depends_on: [context-architecture-overhaul]
---

## Summary

Move all `terraform plan` operations (including targeted replans) into the plan plugin. Apply becomes a pure consumer: receives a plan file, confirms, executes. Implements the domain rule defined in `CONTEXT.md`: Plan = preparation, Apply = confirmation.

## Problem

Apply currently runs `terraform plan -target=X` internally (`StatusReplanning` state) when it receives targets. This creates a hidden second planning path that reads targets from a `SetTargets()` push — the exact pattern that caused the state-leak bug (see context-architecture-overhaul).

The domain violation: Apply produces plans, breaking the data-flow model where Apply should only consume Plan's output.

## Changes

### Plan plugin gains
- Check "do current pins match the plan file?" when user presses `a`
- If mismatch: replan with `-target` flags, then emit apply handoff
- `ApplyRequestMsg` only emitted AFTER plan file is current

### Apply plugin loses
- `StatusReplanning` state
- `runReplan()` method
- `ReplanResultMsg` type
- `SetTargets()` method
- Its only states become: confirming, running, done, error

### Entry points after change

| Entry point | Behavior |
|-------------|----------|
| TUI: user presses `a` in plan | Plan ensures plan file is current (replans if pins changed), then emits `ApplyRequestMsg` with plan file reference. Apply confirms and runs `terraform apply planfile.out`. |
| CLI: `tfui apply --target=X` | Maps directly to `terraform apply -target=X` — terraform's own plan+apply-in-one-shot. Not a replan. |

## Files involved

- `plugins/plan/plan.go` — add replan-before-handoff logic
- `plugins/apply/apply.go` — remove StatusReplanning, runReplan, ReplanResultMsg, SetTargets
- `internal/ui/app.go` — simplify ApplyRequestMsg handler (no more target push)

## Verification

```bash
mise run check:lint && mise run test:unit && mise run check:build && mise run test:macro
```

Key behaviors:
- Plan replans when pins don't match current plan file
- Apply never calls terraform plan in TUI flow
- `tfui apply --target=X` still works (terraform's mode)
- Esc/cancel during plan's replan returns to plan list (not apply)
