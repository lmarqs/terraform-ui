---
layout: default
title: "ADR-0019: Unidirectional data flow"
grand_parent: Development
parent: Architecture
nav_order: 0019
description: Decision that data flows downstream and invalidation flows upstream through strict parent-child chains
---

# Unidirectional data flow

Data flows downstream through strict parent→child chains. Each node receives input from its parent, derives its own state, and resets entirely when the parent signals change. Invalidation flows upstream — a child signals "stale" but never modifies the parent's state directly. No node reaches past its immediate parent.

This applies universally: any layered relationship in the system follows these rules. The primary instance today is Context → Plan → Apply, where Context is the app-owned terraform environment, Plan derives a plan file from it, and Apply consumes only Plan's output.

We chose strict layering over flat shared state because shared mutable state between siblings is structurally unsound. When any node can independently read from a grandparent or a shared pool, it creates parallel data paths with no consistency enforcement. Strict layering makes inconsistency impossible by construction.

## Considered Options

### Flat shared state

All nodes read from the same shared state pool and interpret it independently. Cheaper to wire up, but any node can derive a different interpretation of the same inputs. Rejected — parallel interpretations of shared mutable state caused the target-leak bug.

### Event-sourced state

Nodes emit events, a central reducer builds state. Correct but heavyweight for a bounded set of nodes. Direct parent→child flow is simpler and equally safe when the chain is short.
