---
layout: default
title: "ADR-0018: Context is the single source of truth for terraform inputs"
grand_parent: Development
parent: Architecture
nav_order: 0018
description: Decision to centralize all terraform-affecting state into an atomic Context replaced on context switch
---

# Context is the single source of truth for terraform inputs

All state that flows into a terraform CLI argument (targets, var-files, vars, parallelism, lock, lock-timeout) lives in a single atomic Context owned by the app. On context switch the entire Context is replaced — never patched field-by-field. Plugins read from it but never store local copies.

We chose this over a tactical fix (clear pins at app level, patch individual handlers) because the field-by-field approach is structurally unsound: any new plugin or new field that forgets to clear on context switch silently leaks into the next terraform command. Atomic replacement eliminates the category of bug entirely.

## Considered Options

### Tactical fix: clear state in individual handlers

Cheaper to implement but leaves the structural defect intact. Every future plugin author must remember to implement cleanup for every context-change event. Rejected — treats symptoms.
