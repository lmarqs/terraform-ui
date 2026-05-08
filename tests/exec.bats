#!/usr/bin/env bats

setup() {
  load 'test_helper/common-setup'
  _common_setup
  _TFUI_WORKING_DIR="$BATS_TEST_TMPDIR/workdir"
  _TFUI_OUTPUT_FILE="$BATS_TEST_TMPDIR/output"
  mkdir -p "$_TFUI_WORKING_DIR"
  : > "$_TFUI_OUTPUT_FILE"
}

@test "exec captures stdout" {
  _tfui_exec "echo hello"
  [ "$(cat "$_TFUI_OUTPUT_FILE")" = "hello" ]
}

@test "exec captures stderr" {
  _tfui_exec "echo err >&2"
  [ "$(cat "$_TFUI_OUTPUT_FILE")" = "err" ]
}

@test "exec propagates exit code" {
  run _tfui_exec "exit 42"
  [ "$status" -eq 42 ]
}

@test "exec runs in working directory" {
  touch "$_TFUI_WORKING_DIR/marker.txt"
  _tfui_exec "ls marker.txt"
  [ "$(cat "$_TFUI_OUTPUT_FILE")" = "marker.txt" ]
}

@test "silent strategy delegates to exec" {
  _tfui_strategy_silent "" "echo delegated"
  [ "$(cat "$_TFUI_OUTPUT_FILE")" = "delegated" ]
}

@test "run with progress strategy and empty patterns captures output" {
  _TFUI_STRATEGY="_tfui_strategy_progress"
  _TFUI_UI_LINES="10"
  _tfui_run "Rendering" "" "echo done" 3>/dev/null 2>/dev/null || true
  [ "$(cat "$_TFUI_OUTPUT_FILE")" = "done" ]
}
