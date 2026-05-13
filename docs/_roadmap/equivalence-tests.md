---
title: Path Equivalence Tests
status: planned
priority: high
created: 2026-05-11
effort: medium
tags: [testing, quality, cli, tui]
depends_on: []
---

## Summary

Add integration tests that prove CLI and TUI paths produce identical terraform state for the same operation.

## Problem

The tool claims "all paths to the same operation produce identical outcomes" but this isn't verified by any test. A regression could silently break one path without the other, and we'd never know.

## Proposal

For each core operation, run it via CLI and verify the final .tfstate matches expected outcome:

```go
func TestEquivalence_Apply_CLI(t *testing.T) {
    dir := copyFixture(t, "apply-create")
    runTfui("plan", "--project", dir, "--ci")
    runTfui("apply", "--project", dir, "--ci")
    assertStateContains(t, dir, "local_file.result")
    assertFileExists(t, filepath.Join(dir, "out/result.txt"))
}

func TestEquivalence_Apply_Targeted(t *testing.T) {
    dir := copyFixture(t, "apply-targeted")
    runTfui("plan", "--project", dir, "--ci", "--target", "local_file.alpha")
    runTfui("apply", "--project", dir, "--ci")
    assertStateContains(t, dir, "local_file.alpha")
    assertStateNotContains(t, dir, "local_file.beta")
}

func TestEquivalence_StateRm_CLI(t *testing.T) {
    dir := copyFixture(t, "state-ops")
    runTfui("state", "rm", "local_file.one", "--project", dir)
    assertStateNotContains(t, dir, "local_file.one")
    assertStateContains(t, dir, "local_file.two")
}
```

## Future: Macro Equivalence

When/if macros support `--project` (real terraform execution), add:
```go
func TestEquivalence_Apply_CLIvsMacro(t *testing.T) {
    state1 := runViaCLI(t, "apply-create")
    state2 := runViaMacro(t, "apply-create", applyTape)
    assertStatesEqual(t, state1, state2)
}
```

This requires macros to support live terraform execution (not just static fixtures).

## Status

Partial coverage exists today:
- `TestApply_CreateFixture_SilentMode` verifies CLI apply creates the file
- `TestApply_Targeted_OnlyAppliesTarget` verifies CLI targeting works
- `TestState_Rm_RemovesResource` verifies CLI state rm works

Missing:
- No macro-vs-CLI comparison (macros are read-only today)
- No multi-path verification for the same fixture
