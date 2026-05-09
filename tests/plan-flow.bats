#!/usr/bin/env bats

setup() {
  load 'helpers/common-setup'
  load 'helpers/fixtures'
  _common_setup
}

# -- Plan output scenarios --

@test "given new resources, plan shows + symbols and add count" {
  _fixture_prepare "create"
  run _fixture_plan "Planning"
  assert_line --partial "+ local_file.alpha"
  assert_line --partial "+ local_file.beta"
  assert_line --partial "Plan: 2 to add, 0 to change, 0 to destroy."
}

@test "given changed resource, plan shows ~ symbol and change count" {
  _fixture_prepare "update"
  run _fixture_plan "Planning"
  assert_line --partial "~ terraform_data.doc"
  assert_line --partial "Plan: 0 to add, 1 to change, 0 to destroy."
}

@test "given removed resource, plan shows - symbol and destroy count" {
  _fixture_prepare "delete"
  run _fixture_plan "Planning"
  assert_line --partial "- local_file.to_remove"
  assert_line --partial "Plan: 0 to add, 0 to change, 1 to destroy."
}

@test "given renamed resource, plan shows -/+ replace symbol" {
  _fixture_prepare "replace"
  run _fixture_plan "Planning"
  assert_line --partial "-/+ local_file.moved"
}

@test "given no drift, plan shows up-to-date message" {
  _fixture_prepare "no-changes"
  run _fixture_plan "Planning"
  assert_output --partial "No changes. Infrastructure is up-to-date."
}

@test "given many resources, plan counts all in summary" {
  _fixture_prepare "multi-resource"
  run _fixture_plan "Planning"
  assert_line --partial "Plan: 5 to add, 0 to change, 0 to destroy."
}

# -- Confirm scenarios --

@test "confirm returns 0 when plan has changes" {
  _fixture_prepare "create"
  _fixture_plan "Planning" >/dev/null
  tfui_confirm "$PLAN_FILE" --auto-approve
}

@test "confirm returns 1 when plan has no changes" {
  _fixture_prepare "no-changes"
  _fixture_plan "Planning" >/dev/null
  run tfui_confirm "$PLAN_FILE" --auto-approve
  [ "$status" -eq 1 ]
}

# -- Message preservation --

@test "plan preserves the original status message" {
  _fixture_prepare "create"
  _fixture_plan "Planning module: sa-east-1" >/dev/null
  [ "$_TFUI_MESSAGE" = "Planning module: sa-east-1" ]
}

# -- Progress strategy --

@test "progress strategy tracks resource operations" {
  _fixture_prepare "multi-resource"
  run bash -c '
    source "'"$PROJECT_ROOT"'/lib/tfui.sh"
    exec 3>/dev/null
    _TFUI_WORKING_DIR="'"$BATS_TEST_TMPDIR"'/multi-resource"
    _TFUI_OUTPUT_FILE="'"$BATS_TEST_TMPDIR"'/output"
    : > "$_TFUI_OUTPUT_FILE"
    _TFUI_STRATEGY="_tfui_strategy_progress"
    _TFUI_UI_LINES="10"
    tfui_plan "Planning" --out "'"$BATS_TEST_TMPDIR"'/plan.json"
  '
  assert_line --partial "Plan: 5 to add, 0 to change, 0 to destroy."
}
