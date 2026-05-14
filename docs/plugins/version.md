---
layout: plugin
title: Version
id: version
key:
category: utility
---

# Version

Display tfui version, platform, and the resolved terraform binary version with provider selections.

## Interactive (TUI)

Type `:version` in command mode. The plugin is NavPush — pressing `esc` or `q` returns to the previous view.

### Keybindings

| Key | Action |
|-----|--------|
| `q` / `esc` | Back |

### Display

```
tfui v0.1.0
on linux_amd64

terraform v1.14.9
on linux_amd64
+ provider registry.terraform.io/hashicorp/aws v5.0.0
+ provider registry.terraform.io/hashicorp/local v2.8.0
```

## CLI

```bash
tfui version          # text output
tfui version -json    # structured JSON
```

See [CLI Reference](../cli-reference.md#tfui-version) for full details.
