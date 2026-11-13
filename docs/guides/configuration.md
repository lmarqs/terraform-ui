---
layout: default
title: "Configure tfui with tfui.hcl"
parent: Guides — Getting Started with Terraform UI
nav_order: 2
seo:
  description: "Configure tfui for a Terraform project with an optional tfui.hcl file in the project root. Configuration only adds to the defaults: tfui works with none."
description: "Guides a user through configuring tfui.hcl for a project and its members, and lists every setting no plugin owns."
audience: "Users who know Terraform and want tfui to manage several root modules, workspaces or var-files"
---

# Configuration

tfui runs with no configuration. To work on a project with several root modules, or to give terraform var-files,
variables and lock settings per workspace, add a `tfui.hcl` to the project root:

1. Create `tfui.hcl` in the project root, with one `member` block per root module.
2. Optionally add a `tfui.hcl` inside a member directory for that member's var-files, variables and workspaces.
3. Run `tfui` in the project root and pick a member.

`tfui scaffold` writes the root file for you ([Scaffold](scaffold.md)).

## With and without tfui.hcl

tfui reads `tfui.hcl` from the project directory: `-project`, default the current directory.

| | Without `tfui.hcl` | With `tfui.hcl` |
|---|---|---|
| Where terraform runs | The project directory, or `-chdir <dir>` under it | The member you pick, or `-chdir <dir>` under the project directory |
| Settings | None: terraform's own defaults | The root file, the member's `tfui.hcl` and the matching workspace block |

## The root file

```hcl
# tfui.hcl (project root)

terraform {
  bin = "terraform"    # or "tofu", or a full path
}

member "modules/vpc" {}
member "modules/ecs" {}
member "environments/prod" {}

defaults {
  parallelism = 10
  lock        = true

  var_file "common.tfvars" {}

  plugin "risk" {
    enabled = false
  }
}
```

Every block is optional, and an empty `tfui.hcl` is valid. When the file has no `member` blocks, the project root
itself is the only member.

| Block or attribute | What it does |
|---|---|
| `terraform { bin }` | The binary tfui runs; see [Binary resolution](#binary-resolution) |
| `member "<dir>" {}` | A directory, relative to the project root, that tfui can run terraform in |
| `defaults { parallelism }` | Passed to terraform as `-parallelism` |
| `defaults { lock }` | Passed to terraform as `-lock` |
| `defaults { var_file "<path>" {} }` | Passed to terraform as `-var-file`, before the member's var-files |
| `defaults { plugin "<id>" { … } }` | Enables or disables a plugin, and sets its options |

A `var_file` path is relative to the member directory terraform runs in, at every level.

Plugin settings are read only from `plugin` blocks inside the root file's `defaults` block. `enabled = false` keeps
the plugin out of tfui: its key does nothing, and its command reports that it is disabled and exits 1. Each
[plugin page](../plugins/index.md) lists the options that plugin reads.

## A member's tfui.hcl

A `tfui.hcl` inside a member directory adds var-files, variables and per-workspace settings for that member:

```hcl
# modules/vpc/tfui.hcl

var_file "base.tfvars" {}
var "region" { value = "us-east-1" }

workspace "production" {
  lock_timeout = "30s"
  var_file "prod.tfvars" {}
  var "environment" { value = "prod" }
}

workspace "dev-*" {
  var_file "dev.tfvars" {}
  var "environment" { value = "dev" }
}
```

| Block or attribute | Where | What it does |
|---|---|---|
| `var_file "<path>" {}` | Top level or `workspace` | Passed to terraform as `-var-file` |
| `var "<name>" { value = "…" }` | Top level or `workspace` | Passed to terraform as `-var <name>=<value>` |
| `workspace "<name>" { … }` | Top level | Settings used only when that workspace is selected |
| `lock_timeout` | `workspace` only | Passed to terraform as `-lock-timeout` |

The `terraform`, `member` and `defaults` blocks belong to the root file only. A member's `tfui.hcl` that declares
one of them, or that fails to parse, is ignored as a whole.

### Workspace matching

A `workspace` block matches the selected workspace by name, or by a glob pattern such as `dev-*`:

- An exact name beats a glob.
- Among globs, the first one declared wins.
- With no match, only the member's top-level settings apply.

## How the levels combine

```
root defaults → member top level → matching workspace block → CLI flags
```

- **Var-files** stack in that order, all of them passed to terraform.
- **Variables** merge by name; a later level wins.
- **`lock`** comes from the root `defaults`, and `lock_timeout` from the workspace block; `tfui plan -lock=false`
  and `-lock-timeout=<duration>` override them for one run.

## Binary resolution

| Configured | Binary tfui runs |
|---|---|
| `-terraform-bin <path>` | That binary, whatever the config says |
| `terraform { bin = "<path>" }` | That binary |
| Neither | `terraform` from your `PATH` |

tfui never looks for `tofu` on its own. `tfui scaffold` writes the first of `terraform`, `tofu` and `terragrunt` it
finds on your `PATH` into `bin`.

## Run tfui on the project

```bash
cd my-project
tfui                       # opens the member picker; pick one with Enter
tfui -chdir modules/vpc    # starts in modules/vpc, without the picker
```

Press `C` in tfui to switch member or workspace. Every command and flag is in the
[CLI reference](../reference/cli-reference.md).
