---
layout: default
title: "ADR-0004: Navigation: Replace vs Push with return destination"
parent: Architecture
nav_order: 0004
---

# Navigation: Replace vs Push with return destination

Navigation between plugins uses two declared behaviors — Replace (lateral switch, no history) and Push (subtask with a return destination). The return destination is explicit ("go here when done"), not implicit history ("go back one level").

This distinction exists because navigation isn't linear. Replace means "I'm somewhere else now" (state → plan). Push means "do this subtask, then bring me back" (state → workspace picker → back to state). Workflows can also set return destinations programmatically (plan → apply sets returnTo=plan even though apply wasn't user-pushed).

Each frame within a plugin maintains its own stack (unlimited depth: list → inspect → confirm → ...). Cross-plugin navigation uses Push with a return destination. The return destination should stack — if plugin A pushes B which pushes C, completing C returns to B, completing B returns to A.

## Considered options

- **Full browser-style history stack** — rejected. Navigation isn't linear; Replace transitions shouldn't create history entries. A user switching state → plan → state is not "going back," and a back-stack would accumulate meaningless entries.
- **No return context (all Replace)** — rejected. Subtask patterns (pick a workspace, then continue what you were doing) require knowing where to return. Without it, every subtask dumps you at home.
- **Per-plugin `returnTo`** — rejected in favor of a centralized navigation stack. Plugins shouldn't own routing decisions about other plugins.
