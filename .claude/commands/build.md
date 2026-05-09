---
allowed-tools: Bash(mise run:*), Bash(bash -n:*)
description: Syntax check the library (mise run build)
---

## Mise task: `build`

Run syntax check with `mise run build`.

This runs `bash -n lib/tfui.sh` which checks for syntax errors without executing. Always run this after editing the library.

Related commands: /test-run
