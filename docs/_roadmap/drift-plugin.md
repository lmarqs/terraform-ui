---
title: Drift Detection Plugin
status: idea
priority: medium
created: 2026-05-11
effort: large
tags: [plugin, monitoring]
depends_on: []
---

## Summary

Detect and surface infrastructure drift — resources that changed outside of terraform.

## Need

Teams discover drift only when running `terraform plan` manually. By then:
- Changes have compounded (multiple resources drifted independently)
- Root cause is harder to find (who changed what, when?)
- Surprise in the plan output ("I didn't change that, why is it updating?")

Current workaround: run `tfui plan` periodically and eyeball the output.

## Expected UX

- Home screen shows drift badge: `drift (3)` — 3 resources have drifted
- Press `D` to enter drift plugin
- List view shows drifted resources with what changed
- Detail view shows attribute diff (expected vs actual)
- `ctrl+r` refreshes (re-runs plan to check current drift)
- Configurable check interval in tfui.hcl (e.g., `drift: { interval: 30m }`)

## Advantages

- Catches unmanaged changes before they cause production issues
- Surfaces "who changed this?" questions early
- Integrates naturally with existing plan infrastructure (drift IS a plan with no code changes)
