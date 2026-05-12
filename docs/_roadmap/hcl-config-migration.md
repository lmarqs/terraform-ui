---
title: "Config Migration: YAML → HCL with Workspace Overrides"
status: planned
priority: medium
created: 2026-05-11
effort: large
tags: [feature, config, ux, breaking]
depends_on: []
---

## Summary

The project uses `tfui.yaml` for configuration, but as a terraform ecosystem tool, HCL is the natural fit. Migration to HCL enables workspace-specific override blocks and aligns with the mental model of terraform/tofu/terragrunt users. Also adds terragrunt as a supported binary alongside terraform and tofu.

## Need

- YAML is foreign to terraform users — HCL is their native config language
- No workspace-specific config overrides (e.g., auto_approve=true in staging but not production)
- Binary detection doesn't include terragrunt (common in teams using directory-per-environment)
- Config is flat — no way to vary behavior per workspace without runtime code

## Expected UX

- Clean break: `tfui.hcl` replaces `tfui.yaml`, no dual support
- `tfui init` generates `tfui.hcl` instead of `tfui.yaml`
- Workspace-specific behavior "just works" when switching workspaces in the TUI
- Config errors show HCL-native diagnostics (line:col with context)

## Advantages

- Ecosystem alignment — terraform users feel at home
- Workspace-specific behavior without code changes
- Future expression support (HCL2 supports variables, functions)
- Cascading config for monorepos (like .editorconfig)
- Terragrunt users can use tfui without switching tools

## Effort Justification

**Large** (1-2 weeks) because:
- New HCL parser integration with schema definition
- Complex merge resolution across multiple config sources (base, scope, workspace inline, workspace file, CLI)
- Glob pattern matching with specificity ordering
- Terragrunt auto-detection and integration testing
- Migration path for existing users
- Breaking change requiring docs updates across all examples

## Design

### Benchmarks

Tools analyzed for design inspiration:
- **Terragrunt**: directory-as-environment with `include { path = find_in_parent_folders() }` inheritance
- **Pulumi**: `Pulumi.<stack>.yaml` file-per-stack pattern (most relevant)
- **k9s**: single config with cluster blocks
- **Terraform Cloud**: per-workspace variables managed externally
- **Spacelift**: shared "contexts" attached to multiple stacks

**Chosen approach**: Pulumi-style hybrid — inline workspace blocks for simple cases, file-per-workspace for scale.

### Config Format

**Base config** (`tfui.hcl`):
```hcl
terraform {
  bin = "tofu"
}

scope {
  paths = ["modules/*"]
}

plugin "risk" {
  enabled = true
}

workspace "staging" {
  plugin "apply" {
    auto_approve = true
  }
}

workspace "dev-*" {
  plugin "risk" {
    level = "low"
  }
}
```

**File-per-workspace** (for scale):
```
tfui.hcl                # Base
tfui.production.hcl     # Production overrides
tfui.staging.hcl        # Staging overrides
```

### Resolution Order

Most specific wins:
1. `tfui.hcl` base config
2. Scope-level `tfui.hcl` (in scope directory, optional)
3. Inline `workspace` block (exact match or glob pattern)
4. `tfui.<workspace>.hcl` file (overrides inline block)
5. `--config key=value` CLI flags (always wins)

**Inheritance**: Deep merge — workspace config overlays base, doesn't replace. Matches terraform's `merge()` semantics.

**Workspace glob patterns**: `"dev-*"` matches `dev-alice`, `dev-bob`. Resolved by specificity (exact > longest prefix > glob).

### Binary Support Tiers

| Binary | Support | CI Tested | Notes |
|--------|---------|-----------|-------|
| `terraform` | full | ✓ | Default fallback |
| `tofu` | full | ✓ | Preferred if on PATH |
| `terragrunt` | best-effort | ✓ (plan+state) | Workspace ops may differ |
| custom path | user responsibility | ✗ | Any terraform-compatible binary |

**Auto-detection order** (when `bin` is empty): `tofu` → `terragrunt` → `terraform`

### Terragrunt Limitations

- `workspace list/select/new/delete` may not work (terragrunt uses dir-per-env, which maps to tfui "scopes")
- Plan file `-out` behavior may differ
- `taint`/`untaint` deprecated
- Recommend using scopes instead of workspaces when using terragrunt

## Open Questions

- Migration command (`tfui migrate`) vs migration guide?
- Should scope-level config be mandatory, optional, or disabled by default?
- HCL expression evaluation in v1 (variables, functions) or defer to v2?

## Tasks

- [ ] Add `github.com/hashicorp/hcl/v2` + `github.com/zclconf/go-cty` to `go.mod`
- [ ] Define HCL schema with `hcl:` struct tags on Config types
- [ ] Implement config loading: `findUp()` + `parseHCL()` + deep merge
- [ ] Implement workspace resolution: inline blocks + file-per-workspace + glob matching
- [ ] Implement scope-level config discovery (optional `tfui.hcl` in scope dirs)
- [ ] Update `DetectBinary()` to include terragrunt in auto-detection chain
- [ ] Update `init` command to generate `tfui.hcl`
- [ ] Remove `gopkg.in/yaml.v3` dependency
- [ ] Write tests: HCL parsing, workspace merging, glob patterns, scope cascading, deep merge
- [ ] Add terragrunt integration tests (plan, apply, state operations)
- [ ] Update CLAUDE.md Config section
- [ ] Update all example configs in docs
- [ ] Consider: migration guide or `tfui migrate` subcommand for existing users
