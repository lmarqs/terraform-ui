---
layout: default
title: Install tfui — Terraform TUI Quick Start
parent: Guides — Getting Started with Terraform UI
nav_order: 1
seo:
  description: Install terraform-ui via Homebrew, go install, mise or a release binary, then review your first terraform plan in it.
description: "Guides a new user through installing tfui and reviewing a first plan."
audience: "New users who know Terraform and have not run tfui yet"
---

# Getting started

Install tfui, run `tfui` in a Terraform root module, and press `p`: tfui runs `terraform plan` and shows each
change as a row with a risk badge. This page takes you there, then shows the same plan run from a script.

## Install

Install with one of:

```bash
# Homebrew
brew tap lmarqs/terraform-ui https://github.com/lmarqs/terraform-ui
brew install --cask tfui

# Go
go install github.com/lmarqs/terraform-ui/cmd/tfui@latest

# mise
mise use github:lmarqs/terraform-ui
```

Or download `tfui_<version>_<os>_<arch>.tar.gz` from
[GitHub Releases](https://github.com/lmarqs/terraform-ui/releases), extract it and put `tfui` on your `PATH`.
Builds exist for Linux and macOS, on amd64 and arm64.

tfui runs the `terraform` binary on your `PATH`. To use OpenTofu instead, pass `-terraform-bin tofu`, or set
`terraform { bin = "tofu" }` in [`tfui.hcl`](configuration.md).

## Run tfui on your project

In a directory where `terraform init` has run, start tfui:

```bash
cd my-terraform-project
tfui
```

tfui opens on the Chdir screen, which lists the member directories of a project configured with
[`tfui.hcl`](#projects-with-several-root-modules); without one it is empty. Press `q` to go to the home screen: one
row per screen, each with the key that opens it. Press a key, or move with `j`/`k` and press `Enter`.

![tfui home screen, then a plan review]({{ site.baseurl }}/assets/demo/hero.gif)

## Review a plan

1. Press `p`. tfui runs `terraform plan` and lists the changes as a tree, grouped by module.
2. Each change carries a risk badge: critical, high, medium or low. [Risk analysis](../features/risk-analysis.md)
   gives the rules.
3. Move to a change and press `Enter` to expand its attribute diff.
4. Press `Space` on a change to pin it: the next plan targets only pinned resources.
5. Press `q` to return to the home screen.

Every screen and its keys are in the [plugin catalog](../plugins/index.md).

## Run the same plan from a script

The subcommands run without the TUI when you pass `-ci`, and in CI runners, where stderr is not a terminal:

```bash
# Change list, plan summary and Risk line on stdout
tfui plan -project ./infra -ci

# The plan as JSON (terraform show -json) on stdout
tfui plan -project ./infra -ci -json | jq .

# Apply without a prompt needs explicit approval, as with terraform
tfui apply -project ./infra -ci -auto-approve
```

`tfui plan` exits `2` when the plan has changes, `0` when it has none and `1` on error, so a CI gate can branch on
the exit code. The [CLI I/O contract](../reference/cli-io-contract.md) lists what each command writes and when.

## Projects with several root modules

When one repository holds several root modules, list them in a `tfui.hcl` at the repository root. `tfui scaffold`
writes it for you ([Scaffold](scaffold.md)); written by hand, it is one `member` block per directory:

```hcl
member "envs/prod" {}
member "envs/staging" {}
member "services/api/terraform" {}
```

Run `tfui` in the repository root. It opens a picker of these member directories first; pick one with `Enter`,
and tfui runs terraform in it. Press `C` at any time to switch member or workspace.
[Configuration](configuration.md) covers workspaces, var-files and the other settings.
