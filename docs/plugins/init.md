---
layout: plugin
title: Init
id: init
key: i
description: Generate tfui.hcl configuration interactively
category: setup
default_enabled: true
---

## Overview

The Init plugin provides an interactive wizard to generate a `tfui.hcl` configuration file, similar to `npm init`. It detects the terraform binary, scans for terraform directories, and writes a config file to the current directory.

## Usage

Press `i` from the home screen, or run `tfui init` from the CLI.

### Interactive (TUI)

The wizard progresses through these states:

1. **Detecting** — scans the filesystem for terraform directories
2. **Review** — shows detected members; toggle them on/off
3. **Confirm** — previews the HCL that will be written
4. **Done** — file written successfully

| Key | Action |
|-----|--------|
| `j` / `k` | Navigate detected paths |
| `Space` | Toggle a path on/off |
| `Enter` | Proceed to next step |
| `Esc` | Cancel / go back |

### Non-interactive (CLI)

```bash
tfui init          # fails if tfui.hcl already exists
tfui init --force  # overwrites existing tfui.hcl
```

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
