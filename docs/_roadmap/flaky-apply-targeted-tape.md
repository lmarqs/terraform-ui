---
title: Fix flaky apply_targeted.tape macro test
status: planned
priority: high
created: 2026-05-13
effort: small
tags: [ci, testing, bug]
---

## Summary

The `apply_targeted.tape` macro test consistently times out in CI waiting for "Apply plan" text to appear in the view. This has been failing since at least commit `bc6c3e3` and is unrelated to the CLI restructuring.

## Symptom

```
=== apply_targeted.tape ===
Error: line 6: timeout waiting for view to contain "Apply plan"
FAIL: 1 tape(s) failed
```

The tape navigates: plan → pin resource → switch to apply plugin. The timeout suggests the apply plugin's view doesn't render "Apply plan" within the 5s default wait.

## Possible Causes

- Race condition between plan completion and apply plugin activation
- Macro driver timing — the `wait view` timeout may be too short for CI environments
- Apply plugin requires plan data that hasn't loaded yet when navigated to immediately after pinning

## Files

- `tests/fixtures/tapes/apply_targeted.tape`
- `plugins/apply/` (view rendering)
- `internal/macro/driver.go` (wait timeout)
