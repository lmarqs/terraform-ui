---
layout: default
title: "ADR-0019: Plan owns all replanning"
grand_parent: Development
parent: Architecture
nav_order: 0019
description: Decision that the plan plugin owns all terraform plan operations including targeted replans
---

# Plan owns all replanning

Plan = preparation; Apply = confirmation of what was already prepared. When pins exist and the current plan file doesn't reflect them, Plan replans with `-target` flags before handing off to Apply. Apply never runs `terraform plan` in the TUI flow — it receives a plan file and executes `terraform apply planfile.out`.

This moves the `StatusReplanning` logic currently inside the apply plugin into plan. The domain rule is: if you need a new plan, that's Plan's job. Replan is equivalent to `ctrl+r` refresh but with targets — it's not a special apply-time concern.

The exception is `tfui apply --target=X` (CLI entry point), which maps directly to `terraform apply -target=X` — terraform's own plan+apply-in-one-shot mode. This is not a replan; it's a single terraform command.

## Consequences

- Apply plugin loses `StatusReplanning`, `runReplan()`, `ReplanResultMsg`. Its only states are: confirming, running, done, error.
- Plan plugin gains responsibility to check "do current pins match the plan file?" when user presses `a`. If mismatch, plan replans first, then emits the apply handoff.
- The `ApplyRequestMsg` flow changes: plan only emits it AFTER ensuring the plan file is current.

## Considered Options

### Keep replan in apply (status quo)

Apply runs `terraform plan -target=X` internally when it receives targets. This creates a hidden second planning path that reads targets from a `SetTargets()` push — the exact pattern that caused the state-leak bug. Rejected because it violates the data-flow model (Apply should be a pure consumer of Plan's output, not a producer of plans).
