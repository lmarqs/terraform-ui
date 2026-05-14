---
layout: command
title: Scaffold
id: scaffold
description: Generate tfui.hcl configuration
category: setup
---

## Overview

The `scaffold` command generates a `tfui.hcl` configuration file by detecting terraform project patterns in the working directory. It detects the terraform binary, scans for terraform directories, and writes a config file.

## Usage

```bash
tfui scaffold              # interactive (prompts for member selection)
tfui scaffold --yes        # non-interactive (use detected defaults)
tfui scaffold --force      # overwrite existing tfui.hcl
tfui scaffold --yes --force
```

By default, `tfui scaffold` is interactive when run in a TTY — it shows detected members and lets you toggle them on/off before writing. Use `--yes` to skip prompts and accept all detected defaults.

## Detection Logic

The scanner checks for these common monorepo layouts:

- `modules/*` — each subdirectory containing .tf files
- `envs/*` — environment directories containing .tf files
- `infra/*` — infrastructure directories containing .tf files
- `services/*/terraform` — service-specific terraform directories
- `.` — root directory .tf files

Binary detection order: `terraform` → `tofu` → `terragrunt`.

## Output

```hcl
terraform {
  bin = "terraform"
}

member "modules/vpc" {}
member "envs/prod" {}
```

## Related

- [Context](context.md) — manage project/chdir/workspace selection using the generated config
- [Configuration](../configuration.md) — full config reference
