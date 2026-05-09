---
layout: plugin
title: State Browser
id: state
key: s
description: Browse and inspect terraform state resources
category: navigation
default_enabled: true
---

## Overview

The State Browser plugin loads the current terraform state and presents all managed resources in a filterable list. You can type to filter by address, type, or module, and press Enter to inspect the full resource detail (attributes, dependencies, etc.).

## Usage

Press `s` to open the State Browser. It immediately loads the state from the current working directory.

| Key | Action |
|-----|--------|
| `j` / `k` | Navigate up/down |
| `g` / `G` | Jump to first/last resource |
| `Enter` | Inspect selected resource |
| Any character | Filter resources by address/type/module |
| `Backspace` | Remove last filter character |
| `r` | Refresh state |
| `Esc` / `q` | Go back (or exit detail view) |

## Configuration

```yaml
# tfui.yaml
plugins:
  state:
    enabled: true
```

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enabled` | bool | `true` | Enable/disable the plugin |

## Screenshots/Output

Resource list:

```
State Browser

 aws_instance.web  aws_instance  [module.compute]
 aws_s3_bucket.logs  aws_s3_bucket
 aws_security_group.main  aws_security_group  [module.networking]
 aws_iam_role.deploy  aws_iam_role

4 resources

j/k navigate  Enter inspect  / filter  r refresh  Esc back
```

With filter active:

```
State Browser

filter: s3

 aws_s3_bucket.logs  aws_s3_bucket
 aws_s3_bucket.config  aws_s3_bucket

2/4 resources

j/k navigate  Enter inspect  / filter  r refresh  Esc back
```

Resource detail:

```
Resource Detail
aws_s3_bucket.logs

{
  "bucket": "my-app-logs-prod",
  "acl": "private",
  "region": "us-east-1",
  "tags": {
    "env": "production"
  }
}

Esc/q to go back
```

## Related

- [Workspaces](workspaces.md) -- switch workspace before browsing state
- [Projects](projects.md) -- switch project to browse different state files
