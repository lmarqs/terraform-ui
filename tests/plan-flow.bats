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

@test "tfui_plan produces correct tree view" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  run bash -c '
    source "'"$PROJECT_ROOT"'/lib/tfui.sh"
    exec 3>/dev/null
    PATH="'"$MOCK_DIR"':$PATH"
    _TFUI_WORKING_DIR="'"$_TFUI_WORKING_DIR"'"
    _TFUI_OUTPUT_FILE="'"$_TFUI_OUTPUT_FILE"'"
    _TFUI_STRATEGY="_tfui_strategy_silent"
    _TFUI_UI_LINES="10"
    tfui_plan "Planning" --out "'"$plan_file"'"
  '
  assert_line --partial "+ module.a.resource_b"
  assert_line --partial "~ module.a.resource_a"
  assert_line --partial "- module.b.resource_c"
  assert_line --partial "Plan: 1 to add, 1 to change, 1 to destroy."
}

@test "tfui_confirm detects changes after plan" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  bash -c '
    source "'"$PROJECT_ROOT"'/lib/tfui.sh"
    exec 3>/dev/null
    PATH="'"$MOCK_DIR"':$PATH"
    _TFUI_WORKING_DIR="'"$_TFUI_WORKING_DIR"'"
    _TFUI_OUTPUT_FILE="'"$_TFUI_OUTPUT_FILE"'"
    _TFUI_STRATEGY="_tfui_strategy_silent"
    _TFUI_UI_LINES="10"
    tfui_plan "Planning" --out "'"$plan_file"'"
  ' >/dev/null 2>&1
  tfui_confirm "$plan_file" --auto-approve
}

@test "plan preserves original message" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  bash -c '
    source "'"$PROJECT_ROOT"'/lib/tfui.sh"
    exec 3>/dev/null
    PATH="'"$MOCK_DIR"':$PATH"
    _TFUI_WORKING_DIR="'"$_TFUI_WORKING_DIR"'"
    _TFUI_OUTPUT_FILE="'"$_TFUI_OUTPUT_FILE"'"
    _TFUI_STRATEGY="_tfui_strategy_silent"
    _TFUI_UI_LINES="10"
    tfui_plan "Planning module: sa-east-1" --out "'"$plan_file"'"
    echo "$_TFUI_MESSAGE"
  ' >/dev/null 2>&1
  # Verify in-process (the subshell above can't export back)
  # Use silent strategy which doesn't touch fd3
  tfui_plan "Planning module: sa-east-1" --out "$plan_file" 3>/dev/null 2>/dev/null
  [ "$_TFUI_MESSAGE" = "Planning module: sa-east-1" ]
}

@test "plan with progress strategy produces summary" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  run bash -c '
    source "'"$PROJECT_ROOT"'/lib/tfui.sh"
    exec 3>/dev/null
    PATH="'"$MOCK_DIR"':$PATH"
    _TFUI_WORKING_DIR="'"$_TFUI_WORKING_DIR"'"
    _TFUI_OUTPUT_FILE="'"$_TFUI_OUTPUT_FILE"'"
    _TFUI_STRATEGY="_tfui_strategy_progress"
    _TFUI_UI_LINES="10"
    tfui_plan "Planning" --out "'"$plan_file"'"
  '
  assert_line --partial "Plan: 1 to add, 1 to change, 1 to destroy."
}
