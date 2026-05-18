---
title: ContentPanel — converge on pure formatting contract
status: planned
priority: medium
created: 2026-05-17
effort: medium
tags: [sdk, architecture, rendering]
depends_on: []
---

## Summary

Remove the `BuildRow` generator path from `ContentPanel`, making it a pure formatting layer that receives pre-windowed `Rows` and applies horizontal concerns only. Extract the "compute visible height budget" helper that every plugin duplicates.

## Scope

- Remove `BuildRow func(index int) string` from `RenderParams`
- Remove internal iteration logic from `Render()` (the `for i := params.ViewOffset` loop that pulls items)
- Add `HeightBudget(total, ...deductions int) int` SDK helper so plugins don't re-derive `maxVisible := height - filterHeight - summaryHeight - ActionsBarHeight`
- Migrate `plugins/plan/` flat-list path from `BuildRow` to pre-windowed `Rows` (tree path already uses `Rows`)
- Remove `ViewOffset` from `RenderParams` — it becomes redundant when the panel doesn't window
- Deprecate `ContentWidth` public method — the panel should compute gutter internally without callers needing to pre-query (removes the consistency hazard where `ContentWidth(w,h,n)` params can diverge from `Render(w,h,n)`)
- Update `Render()` to accept only: `Rows []string`, `Width int`, `Height int`, `TotalItems int`, `Cursor int` (relative to Rows, not absolute)
- Deprecate and migrate `pkg/sdk/viewport.go` — InspectFrame should use ContentPanel + explicit vertical scroll state
- Remove dead `wrapLines()` helpers from plan/state plugins
- Update all tests to reflect the narrower contract

## Why

The current API has two input paths (`BuildRow` vs `Rows`) with different mental models. `BuildRow` makes the panel responsible for vertical windowing (iterating from ViewOffset), which contradicts the stated design: "horizontal-only." This forces plugin authors to understand when each path applies. A single `Rows`-only contract makes the panel's responsibility obvious: "I format what you give me."

The `ContentWidth` pre-query is a consistency hazard — callers must pass matching `(width, height, totalItems)` to both `ContentWidth()` and `Render()`. If the panel computed content width internally and exposed it *after* rendering (or took the width hint from the rows themselves), the hazard disappears.

## Success Criteria

- `RenderParams` has exactly 5 fields: `Rows`, `Width`, `Height`, `TotalItems`, `Cursor`
- No plugin computes `height - X - Y - Z` manually — they use a shared helper
- `pkg/sdk/viewport.go` is deleted or reduced to a 10-line wrapper
- The panel's role is stateable in one sentence: "Formats pre-windowed rows with ANSI-safe truncation, horizontal scroll, cursor highlight, and scroll gutter."
