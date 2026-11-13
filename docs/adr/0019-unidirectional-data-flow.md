---
layout: default
title: "ADR-0019: Unidirectional data flow"
grand_parent: Development
parent: Architecture
nav_order: 0019
seo:
  description: Decision that data flows downstream and invalidation flows upstream through strict parent-child chains
description: "Records why data flows down the plugin chain and invalidation flows up, and why apply in the TUI pipeline never passes -target."
audience: "Contributors changing how plan and apply pass data"
---

# Unidirectional data flow

Data flows downstream through strict parent→child chains. Each node receives input from its parent, derives its own state, and resets entirely when the parent signals change. Invalidation flows upstream — a child signals "stale" but never modifies the parent's state directly. No node reaches past its immediate parent.

This applies universally: any layered relationship in the system follows these rules. The primary instance today is Context → Plan → Apply, where Context is the complete terraform operating environment, Plan derives a plan file from it, and Apply consumes only Plan's output.

We chose strict layering over flat shared state because shared mutable state between siblings is structurally unsound. When any node can independently read from a grandparent or a shared pool, it creates parallel data paths with no consistency enforcement. Strict layering makes inconsistency impossible by construction.

The target-leak bug forced the choice. Apply kept its own target list, pushed to it apart from the plan, and re-planned with those targets on its own. Pinned targets from one context survived a context switch and reached terraform in another: the user applied with stale targets the TUI did not show, and terraform modified the wrong resources in the wrong module.

## Consequences

- The rule governs every path that reviews a plan before applying it: the TUI and the interactive `tfui apply`, which enters the same chain at the plan plugin. Targets are baked into the plan file at plan time; apply consumes only the plan artifact and never passes `-target` to terraform. Terraform ignores `-target` on a saved plan and applies the whole plan, so a target passed at apply time would do nothing while appearing to scope the run.
- Per-run settings the plan ran with, such as `-lock` and `-lock-timeout`, travel downstream with the plan file.
- `tfui apply -auto-approve` has no review step. It invokes terraform's native plan-and-apply-in-one-shot mode, so no intermediate plan artifact exists and `-target` goes to the apply.

## Considered Options

### Flat shared state

All nodes read from the same shared state pool and interpret it independently. Cheaper to wire up, but any node can derive a different interpretation of the same inputs. Rejected — parallel interpretations of shared mutable state caused the target-leak bug.

### Event-sourced state

Nodes emit events, a central reducer builds state. Correct but heavyweight for a bounded set of nodes. Direct parent→child flow is simpler and equally safe when the chain is short.
