#!/usr/bin/env bash

_fixture_prepare() {
  local fixture_name="$1"
  local fixture_src="$PROJECT_ROOT/tests/fixtures/$fixture_name"
  FIXTURE_DIR="$BATS_TEST_TMPDIR/$fixture_name"

  cp -r "$fixture_src" "$FIXTURE_DIR"
  mkdir -p "$FIXTURE_DIR/out"

  terraform -chdir="$FIXTURE_DIR" init -backend=false -input=false >/dev/null 2>&1

  _TFUI_WORKING_DIR="$FIXTURE_DIR"
  _TFUI_OUTPUT_FILE="$BATS_TEST_TMPDIR/output"
  : > "$_TFUI_OUTPUT_FILE"
  _TFUI_STRATEGY="_tfui_strategy_silent"
  _TFUI_UI_LINES="10"
}

_fixture_plan() {
  PLAN_FILE="$BATS_TEST_TMPDIR/plan.json"
  tfui_plan "$@" --out "$PLAN_FILE" 3>/dev/null
}
