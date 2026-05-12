---
title: Wire New Macro Tapes into Integration Tests
status: planned
priority: high
created: 2026-05-11
effort: small
tags: [testing, macros]
depends_on: []
---

## Summary

8 new macro tape files exist in `tests/fixtures/tapes/` but are not yet registered in `tests/integration/macro_test.go`. Wire them in.

## Tapes to Register

| Tape | Required flags | Tests |
|------|---------------|-------|
| `plan_expand.tape` | `--plan plan.json` | Expand attribute diffs in plan |
| `plan_pin.tape` | `--plan plan.json` | Pin resource shows `*` marker |
| `plan_to_apply.tape` | `--plan plan.json` | Plan→Apply transition works |
| `state_browse.tape` | `--state state.json` | State plugin shows resources |
| `state_inspect.tape` | `--state state.json` | Detail view shows attributes |
| `state_tree.tape` | `--state state.json` | Tree mode toggle |
| `apply_idle.tape` | `--plan plan.json` | Apply shows idle message |
| `navigate_all.tape` | `--plan plan.json --state state.json` | Full plugin navigation cycle |

## Implementation

Add test cases to `tests/integration/macro_test.go`:

```go
{name: "plan expand diffs", tape: planExpandTape, args: []string{"--plan", planFixture}, wantExit: 0},
{name: "plan pin resource", tape: planPinTape, args: []string{"--plan", planFixture}, wantExit: 0},
{name: "plan to apply transition", tape: planToApplyTape, args: []string{"--plan", planFixture}, wantExit: 0},
{name: "state browse", tape: stateBrowseTape, args: []string{"--state", stateFixture}, wantExit: 0},
{name: "state inspect", tape: stateInspectTape, args: []string{"--state", stateFixture}, wantExit: 0},
{name: "state tree mode", tape: stateTreeTape, args: []string{"--state", stateFixture}, wantExit: 0},
{name: "apply idle message", tape: applyIdleTape, args: []string{"--plan", planFixture}, wantExit: 0},
{name: "navigate all plugins", tape: navigateAllTape, args: []string{"--plan", planFixture, "--state", stateFixture}, wantExit: 0},
```

Either inline tape content or read from file paths.
