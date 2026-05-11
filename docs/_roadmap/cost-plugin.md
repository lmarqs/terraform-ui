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

Integrate Infracost to show cost impact of planned changes directly in the plan view.

## Problem

Users review plans without knowing the cost impact. They switch to a separate tool (Infracost CLI or web) to check costs. This breaks the review flow.

## Design (sketch)

- New plugin: `plugins/cost/`
- Calls `infracost breakdown --path .` or parses existing Infracost JSON output
- Decorates plan changes with monthly cost delta
- Summary shows total cost change
