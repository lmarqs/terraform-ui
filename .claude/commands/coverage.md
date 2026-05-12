---
allowed-tools: Bash(mise run:*), Bash(go test:*), Bash(go tool:*)
description: Run Go coverage report (mise run test:coverage)
---

## Mise task: `test:coverage`

Run `mise run test:coverage` to generate a coverage report with 90% threshold enforcement.

Excludes cmd/ and internal/terraform from coverage.

Related commands: /test, /lint
