---
allowed-tools: Bash(bats:*), Bash(mise run:*), Read, Edit, Write
description: Add a new test for a function or scenario
---

## Context

- Library source: !`cat lib/tfui.sh | head -75`
- Existing test files: !`ls tests/*.bats`
- Test helper: !`cat tests/helpers/common-setup.bash`
- Fixture helper: !`cat tests/helpers/fixtures.bash`

## Instructions

Add a new BATS test. Follow these conventions:

1. Place the test in the appropriate existing `.bats` file by feature area
2. Name tests as BDD scenarios: "given X, Y happens" or "function: description of behavior"
3. Use `_common_setup` in setup(), load helpers as needed
4. For functions that write to fd3, use `3>/dev/null` or `3>"$tmpfile"` on the call
5. For integration tests with terraform, use `_fixture_prepare "name"` + `_fixture_plan "msg"`
6. Use `run` + `assert_output`/`assert_line` for output assertions
7. Use `[ "$var" = "expected" ]` for state assertions

After writing the test, run it with `bats tests/<file>.bats` to verify it passes.
