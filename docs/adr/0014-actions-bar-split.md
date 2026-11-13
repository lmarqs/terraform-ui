---
layout: default
title: "ADR-0014: Actions bar — separate terraform mutations from UI hints"
grand_parent: Development
parent: Architecture
nav_order: 0014
seo:
  description: Decision to split terraform action keys into a dedicated actions bar inside the plugin frame
description: "Records why terraform action keys sit in an actions bar inside the plugin frame, apart from the hint bar."
audience: "Contributors adding a key binding or changing the bars"
---

# Actions bar — separate terraform mutations from UI hints

Terraform mutation keys (bare alpha: `d`, `t`, `T`, `e`, `m`, `n`, `a`, `A`, `!`) are separated from the hint bar into a dedicated **actions bar** rendered inside the bordered plugin frame. The hint bar (outside the border) shows only UI/navigation keys (`Enter`, `/`, `Space`, `^t`, `^r`, etc.).

The split follows a single rule: **bare key = actions bar, ctrl+key or punctuation = hint bar.** The modifier is the visual signal. This removes cognitive mixing — terraform verbs that change infrastructure live in a visually distinct zone from interface controls that change what you see.

The actions bar is an SDK rendering primitive owned by the plugin. It renders pinned to the bottom of the available frame space (after a blank separator line), styled as two-tone button chips (see Actions bar rules in [TUI UX](../reference/tui-ux.md)). Plugins decide when and whether to show it. Plugins with no terraform actions (output, validate, version) don't render one.

## Consequences

- A frame that shows the actions bar gives up two lines of its height to it (`ActionsBarHeight = 2`: the blank separator and the chips).
- A new terraform mutation key is a bare key and goes in the actions bar; a new UI key is ctrl+key or punctuation and goes in the hint bar.

## Considered Options

### Keep everything in a single hint bar

Status quo. Rejected — as the action set grows, the hint bar becomes a wall of undifferentiated text. Users can't quickly distinguish "what changes infrastructure" from "what changes the view." The cognitive load compounds because both categories compete for the same 1-line budget.

### Two-line hint bar (actions above, UI below)

Rejected — violates the rule that the hint bar is a single line (Layout Structure in [TUI UX](../reference/tui-ux.md)). Two lines create layout jitter when one category is empty. The actions bar inside the bordered frame is visually separated by the border itself, making the hierarchy clearer than stacking two text lines.

### App-level actions bar (rendered by the app, not the plugin)

Rejected — plugins own their frame content. An app-rendered actions bar would require the app to know which actions each frame supports, breaking the plugin encapsulation. The SDK provides the rendering primitive; plugins compose it in their `View()`.
