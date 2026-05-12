# Terraform Compatibility Decisions

**Status**: Active  
**Priority**: Critical  
**Date**: 2026-05-12

## Guiding Principle

Zero translation required. If a terraform user knows the terraform concept or flag, it works identically in tfui. We do not invent new names for existing terraform concepts.

---

## Decision Log

### D1: Flag Syntax — Accept both single and double dash

**Decision**: Support both `-target` (terraform native) and `--target` (Go convention).

**Reasoning**: Terraform uses single-dash for all flags (`-target`, `-var-file`, `-destroy`). Go/cobra uses double-dash. A terraform user's first instinct is single-dash. If it doesn't work, they leave.

**Implementation**: Pre-parser normalizes `-flag` → `--flag` before cobra processes args. Both forms documented in help.

**Breaking**: No.

---

### D2: No binary auto-detection

**Decision**: Remove existing tofu auto-detection. `terraform.bin` must be explicitly configured via `tfui.hcl` or `--terraform-bin` flag.

**Reasoning**: Auto-detection is magic. Magic surprises users. If someone has both tofu and terraform installed, they should explicitly choose. Explicit is better than implicit.

**Implementation**: 
- Delete `DetectBinary()` logic
- Error with clear message when `terraform.bin` not set
- `tfui init` asks which binary to use

**Breaking**: Yes. Users relying on tofu auto-detection must add `terraform { bin = "tofu" }` to config.

---

### D3: Configuration format — HCL, not YAML

**Decision**: Replace `tfui.yaml` with `tfui.hcl`. Delete all YAML support. No migration tool.

**Reasoning**: Every tool in the terraform ecosystem uses HCL. YAML signals "this doesn't belong here." The config file is the first thing a user sees when adopting the tool.

**Full architecture**: See `docs/_roadmap/hcl-config-architecture.md`.

**Breaking**: Yes. All existing `tfui.yaml` files stop working.

---

### D4: Rename "scope" → "chdir"

**Decision**: Rename the "scope" concept entirely to "chdir" across the codebase.

**Reasoning**: Terraform users know `-chdir` — it changes the working directory to a different root module. tfui's "scope" is the same concept (selecting a subdirectory within a monorepo). "Scope" has no precedent in the terraform ecosystem and forces users to learn a new term for a familiar concept.

**What changes**:
| Before | After |
|--------|-------|
| `--scope` flag | `--chdir` flag |
| `scope:` config block | `chdir:` config block |
| `scope.paths` config key | `chdir.members` config key |
| `plugins/scope/` package | `plugins/chdir/` package |
| `SessionKeyActiveScope` | `SessionKeyActiveChdir` |
| `SessionKeyActiveScopeAbs` | `SessionKeyActiveChdirAbs` |
| `scope.active` session key | `chdir.active` session key |
| `ScopeGuard` SDK utility | `ChdirGuard` SDK utility |
| `ProjectContext.Scopes` | `ProjectContext.Chdirs` |
| `ProjectContext.ActiveScope` | `ProjectContext.ActiveChdir` |
| `ProjectContext.ActiveScopeAbs` | `ProjectContext.ActiveChdirAbs` |
| `DiscoverScopes()` | `DiscoverChdirs()` |
| Plugin ID: `"scope"` | Plugin ID: `"chdir"` |
| Keybinding: `c` for scope | Keybinding: `c` for chdir |

**Breaking**: Yes. Config key changes, plugin references change.

---

### D5: Raw passthrough via `--`

**Decision**: Everything after `--` separator is passed directly to the terraform binary.

**Reasoning**: Standard Unix convention. kubectl, docker, git all use this. Provides escape hatch for terraform flags we haven't explicitly mapped yet. Users never get stuck.

**Implementation**: Cobra already splits at `--`. Collect remaining args into `ExtraArgs` field on PlanOptions/ApplyOptions.

**Breaking**: No.

---

### D6: --workspace flag

**Decision**: Add `--workspace` as a persistent CLI flag that selects workspace before any operation.

**Reasoning**: Common CI/CD pattern is `terraform workspace select prod && terraform plan`. tfui should support `tfui --workspace prod plan` without requiring env vars.

**Implementation**: Persistent flag that calls `WorkspaceSelect(name)` during service initialization.

**Breaking**: No.

---

### D7: All terraform flags supported

**Decision**: Support all terraform plan/apply flags as typed options, not just `-target`.

**Flags to support**:
- `-var` / `-var-file` (input variables)
- `-destroy` (destruction planning)
- `-replace` (force replacement)
- `-refresh-only` (drift check)
- `-parallelism` (concurrency)
- `-lock` / `-lock-timeout` (state locking)
- `-compact-warnings` (output formatting)
- `-input=false` (non-interactive)

**Implementation**: `PlanOptions` / `ApplyOptions` structs, mapped to `tfexec.*Option` constructors. `--var-file`/`--var` are tfui-aware (tracked, shown in TUI). `-- -flag` is opaque passthrough.

**Breaking**: Yes (internal). Service interface signature changes from `Plan(ctx, []string)` to `Plan(ctx, PlanOptions)`.

---

### D8: Terragrunt and Tofu — future roadmap only

**Decision**: No auto-detection for any binary. Create roadmap items for smart detection based on project files in the future.

**Future detection logic** (not implemented now):
- `terragrunt.hcl` present → suggest terragrunt
- `.opentofu.lock.hcl` present → suggest tofu
- `.terraform.lock.hcl` present → suggest terraform

For now: user explicitly sets `terraform { bin = "..." }`.

---

### D9: Config is optional

**Decision**: `tfui --terraform-bin terraform` works without any config file.

**Reasoning**: Zero-friction first run must be preserved. A user should be able to try tfui instantly without creating files. Config adds power but must not be a gate.

**Behavior**:
- No `tfui.hcl` + no `--terraform-bin` → error with actionable message
- No `tfui.hcl` + `--terraform-bin terraform` → works (single-module mode)
- `tfui.hcl` exists → `terraform.bin` required within it

**Breaking**: No.

---

### D10: Explicit chdir members (no globs)

**Decision**: `chdir.members` is an explicit list of paths. No glob patterns in runtime config.

**Reasoning**: Globs surprise users when new directories appear. Reading the config should tell you exactly what your project contains. Explicit is predictable, auditable, git-diffable. Globs are only used during `tfui init` wizard for discovery — the output is always an explicit list.

**Breaking**: Yes. Users with `scope.paths = ["modules/*"]` must list members explicitly.

---

### D11: Split config files (root + per-chdir)

**Decision**: Root `tfui.hcl` + optional per-chdir `tfui.hcl` overrides.

**Reasoning**: Lowest coupling. Each module owns its config. Scales to 50+ modules without bloating a single file. Clean git blame per module. Follows Cargo workspace model.

**Full architecture**: See `docs/_roadmap/hcl-config-architecture.md`.

---

### D12: 3-tier override model

**Decision**: Locked fields (root-only), inheritable defaults, child-only fields. Parser enforces boundaries.

**Reasoning**: Inspired by Cargo (locked profiles), Gradle (graduated enforcement), Kustomize (identity vs behavior). Principle: "If child A changes this, could child B break?" → locked.

**Full architecture**: See `docs/_roadmap/hcl-config-architecture.md`.

---

### D13: Workspace blocks in child files only

**Decision**: Per-workspace config lives in child `tfui.hcl` files, not at root level.

**Reasoning**: Avoids 2D override matrix (root × workspace × chdir). Workspace blocks solve terraform's biggest pain point (per-workspace var-files) without adding root-level complexity. Child top-level applies to all workspaces; workspace blocks override.

---

### D14: Var-file paths relative to declaring file

**Decision**: Paths in `var_file "path" {}` resolve relative to the directory containing the `tfui.hcl` that declares them.

**Reasoning**: Same as terraform's own `-var-file` resolution (relative to CWD). Root var-files resolve from project root. Child var-files resolve from child dir. Intuitive, predictable.

---

### D15: Single-module projects don't need chdir block

**Decision**: If no `chdir` block exists in root config, tfui operates in project root directly. No picker shown.

**Reasoning**: Simple projects shouldn't need extra config. A lone `terraform { bin = "terraform" }` should be a complete valid config.

---

## Terminology Alignment

| terraform concept | tfui term (old) | tfui term (new) | Notes |
|-------------------|----------------|-----------------|-------|
| `-chdir` | scope | chdir | Full rename |
| root module | scope/project | chdir target | Directory with .tf files |
| `-target` | pin | pin | Keep — pins are richer (persistent, bulk) |
| workspace | workspace | workspace | Already aligned |
| backend | (not shown) | backend | Future: show in header |
| `.tfvars` auto-load | (not supported) | (detected, shown in header) | Future |

**Note on "pin" vs "-target"**: We keep "pin" because it's a superset. Pins persist across views, enable batch operations, and scope both plan AND apply. They map TO `-target` flags, but the UX concept is richer. The header/hints should show: `[3 pinned → -target]` to make the terraform connection explicit.

---

## Migration Impact

All breaking changes (HCL config, no auto-detect, scope→chdir, explicit members) ship as a single BREAKING CHANGE commit. No deprecation period, no dual support, no migration tooling.

Users upgrading will need to:
1. Delete `tfui.yaml`, create `tfui.hcl`
2. Explicitly set `terraform { bin = "terraform" }` (or "tofu")
3. Replace `scope.paths` globs with explicit `chdir.members` list

This is acceptable because:
- The tool is pre-1.0
- The user base is small
- The long-term benefit of terraform alignment outweighs short-term churn
- A clear error message tells users exactly what to do

---

## Related Documents

- `docs/_roadmap/hcl-config-architecture.md` — Full HCL config architecture (schema, resolution, testing, implementation)
- `docs/_roadmap/hcl-config-migration.md` — Original YAML→HCL migration proposal (superseded by architecture doc)
- `docs/_roadmap/terraform-flags.md` — Original terraform flags proposal (absorbed into this plan)
