---
allowed-tools: Bash(mise run:*), Bash(docker:*), Bash(ls:*)
description: Run code coverage via Docker + kcov (mise run coverage)
---

## Mise task: `coverage`

Run coverage with `mise run coverage`.

This builds and runs `Dockerfile.coverage` (kcov + bats + jq) against the test suite. Output goes to `coverage/` directory with HTML report at `coverage/index.html`.

Requirements: Docker must be running.

If the build fails, check Dockerfile.coverage. If tests fail under kcov, note that kcov can interfere with signal handling and process tracing — some animation/lifecycle tests may behave differently.
