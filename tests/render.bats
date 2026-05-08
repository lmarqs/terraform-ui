#!/usr/bin/env bats

setup() {
  load 'test_helper/common-setup'
  _common_setup
}

@test "render_header writes spinner, message, and elapsed to fd3" {
  local tmpfile="$BATS_TEST_TMPDIR/ui-out"
  _tfui_ui_render_header 0 "Planning" 5 3>"$tmpfile"
  result=$(cat "$tmpfile")
  [[ "$result" == *"⠋"* ]]
  [[ "$result" == *"Planning"* ]]
  [[ "$result" == *"(5s)"* ]]
}

@test "render_progress writes counts and percentage to fd3" {
  local tmpfile="$BATS_TEST_TMPDIR/ui-out"
  _tfui_ui_render_progress 10 100 3>"$tmpfile"
  result=$(cat "$tmpfile")
  [[ "$result" == *"10/100"* ]]
  [[ "$result" == *"10%"* ]]
}

@test "render_status writes label and dots to fd3" {
  local tmpfile="$BATS_TEST_TMPDIR/ui-out"
  _tfui_ui_render_status "Calculating" ".." 3>"$tmpfile"
  result=$(cat "$tmpfile")
  [[ "$result" == *"Calculating"* ]]
  [[ "$result" == *".."* ]]
}
