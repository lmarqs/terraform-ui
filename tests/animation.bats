#!/usr/bin/env bats

setup() {
  load 'test_helper/common-setup'
  _common_setup
}

@test "advance_dots: tick not on interval preserves state" {
  run _tfui_ui_advance_dots "." 3
  assert_output "."
}

@test "advance_dots: tick on interval adds a dot" {
  run _tfui_ui_advance_dots "." 8
  assert_output ".."
}

@test "advance_dots: tick on interval with empty dots starts with one" {
  run _tfui_ui_advance_dots "" 16
  assert_output "."
}

@test "advance_dots: at max length resets to one dot" {
  run _tfui_ui_advance_dots "....." 24
  assert_output "."
}

@test "advance_dots: just below max grows" {
  run _tfui_ui_advance_dots "...." 32
  assert_output "....."
}

@test "advance_dots: non-interval tick preserves existing dots" {
  run _tfui_ui_advance_dots "..." 7
  assert_output "..."
}
