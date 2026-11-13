---
layout: default
title: "ADR-0004: Navigation: Replace vs Push with return destination"
grand_parent: Development
parent: Architecture
nav_order: 0004
seo:
  description: Decision to model plugin navigation as Replace or Push with a single return address
description: "Records why plugin navigation is either Replace or Push, with a single return address."
audience: "Contributors changing navigation between plugins"
---

# Navigation: Replace vs Push with return destination

Navigation between plugins uses two declared behaviors -- Replace (lateral switch, no history) and Push (subtask with a return destination). The return destination is explicit ("go here when done"), not implicit history ("go back one level").

This distinction exists because navigation isn't linear. Replace means "I'm somewhere else now" (state → plan). Push means "do this subtask, then bring me back" (state → workspace picker → back to state). Workflows can also set return destinations programmatically (plan → apply pushes plan as the return destination even though the user did not push apply).

## Consequences

- Each plugin keeps its frames on a stack (unlimited depth: list → inspect → confirm → ...). Cross-plugin navigation uses Push with a return destination.
- Return destinations stack: the app keeps one LIFO stack of them. If plugin A pushes B which pushes C, completing C returns to B, completing B returns to A. A Replace clears the stack.

## Considered Options

### Full browser-style history stack

Rejected -- navigation isn't linear; Replace transitions shouldn't create history entries. A user switching state → plan → state is not "going back," and a back-stack would accumulate meaningless entries.

### No return context (all Replace)

Rejected -- subtask patterns (pick a workspace, then continue what you were doing) require knowing where to return. Without it, every subtask dumps you at home.

### Per-plugin `returnTo`

Rejected in favor of a centralized navigation stack. Plugins shouldn't own routing decisions about other plugins.
