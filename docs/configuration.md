---
layout: default
title: Configuration
description: Configure terraform-ui for your project
---

# Configuration

terraform-ui uses a `tfui.yaml` file for project configuration. Place it in your repository root — tfui walks up the directory tree to find it (like `.gitignore`).

## Config File

```yaml
# tfui.yaml

# Path to terraform (or tofu) binary
terraform_binary: terraform

# Project discovery for monorepos
projects:
  paths:
    - "modules/*"
    - "envs/*"
```

## Options

### `terraform_binary`

Path to the terraform binary. Defaults to `terraform`. Set to `tofu` for OpenTofu support.

```yaml
terraform_binary: tofu
```

### `projects`

Defines glob patterns to discover independent terraform projects in a monorepo. Each matched directory that contains `.tf` or `.tofu` files is treated as a selectable project.

```yaml
projects:
  paths:
    - "modules/*"        # matches modules/vpc, modules/ecs, etc.
    - "envs/**"          # matches envs/dev, envs/staging/us-east-1, etc.
    - "infra/shared"     # exact path
```

When configured, the TUI shows a project selector (`m` key) that lets you switch between terraform roots without leaving the app.

## CLI Flags

CLI flags override config file values:

```bash
tfui --dir ./infra                    # override working directory
tfui --terraform-bin /usr/local/bin/tofu   # override binary path
tfui plan --mode agent                # non-interactive mode
tfui plan --target aws_instance.web   # target specific resources
```

## Monorepo Examples

### Regional modules (like medprev-cloud-iac)

```yaml
projects:
  paths:
    - "modules/*"
```

Discovers: `modules/global`, `modules/sa-east-1`, `modules/us-east-1`, `modules/us-east-2`

### Environment-based (staging/production)

```yaml
projects:
  paths:
    - "envs/*"
```

Discovers: `envs/dev`, `envs/staging`, `envs/production`

### Mixed layout

```yaml
projects:
  paths:
    - "infra/*"
    - "services/*/terraform"
    - "platform/**"
```
