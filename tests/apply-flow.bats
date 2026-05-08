#!/usr/bin/env bats

setup() {
  load 'test_helper/common-setup'
  load 'test_helper/mock-terraform'
  _common_setup
  _mock_terraform_setup
  _TFUI_WORKING_DIR="$BATS_TEST_TMPDIR/workdir"
  _TFUI_OUTPUT_FILE="$BATS_TEST_TMPDIR/output"
  mkdir -p "$_TFUI_WORKING_DIR"
  : > "$_TFUI_OUTPUT_FILE"
  _TFUI_STRATEGY="_tfui_strategy_silent"
  _TFUI_UI_LINES="10"
}

@test "tfui_apply completes successfully" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  touch "$_TFUI_WORKING_DIR/tfplan.out"
  tfui_apply "$plan_file" "Applying" 3>/dev/null 2>/dev/null
  [ "$(head -1 "$_TFUI_OUTPUT_FILE")" = "module.a.resource_b: Creating..." ]
}

@test "run_sub propagates failure as exit 1" {
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
