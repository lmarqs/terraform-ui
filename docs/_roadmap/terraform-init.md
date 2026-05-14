---
title: Terraform Init Passthrough
status: idea
priority: medium
created: 2026-05-14
effort: small
tags: [cli, passthrough]
depends_on: []
---

## Summary

`tfui init` should run `terraform init` — either as a pure passthrough or as a smart multi-member init that initializes all configured members with progress feedback.

## Need

With the rename of config-generation to `tfui scaffold`, the `init` subcommand is now free. Users expect `init` to mean terraform init. Multi-member init (running terraform init across all configured members) would add real value for monorepo setups.

## Options

1. **Smart multi-member init**: `tfui init` runs terraform init on all members, shows progress per-member. Single-member via `--chdir`.
2. **Pure passthrough**: `tfui init` = `terraform init` exactly (same flags, same output). Adds no value over calling terraform directly.

## Design Notes

- The `sdk.Service.Init()` method already exists and wraps terraform-exec
- Multi-member init should respect `member` blocks from `tfui.hcl`
- Exit code: 0 = all succeeded, 1 = any failed
- Consider `--parallel` flag for concurrent init across members
