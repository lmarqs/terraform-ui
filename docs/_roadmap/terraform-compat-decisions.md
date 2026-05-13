---
title: Terraform Compatibility Decisions
status: completed
priority: critical
created: 2026-05-12
completed: 2026-05-12
effort: large
tags: [architecture, config, breaking]
---

# Terraform Compatibility Decisions

**Status**: Completed  
**Date**: 2026-05-12

## Guiding Principle

Zero translation required. If a terraform user knows the terraform concept or flag, it works identically in tfui. We do not invent new names for existing terraform concepts.

tfui is a thin wrapper — it never overrides library behavior.

## Decisions Implemented

### D1: Accept both single and double dash flags

`normalizeArgs()` in `cmd/tfui/normalize.go` rewrites terraform-style single-dash flags to double-dash for cobra. Users can write `-target=X` or `--target=X`.

### D2: No binary auto-detection

`DetectBinary()` only exists for the `tfui init` wizard. Normal operation passes `terraform.bin` from config to terraform-exec as-is. Empty = let terraform-exec handle/error naturally.

### D3: HCL format, not YAML

Config file is `tfui.hcl`. All YAML support removed. Config struct has no yaml tags.

### D4: Rename "scope" → "chdir"

Plugin renamed from `plugins/scope/` to `plugins/chdir/`. Session keys: `chdir.active`, `chdir.active_abs`, `chdir.count`. `ScopeGuard` → `ChdirGuard`.

### D5: Raw passthrough via `--`

`splitPassthrough()` extracts args after `--` before normalization. Stored in `Config.ExtraArgs`, threaded through `PlanOptions.ExtraArgs` / `ApplyOptions.ExtraArgs`.

### D6: --workspace flag

Persistent flag on root command. Workspace plugin sets `SessionKeyWorkspace` on switch.

### D7: PlanOptions/ApplyOptions

Service interface changed from `Plan(ctx, targets)` to `Plan(ctx, PlanOptions)`. Options carry: targets, var-files, vars, replace, destroy, refresh-only, parallelism, lock, lock-timeout, extra-args.

### D8: Config is purely optional

No config file required. Empty file valid. No mandatory blocks. Everything works with zero config (standalone mode).

### D9: No walk-up directory discovery

`LoadRoot(dir)` checks CWD only. No traversal. `--project` overrides CWD for config location.

### D10: Explicit chdir members

`chdir.members` is an explicit list. No globs, no auto-discovery. The chdir picker renders this list directly.

### D11: Split config files (root + per-member)

Root `tfui.hcl` at project root. Optional child `tfui.hcl` per chdir member. Locked fields prevent children from overriding root concerns.

### D12: 3-tier override model

Locked (root only) → Inheritable (defaults block) → Child-only (workspace blocks, var_file, plugin overrides).

### D13: Workspace blocks in child files only

`workspace "name" {}` blocks appear in child configs. Glob matching supported (`dev-*`). Exact match beats glob.

### D14: Var-file concatenation (not replacement)

All levels concatenate: root defaults → child top-level → workspace → CLI. For vars (map), later level wins for same key.

### D15: Two operational modes

- **Standalone** (no tfui.hcl in CWD, no --project): just a TUI over terraform. `--chdir` = raw passthrough.
- **Project** (tfui.hcl found or --project): full config resolution. `--chdir` validates against members.

### D16: Live workspace re-resolve

Switching workspace triggers `Resolve()` again. Plan/apply read resolved config from session at call time via `BuildPlanOptions()`/`BuildApplyOptions()`.
