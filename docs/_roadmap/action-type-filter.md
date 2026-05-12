---
title: Action Type Filter (+, ~, -, -/+)
status: planned
priority: low
created: 2026-05-11
effort: small
tags: [ux, plan, filter]
depends_on: []
---

## Summary

Allow filtering plan changes by action type using the same symbols used in the plan display: `+` (create), `~` (update), `-` (delete), `-/+` (replace).

## Problem

In a plan with 30 changes, you often want "show me only the deletes" or "show me only the creates." Currently you must scroll through everything.

## Proposal

In the Plan view filter mode (`/`):
- Type `+` → show only creates
- Type `~` → show only updates
- Type `-` → show only deletes
- Type `-/+` → show only replacements

These are special-cased: if the entire filter text is exactly one of these symbols, it filters by action type instead of fuzzy address matching. Any other text uses normal fuzzy matching.

## Precedent

- vim quickfix: filter by type
- fzf: prefix syntax for special matching modes
- k9s: filter pods by status

Not novel — uses existing symbols the user already sees in the plan output.

## Implementation

In Plan plugin's filter logic:
```go
switch filter {
case "+":
    // filter to ActionCreate only
case "~":
    // filter to ActionUpdate only
case "-":
    // filter to ActionDelete only
case "-/+", "+/-":
    // filter to ActionReplace only
default:
    // normal fuzzy address matching
}
```

## UX

- Filter mode activated by `/` (same as state)
- Typing `+` immediately shows only creates
- Typing `+` then more chars switches to fuzzy mode ("+aws" = fuzzy for "+aws")
- `Esc` clears filter
- Hint bar shows: `+ create  ~ update  - delete  -/+ replace` when filter is empty
