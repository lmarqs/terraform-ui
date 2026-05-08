#!/usr/bin/env bash

_common_setup() {
  PROJECT_ROOT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"

  load "$PROJECT_ROOT/tests/test_helper/bats-support/load"
  load "$PROJECT_ROOT/tests/test_helper/bats-assert/load"
  load "$PROJECT_ROOT/tests/test_helper/bats-file/load"

  source "$PROJECT_ROOT/lib/tfui.sh"

  _TFUI_UI_LINES="10"
  _TFUI_MESSAGE=""
  _TFUI_START_TIME=0
  _TFUI_ANIMATION_PID=""
}

_tfui_capture_fd3() {
  TFUI_UI_CAPTURE="$BATS_TEST_TMPDIR/ui-output"
}

_tfui_get_ui_output() {
  cat "$TFUI_UI_CAPTURE"
}
