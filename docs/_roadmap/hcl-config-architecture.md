# HCL Configuration Architecture

**Status**: Designed (ready for implementation)  
**Priority**: Critical  
**Effort**: Large (1-2 weeks)  
**Date**: 2026-05-12  
**Depends on**: None (first phase of terraform compatibility)  
**Blocks**: All other terraform compatibility work

---

## Problem Statement

tfui uses `tfui.yaml` for configuration. This creates three adoption barriers:

1. **Format signal**: YAML says "generic DevOps tool." HCL says "terraform ecosystem." Every terraform-adjacent tool uses HCL (terraform, tofu, terragrunt, packer, vault, nomad, waypoint, boundary). YAML is the outlier.

2. **Feature limitation**: YAML cannot express the config hierarchy that real monorepo projects need — per-module variable files, workspace-specific overrides, ordered variable declarations.

3. **Magic binary detection**: The current auto-detection (`tofu` preferred over `terraform`) surprises users who have both installed. Configuration should be explicit.

---

## Research: How Other Tools Handle Hierarchical Config

### Tools Analyzed

| Tool | Config model | Inheritance | Locking | Discovery |
|------|-------------|-------------|---------|-----------|
| **Terragrunt** | File-per-directory, `include` blocks | Merge-based (3 strategies: shallow, deep, no_merge) | None — child controls everything | `find_in_parent_folders()` walks up |
| **Gradle** | `settings.gradle` + per-project `build.gradle` | Injection (allprojects) or opt-in (convention plugins) | Graduated: PREFER_PROJECT → PREFER_SETTINGS → FAIL_ON_PROJECT | Explicit `include()` |
| **Nx** | Root `nx.json` + per-project `project.json` | targetDefaults apply unless overridden | Infrastructure fields (cache, parallel) locked to root | Plugin scans for config files |
| **Bazel** | `WORKSPACE` + per-package `BUILD` | None — deliberate isolation | All inheritance is explicit `load()` | BUILD file presence |
| **Cargo** | Root `Cargo.toml` [workspace] + member `Cargo.toml` | Opt-in per field (`key.workspace = true`) | `[profile]`, `[patch]`, `[replace]` workspace-only | `members` globs |
| **Docker Compose** | Base + override files | Scalar: replace. Lists: concatenate. Maps: merge per-key | None | Explicit file list |
| **Kustomize** | Base + overlays | Overlay owns behavior, base owns identity | Identity fields (apiVersion, kind, name) immutable | Explicit `resources:` list |

### Key Findings

**1. Lock what's shared / singular**  
If there's a shared resource (Cargo.lock, Nx cache, Gradle dependency graph), the config controlling it must be singular. Multiple writers to a shared resource create conflicts.

**2. Lock infrastructure, free behavior**  
Orchestration (binary, cache, parallelism defaults) is root-only. What individual units DO (their vars, their plugins, their risk levels) is customizable.

**3. Lock identity, free configuration**  
Identity fields (what projects exist, project structure) are root-controlled. Behavioral fields (how they're configured) are child-controlled.

**4. Match enforcement to trust model**
- High-trust (Terragrunt): No locks, everything overridable
- Medium-trust (Gradle): Graduated enforcement
- Low-trust (Cargo, Nx): Hard locks at infrastructure level

**5. Inheritance should be explicit, not surprising**  
Cargo's `workspace = true` pattern and Gradle's convention plugins are preferred over Terragrunt's implicit merge. The industry trend is away from implicit inheritance toward explicit opt-in.

**6. The litmus test for locking**  
"If child A changes this, could child B break?" → Yes = locked. No = child can override.

### Why We Chose This Model

Our model combines:
- **Cargo**: Root defines workspace membership explicitly, members opt into shared values
- **Gradle**: Graduated enforcement with clear locked vs. inheritable boundaries
- **Kustomize**: Base owns identity (what exists), overlay owns behavior (how it runs)
- **Terragrunt**: Per-directory config file for per-module customization (but without Terragrunt's merge complexity)

We rejected:
- **Terragrunt's full merge model**: No locking means a child can change the binary, bypass safety thresholds, or break siblings. Too dangerous.
- **Bazel's zero inheritance**: Too verbose. Every module repeating everything is unacceptable for a TUI tool config.
- **Nx's auto-discovery**: Magic that surprises users. We use explicit member lists.
- **Single-file with named blocks**: Doesn't scale. One person's typo in a shared file breaks everyone. Can't have per-module git ownership.

---

## Architecture: The 3-Tier Override Model

### Tiers

| Tier | Where declared | What it controls | Design principle |
|------|---------------|-----------------|-----------------|
| **Locked** | Root `tfui.hcl` only | `terraform.bin`, `chdir.members`, `cache.*`, `ai.*` | Could break sibling chdirs if changed per-child |
| **Inheritable** | Root `defaults {}` block | `parallelism`, `lock`, `lock_timeout`, `var_file`, `var`, `plugin.*` | Root provides safe defaults, child can specialize |
| **Child-only** | Child `tfui.hcl` | `var_file`, `var`, `workspace` blocks, `plugin` overrides | Only meaningful for that specific module |

### Why Each Field Is Where It Is

**Locked fields:**

| Field | Why locked | What breaks if child overrides |
|-------|-----------|-------------------------------|
| `terraform.bin` | All chdirs share state format | Mixing terraform + tofu within one project can corrupt state files |
| `chdir.members` | Identity — what exists in the project | A child shouldn't add/remove siblings |
| `cache.staleness_threshold` | Safety invariant | A child bypassing staleness check could apply on stale data (dangerous) |
| `ai.enabled`, `ai.region` | Credential scope is project-wide | AWS region mismatch would cause auth failures |

**Inheritable defaults:**

| Field | Why inheritable | Use case for override |
|-------|----------------|----------------------|
| `parallelism` | Reasonable project-wide default | Module with many resources needs higher parallelism |
| `lock` | Safety default (true) | Dev modules may safely run unlocked |
| `var_file` | Shared variable files (tags, common config) | Module may not need shared vars |
| `plugin "risk"` | Risk level defaults | Prod modules need critical, dev needs low |
| `plugin "apply"` | Auto-approve defaults | CI environments may auto-approve |

**Child-only:**

| Field | Why child-only | Makes no sense at root |
|-------|---------------|------------------------|
| `var_file` (module-specific) | VPC vars don't apply to ECS | Root doesn't run terraform directly |
| `var` | Module-specific variable values | Root has no terraform context |
| `workspace` blocks | Per-workspace behavior for THIS module | Workspaces are per-module concept |

### Resolution Chain

```
Root defaults → Child top-level → Child workspace block → CLI flags → [-- passthrough]
     ↑                ↑                    ↑                   ↑              ↑
  Biggest scope    Module-wide        Per-workspace     User intent     Escape hatch
```

For `var_file` and `var` specifically: all levels are **concatenated** (not replaced). Order = precedence. Later values override earlier for the same terraform variable name.

For `plugin` settings and scalars (`parallelism`, `lock`): each level **overrides** the previous (last writer wins).

---

## File Format Specification

### Root `tfui.hcl` Schema

```hcl
# LOCKED BLOCKS — cannot appear in child files

terraform {
  bin = "terraform"               # Required. Path or name of binary. No auto-detection.
}

chdir {
  members = [                     # Required for monorepos. Explicit list. No globs.
    "modules/vpc",
    "modules/ecs",
    "environments/prod",
    "environments/staging",
  ]
}

cache {
  staleness_threshold = "5m"      # Duration. Prompt before destructive ops on stale data.
}

ai {
  enabled  = true                 # Master switch for AI features
  provider = ""                   # "bedrock" or "anthropic" (auto-detect from credentials)
  model    = ""                   # Model ID (auto-detect per provider if empty)
  region   = "us-east-1"         # AWS region for Bedrock
}

# INHERITABLE BLOCK — defaults for all chdirs, all workspaces

defaults {
  parallelism = 10                # Terraform parallelism (-parallelism=N)
  lock        = true              # State locking (-lock=true/false)
  lock_timeout = ""               # Lock wait duration (-lock-timeout=Xs)

  # Ordered var-files applied to all chdirs (paths relative to project root)
  var_file "common/tags.tfvars" {}
  var_file "common/provider.tfvars" {}

  # Ordered vars applied to all chdirs
  var "project_name" { value = "my-infra" }

  # Plugin defaults
  plugin "risk" {
    enabled = true
    level   = "high"              # Default risk sensitivity
  }

  plugin "apply" {
    auto_approve = false          # Require confirmation by default
  }

  plugin "phantom" {
    enabled = true
  }
}
```

### Child `tfui.hcl` Schema

```hcl
# TOP-LEVEL — applies to ALL workspaces in this child
# Overrides root defaults for this module

# Plugin overrides (inheritable tier — overrides root defaults)
plugin "risk" {
  level = "critical"              # This module is always critical
}

# Module-wide var-files (child-only tier — paths relative to THIS directory)
var_file "base.tfvars" {}

# Module-wide vars (child-only tier)
var "environment_type" { value = "production" }

# PER-WORKSPACE BLOCKS — override child top-level for specific workspace

workspace "default" {
  var_file "dev.tfvars" {}
}

workspace "staging" {
  var_file "staging.tfvars" {}
  var "deploy_target" { value = "staging-cluster" }
  plugin "risk" { level = "high" }
}

workspace "production" {
  var_file "prod.tfvars" {}
  var_file "prod-secrets.tfvars" {}
  var "deploy_target" { value = "prod-cluster" }
  var "lock_timeout" { value = "60s" }
  plugin "apply" { auto_approve = false }
}

# FORBIDDEN — parser rejects these with clear error messages:
# terraform { ... }
# chdir { ... }
# cache { ... }
# ai { ... }
# defaults { ... }
```

### Block Syntax Details

**`var_file "path" {}`**
- Label is the file path
- Path is relative to the file declaring it (root var_files relative to root, child var_files relative to child dir)
- Empty body today. Extensible: `optional = true` for files that may not exist
- Declaration order = terraform invocation order within that scope level

**`var "name" { value = "..." }`**
- Label is the terraform variable name
- `value` attribute is the value (string; terraform handles type coercion)
- Extensible: could add `sensitive = true`, `description = "..."` later

**`workspace "name" {}`**
- Label is the terraform workspace name (exact match or glob: `"dev-*"`)
- Contains: `var_file`, `var`, `plugin` blocks
- Cannot contain: other `workspace` blocks (no nesting)
- Globs match using `filepath.Match` rules

**`plugin "id" {}`**
- Label is the plugin ID (e.g., "risk", "apply", "phantom")
- Body is plugin-specific key-value pairs
- `enabled` attribute is special (controls whether plugin is active)
- All other attributes are opaque to the config system (passed to plugin's `Configure()`)

---

## Variable Precedence Deep Dive

### Terraform's Native Rules (researched from source code)

Terraform processes variables in this order (lowest → highest priority):

1. Environment variables (`TF_VAR_*`)
2. `terraform.tfvars` (auto-loaded if present)
3. `terraform.tfvars.json` (auto-loaded if present)
4. `*.auto.tfvars` / `*.auto.tfvars.json` (alphabetical order)
5. `-var-file` and `-var` CLI flags (in command-line order, interleaved)

**Critical finding**: Within tier 5, `-var` and `-var-file` share the same precedence tier. They are processed left-to-right as they appear on the command line. There is NO inherent priority of `-var` over `-var-file`.

**However**: `terraform-exec` (the Go library tfui uses) always emits all `-var-file` flags before all `-var` flags, regardless of API call order. This means when using terraform-exec, `-var` always beats `-var-file` (because it comes later in the command line).

**For complex types** (maps, lists, objects): Full replacement, no merging. Since Terraform 0.12+, if a higher-precedence source defines a map variable, it completely replaces the lower-precedence definition. No key-level merging.

### tfui's Concatenation Model

Given the above, tfui concatenates var-files and vars from all config levels:

```
terraform plan \
  -var-file=common/tags.tfvars          \  # ← Root defaults (position 1-N)
  -var-file=common/provider.tfvars      \
  -var-file=base.tfvars                 \  # ← Child top-level (position N+1...)
  -var-file=prod.tfvars                 \  # ← Child workspace (position ...)
  -var-file=prod-secrets.tfvars         \
  -var-file=hotfix.tfvars               \  # ← CLI --var-file (position ...)
  -var 'project_name=my-infra'          \  # ← Root vars (always after var-files)
  -var 'environment_type=production'    \  # ← Child vars
  -var 'deploy_target=prod-cluster'     \  # ← Workspace vars
  -var 'region=us-west-2'                  # ← CLI --var
```

**Rules:**
1. All var-files emitted before all vars (terraform-exec constraint)
2. Within var-files: root → child → workspace → CLI order
3. Within vars: same order
4. Later values override earlier for same variable name
5. Result: CLI > workspace > child > root (smallest scope wins)

### Interaction with Terraform's Auto-Loading

Terraform auto-loads `terraform.tfvars` and `*.auto.tfvars` from the working directory BEFORE any `-var-file` flags. This means:

```
terraform.tfvars (auto)  →  *.auto.tfvars (auto)  →  [tfui var-files in order]  →  [tfui vars]
```

tfui does NOT suppress terraform's auto-loading. They coexist. If a user has `terraform.tfvars` in their module AND var-files in tfui config, the tfui var-files win (they come later).

### The `--` Passthrough

Everything after `--` is appended raw to the terraform command. tfui does not parse, track, or display these. They exist for:
- Experimental flags tfui hasn't mapped yet
- One-off debugging (e.g., `-- -input=false`)
- Provider-specific flags

```bash
tfui plan --var-file=tracked.tfvars -- -var-file=untracked.tfvars -some-future-flag
```

In this example:
- `tracked.tfvars` is shown in the TUI header, participates in resolution
- `untracked.tfvars` is invisible to tfui, just forwarded

---

## First-Run Experience

### Design Goals

1. `tfui --terraform-bin terraform` must work with zero config files (instant start)
2. Error messages when config is missing must be actionable ("Run `tfui init` to get started")
3. `tfui init` wizard should feel like `terraform init` — fast, helpful, non-blocking

### Behavior Matrix

| State | Behavior |
|-------|----------|
| No `tfui.hcl` + no `--terraform-bin` | Error: "No terraform binary configured. Use --terraform-bin or run tfui init" |
| No `tfui.hcl` + `--terraform-bin terraform` | Works. Single-module mode. No chdir picker. |
| `tfui.hcl` exists + `terraform.bin` set | Full config loaded. Chdir picker if members > 1. |
| `tfui.hcl` exists + `terraform.bin` missing | Error: "terraform.bin is required in tfui.hcl" |
| `tfui.hcl` exists + `--terraform-bin` flag | CLI flag overrides config file value |

### `tfui init` Wizard Flow

1. **Detect binary**: Scan PATH for `terraform`, `tofu`, `terragrunt`. Present findings. User picks one.
2. **Discover modules**: Glob for directories containing `.tf` or `.tofu` files (one level deep + configured patterns). Present as checklist.
3. **User confirms/edits**: Remove unwanted dirs, add missed ones. Reorder if desired.
4. **Generate `tfui.hcl`**: Write explicit config with confirmed members and binary.
5. **Optionally generate child configs**: For each member, ask "Does this module need per-workspace config?" If yes, scaffold a child `tfui.hcl` with workspace blocks.

The wizard uses globs for discovery (convenience during setup) but the OUTPUT is always an explicit list (no globs in the final config).

---

## Scope → Chdir Rename

### Rationale

terraform's `-chdir` flag changes the working directory before any operation. tfui's "scope" does the same thing (select a subdirectory within a monorepo) plus:
- Discovery via glob patterns → replaced by explicit members list
- Interactive picker (fuzzy filter) → stays, just renamed
- Session persistence → stays, keys renamed

The extra features don't justify a different name. The base concept IS chdir.

### Rename Table

| Before | After | Location |
|--------|-------|----------|
| `--scope` | `--chdir` | CLI flag |
| `scope:` / `scope.paths` | `chdir.members` | Config |
| `plugins/scope/` | `plugins/chdir/` | Package |
| `ScopeGuard` | `ChdirGuard` | `pkg/sdk/` |
| `NewScopeGuard` | `NewChdirGuard` | All plugins |
| `SessionKeyActiveScope` | `SessionKeyActiveChdir` | `pkg/sdk/` |
| `SessionKeyActiveScopeAbs` | `SessionKeyActiveChdirAbs` | `pkg/sdk/` |
| `scope.active` | `chdir.active` | Session key |
| `scope.active_abs` | `chdir.active_abs` | Session key |
| `scope.count` | `chdir.count` | Session key |
| `ProjectContext.Scopes` | `ProjectContext.Chdirs` | `pkg/sdk/app_context.go` |
| `ProjectContext.ActiveScope` | `ProjectContext.ActiveChdir` | `pkg/sdk/app_context.go` |
| `ProjectContext.ActiveScopeAbs` | `ProjectContext.ActiveChdirAbs` | `pkg/sdk/app_context.go` |
| `DiscoverScopes()` | `DiscoverChdirs()` | `internal/config/` |
| Plugin ID: `"scope"` | `"chdir"` | Registration |
| Menu label: `scope` | `chdir` | Home menu |
| Keybinding hint: `c scope` | `c chdir` | Status bar |

### What Stays the Same

- Keybinding (`c`) — muscle memory preserved
- Interactive picker UI — same fuzzy filter behavior
- Session persistence — values persist across plugin switches
- `ChdirGuard` pattern — same check-and-rescope logic
- `WithDir()` on service — creates fresh instance per chdir

---

## Flag Compatibility Layer

### The Problem with pflag

Go's `spf13/pflag` (cobra's parser) interprets `-var` as three shorthand flags `-v -a -r`. It cannot register multi-character single-dash flags. This is a fundamental library limitation.

### Solution: Argument Pre-Processor

A `normalizeArgs(args []string) []string` function rewrites terraform-style flags before cobra sees them:

```
Input:  ["tfui", "plan", "-target=aws_instance.web", "-var-file", "prod.tfvars", "-destroy"]
Output: ["tfui", "plan", "--target=aws_instance.web", "--var-file", "prod.tfvars", "--destroy"]
```

**Precedent**: kubectl uses `normalizeFunc` for legacy flag compatibility. helm does similar rewriting.

**Known flag map:**
```
-target         → --target
-var            → --var
-var-file       → --var-file
-destroy        → --destroy
-replace        → --replace
-refresh-only   → --refresh-only
-parallelism    → --parallelism
-lock           → --lock
-lock-timeout   → --lock-timeout
-chdir          → --chdir
-workspace      → --workspace
-input          → --input
-compact-warnings → --compact-warnings
```

**Rules:**
- Only normalize flags in the known map (unknown flags left for cobra to error on)
- Handle both `-flag=value` and `-flag value` forms
- Boolean flags (`-destroy`, `-refresh-only`, `-compact-warnings`) don't consume next arg as value
- Never touch anything after bare `--` (passthrough boundary)
- Never touch `-` alone (stdin indicator for --plan/--state)
- Already double-dashed flags pass through unchanged

### PlanOptions / ApplyOptions

The `Service` interface changes from:
```go
Plan(ctx context.Context, targets []string) (*PlanSummary, error)
Apply(ctx context.Context, targets []string) error
```
To:
```go
Plan(ctx context.Context, opts PlanOptions) (*PlanSummary, error)
Apply(ctx context.Context, opts ApplyOptions) error
```

Options structs carry all terraform flags + config-resolved var-files:
```go
type PlanOptions struct {
    Targets     []string          // -target flags
    VarFiles    []string          // -var-file flags (resolved from all config levels + CLI)
    Vars        map[string]string // -var flags (resolved from all levels + CLI)
    Replace     []string          // -replace flags
    Destroy     bool              // -destroy
    RefreshOnly bool              // -refresh-only
    Refresh     *bool             // -refresh=true/false (nil = default)
    Parallelism int               // -parallelism=N (0 = default)
    Lock        *bool             // -lock=true/false (nil = default)
    LockTimeout string            // -lock-timeout=Xs
    ExtraArgs   []string          // raw passthrough from --
}
```

The `VarFiles` and `Vars` fields contain the FINAL resolved list (all config levels concatenated + CLI appended). The service just passes them through to terraform-exec.

---

## Tradeoffs Accepted

| Tradeoff | Chose | Over | Why |
|----------|-------|------|-----|
| Explicit members vs globs | Explicit | Globs | Predictability > convenience. Globs surprise on new dirs. |
| HCL vs YAML+HCL dual support | HCL only | Dual | Clean break. No maintenance burden of two parsers. |
| Split files vs single file | Split | Single | Scales better, lower coupling, cleaner git ownership. |
| Locked fields vs everything-overridable | Locked critical fields | Full freedom | Safety. A child bypassing staleness or binary = danger. |
| terraform-exec `-var` always wins | Accept | Custom ordering | Library constraint. Not fighting it. Vars should win anyway. |
| No auto-detection | Explicit binary | Magic detection | "It picked tofu when I wanted terraform" = immediate trust loss. |
| Pre-parser normalization | Rewrite args | Fork pflag | Simpler, testable, no library maintenance burden. |

---

## Open Questions (To Resolve During Implementation)

### Q1: Root `defaults` var_file paths — relative to what?

**Recommendation**: Relative to project root (where root `tfui.hcl` lives). Reason: that's where the file is declared. Same principle as child paths being relative to child dir.

**Impact**: `var_file "common/tags.tfvars" {}` in root defaults resolves to `<project-root>/common/tags.tfvars`. When a child in `modules/vpc` uses this, the resolved absolute path is passed to terraform (which runs in `modules/vpc` dir, so a relative path would break).

**Rule**: All var-file paths are resolved to absolute paths before being passed to terraform.

### Q2: Workspace glob matching — supported or exact only?

**Recommendation**: Support globs. `workspace "dev-*" {}` matches `dev-us-east-1`, `dev-eu-west-1`. Use `filepath.Match` semantics. Exact match takes priority over glob match. Multiple glob matches = first declared wins.

**Why**: Terragrunt users expect this. Projects with many regional workspaces (`prod-us-east-1`, `prod-eu-west-1`) would need redundant blocks without globs.

### Q3: What if both child top-level and workspace block set same plugin field?

**Rule**: Workspace block wins (it's the more specific scope). Resolution is child-top-level THEN workspace.

Example: Child sets `plugin "risk" { level = "critical" }`. Workspace "dev" sets `plugin "risk" { level = "low" }`. When workspace=dev, risk level = low.

### Q4: What if no workspace block matches the active workspace?

**Rule**: Fall through to child top-level (and then to root defaults). No error. The child's workspace blocks are optional convenience — absence means "use the defaults."

### Q5: Can a child declare a workspace block for a workspace that doesn't exist?

**Rule**: No validation against terraform's actual workspace list. Config is static; workspaces are dynamic. The block simply won't match until that workspace is created.

---

## Testing Strategy

### Philosophy

Every architectural decision must have a corresponding test that would FAIL if the decision were violated. Tests are the executable specification of this document.

**TDD enforcement**: Write failing tests first. The test file defines the API surface. Implementation makes tests pass.

**Four layers**, each testing different concerns:

```
Unit tests       → Logic correctness (parsing, resolution, normalization)
Integration tests → Real filesystem + real terraform (end-to-end config loading)
Macro tapes      → TUI renders correct state given resolved config
Exploratory      → Real monorepo projects with diverse patterns
```

---

### Layer 1: Unit Tests

**Package**: `internal/config`  
**Files**: `internal/config/hcl_test.go`, `internal/config/resolve_test.go`

#### 1.1 Root Parsing (`TestLoadRoot_*`)

| Test | Input | Expected | Decision validated |
|------|-------|----------|-------------------|
| `ValidFullConfig` | Complete root tfui.hcl with all blocks | All fields populated correctly | D3: HCL format works |
| `MinimalConfig` | Only `terraform { bin = "terraform" }` | Valid config, empty defaults | D15: single-module doesn't need chdir |
| `WithChdirMembers` | chdir block with 5 members | Members list populated, no glob expansion | D10: explicit members |
| `WithDefaults` | defaults block with parallelism, lock, var_file, plugin | All defaults accessible | Inheritable tier |
| `VarFileOrderPreserved` | defaults with 3 var_file blocks | Slice preserves declaration order | Var ordering design |
| `VarBlockParsed` | `var "name" { value = "x" }` inside defaults | Name and value extracted | Block syntax decision |
| `PluginConfig` | Multiple named plugin blocks | Each plugin has correct settings | Plugin config pattern |
| `EmptyChdirBlock` | `chdir {}` (no members) | Valid — single-module mode | D15 |
| `DuplicatePluginBlock` | Two `plugin "risk" {}` blocks | HCL error (block labels must be unique) | HCL2 semantics |
| `UnknownBlock` | `unknown_block {}` in root | HCL error (strict schema) | Schema enforcement |

#### 1.2 Root Validation (`TestLoadRoot_Validation_*`)

| Test | Input | Expected error | Decision validated |
|------|-------|---------------|-------------------|
| `MissingTerraformBin` | terraform block without bin | "terraform.bin is required" | D2: no auto-detection |
| `EmptyTerraformBin` | `terraform { bin = "" }` | "terraform.bin cannot be empty" | D2 |
| `MissingTerraformBlock` | Root file with no terraform block | "terraform block is required" | D2 |
| `NegativeParallelism` | `parallelism = -1` | "parallelism must be positive" | Validation |
| `InvalidStaleness` | `staleness_threshold = "not-a-duration"` | "invalid duration" | Validation |
| `InvalidHCLSyntax` | Malformed HCL | Error with line number | UX: actionable errors |
| `PermissionDenied` | Unreadable file | OS error wrapped | Error handling |

#### 1.3 Child Parsing (`TestLoadChild_*`)

| Test | Input | Expected | Decision validated |
|------|-------|----------|-------------------|
| `ValidWithWorkspaces` | Child with 3 workspace blocks | All workspace configs parsed | D13: workspace in child |
| `ValidWithoutWorkspaces` | Child with only top-level plugin/var_file | Top-level overrides parsed | Child top-level design |
| `TopLevelPlusWorkspace` | Plugin at top + workspace with same plugin | Both parsed, resolution tested separately | 4-level model |
| `VarFileOrder` | 3 var_file blocks in workspace | Order preserved | Var ordering |
| `EmptyWorkspaceBlock` | `workspace "prod" {}` | Valid empty workspace | Edge case |
| `NestedWorkspace` | workspace inside workspace | HCL error | No nesting |
| `WorkspaceWithGlob` | `workspace "dev-*" {}` | Parsed as glob pattern | Q2: glob matching |

#### 1.4 Child Validation — Locked Field Rejection (`TestLoadChild_Locked_*`)

| Test | Locked field in child | Expected error | Decision validated |
|------|----------------------|---------------|-------------------|
| `RejectsTerraformBlock` | `terraform { bin = "tofu" }` | "terraform block cannot appear in per-chdir config (it's project-wide)" | D12: 3-tier enforcement |
| `RejectsChdirBlock` | `chdir { members = [...] }` | "chdir block cannot appear in per-chdir config" | D12 |
| `RejectsCacheBlock` | `cache { staleness_threshold = "0s" }` | "cache block cannot appear in per-chdir config (safety invariant)" | D12 |
| `RejectsAIBlock` | `ai { enabled = false }` | "ai block cannot appear in per-chdir config" | D12 |
| `RejectsDefaultsBlock` | `defaults { ... }` | "defaults block cannot appear in per-chdir config" | D12 |

Each error message must be clear and explain WHY: `"(it's project-wide)"`, `"(safety invariant)"`.

#### 1.5 Resolution Logic (`TestResolve_*`)

| Test | Levels present | Active workspace | Expected result | What it proves |
|------|---------------|-----------------|-----------------|----------------|
| `RootDefaultsOnly` | Root with defaults | "default" | Root defaults applied | Base case |
| `RootPlusChild` | Root defaults + child top-level | "default" | Child overrides root | Inheritance works |
| `RootPlusChildPlusWorkspace` | All three | "production" | Workspace overrides child overrides root | Full chain |
| `NoChildFile` | Root only, no child tfui.hcl exists | any | Root defaults used | Missing child = OK |
| `NoMatchingWorkspace` | Child has workspace "prod" block | "staging" | Falls to child top-level | Q4: no-match fallback |
| `ExactMatchBeatsGlob` | workspace "prod" + workspace "prod-*" | "prod" | Exact match wins | Q2: priority |
| `GlobMatch` | workspace "dev-*" | "dev-us-east-1" | Glob matches | Q2: glob support |
| `MultipleGlobsFirstWins` | workspace "dev-*" + workspace "d*" | "dev-x" | First declared glob wins | Q2: ordering |
| `PluginOverrideField` | Root: level="high", Child: level="critical" | — | level="critical" | Plugin merge |
| `PluginAddField` | Root: level="high", Child: threshold=5 | — | Both fields present | Plugin merge |
| `PluginDisable` | Root: enabled=true, Child: enabled=false | — | Plugin disabled | Plugin override |
| `WorkspaceOverridesChildTopLevel` | Child top: level="critical", WS: level="low" | matching ws | level="low" | Q3: ws beats child |

#### 1.6 Variable File Concatenation (`TestResolveVarFiles_*`)

| Test | Config state | Expected var_files slice | What it proves |
|------|-------------|------------------------|----------------|
| `RootOnly` | Root: [a.tfvars, b.tfvars] | [a, b] | Root order preserved |
| `RootPlusChild` | Root: [a], Child: [b, c] | [a, b, c] | Concatenation |
| `AllLevels` | Root: [a], Child: [b], WS: [c, d] | [a, b, c, d] | Full chain |
| `EmptyLevels` | Root: [a], Child: (none), WS: [c] | [a, c] | Gaps OK |
| `RootPathResolution` | Root: var_file "common/x.tfvars" | Absolute path from project root | D14: relative to declaring file |
| `ChildPathResolution` | Child (in modules/vpc): var_file "vpc.tfvars" | Absolute path from modules/vpc/ | D14 |
| `WorkspacePathResolution` | WS block in child: var_file "ws.tfvars" | Same as child dir (WS is inside child file) | D14 |
| `RelativePathUp` | Child: var_file "../../shared/x.tfvars" | Resolves correctly | Path traversal |
| `CLIAppends` | Config: [a, b], CLI: [c] | [a, b, c] | CLI is highest layer |
| `VarsAfterVarFiles` | Mix of var and var_file at all levels | var_files first, then vars | terraform-exec constraint |

#### 1.7 Variable Concatenation (`TestResolveVars_*`)

| Test | Config state | Expected vars map | Order |
|------|-------------|-------------------|-------|
| `RootOnly` | Root: {a=1, b=2} | {a=1, b=2} | Root order |
| `ChildOverridesRoot` | Root: {a=1}, Child: {a=2} | {a=2} | Child wins |
| `WorkspaceOverridesChild` | Root: {a=1}, Child: {a=2}, WS: {a=3} | {a=3} | WS wins |
| `CLIOverridesAll` | All levels set {a=?}, CLI: {a=final} | {a=final} | CLI wins |
| `DifferentKeys` | Root: {a=1}, Child: {b=2}, WS: {c=3} | {a=1, b=2, c=3} | All present |

#### 1.8 Optional Config Mode (`TestOptionalConfig_*`)

| Test | State | Expected | Decision validated |
|------|-------|----------|-------------------|
| `NoBinaryNoFile` | No tfui.hcl, no --terraform-bin | Error: actionable message | D9 |
| `BinaryFlagNoFile` | No tfui.hcl, --terraform-bin=terraform | Valid minimal config | D9: config optional |
| `BinaryFlagOverridesConfig` | tfui.hcl has bin=terraform, flag=tofu | Uses tofu | CLI > config |

#### 1.9 File Discovery (`TestFindConfig_*`)

| Test | Directory structure | Start dir | Expected |
|------|-------------------|-----------|----------|
| `InCurrentDir` | ./tfui.hcl | . | Found |
| `InParentDir` | ../tfui.hcl | ./subdir | Found (walk up) |
| `InGrandparentDir` | ../../tfui.hcl | ./a/b | Found (walk up) |
| `NotFound` | No tfui.hcl anywhere | . | Not found (no error for root, error for binary) |
| `StopsAtFsRoot` | No tfui.hcl | /tmp/deep/path | Not found |
| `ChildInMemberDir` | Root lists "modules/vpc", modules/vpc/tfui.hcl exists | modules/vpc | Found |
| `ChildMissingInMemberDir` | Root lists "modules/ecs", no modules/ecs/tfui.hcl | modules/ecs | Not found (OK, uses defaults) |

---

### Layer 2: Unit Tests — Argument Normalizer

**Package**: `main` (cmd/tfui)  
**File**: `cmd/tfui/normalize_test.go` (already written)

| Category | Count | What it proves |
|----------|-------|----------------|
| Empty/minimal args | 4 | No panic on edge inputs |
| Known flags (= form) | 10 | Each terraform flag individually normalized |
| Known flags (space form) | 10 | Space-separated values handled |
| Boolean flags | 3 | Don't consume next arg as value |
| Already double-dashed | 3 | Pass through unchanged |
| `--` passthrough boundary | 3 | Normalization stops after `--` |
| `-` stdin indicator | 2 | Never touched |
| Unknown flags | 3 | Left unchanged |
| Repeated flags | 2 | All instances normalized |
| Mixed flags | 2 | Known + unknown + double-dashed combined |
| Complex real-world | 3 | Full commands with many flags |
| Special characters | 3 | Equals, spaces, JSON in values |
| Values that look like flags | 2 | `-var=-something` handled correctly |
| Input immutability | 1 | Original slice not mutated |

---

### Layer 3: Unit Tests — Service Options

**Package**: `terraform` (internal/terraform)  
**File**: `internal/terraform/options_test.go` (already written)

| Category | What it proves |
|----------|----------------|
| Zero-value PlanOptions | Backward compat: no extra flags when options empty |
| Each PlanOptions field individually | Correct flag emitted for each field |
| All fields combined | Full command string correct |
| ApplyOptions fields | Same pattern for apply |
| Binary prefix | Works with terraform and tofu |
| Interface compliance | StaticService satisfies sdk.Service with new signatures |
| Command.String() | Serialized output matches what user would type |

---

### Layer 4: Integration Tests

**Package**: `integration` (tests/integration)  
**File**: `tests/integration/compat_test.go` (already written, needs expansion)  
**Requirements**: terraform binary on PATH, test fixtures

#### Config Loading Integration

| Test | Fixture | What it proves |
|------|---------|----------------|
| `TestConfig_RootAndChild_ResolvesCorrectly` | Multi-dir project with root + child tfui.hcl | End-to-end config loading from real filesystem |
| `TestConfig_LockedFieldInChild_FailsWithMessage` | Child file with terraform block | Parser error is actionable |
| `TestConfig_WorkspaceSwitch_ChangesVarFiles` | Child with workspace blocks | Different var-files per workspace |
| `TestConfig_SingleModuleMode_NoChdir` | Simple project, root tfui.hcl, no chdir block | Works without members |
| `TestConfig_NoBinaryConfigured_FailsWithMessage` | Empty tfui.hcl | Error suggests --terraform-bin or tfui init |
| `TestConfig_CLIBinaryOverridesConfig` | Root has bin=terraform, test passes --terraform-bin=tofu | CLI wins |

#### Terraform Execution Integration

| Test | What it proves | Requires |
|------|----------------|----------|
| `TestPlan_WithVarFile_PassesToTerraform` | Var-file reaches terraform, plan uses the variable values | Real terraform + fixture with variables |
| `TestPlan_WithVar_OverridesVarFile` | -var beats -var-file for same variable | Real terraform |
| `TestPlan_Destroy_ProducesDeleteActions` | -destroy flag changes plan type | Real terraform |
| `TestPlan_WithWorkspace_UsesCorrectState` | --workspace selects workspace before planning | Fixture with multiple workspaces |
| `TestPlan_Passthrough_ReachesTerraform` | Args after -- are forwarded | Real terraform |
| `TestPlan_VarFileConcatenation_OrderCorrect` | Root + child + workspace var-files in right order | Real terraform + multi-config fixture |

#### Flag Normalization Integration

| Test | What it proves |
|------|----------------|
| `TestCLI_SingleDashTarget_Works` | `tfui plan -target=X` doesn't error |
| `TestCLI_DoubleDashTarget_Works` | `tfui plan --target=X` still works |
| `TestCLI_MixedDashes_Works` | `-target=X --var-file=y` both work in same command |
| `TestCLI_UnknownSingleDash_Errors` | `-unknown-flag` produces cobra error (not silently dropped) |

---

### Layer 5: Macro Tapes (E2E TUI)

**Location**: `tests/fixtures/tapes/compat/`  
**Requirements**: Built binary + StaticService fixtures

| Tape | What it renders | Config state | Decision validated |
|------|----------------|-------------|-------------------|
| `config_header_varfiles.tape` | Header shows resolved var-files | Root + child config with var_files | TUI integration |
| `config_header_workspace.tape` | Header shows active workspace name | Workspace selected | Workspace display |
| `config_chdir_picker.tape` | Chdir picker shows exact member list | Root with 3 members | D10: explicit, no globs |
| `config_destroy_mode.tape` | Plan view shows DESTROY badge | --destroy flag | Destroy UX |
| `config_no_config_error.tape` | Error message displayed in TUI | No tfui.hcl, no --terraform-bin | D9: helpful error |

---

### Layer 6: Exploratory Testing (Manual/Agent-Driven)

**When**: After implementation, before merge. Run against real-world project patterns.

| Scenario | What to observe | Risk if untested |
|----------|----------------|------------------|
| **Large monorepo** (20+ modules) | Chdir picker renders all members, no lag | Performance regression |
| **Deep nesting** (project/env/region/module) | Root found via walk-up from deep path | File discovery fails |
| **Workspace-heavy project** (10+ workspaces) | Correct var-files resolve per workspace | Wrong vars applied to wrong env |
| **Terragrunt project** | bin=terragrunt, plan works, workspace graceful error | Crash on workspace ops |
| **Tofu project** | bin=tofu, full plan/apply cycle | Binary invocation differs |
| **No chdir (single module)** | No picker shown, operates in root | Picker shows with empty list |
| **All config levels active** | Root + child + workspace + CLI | Precedence correct end-to-end |
| **Conflicting vars across levels** | Same variable in root and workspace | Workspace wins (later in concat) |
| **Permission issues** | Child tfui.hcl unreadable | Clear error, not crash |
| **Concurrent workspace switch** | Change workspace while plan is running | No race condition in resolution |

---

### Test Fixtures Needed

```
tests/fixtures/
  compat/
    plan_destroy.json              # (exists) Destroy-only plan
    state.json                     # (exists) Matching state
  config/
    single-module/
      tfui.hcl                     # Minimal: just terraform.bin
      main.tf                      # Simple terraform config
    monorepo/
      tfui.hcl                     # Root with 3 members + defaults
      common/
        tags.tfvars                # Shared var-file
      modules/
        vpc/
          main.tf
          tfui.hcl                 # Child with workspace blocks
          base.tfvars
          tfvars/
            dev.tfvars
            prod.tfvars
        ecs/
          main.tf                  # No child config (uses defaults)
    locked-field-violation/
      tfui.hcl                     # Root
      modules/
        bad/
          tfui.hcl                 # Child that sets terraform.bin (should fail)
    workspace-heavy/
      tfui.hcl                     # Root
      modules/
        api/
          tfui.hcl                 # Child with 5+ workspace blocks including globs
```

---

### Coverage Targets

| Package | Target | Rationale |
|---------|--------|-----------|
| `internal/config` (HCL loading) | 100% | Core system, all branches matter |
| `internal/config` (resolution) | 100% | Precedence bugs = wrong terraform invocation |
| `cmd/tfui` (normalizer) | 100% | Pure function, full table-driven coverage |
| `internal/terraform` (options mapping) | 100% | Flag mapping bugs = silent wrong behavior |
| `pkg/sdk` (ChdirGuard) | 100% | State detection is critical path |

---

### Regression Prevention

Each bug found during exploratory testing must be converted to a unit test that reproduces the exact scenario. The test must fail without the fix and pass with it. This prevents regressions as the config system evolves.

**Pattern**: Bug → minimal reproduction fixture → unit test → fix → verify → commit.

---

## Implementation Sequence

```
1. Add github.com/hashicorp/hcl/v2 dependency
2. Define Go structs for root schema (hcl tags)
3. Define Go structs for child schema (hcl tags)
4. Implement root parser (LoadRoot)
5. Implement child parser (LoadChild) with locked-field rejection
6. Implement resolution logic (merge root → child → workspace)
7. Implement var-file path resolution (relative → absolute)
8. Implement workspace glob matching
9. Implement dot-notation accessors (GetString, GetBool, etc.)
10. Implement optional-config mode (no file + --terraform-bin)
11. Wire into main.go (replace YAML loader)
12. Delete YAML loader + yaml.v3 dependency
13. Delete DetectBinary()
14. Update tfui init (wizard generates HCL)
15. Rename scope → chdir throughout
16. Update all tests
```

Each step should have a failing test written BEFORE implementation (TDD).

---

## Future Extensions (Not This Phase)

| Extension | How the architecture supports it |
|-----------|----------------------------------|
| `var_file "path" { optional = true }` | Add attribute to var_file block body |
| `var "name" { sensitive = true }` | Add attribute to var block body |
| Remote var-files (`var_file "s3://bucket/vars.tfvars" {}`) | Source abstraction resolves URI |
| Per-chdir hooks (`hook "pre_plan" { command = "..." }`) | New block type in child schema |
| Policy enforcement (`policy { max_parallelism = 20 }`) | New locked block in root |
| Config validation command (`tfui config validate`) | Schema is strongly typed, validate at parse time |
| Config show command (`tfui config show --resolved`) | Resolution logic is pure function, just print result |
| IDE support (LSP for tfui.hcl) | HCL2 has built-in LSP tooling |
