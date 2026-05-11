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

A plugin that periodically runs `terraform plan` and reports drift between actual infrastructure and desired state. Shows which resources have changed outside of terraform.

## Problem

Teams discover drift only when running plan manually. By then, changes may have compounded. A background drift check would surface issues early.

## Design (sketch)

- New plugin: `plugins/drift/`
- Configurable interval (e.g., every 30m)
- Stores last-known drift state in session
- Badge/count on home screen showing drift count
- Detail view shows drifted resources with diff
