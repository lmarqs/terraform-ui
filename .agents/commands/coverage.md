---
allowed-tools: Bash(mise run:*)
description: Run Go coverage report (mise run test:coverage)
---

## Mise task: `test:coverage`

Run `mise run test:coverage` to generate a coverage report with 100% threshold enforcement.

Excludes cmd/ and internal/terraform/exec from coverage.

Related commands: /test, /lint
