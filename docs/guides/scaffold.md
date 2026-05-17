---
layout: default
parent: Getting Started
title: Scaffold
nav_order: 3
description: Generate tfui.hcl configuration by detecting project patterns
---

## Overview

The `tfui scaffold` command generates a `tfui.hcl` configuration file by detecting terraform project patterns in the working directory. It detects the terraform binary, scans for terraform directories, and writes a config file.

## Usage

```bash
tfui scaffold              # Interactive (prompts for member selection)
tfui scaffold --yes        # Non-interactive (use detected defaults)
tfui scaffold --force      # Overwrite existing tfui.hcl
tfui scaffold --yes --force
```

By default, `tfui scaffold` is interactive when run in a TTY -- it shows detected members and lets you toggle them on/off before writing. Use `--yes` to skip prompts and accept all detected defaults.

| Flag | Description |
|------|-------------|
| `--yes` | Skip prompts, accept all detected defaults |
| `--force` | Overwrite existing tfui.hcl |

| Code | Meaning |
|------|---------|
| 0 | Config generated successfully |
| 1 | Error (no terraform files found, write failed) |

## Detection Logic

The scanner checks for these common monorepo layouts:

- `modules/*` -- each subdirectory containing .tf files
- `envs/*` -- environment directories containing .tf files
- `infra/*` -- infrastructure directories containing .tf files
- `services/*/terraform` -- service-specific terraform directories
- `.` -- root directory .tf files

Binary detection order: `terraform` → `tofu` → `terragrunt`.

## Output

Scaffold generates a `tfui.hcl` in the current directory:

```hcl
terraform {
  bin = "terraform"
}

member "modules/vpc" {}
member "envs/prod" {}
```

## Example

```
$ tfui scaffold
Detected terraform binary: terraform
Found 3 terraform directories:
  [x] modules/vpc
  [x] modules/ecs
  [ ] modules/deprecated

Toggle with space, confirm with enter.
```

## Related

- [Quick Start](getting-started.md) -- getting started with tfui
- [Configuration](configuration.md) -- full config reference
