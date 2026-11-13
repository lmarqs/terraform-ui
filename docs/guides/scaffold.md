---
layout: default
parent: Guides — Getting Started with Terraform UI
title: "Generate tfui.hcl with tfui scaffold"
nav_order: 3
seo:
  description: "tfui scaffold detects the terraform binary and the Terraform directories in your project, then writes a tfui.hcl configuration file for them."
description: "Guides a user through generating tfui.hcl with scaffold, and what it detects."
audience: "Users with an existing Terraform project of several root modules who have not written a tfui.hcl"
---

# Generate tfui.hcl with scaffold

Run `tfui scaffold` in the project root. It finds the directories that hold Terraform files, lets you untick any,
and writes `tfui.hcl` with one `member` block for each.

## What it finds

Scaffold looks for `.tf` files in these places, relative to the project directory:

- `modules/*`
- `envs/*`
- `infra/*`
- `services/*/terraform`
- `.`, the project directory itself

Directories elsewhere are not detected; add them to `tfui.hcl` by hand as `member` blocks.

The `bin` value is the first of `terraform`, `tofu` and `terragrunt` found on your `PATH`, or `terraform` if none
is.

## Run it

```bash
cd my-project
tfui scaffold
```

In a terminal, scaffold asks three things in turn:

1. **Select terraform binary:** the detected binary is preselected; move with `j`/`k`, confirm with `Enter`.
2. **Detected members:** every detected directory starts ticked; `Space` toggles the one under the cursor,
   `Enter` confirms.
3. **Preview:** shows the file; `y` or `Enter` writes it, `n` or `Esc` goes back to the members.

`Esc` in the first two steps, or `q` in any, aborts without writing. With `-yes`, or when stdin is not a terminal, scaffold skips the questions and keeps
everything it detected.

Scaffold writes `tfui.hcl` to the project directory (`-project`, default the current directory) and prints it.
For a project with `envs/prod` and `modules/vpc`:

```hcl
terraform {
  bin = "terraform"
}

member "envs/prod" {}

member "modules/vpc" {}
```

When it finds no directories, it writes a file with only the `terraform` block.

## Flags and exit codes

| Flag | Description |
|------|-------------|
| `-yes` | Skip the questions and keep every detected directory |
| `-force` | Overwrite an existing `tfui.hcl` |

| Code | Meaning |
|------|---------|
| 0 | `tfui.hcl` written, or the wizard aborted |
| 1 | `tfui.hcl` already exists (without `-force`), or the write failed |

## Next

Run `tfui` in the same directory: it opens the member picker. Pick a member with `Enter`. To add var-files,
variables or workspace settings, see [Configuration](configuration.md).
