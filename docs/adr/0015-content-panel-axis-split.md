---
layout: default
title: "ADR-0015: Axis-split box model for content rendering"
grand_parent: Development
parent: Architecture
nav_order: 0015
seo:
  description: The project's rendering paradigm — axis separation for all scrollable content surfaces
description: "Records why every scrollable content surface splits rendering by axis, and why the panel picks a wrapped list's window."
audience: "Contributors writing a View() or a scrollable panel"
---

# Axis-split box model for content rendering

**The component that decides which rows exist and where the cursor is must not also decide how those rows are drawn.** These are always separate responsibilities.

This project separates rendering of scrollable content into axis-specific concerns. **Horizontal formatting** (truncation, scrolling, wrapping, gutter alignment) is mechanical and identical across all views; a shared formatting layer, `ui.ContentPanel` (`pkg/sdk/ui/content_panel.go`), owns it. **Vertical navigation** (which rows exist, their order, collapse-aware traversal, the cursor) requires domain semantics that vary per component; the domain component owns it.

Where the visible window sits depends on both axes. With wrap off a row is one line, and the plugin can window its rows and hand the slice to `Render`. With wrap on a row's height comes from the content width and the wrap state, which only the panel knows: Risk windowed in rows while the panel filled its height with wrapped lines, so on a narrow terminal `G` moved the cursor onto a row the panel had clipped away and Enter opened a row nobody could see ([#142](https://github.com/lmarqs/terraform-ui/issues/142)). So a list can pass the panel every row with its cursor, and `RenderWindow` picks the window, in rows with wrap off and in wrapped lines with wrap on.

## Consequences

- All scrollable content surfaces use the same horizontal formatting layer
- The rendering pipeline is fixed once (ANSI safety, gutter alignment, truncation) and benefits all components
- New plugins or view types only implement vertical navigation — horizontal formatting is inherited
- The formatting layer can evolve independently of domain components (new truncation strategies, gutter styles, etc.) without touching plugin code
- The panel holds no vertical state but one: how far the window has scrolled inside a cursor block taller than it, which `ScrollBlock` moves. Whether the block's last line is in view depends on its wrapped height, which only the panel knows, so `j`/`k` ask the panel before moving the cursor; otherwise an expanded diagnostic or a multi-line output value taller than the view kept its tail out of reach ([#170](https://github.com/lmarqs/terraform-ui/issues/170)).
- `RenderWindow` returns the window's top row in both wrap modes, and the plugin passes it back next frame, so a wrap toggle or a resize never reads it in the wrong unit.
- Lists that still window their own rows for `Render` are line-accurate only inside their slice with wrap on ([#159](https://github.com/lmarqs/terraform-ui/issues/159)).
- The contracts each side keeps, the rules for a new view and the anti-patterns are in [TUI UX § Scrollable content](../reference/tui-ux.md#scrollable-content-split-by-axis).

## Considered Options

### A shared component owns both axes

Fails because vertical navigation requires domain semantics that vary per plugin — tree collapse, pin filtering, fuzzy scoring, grouping. The component either becomes a god object that understands every domain's vertical logic (coupling plugins to its opinion), or plugins fight against it by overriding its vertical behavior and feeding it pre-windowed data anyway — at which point it's a formatting layer pretending to be a Model.

### The domain component owns both axes

Fails because horizontal formatting is identical everywhere and gets duplicated across every plugin. Each caller reinvents ANSI-safe truncation, gutter alignment, and pan state. Bugs multiply and fixes must be applied N times.

### Split by axis (chosen)

Vertical to the domain component (who knows which rows exist and where the cursor is), horizontal to a shared formatting layer (who knows how to render safely and how tall a wrapped row is). Neither side has responsibility it cannot fulfill. The domain never struggles with ANSI sequences; the formatting layer never needs to understand collapse state.

### Stateless formatting functions (no shared state)

A variant of the split approach where horizontal formatting has no state. Works for output but forces every caller to declare horizontal scroll and wrap-mode fields and duplicate the pan/toggle logic identically. Making the formatting layer stateful for horizontal concerns only eliminates this duplication without taking ownership from the domain.

### Each plugin wraps its rows to measure them

The plugin would call the panel's wrap logic, sum line heights and window in lines. Every list would repeat the same arithmetic against the panel's width and gutter rules, which is the duplication this split removes, and the gutter's wrap decision would leak into every plugin.

### `Render` alone keeps the cursor visible inside the slice

This draws the cursor row, but the window still starts from a slice cut in rows, so the gutter thumb and the lines above the slice stay wrong, and the next frame's window cannot follow from this one.
