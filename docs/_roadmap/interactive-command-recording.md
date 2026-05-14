---
title: Interactive Command Recording (dry-run mode)
status: planned
priority: medium
created: 2026-05-14
effort: small
tags: [cli, ux, macro, architecture]
depends_on: [service-layer-simplification]
---

## Summary

Allow users to run the full interactive TUI without executing mutations, printing recorded commands to stdout on exit. This is the keyboard-input counterpart to `--macro` (tape-input).

## Need

A user wants to explore their infrastructure interactively, plan changes, select targets, queue operations — then review the generated commands before executing. Currently they must either: commit to real execution, or write a macro tape in advance. There's no "try it interactively, execute later" workflow.

## Decision: Flag Design

The feature needs a CLI flag. Two approaches:

### Option A: `--dry-run`

```bash
tfui --dry-run
tfui --dry-run | sh
```

**Pros:**
- Universally understood concept
- Discoverable (users guess it exists)
- Short, clean

**Cons:**
- Terraform doesn't have `--dry-run` (terraform's dry-run IS `plan`)
- Could confuse terraform users: "isn't plan already dry-run?"
- Implies "show what would happen" — ours is "record what I would do"

### Option B: `--macro` without tape (empty macro mode)

```bash
tfui --macro
tfui --macro | sh
```

`--macro` with a path = tape input. `--macro` without a path = keyboard input, same recording behavior.

**Pros:**
- No new flag — extends existing concept
- Consistent mental model: `--macro` means "record mode" regardless of input source
- The macro docs already explain the stdout contract

**Cons:**
- Overloading: `--macro` with and without an argument have different input behaviors
- Less discoverable than `--dry-run`
- `--macro` sounds like "playback", not "record"

### Option C: `--record`

```bash
tfui --record
tfui --record | sh
```

**Pros:**
- Accurately describes the behavior (recording commands)
- No semantic collision with terraform concepts
- Pairs naturally with `--macro` (record vs playback)

**Cons:**
- Less familiar than `--dry-run`
- Another novel flag to document

### Option D: `--emit`

```bash
tfui --emit
tfui --emit | sh
```

**Pros:**
- Describes the stdout behavior precisely
- Novel, no baggage

**Cons:**
- Obscure, not self-explanatory
- Users won't guess it exists

## Recommendation

Decide after implementing the service-layer-simplification. The implementation is the same regardless of flag name — it's purely a UX/naming decision. Test with real users if possible.

## Implementation

Depends on service-layer-simplification. Once `MacroService` exists as a standalone type:

```go
// In runTUI, after service creation:
if dryRun {
    svc = terraform.NewMacroService(binary, dir, cache)
}
// After TUI exits:
if dryRun {
    for _, cmd := range svc.Commands() {
        fmt.Println(cmd.String())
    }
}
```

~5 lines of glue code regardless of which flag name is chosen.
