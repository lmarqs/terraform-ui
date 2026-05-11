---
title: Variable Support (--var, --var-file)
status: planned
priority: high
created: 2026-05-11
effort: small
tags: [cli, config, plan, apply]
depends_on: []
---

## Summary

Support `--var` and `--var-file` flags so users whose terraform workflows require variable files get correct plan/apply results from tfui.

## Need

tfui currently doesn't support `--var` or `--var-file`. Users whose terraform workflow requires `-var-file=prod.tfvars` get wrong plan results from tfui. tfui is unusable for them.

Current workaround: set `TF_CLI_ARGS_plan` environment variable (terraform-native), which is fragile and easy to forget.

## Expected UX

**CLI flags:**

```bash
tfui plan --var-file ./prod.tfvars --var instance_type=t3.large
tfui apply --var-file ./prod.tfvars
```

**Config (tfui.yaml):**

```yaml
terraform:
  var_files:
    - environments/prod.tfvars
  vars:
    region: us-east-1
```

**Header indication:**

```
Project: medprev-cloud-iac
Scope: modules/sa-east-1  [prod.tfvars]
Workspace: default
```

**Read-only mode:** vars flow through to Command type output when serializing.

## Advantages

- **Correct results** — users with var-file workflows can actually use tfui
- **Zero new UI** — vars are configuration, not interactive; no new screen needed
- **Familiar interface** — mirrors terraform's own `--var` / `--var-file` flags exactly
- **Composable** — CLI flags override/extend config defaults

## Design

**Merge strategy:** CLI flags extend config values. If config defines `var_files: [base.tfvars]` and CLI passes `--var-file ./prod.tfvars`, both are sent to terraform (config first, CLI second — later values override earlier for the same key).

**Passthrough:** vars are passed to both `plan` AND `apply` (terraform requires them on both).

```go
// In service.Plan() and service.Apply()
for _, vf := range opts.VarFiles {
    planOpts = append(planOpts, tfexec.VarFile(vf))
}
for _, v := range opts.Vars {
    planOpts = append(planOpts, tfexec.Var(v))
}
```

**Config schema addition:**

```go
// Accessed via:
ctx.Config.GetStringSlice("terraform.var_files", nil)
ctx.Config.GetMap("terraform.vars", nil)
```

**Command serialization:** include in `Command.Flags` so read-only mode preserves the var context.

## Tasks

- [ ] Add `--var` and `--var-file` flags to plan and apply cobra commands
- [ ] Add `terraform.var_files` and `terraform.vars` to config schema
- [ ] Pass as `tfexec.Var()` and `tfexec.VarFile()` options in `service.Plan()` and `service.Apply()`
- [ ] Merge CLI flags with config values (config first, CLI extends)
- [ ] Show active var files in header/status bar
- [ ] Include in `Command.Flags` when serializing for read-only mode
- [ ] Tests for merge logic (config-only, CLI-only, both, duplicates)
- [ ] Document in tfui.yaml example config

## Open Questions

- Should `--var-file` paths be resolved relative to CWD or project root?
- Should config `var_files` paths be relative to tfui.yaml location?
- Support glob patterns in config var_files (e.g., `environments/*.auto.tfvars`)?
