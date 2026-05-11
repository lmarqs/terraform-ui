---
title: Yank to Filter
status: planned
priority: medium
created: 2026-05-11
effort: small
tags: [ux, state, inspect]
depends_on: []
---

## Summary

From the inspect/detail view, allow the user to yank a value (e.g. a UUID, ARN, or name) directly into the filter. Solves cross-referencing resources that reference each other by opaque IDs.

## Problem

Terraform state encodes relationships by opaque ID (e.g. `member_id: "84a834b8-..."`). To find all resources referencing a user, the current workflow requires memorizing the UUID, escaping inspect, and manually typing it into filter. This is the kind of friction that sends users to `grep`.

## Design

In inspect view, a key (e.g. `y`) yanks the value under cursor into the filter:

```
Inspect: aws_identitystore_user.this["ronaldo.brisa"]
  user_id: "84a834b8-9081-7010-a941-45085221bc30"
                    ^ cursor here

Press `y` → exits inspect → filter is set to "84a834b8"
→ all resources referencing ronaldo appear
```

### UX Flow

1. User inspects a resource (Enter)
2. User navigates to a field value
3. Press `y` → value is extracted and set as the active filter query
4. View returns to filtered list showing all resources containing that value

### Open Questions

- What counts as "the value under cursor"? Full JSON value? Just the string content?
- Should it yank the full value or a prefix (UUIDs are long)?
- Should it append to existing filter or replace?
- Does the inspect view need line-level cursor navigation first?

## Precedent

- vim: `y` yanks text, `/` pastes into search with `ctrl+r"`
- k9s: no equivalent (no cross-reference)
- lazygit: copy SHA, filter by SHA in other views
