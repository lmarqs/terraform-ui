---
title: Cost Estimation Plugin (Infracost)
status: idea
priority: low
created: 2026-05-11
effort: medium
tags: [plugin, cost]
depends_on: []
---

## Summary

Show cost impact of planned changes directly alongside the plan view.

## Need

Users review plans without knowing the financial impact. They:
- Switch to Infracost CLI or web dashboard separately
- Lose context switching between "what changes" and "what it costs"
- Sometimes approve expensive changes unknowingly (e.g., upgrading instance type 10x)

Current workaround: run `infracost breakdown` in a separate terminal.

## Expected UX

- Plan view shows cost delta per resource: `+ aws_instance.web  create  +$42/mo`
- Summary line: `Plan: 1 to add (+$42/mo), 1 to change (+$8/mo), 0 to destroy`
- Detail view shows cost breakdown by component (compute, storage, network)
- Works with existing `--plan ./plan.json` (Infracost can analyze plan JSON)
- Gracefully hides when Infracost is not installed

## Advantages

- Cost visibility at decision time, not after the fact
- No context switching — cost is inline with the change
- Prevents costly mistakes ("this instance type costs $3000/mo not $30/mo")
