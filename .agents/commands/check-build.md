---
allowed-tools: Bash(mise run:*)
description: Compile check without artifacts (mise run check:build)
---

## Mise task: `check:build`

Run `mise run check:build` to verify the project compiles (go build ./...).

Faster than `mise run build` — no goreleaser, no artifacts.

Related commands: /build, /lint
