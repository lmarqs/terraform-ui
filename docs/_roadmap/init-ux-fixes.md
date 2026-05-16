---
title: Init Plugin UX Fixes
status: in-progress
priority: high
created: 2026-05-15
effort: medium
tags: [ux, cli, tui, plugin, init]
depends_on: []
---

## Summary

UX audit of the init plugin found 17 issues across TUI and CLI surfaces. This roadmap tracks fixes item by item.

## Critical

### 1. [TUI] Empty stack after form submit makes Loading/Done/Error non-interactive

- **File**: `plugins/init/init.go`
- **Problem**: `p.stack.Reset()` emptied the stack before entering Loading. The app routes keys through `Stack.Update()` for Stackable plugins — empty stack meant immediate deactivation on any keypress.
- **Fix**: Introduced `resultFrame` that lives on the stack during execution. Form stays as root frame. On error, Enter pops resultFrame back to form. On success, auto-deactivates.
- **Status**: done

### 2. [TUI] `Hints()` method is dead code

- **File**: `plugins/init/init.go`
- **Problem**: App checks Stackable first, calls `Stack().Hints()`. Empty stack returned nil.
- **Fix**: Removed plugin-level `Hints()`. ResultFrame provides its own hints via the stack. Form provides its own via `FormFrame.Hints()`.
- **Status**: done

### 3. [TUI] "Esc cancel" hint shown during Loading but cancel is not implemented

- **File**: `plugins/init/result_frame.go`
- **Problem**: No cancel context, no abort mechanism.
- **Fix**: Loading state shows `HintSetBack` (only `q back`). No false "Esc cancel" promise. Cancel can be added later if needed.
- **Status**: done

### 4. [CLI] Stderr success message not gated on `--ci`

- **File**: `cmd/tfui/cli.go`
- **Problem**: Success message printed unconditionally, violating the I/O contract.
- **Fix**: Gated on `!ciMode`. Also changed message to active voice: "Initialized successfully."
- **Status**: done

### 5. [CLI] `--chdir` not respected

- **File**: `cmd/tfui/cli.go`
- **Problem**: Used `cfg.WorkingDir()` instead of `effectiveWorkDir(cfg)`.
- **Fix**: Now uses `effectiveWorkDir(*cfg)` + `validateChdir(*cfg)` like plan/apply.
- **Status**: done

### 6. [CLI] `tfui init` missing from `docs/cli-io-contract.md`

- **File**: `docs/cli-io-contract.md`
- **Problem**: The stdout/stderr/exit contract was unspecified.
- **Fix**: Added init rows to both terraform baseline and tfui I/O tables.
- **Status**: done

## Warning

### 7. [TUI] `Enter` used for "re-run" violates spec

- **File**: `plugins/init/result_frame.go`
- **Problem**: `Enter` must always mean inspect or confirm — "re-run" is neither.
- **Fix**: Redesigned flow eliminates "re-run" concept entirely. On error, `Enter` means "acknowledge" (back to form). On success, auto-returns home. Enter is used as confirm/acknowledge, consistent with the spec.
- **Status**: done

### 8. [TUI] Redundant re-run mechanisms

- **File**: `plugins/init/init.go`
- **Problem**: Done state showed both `Enter re-run` and `^r refresh`.
- **Fix**: No re-run mechanisms needed. Success auto-returns. Error has single path: Enter → back to form (pre-filled). No ctrl+r.
- **Status**: done

### 9. [TUI] Missing `Busy` interface

- **File**: `plugins/init/init.go`
- **Problem**: `terraform init` can acquire backend locks, but `:q` during a running init will force-quit without warning.
- **Fix**: Implemented `Busy()` — checks if the result frame is in Loading state.
- **Status**: done

### 10. [CLI] Spinner uses 2-condition gate instead of 3

- **File**: `cmd/tfui/cli.go`
- **Problem**: Uses `!ciMode && isStderrTTY()` instead of the documented 3-condition pattern.
- **Fix**: Won't fix — init has no JSON output mode and never will (it produces no structured data). The third condition is only meaningful when `--json` is a valid flag.
- **Status**: won't fix

### 11. [CLI] Undocumented flags in normalization

- **File**: `cmd/tfui/normalize.go`
- **Problem**: Init-related flags added to normalization maps but not documented in docs/cli-ux.md.
- **Fix**: Updated docs/cli-ux.md §4 flag normalization sets with all init-related flags.
- **Status**: done

### 12. [CLI] Missing `--backend` cobra flag registration

- **File**: `cmd/tfui/cli.go`
- **Problem**: Normalization handled `backend` but cobra rejected it as unrecognized.
- **Fix**: Registered `--backend` flag (defaults to true, `-backend=false` disables backend init).
- **Status**: done

### 13. [CLI] Missing `--chdir` validation

- **File**: `cmd/tfui/cli.go`
- **Problem**: Other commands call `validateChdir(cfg)` but init did not.
- **Fix**: Added `validateChdir(*cfg)` call (resolved together with #5).
- **Status**: done

## Info

### 14. [TUI] FormFrame uses lowercase "esc"

- **File**: `pkg/sdk/frames/form.go`
- **Problem**: SDK convention is "Esc" (capitalized). Shared frame issue affecting init and chdir.
- **Fix**: FormFrame now uses `sdk.HintCancel` constant instead of inline strings. Also removed dead `q` handling from FormFrame (app intercepts it globally).
- **Status**: done

### 15. [TUI] No terraform output captured

- **File**: `plugins/init/init.go`
- **Problem**: Shows only a one-line message. Roadmap spec (`terraform-init.md`) calls for scrollable output view.
- **Fix**: Capture full terraform output and display in scrollable view. Requires `sdk.Service.Init()` to return output.
- **Status**: pending (feature gap)

### 16. [TUI] `q` handling in FormFrame is dead code

- **File**: `pkg/sdk/frames/form.go`
- **Problem**: App intercepts `q` globally before it reaches the stack.
- **Fix**: Removed `"q"` from FormFrame's key handler (resolved with #14).
- **Status**: done

### 17. [CLI] Success message style

- **File**: `cmd/tfui/cli.go`
- **Problem**: "Terraform has been successfully initialized." uses passive voice.
- **Fix**: Changed to "Initialized successfully." (resolved with #4).
- **Status**: done
