---
allowed-tools: Bash(mise run:*)
description: Build Go binary (mise run go:build)
---

## Mise task: `go:build`

Run `mise run go:build` to compile the Go binary to `dist/tfui`.

Accepts an optional version argument: `mise run go:build 1.0.0`

Injects version via ldflags. Verify with `./dist/tfui version`.

Related commands: /go-test, /go-run
