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
places the gutter thumb, measuring in rows with wrap off and in wrapped lines with wrap on, and returns the window's
top row for the caller to keep. `Render`, which takes pre-windowed rows, stays. This amends
[ADR-0015](0015-content-panel-axis-split.md); recorded 2026-11-19.

ADR-0015 gave viewport windowing to the plugin. With wrap on, a row's height depends on the content width and the
panel's wrap state, which only the panel knows. Risk windowed in rows and the panel filled its height with wrapped
lines from the top of that slice, so on a narrow terminal with a long first row, `G` moved the cursor onto a row the
panel clipped away, and Enter opened a row nobody could see
([#142](https://github.com/lmarqs/terraform-ui/issues/142)).

## Consequences

- The plugin still owns the domain side of vertical navigation: which rows exist, their order, collapse, filtering,
  and the cursor. The panel owns where the window sits once the rows are known; it does not store the offset.
- The offset `RenderWindow` returns is the window's top row in both wrap modes, so a wrap toggle or a resize cannot
  read it in the wrong unit. With wrap on the window can start partway through that row; the next frame starts from
  the row's first line and scrolls again only as far as the cursor block needs.
- With wrap off, `RenderWindow` gives the same output as windowing in the plugin and calling `Render`.
- `Render` with wrap on also scrolls inside the slice it was given until the cursor row's first line shows, and
  counts rows outside the slice as one line each for the gutter. Without a cursor it keeps counting rows for the
  gutter, because those views scroll by rows and a narrower wrap would push lines past their reach. Risk, Blast Radius and Phantom use `RenderWindow`; the Plan, State and
  Workspace lists still pass slices to `Render`, so with wrap on their window and gutter thumb are only line-accurate
  inside the slice ([#159](https://github.com/lmarqs/terraform-ui/issues/159)).
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
