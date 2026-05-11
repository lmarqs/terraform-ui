---
title: Macro CLI Integration
status: planned
priority: high
created: 2026-05-11
effort: medium
tags: [macro, cli, testing]
depends_on: [source-abstraction]
---

## Summary

Wire the existing macro engine to a `--macro` CLI flag so tape scripts can be executed from the command line — enabling CI visual regression testing and automated workflows.

## Need

The macro engine exists (`internal/macro/`) with a Driver and tape parser, but nothing invokes it from the CLI. Currently:

1. **Claude can't verify UI changes** — after modifying a plugin's View(), there's no way to programmatically check the result
2. **No CI visual regression** — layout breaks go undetected until a human looks at the TUI
3. **Users can't automate repetitive flows** — "check each workspace for drift" requires manual keypresses every time
4. **Plugin developers can't test interactively** — writing a new plugin requires running the TUI manually to verify

## Expected UX

```bash
# Run a tape file
tfui --macro ./tests/verify-plan.tape --plan ./plan.json

# CI pipeline: generate plan, verify rendering
terraform show -json tfplan.out > plan.json
tfui --plan ./plan.json --macro ./tests/check-risk.tape
# Exit code 0 = all assertions pass, 1 = failure

# Capture screenshots for golden file comparison
tfui --plan ./plan.json --macro ./tests/capture-views.tape
# Screenshots saved to paths specified in the tape file

# Pipe tape from stdin
echo "key p; wait ready; assert view create" | tfui --plan ./plan.json --macro -
```

**Exit codes (clear, actionable):**

| Code | Meaning | Example output |
|------|---------|----------------|
| 0 | All assertions passed | (silent) |
| 1 | Assertion failed | `FAIL line 5: assert view "create" — not found in view` |
| 2 | Syntax error in tape | `ERROR line 3: unknown command "wai" (did you mean "wait"?)` |
| 3 | Timeout | `TIMEOUT line 4: wait view "ready" — not found after 5s` |

**No TTY required.** Macro mode never opens a terminal — it drives the model directly via the Driver.

## Advantages

- **Claude becomes self-verifying** — write code, run macro, check output, iterate
- **CI catches visual regressions** — golden file diffs on every PR
- **Reproducible demos** — tape file generates consistent screenshots for docs
- **Low barrier** — tape DSL is 7 commands, learnable in 2 minutes

## Effort Justification

**Medium** because:
- Driver and tape parser already exist and are tested
- Main work: wiring to CLI flag + fixing the `WaitUntil` design flaw
- `WaitUntil` fix is the tricky part (single-threaded model needs command processing during wait)
- No new packages needed, just orchestration code in `cmd/tfui/`

## Design

```
--macro flag → Resolver.Resolve(uri) → ParseTape(bytes) → Runner.Execute(commands)
                                                              ↓
                                         NewDriver(app, width, height)
                                         driver.Init()
                                         for each command:
                                           driver.SendKey / WaitUntil / Assert / Screenshot
```

**Critical fix needed:** `WaitUntil` currently busy-waits but nothing changes the view (driver is single-threaded). Must process pending `tea.Cmd` results during the wait loop.

## Open Questions

- `--width` / `--height` flags? (default 80x24, affects golden file comparison)
- Built-in golden comparison (`assert screenshot ./golden/expected.txt`)?
- Timeout per command vs global timeout?

## Tasks

- [ ] Fix `WaitUntil`: execute pending commands in a loop until predicate or timeout
- [ ] Add `--macro` flag to root command
- [ ] Create `macro.Runner` (orchestrator: resolve → parse → drive)
- [ ] Error reporting with line numbers and context
- [ ] Exit codes (0/1/2/3)
- [ ] Integration test: macro against fixture plan
- [ ] `--width`/`--height` flags

## References

- Current macro engine: `internal/macro/driver.go`, `internal/macro/tape.go`
- Tape DSL docs: `docs/macro-language.md`
- VHS (charmbracelet) for prior art on tape file format
