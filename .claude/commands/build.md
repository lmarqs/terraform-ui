---
allowed-tools: Bash(mise run:*)
description: Build Go binary (mise run build)
---

## Mise task: `build`

Run `mise run build` to compile the Go binary to `dist/tfui`.

Runs `fmt` and `lint` first as dependencies.

Accepts an optional version argument: `mise run build 1.0.0`

Related commands: /fmt, /lint, /test, /run
