---
layout: default
parent: Plugin Catalog — All Terraform UI Features
title: "Terraform Phantom Changes: Spot No-Op Diffs"
id: phantom
key: P
seo:
  description: "Find Terraform plan changes that modify no infrastructure, such as JSON field reordering or tag order differences, and see why each one is cosmetic."
description: "Describes how the Phantom Changes plugin detects and explains no-op changes in a plan."
audience: "Users reviewing a plan full of diffs that change nothing"
category: analysis
default_enabled: true
---

# Phantom Changes

## Overview

The Phantom Changes plugin identifies plan changes that are cosmetic only -- they appear in the plan output but result in no actual infrastructure modification. Each phantom change includes an explanation of why it is a no-op. [What counts as phantom](../features/phantom-changes.md) lists the causes tfui detects.

## Screenshot

![Phantom Changes]({{ site.baseurl }}/assets/demo/phantom.gif)

## Interactive (TUI)

Press `P` (uppercase) to open the Phantom Changes view. Its screen fills as soon as a plan exists:

- After a plan ran in this session (`:plan`), the view lists that plan's phantom changes at once.
- With no plan in this session, opening the view asks tfui to run `terraform plan` for the active chdir and
  workspace, with the targets `:plan` would use, and shows `Loading plan...` with the elapsed time until it ends.
  With `-plan`, the plan is served from the loaded file instead of running terraform.
- If that plan cannot be produced, the view shows `plan unavailable` with the reason. `Ctrl+r` retries, and `Esc`
  goes back to Home.
- A change of chdir, workspace or pins drops the plan, and the view returns to its empty state until the next plan
  or the next opening with `P`.

Known limitation: Phantom gets its plan through the host's assessment request, the one [Risk](risk.md) uses. A risk
policy that fails to load stops that request before it plans, so Phantom shows `plan unavailable` with the policy's
failure, as Risk does. Issue #156 tracks a plan-only request.

### Keybindings

| Key | Action | When |
|-----|--------|------|
| `j` / `k` | Navigate up/down | List |
| `Enter` / `i` | Expand/collapse phantom details | List |
| `←` / `→` | Scroll the list horizontally when wrapping is off | List |
| `Ctrl+w` | Toggle wrapping | List |
| `Ctrl+r` | Ask for the plan again | Plan unavailable |
| `Esc` | Back to Home, cancelling a running plan | Loading, or plan unavailable |
| `q` | Back to Home | Always |

### Flow

```
Home ──P──→ Phantom Changes (loading) ──→ Phantom Changes (list)
                  │                         │
                  │                         ├── Enter/i → Expand phantom details
                  │                         └── q → Home
                  ├── Esc → Home (cancels the plan)
                  └── plan unavailable → Ctrl+r retries, Esc → Home
```

## Command Line (CLI)

Phantom Changes is an analysis view within the TUI. It has no CLI command of its own, and phantom analysis is available only in the TUI.

## Equivalence

| Goal | CLI | TUI |
|------|-----|-----|
| Identify phantom changes | — | Press `P` |
| Spot phantoms while reviewing | — | `(phantom)` mark on the change in Plan |
| See phantom explanations | — | `P` → expand change |

## Related

- [Plan](plan.md) -- see all changes including phantoms
- [Risk Analysis](risk.md) -- phantoms are marked as cosmetic in risk view
- [Blast Radius](blastradius.md) -- phantoms are flagged in module groups
