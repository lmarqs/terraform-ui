---
layout: default
title: "ADR-0015: ContentPanel — horizontal-only rendering pipeline"
grand_parent: Development
parent: Architecture
nav_order: 0015
description: Decision to split rendering concerns by axis, making ContentPanel own horizontal layout without vertical navigation
---

# ContentPanel — horizontal-only rendering pipeline

ContentPanel (`pkg/sdk/ui/content_panel.go`) owns the **horizontal rendering pipeline** for scrollable content areas: ANSI-safe truncation, horizontal scroll, wrap toggle, cursor highlight, and scroll gutter alignment. It deliberately refuses to own vertical navigation — cursor position, viewport offset, and collapse-aware traversal remain with the caller (tree, cursor, or plugin).

This axis split exists because vertical navigation requires domain semantics that vary per plugin (tree collapse, pin filtering, fuzzy scoring), while horizontal concerns are mechanical and identical everywhere. Bundling both axes into one component forces plugins into either using an over-opinionated Model (like `bubbles/list`) or duplicating the horizontal logic they can't use from it.

The panel is a **formatting layer**, not a Model. It has no `Init`/`Update`/`View` lifecycle — plugins call `HandleKey()` for horizontal input and `Render()` for output. This makes it composable: plugins combine panel output with actions bars, filter lines, and summary sections however they need.

## Considered Options

### Extend the existing Viewport (`pkg/sdk/viewport.go`)

The Viewport owns both axes and uses byte-level `len(line)` for width/truncation. This breaks ANSI sequences and misaligns gutters with styled content. Extending it to be ANSI-safe would fix rendering but not the ownership problem — tree-based plugins can't delegate vertical scroll to a component that doesn't understand collapse state.

### Use `bubbles/viewport` from the Charm ecosystem

Charm's viewport owns the full `tea.Model` lifecycle, uses byte-level truncation, has no cursor concept, and doesn't render a scroll gutter. Adopting it would require fighting against its ownership model for every plugin that has a tree, filter, or custom vertical navigation — which is all of them.

### Make the tree own all rendering

The tree already has `RenderLeaf`/`RenderBranch`/`SelectedStyle` callbacks. Pushing gutter, hscroll, and wrap into the tree would consolidate rendering for tree-mode views but force flat-list views and detail/inspect views to either use a tree (wrong abstraction) or duplicate the rendering pipeline.

### Stateless pure-function approach

An earlier iteration had `ContentPanel` as a set of pure rendering functions with no stored state. This worked for output but forced every plugin to declare `hScroll int` and `wrapMode bool` fields and duplicate the pan/toggle logic. Making the panel stateful for horizontal concerns only eliminated ~30 lines of identical code per plugin without taking ownership away from anything that should own itself.

## Consequences

The existing `Viewport` (`pkg/sdk/viewport.go`) is now semantically superseded for list views. It retains value only for detail/inspect views where the plugin has no cursor and wants a simple both-axes component. Long-term, it should either delegate its horizontal pipeline to ContentPanel or be deprecated in favor of ContentPanel + explicit vertical scroll (which is what InspectFrame already does manually).

The `BuildRow` generator path exists as a convenience but leaks vertical windowing into a horizontal-only abstraction. Callers using `BuildRow` trust the panel to iterate from `ViewOffset` — mixing concerns. The architecturally pure contract is `Rows`-only: the caller always provides the visible window, and the panel is purely a formatting/gutter layer. `BuildRow` should be treated as a convenience shortcut, not the canonical path.
