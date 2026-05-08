#!/usr/bin/env bats

setup() {
  load 'test_helper/common-setup'
  _common_setup
}

@test "open renders initial header with message" {
  local tmpfile="$BATS_TEST_TMPDIR/ui-out"
  _TFUI_UI_LINES="10"
  _TFUI_MESSAGE="Testing"
  _tfui_ui_open 3>"$tmpfile"
  result=$(cat "$tmpfile")
  [[ "$result" == *"Testing"* ]]
  [[ "$result" == *"(0s)"* ]]
}

@test "close restores cursor visibility" {
  local tmpfile="$BATS_TEST_TMPDIR/ui-out"
  _TFUI_UI_LINES="10"
  _tfui_ui_close 3>"$tmpfile"
  result=$(cat "$tmpfile")
  [[ "$result" == *$'\033[?25h'* ]]
}
