---
allowed-tools: Bash(mise run:*), Bash(go run:*)
description: Run tfui TUI in development mode (mise run go:run)
---

## Mise task: `go:run`

Run `mise run go:run` to launch the interactive TUI in development mode.

Pass flags after `--`: `mise run go:run -- --dir ../medprev-cloud-iac`

Non-interactive modes: `go run ./cmd/tfui plan --dir ./tests/fixtures/simple --mode agent`

Related commands: /go-build, /go-test
