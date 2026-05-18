---
layout: default
title: "ADR-0015: Axis-split box model for content rendering"
grand_parent: Development
parent: Architecture
nav_order: 0015
description: The project's rendering paradigm — axis separation for all scrollable content surfaces
---

# Axis-split box model for content rendering

This project separates rendering of scrollable content into axis-specific concerns. **Horizontal formatting** (truncation, scrolling, wrapping, gutter alignment) is mechanical and identical across all views. **Vertical navigation** (cursor position, viewport windowing, collapse-aware traversal) requires domain semantics that vary per component. No single abstraction may own both axes.

This paradigm applies to all components that render scrollable content: list views, tree views, detail/inspect views, and any future scrollable surface.

## Box Model

```
┌──────────────────── Border (app) ─────────────────────┐
│  ┌──── Horizontal Formatting Layer ────────────────┐  │
│  │  Row content (truncated/wrapped/scrolled)  │ G  │  │
│  │  Row content                               │ u  │  │
│  │  Row content [cursor highlighted]          │ t  │  │
│  │  Row content                               │ t  │  │
│  │  Row content                               │ e  │  │
│  │                                            │ r  │  │
│  └─────────────────────────────────────────────────┘  │
│  Actions bar (sibling — not owned by the layer)       │
└───────────────────────────────────────────────────────┘
  Hint bar (footer — outside border)
```

Layers are composed vertically. Each layer's boundaries are fixed — components never reach into a sibling's or parent's rendering space.

## Contracts

### Horizontal formatting layer promises

- ANSI-safe truncation — never breaks escape sequences mid-stream
- ANSI-safe horizontal scrolling (pan left/right)
- ANSI-safe wrapping at visual character boundaries
- Scroll gutter rendering when content overflows the viewport
- Cursor row highlighting via caller-provided style
- Consistent width math: content width = total width minus gutter budget

### Horizontal formatting layer requires

- **Pre-windowed visible rows** — the caller decides WHAT is visible
- **Total item count** — for gutter thumb proportionality
- **Cursor index** — relative to the visible window, or absent
- **Scroll offset** — where this window sits in the full list (gutter positioning)

### Vertical layer (caller) owns

- Domain-aware traversal (collapse state, pin filtering, fuzzy scoring)
- Viewport windowing (which subset of items to show based on cursor and data structure)
- Cursor management (position, bounds, collapse-aware movement)
- Height budgeting (computing available space by deducting sibling elements)

## The Rule

**The component that decides WHICH rows are visible must not also decide HOW those rows are formatted for display.** These are always separate responsibilities.

## Design Principles

**No lifecycle, no autonomy.** The formatting layer has no Init/Update/View cycle. It acts only when called — rendering on demand and handling input when delegated. This makes it composable: callers combine its output with sibling elements in whatever layout they need.

**No dual-axis components.** No component in this project bundles both scroll axes into a single opinionated Model. This prevents the "framework trap" where plugins must fight against a component's lifecycle to implement domain-specific vertical behavior.

**Single input path.** The layer accepts pre-windowed rows. There is no generator or pull-based alternative. The caller always decides what to feed; the layer formats what it receives.

**Stateful only where universal.** The layer holds horizontal scroll offset and wrap-mode toggle — state that is identical across all callers. Vertical state remains external because its semantics vary (tree offset vs. flat cursor vs. detail scroll).

## Anti-patterns (do NOT introduce)

- Byte-level width math on styled content (`len(line)` instead of visual width)
- A single "viewport" owning both scroll axes
- Forcing callers to replicate horizontal state and pan/toggle logic
- Domain-specific rendering (e.g., tree connectors) inside the formatting layer
- Components reaching into a sibling's rendering space

## Considered Options

The core question is: **why can't a single component own both axes?**

### A shared component owns both axes

Fails because vertical navigation requires domain semantics that vary per plugin — tree collapse, pin filtering, fuzzy scoring, grouping. The component either becomes a god object that understands every domain's vertical logic (coupling plugins to its opinion), or plugins fight against it by overriding its vertical behavior and feeding it pre-windowed data anyway — at which point it's a formatting layer pretending to be a Model.

### The domain component owns both axes

Fails because horizontal formatting is identical everywhere and gets duplicated across every plugin. Each caller reinvents ANSI-safe truncation, gutter alignment, and pan state. Bugs multiply and fixes must be applied N times.

### Split by axis (chosen)

Vertical to the domain component (who knows what's visible), horizontal to a shared formatting layer (who knows how to render safely). Neither side has responsibility it cannot fulfill. The domain never struggles with ANSI sequences; the formatting layer never needs to understand collapse state.

### Stateless formatting functions (no shared state)

A variant of the split approach where horizontal formatting has no state. Works for output but forces every caller to declare horizontal scroll and wrap-mode fields and duplicate the pan/toggle logic identically. Making the formatting layer stateful for horizontal concerns only eliminates this duplication without taking ownership from the domain.

## Consequences

- All scrollable content surfaces use the same horizontal formatting layer
- The rendering pipeline is fixed once (ANSI safety, gutter alignment, truncation) and benefits all components
- New plugins or view types only implement vertical navigation — horizontal formatting is inherited
- The formatting layer can evolve independently of domain components (new truncation strategies, gutter styles, etc.) without touching plugin code

## Applicability

**Always applies** when a component renders scrollable content — whether it's a tree, a flat list, a detail/inspect view, or a future log-tail.

**Does not apply** to static views (e.g., a status message or a 3-line summary). If content doesn't scroll, there is no axis to split.

**For new vertical paradigms** (virtual scroll, paginated views, auto-scrolling logs): the vertical logic lives in the domain component. The formatting layer doesn't change — it still receives pre-windowed rows and formats them.
