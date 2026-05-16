---
title: Apply Replan for Targeted Resources
status: planned
priority: critical
created: 2026-05-15
effort: medium
tags: [ux, workflow, apply, plan]
depends_on: []
---

## Summary

When the user pins resources and presses `a` (apply) from plan, the system must replan with those targets before applying — never apply a targeted subset without showing the user what that targeted plan produces.

## Problem

### Terraform's constraint

Terraform does NOT allow `-target` on a saved plan file. A saved plan is a complete, verified execution plan with all dependencies resolved. Subsetting it at apply time could break dependency chains.

Therefore terraform requires: `terraform plan -target=X` → review → `terraform apply` (applies the saved targeted plan).

### Current behavior (broken)

Today when targets exist:
1. Plan runs (full, no targets)
2. User pins 2 resources and presses `a`
3. Apply runs `terraform apply -target=X -target=Y` **without a saved plan**

This skips the review of the targeted plan. The user saw the FULL plan, but the targeted apply may produce DIFFERENT changes because targeting changes dependency resolution.

### Why this matters

- The user reviews plan showing 10 changes
- They pin 2 resources, thinking "I'll just apply these 2"
- But targeting may pull in dependencies or exclude resources the user expected
- The apply executes changes the user never reviewed

## Design

### Revised Flow

```
Plan (full) → user pins resources → presses a
  → System detects: targets ≠ full plan scope
  → REPLAN with targets (terraform plan -target=X -target=Y)
  → Show targeted plan for review
  → User confirms
  → Apply the saved targeted plan file (terraform apply tfplan.out)
```

### Apply Plugin States (revised)

1. **Replanning** — running `terraform plan -target=X -target=Y` (new state)
2. **Confirming** — showing targeted plan summary for approval
3. **Loading** — executing `terraform apply tfplan.out`
4. **Done/Error** — result

### Views

**Replanning:**
```
Replanning with 2 targets...  ⠋ 4s

  -target=aws_instance.web
  -target=aws_subnet.private
```

**Confirming (no targets — unchanged):**
```
Apply all changes?
  3 to add, 1 to change, 1 to destroy

[y]es / [n]o / Esc cancel
```

**Confirming (targeted — after replan):**
```
Apply targeted plan?
  2 resources targeted
  1 to add, 1 to change

[y]es / [n]o / Esc cancel
```

### Decision Logic in App Handler

```
case ApplyRequestMsg:
    targets := pins.All()
    if len(targets) == 0:
        // No targets: apply saved plan file directly
        → apply plugin (StatusConfirming)
    else:
        // Targets: must replan first
        → apply plugin (StatusReplanning)
        → run terraform plan -target=X -target=Y
        → on success → transition to StatusConfirming
        → on failure → StatusError with replan error
```

### Service Layer Change

The `Apply()` method no longer needs the "targets present → skip plan file" workaround. With replan:
- Targeted replan saves a new `tfplan.out`
- Apply always uses the saved plan file: `terraform apply tfplan.out`
- Removes the divergent code path entirely

### Edge Cases

- **Replan shows different changes**: Expected. That's why we replan — the user MUST review.
- **Replan shows no changes**: Show "No changes for targeted resources." with escape.
- **Replan fails** (lock, auth, etc.): Show error with retry/escape, same as plan errors.
- **User cancels during replan**: Esc → DeactivateMsg → return to plan (no state change).

## Migration

1. Add `StatusReplanning` state to apply plugin
2. When targets present, apply plugin runs plan internally before confirming
3. Remove the "targets present → direct apply with -target" code path from ExecService
4. Apply always uses saved plan file after replan
5. Update apply views to show replan progress

## Workflow Examples

### Before (broken):
```
plan (full) → pin 2 → a → confirm → terraform apply -target=X -target=Y (NO plan file, NO review)
```

### After (correct):
```
plan (full) → pin 2 → a → replan -target=X -target=Y → review changes → confirm → terraform apply tfplan.out
```
