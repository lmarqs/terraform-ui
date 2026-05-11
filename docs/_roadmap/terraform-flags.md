---
title: Terraform Flag Passthrough
status: planned
priority: high
---

# Terraform Flag Passthrough

## Problem

tfui currently passes only `-target` and `-out` to terraform. Users whose workflows require `-var-file`, `-destroy`, `-replace`, `-parallelism`, or other flags cannot use tfui without the `TF_CLI_ARGS_*` workaround.

This makes tfui unusable for most real-world terraform workflows where variables are environment-specific.

## Gap Analysis

| Flag | Status | Priority | Use case |
|------|--------|----------|----------|
| `-var` | Missing | HIGH | Pass individual variables |
| `-var-file` | Missing | HIGH | Load variable files (prod.tfvars, etc.) |
| `-destroy` | Missing | HIGH | Plan infrastructure teardown |
| `-replace` | Missing | HIGH | Force recreation (modern taint replacement) |
| `-refresh=false` | Missing | HIGH | Skip state refresh for speed |
| `-refresh-only` | Missing | HIGH | Update state without changing infra |
| `-parallelism` | Missing | MEDIUM | Control concurrency for large configs |
| `-lock=false` | Missing | MEDIUM | Disable locking in CI pipelines |
| `-lock-timeout` | Missing | MEDIUM | Wait for lock release |
| `-target` | Supported | — | Scope to specific resources |
| `-out` | Internal | — | Plan file management (automatic) |
| `-no-color` | N/A | — | tfui handles output rendering |
| `-input=false` | N/A | — | tfui is non-interactive by design |

## Proposal

### Service Interface Change

Current:
```go
Plan(ctx context.Context, targets []string) (*PlanSummary, error)
Apply(ctx context.Context, targets []string) error
```

Proposed:
```go
Plan(ctx context.Context, opts PlanOptions) (*PlanSummary, error)
Apply(ctx context.Context, opts ApplyOptions) error
```

```go
type PlanOptions struct {
    Targets     []string
    VarFiles    []string
    Vars        map[string]string
    Replace     []string
    Destroy     bool
    Refresh     *bool  // nil = default, false = skip, true = force
    RefreshOnly bool
    Parallelism int    // 0 = default (10)
    Lock        *bool  // nil = default
    LockTimeout string
}

type ApplyOptions struct {
    Targets     []string
    Parallelism int
    Lock        *bool
    LockTimeout string
}
```

### CLI Flags

```bash
tfui plan --var-file prod.tfvars --var region=us-east-1 --replace aws_instance.web
tfui plan --destroy
tfui plan --refresh=false
tfui plan --parallelism 5
tfui apply --parallelism 5 --lock-timeout 5m
```

### Config (tfui.yaml)

```yaml
terraform:
  var_files:
    - environments/prod.tfvars
  vars:
    region: us-east-1
  parallelism: 5
```

CLI flags override/extend config values.

### TUI Impact

- **Vars:** No new screen. Show active var-files in header: `[prod.tfvars]`
- **Destroy:** Plan screen shows "DESTROY MODE" warning prominently
- **Replace:** Pin resources for replacement (like pin for target, but with replace semantics)
- **Parallelism:** No UI impact, affects execution speed only

### Command Type Integration

All flags serialize naturally via `sdk.Command.Flags`:

```go
Command{
    Binary: "terraform",
    Verb:   "plan",
    Flags:  []string{"-var-file=prod.tfvars", "-var=region=us-east-1", "-target=aws_instance.web", "-destroy"},
}
```

## Implementation Order

1. **Phase 1:** `-var`, `-var-file` (unblocks most users)
2. **Phase 2:** `-destroy`, `-replace`, `-refresh` (workflow completeness)
3. **Phase 3:** `-parallelism`, `-lock`, `-lock-timeout` (operational control)

## Workaround (today)

Users can set `TF_CLI_ARGS_plan` and `TF_CLI_ARGS_apply` environment variables. Terraform reads these natively:

```bash
export TF_CLI_ARGS_plan="-var-file=prod.tfvars"
tfui plan
```

This works because terraform-exec passes the environment through to the terraform subprocess.
