---
allowed-tools: Bash(mise run:*), Bash(go test:*)
description: Run Go unit tests (mise run go:test)
---

## Mise task: `go:test`

Run `mise run go:test` to execute all Go unit tests.

For a single package: `go test ./internal/terraform/...`

Key facts:
- Tests are in internal/terraform/*_test.go
- Table-driven tests for risk, phantom detection, and grouping
- Run `mise run go:coverage` for coverage report

Related commands: /go-build, /go-coverage
