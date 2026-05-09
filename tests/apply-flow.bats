#!/usr/bin/env bats

setup() {
  load 'helpers/common-setup'
  load 'helpers/fixtures'
  _common_setup
}

@test "apply creates the planned files on disk" {
  _fixture_prepare "create"
  _fixture_plan "Planning" >/dev/null
  tfui_apply "$PLAN_FILE" "Applying" 3>/dev/null 2>/dev/null

  [ -f "$FIXTURE_DIR/out/alpha.txt" ]
  [ -f "$FIXTURE_DIR/out/beta.txt" ]
}

@test "apply removes files for destroy plan" {
  _fixture_prepare "delete"
  _fixture_plan "Planning" >/dev/null
  tfui_apply "$PLAN_FILE" "Applying" 3>/dev/null 2>/dev/null

  [ ! -f "$FIXTURE_DIR/out/to_remove.txt" ]
}

@test "apply captures terraform output" {
  _fixture_prepare "create"
  _fixture_plan "Planning" >/dev/null
  tfui_apply "$PLAN_FILE" "Applying" 3>/dev/null 2>/dev/null

  [[ "$(cat "$_TFUI_OUTPUT_FILE")" == *"Creation complete"* ]]
}

@test "run_sub propagates command failure as exit 1" {
  run bash -c '
    source "'"$PROJECT_ROOT"'/lib/tfui.sh"
    exec 3>/dev/null
    _TFUI_WORKING_DIR=$(mktemp -d)
    _TFUI_OUTPUT_FILE=$(mktemp)
    _TFUI_UI_LINES="10"
    _TFUI_MESSAGE="Test"
    _TFUI_START_TIME=$SECONDS
    _tfui_run_sub "Failing" "exit 99"
  '
  [ "$status" -eq 1 ]
}
