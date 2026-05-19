---
layout: default
title: Configuration
parent: Getting Started
nav_order: 2
description: Configure terraform-ui for your project
---

# Configuration

terraform-ui uses an optional `tfui.hcl` file for project configuration. Place it in your project root directory. Config is purely additive — tfui works with zero configuration.

## Two Modes

| Mode | When | Behavior |
|------|------|----------|
| **Standalone** | No `tfui.hcl` in CWD, no `-project` | TUI skin over terraform. CWD = working dir. |
| **Project** | `tfui.hcl` in CWD (or `-project`) | Full config resolution, member directories, workspace overrides. |

## Config File

```hcl
# tfui.hcl

terraform {
  bin = "terraform"    # or "tofu", or full path
}

member "modules/vpc" {}
member "modules/ecs" {}
member "environments/prod" {}

cache {
  staleness_threshold = "5m"
}

ai {
  enabled  = true
  provider = "bedrock"
  region   = "us-east-1"
}

defaults {
  parallelism = 10
  lock        = true

  var_file "common/tags.tfvars" {}

  plugin "risk" {
    enabled = true
    level   = "high"
  }
}
```

Every block is optional. An empty `tfui.hcl` is valid.

## Child Config (per member)

Place a `tfui.hcl` in any member directory for per-module overrides:

```hcl
# modules/vpc/tfui.hcl

var_file "base.tfvars" {}

plugin "risk" {
  level = "critical"
}

workspace "production" {
  var_file "prod.tfvars" {}
  var "environment" { value = "prod" }
}

workspace "staging" {
  var_file "staging.tfvars" {}
  var "environment" { value = "staging" }
}

workspace "dev-*" {
  plugin "risk" {
    level = "low"
  }
}
```

### Locked Fields

Child configs **cannot** declare these blocks (they're root-only):
- `terraform` — binary must be consistent across all members
- `member` — membership is project-level identity
- `cache` — safety invariants apply project-wide
- `ai` — credential scope is project-wide
- `defaults` — inheritance root

## Resolution Chain

```
Root defaults → Child top-level → Workspace block → CLI flags → [-- passthrough]
```

- **Var-files**: concatenated in order (all levels stacked)
- **Vars**: map merge (later level wins for same key)
- **Plugins**: per-plugin option merge (later wins)
- **Scalars** (parallelism, lock): last writer wins

## Workspace Matching

Workspace blocks match by name. Glob patterns supported:

```hcl
workspace "production" { ... }  # exact match
workspace "dev-*" { ... }       # matches dev-us-east-1, dev-staging, etc.
```

Rules:
- Exact match beats glob
- First glob match wins (declaration order)
- No match → workspace block skipped, child top-level still applies

## CLI Flags

```bash
# Project mode
tfui -project ./infra                 # explicit project root
tfui -chdir modules/vpc               # select member (validated against member blocks)
tfui -terraform-bin /usr/local/bin/tofu

# Terraform flags (single or double dash)
tfui plan -target=aws_instance.web     # terraform-style
tfui plan -target=aws_instance.web    # cobra-style (both work)
tfui plan -var-file=prod.tfvars -var=env=prod
tfui plan -destroy
tfui plan -parallelism=5 -lock=false

# Passthrough (everything after -- goes to terraform unmodified)
tfui plan -- -no-color -compact-warnings
```

## Standalone Mode

When no `tfui.hcl` exists and no `-project` is passed:

- tfui runs from CWD
- `-chdir` acts like terraform's `-chdir` (raw dir change)
- No member validation, no child configs, no resolution chain
- Just a TUI shell over terraform

## Binary Resolution

| State | What happens |
|-------|-------------|
| `terraform.bin = "tofu"` in config | Passed to terraform-exec |
| `-terraform-bin terraform` flag | Flag wins over config |
| Nothing configured | Empty string → terraform-exec errors → tfui appends hint |

No auto-detection. Only `tfui scaffold` can detect binaries.

## Examples

### Single-module project (minimal)

```bash
cd my-terraform-project
tfui                        # just works, no config needed
```

### Monorepo with workspaces

```hcl
# tfui.hcl (project root)
terraform {
  bin = "terraform"
}

member "modules/vpc" {}
member "modules/ecs" {}
member "modules/rds" {}

defaults {
  var_file "common/tags.tfvars" {}
}
```

```hcl
# modules/vpc/tfui.hcl (child)
workspace "production" {
  var_file "prod.tfvars" {}
  var "lock_timeout" { value = "30s" }
}
```

Running:
```bash
tfui -chdir modules/vpc           # selects vpc member
tfui                                # shows chdir picker
```
