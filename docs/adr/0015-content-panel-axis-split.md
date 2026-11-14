---
layout: default
title: "ADR-0015: Axis-split box model for content rendering"
grand_parent: Development
parent: Architecture
nav_order: 0015
seo:
  description: The project's rendering paradigm — axis separation for all scrollable content surfaces
description: "Records why every scrollable content surface splits rendering by axis."
audience: "Contributors writing a View() or a scrollable panel"
---

# Axis-split box model for content rendering

**The component that decides WHICH rows are visible must not also decide HOW those rows are formatted for display.** These are always separate responsibilities.

This project separates rendering of scrollable content into axis-specific concerns. **Horizontal formatting** (truncation, scrolling, wrapping, gutter alignment) is mechanical and identical across all views; a shared formatting layer, `ui.ContentPanel` (`pkg/sdk/ui/content_panel.go`), owns it. **Vertical navigation** (cursor position, viewport windowing, collapse-aware traversal) requires domain semantics that vary per component; the domain component owns it.

## Consequences

- All scrollable content surfaces use the same horizontal formatting layer
- The rendering pipeline is fixed once (ANSI safety, gutter alignment, truncation) and benefits all components
- New plugins or view types only implement vertical navigation — horizontal formatting is inherited
- The formatting layer can evolve independently of domain components (new truncation strategies, gutter styles, etc.) without touching plugin code
- The contracts each side keeps, the rules for a new view and the anti-patterns are in [TUI UX § Scrollable content](../reference/tui-ux.md#scrollable-content-split-by-axis).

## Considered Options

### A shared component owns both axes

Fails because vertical navigation requires domain semantics that vary per plugin — tree collapse, pin filtering, fuzzy scoring, grouping. The component either becomes a god object that understands every domain's vertical logic (coupling plugins to its opinion), or plugins fight against it by overriding its vertical behavior and feeding it pre-windowed data anyway — at which point it's a formatting layer pretending to be a Model.

### The domain component owns both axes

Fails because horizontal formatting is identical everywhere and gets duplicated across every plugin. Each caller reinvents ANSI-safe truncation, gutter alignment, and pan state. Bugs multiply and fixes must be applied N times.

### Split by axis (chosen)

Vertical to the domain component (who knows what's visible), horizontal to a shared formatting layer (who knows how to render safely). Neither side has responsibility it cannot fulfill. The domain never struggles with ANSI sequences; the formatting layer never needs to understand collapse state.

### Stateless formatting functions (no shared state)

A variant of the split approach where horizontal formatting has no state. Works for output but forces every caller to declare horizontal scroll and wrap-mode fields and duplicate the pan/toggle logic identically. Making the formatting layer stateful for horizontal concerns only eliminates this duplication without taking ownership from the domain.
