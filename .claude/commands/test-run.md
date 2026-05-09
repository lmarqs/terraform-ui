---
allowed-tools: Bash(mise run:*), Bash(bats:*)
description: Run the test suite (mise run test:run)
---

## Mise task: `test:run`

Run the test suite with `mise run test:run`.

If tests fail, analyze the output and fix the failing tests. Run only the failing file first with `bats tests/<file>.bats` for faster iteration.

Key facts:
- Tests use BATS framework (tests/*.bats)
- Helpers in tests/helpers/ (common-setup.bash, fixtures.bash)
- fd3 conflict: never use `exec 3>` in tests — use `3>/dev/null` or `3>"$file"` on function calls
- Flow tests use real terraform fixtures from tests/fixtures/
- JUnit XML reports output to reports/

Related commands: /test-add, /fixture-add, /coverage-run
