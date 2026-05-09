#!/usr/bin/env bats

setup() {
  load 'helpers/common-setup'
  _common_setup
}

@test "line matches refresh pattern" {
  _tfui_progress_line_matches "module.a.resource_b: Refreshing state..." ": Refreshing state\.\.\."
}

@test "unrelated line does not match" {
  run _tfui_progress_line_matches "Terraform initialized" ": Refreshing state\.\.\."
  [ "$status" -eq 1 ]
}

@test "multi-pattern matches first alternative" {
  _tfui_progress_line_matches "module.x: Creating..." ": Creating\.\.\.|: Modifying\.\.\."
}

@test "multi-pattern matches second alternative" {
  _tfui_progress_line_matches "module.x: Modifying..." ": Creating\.\.\.|: Modifying\.\.\."
}

@test "empty line does not match" {
  run _tfui_progress_line_matches "" ": Creating\.\.\."
  [ "$status" -eq 1 ]
}
