---
layout: default
title: "ADR-0027: The content panel windows a list when wrap is on"
grand_parent: Development
parent: Architecture
nav_order: 0027
seo:
  description: Decision that ui.ContentPanel can pick a list's visible window, because only it knows how tall wrapped rows are
description: "Records why ui.ContentPanel.RenderWindow picks which rows a list shows, amending ADR-0015's axis split."
audience: "Contributors writing a View() that scrolls a list, or changing ui.ContentPanel"
---

# The content panel windows a list when wrap is on

`ui.ContentPanel` has a second input path, `RenderWindow`. The caller passes every row, the cursor and the end of the
cursor's block, the size, and the offset the panel returned last frame. The panel picks the window, clips it and
places the gutter thumb, measuring in rows with wrap off and in wrapped lines with wrap on, and returns the new offset
for the caller to keep. `Render`, which takes pre-windowed rows, stays. This amends
[ADR-0015](0015-content-panel-axis-split.md); recorded 2026-11-19.

ADR-0015 gave viewport windowing to the plugin. With wrap on, a row's height depends on the content width and the
panel's wrap state, which only the panel knows. Risk windowed in rows and the panel filled its height with wrapped
lines from the top of that slice, so on a narrow terminal with a long first row, `G` moved the cursor onto a row the
panel clipped away, and Enter opened a row nobody could see
([#142](https://github.com/lmarqs/terraform-ui/issues/142)).

## Consequences

- The plugin still owns the domain side of vertical navigation: which rows exist, their order, collapse, filtering,
  and the cursor. The panel owns where the window sits once the rows are known; it does not store the offset.
- The offset `RenderWindow` returns is in rows with wrap off and in wrapped lines with wrap on. A caller keeps it as
  an opaque value and passes it back; the panel re-clamps it after a resize or a wrap toggle, keeping the cursor
  block in view.
- With wrap off, `RenderWindow` gives the same output as windowing in the plugin and calling `Render`.
- `Render` with wrap on also scrolls inside the slice it was given until the cursor row's first line shows, and
  counts rows outside the slice as one line each for the gutter. Lists that pass a slice move to `RenderWindow` to
  get a line-accurate window and thumb.
- The contracts are in [TUI UX § Scrollable content](../reference/tui-ux.md#scrollable-content-split-by-axis).

## Considered Options

### Each plugin wraps its rows to measure them

The plugin would call the panel's wrap logic, sum line heights and window in lines. It keeps ADR-0015 literal, but
every list repeats the same arithmetic against the panel's width and gutter rules, which is the duplication ADR-0015
set out to remove, and the gutter decision (wrap at full width, re-wrap narrower on overflow) leaks into every plugin.

### `Render` alone keeps the cursor visible inside the slice

This draws the cursor row, but the window still starts from a slice cut in rows, so the gutter thumb and the lines
above the slice stay wrong, and the next frame's window cannot follow from this one.

### The panel computes the window from every row (chosen)

One computation decides which lines show, where they clip and where the thumb sits, in the unit that is true for the
current wrap state. The panel stays without vertical state: the caller holds the offset.
