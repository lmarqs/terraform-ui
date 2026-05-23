---
layout: default
title: "ADR-0018: Atomic Context replacement on context switch"
grand_parent: Development
parent: Architecture
nav_order: 0018
description: Decision to centralize all terraform-affecting state into an atomic Context replaced on context switch
---

# Atomic Context replacement on context switch

All terraform-affecting state (targets, var-files, vars, parallelism, lock, lock-timeout) is centralized into a single Context owned by the app. On context switch, the entire Context is replaced — never patched field-by-field. Plugins read from it but never store local copies.

We chose atomic replacement over field-by-field mutation because the patch approach is structurally unsound: any new field that forgets to clear on context switch silently leaks into the next terraform command. Atomic replacement eliminates the category of bug entirely.

## Considered Options

### Tactical fix: clear state in individual handlers

Cheaper to implement but leaves the structural defect intact. Every future plugin author must remember to implement cleanup for every context-change event. Rejected — treats symptoms.
