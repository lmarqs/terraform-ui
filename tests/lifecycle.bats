#!/usr/bin/env bats

setup() {
  load 'helpers/common-setup'
  _common_setup
}

@test "lifecycle_die prints captured output to stderr" {
  run bash -c '
    source "'"$PROJECT_ROOT"'/lib/tfui.sh"
    exec 3>/dev/null
    _TFUI_OUTPUT_FILE=$(mktemp)
    _TFUI_ANIMATION_PID=""
    _TFUI_UI_LINES="10"
    echo "error details" > "$_TFUI_OUTPUT_FILE"
    _tfui_lifecycle_die
  '
  [ "$status" -eq 1 ]
  assert_output --partial "error details"
}

@test "lifecycle_on_exit removes temp files" {
  local tmpdir="$BATS_TEST_TMPDIR/wd"
  mkdir -p "$tmpdir"
  local output_file="$BATS_TEST_TMPDIR/output"
  touch "$output_file"
  touch "$tmpdir/tfplan.out"

  bash -c '
    source "'"$PROJECT_ROOT"'/lib/tfui.sh"
    exec 3>/dev/null
    _TFUI_OUTPUT_FILE="'"$output_file"'"
    _TFUI_WORKING_DIR="'"$tmpdir"'"
    _TFUI_ANIMATION_PID=""
    _tfui_lifecycle_on_exit
  '

  [ ! -f "$output_file" ]
  [ ! -f "$tmpdir/tfplan.out" ]
}

@test "lifecycle_on_exit writes show-cursor escape to fd3" {
  local tmpfile="$BATS_TEST_TMPDIR/ui-out"
  _TFUI_OUTPUT_FILE="$BATS_TEST_TMPDIR/output"
  _TFUI_WORKING_DIR="$BATS_TEST_TMPDIR"
  touch "$_TFUI_OUTPUT_FILE"
  _tfui_lifecycle_on_exit 3>"$tmpfile"
  result=$(cat "$tmpfile")
  [[ "$result" == *$'\033[?25h'* ]]
}
