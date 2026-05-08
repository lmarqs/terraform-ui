#!/usr/bin/env bats

setup() {
  load 'test_helper/common-setup'
  _common_setup
}

@test "set message stores the value" {
  _tfui_state_set_message "hello"
  [ "$_TFUI_MESSAGE" = "hello" ]
}

@test "set message handles special characters" {
  _tfui_state_set_message "Planning module: sa-east-1"
  [ "$_TFUI_MESSAGE" = "Planning module: sa-east-1" ]
}

@test "reset timer sets start time to SECONDS" {
  _tfui_state_reset_timer
  [ "$_TFUI_START_TIME" = "$SECONDS" ]
}

@test "clear output truncates the file" {
  _TFUI_OUTPUT_FILE="$BATS_TEST_TMPDIR/output"
  echo "some content" > "$_TFUI_OUTPUT_FILE"
  _tfui_state_clear_output
  [ "$(wc -c < "$_TFUI_OUTPUT_FILE")" -eq 0 ]
}
