---
layout: default
parent: Plugins
title: Init
id: init
description: Run terraform init with form-based options
category: operations
---

## Overview

The `init` plugin provides a TUI interface for `terraform init`. It presents a form for common init options (upgrade, migrate-state, reconfigure, backend-config) and shows real-time progress.

## Keybinding

| Key | Context | Action |
|-----|---------|--------|
| `i` | Home | Open init plugin |

## Form Options

| Field | Flag | Default |
|-------|------|---------|
| Upgrade | `-upgrade` | false |
| Migrate State | `-migrate-state` | false |
| Reconfigure | `-reconfigure` | false |
| Backend Config | `-backend-config=...` | (empty) |

Form values are preserved across runs within a session for convenient re-initialization.

## Flow

```
Home → [i] Init → Form (options) → Running (spinner + timer) → Result (success/error)
```

- `Enter` submits the form
- `Esc` cancels and returns home
- On completion, shows duration and any error output

## Related

- [Scaffold](scaffold.md) — generate `tfui.hcl` config (CLI-only)
- [Validate](validate.md) — run terraform validate
