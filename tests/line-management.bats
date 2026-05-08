#!/usr/bin/env bats

setup() {
  load 'test_helper/common-setup'
  _common_setup
}

@test "default line state has header=1 status=0" {
  _TFUI_UI_LINES="10"
  [ "${_TFUI_UI_LINES:0:1}" = "1" ]
  [ "${_TFUI_UI_LINES:1:1}" = "0" ]
}

@test "enable line sets status bit to 1" {
  _TFUI_UI_LINES="10"
  _tfui_ui_enable_line $_TFUI_LINE_STATUS
  [ "$_TFUI_UI_LINES" = "11" ]
}

@test "is_line_enabled returns 0 when enabled" {
  _TFUI_UI_LINES="11"
  _tfui_ui_is_line_enabled $_TFUI_LINE_STATUS
}

@test "disable line sets status bit to 0" {
  _TFUI_UI_LINES="11"
  _tfui_ui_disable_line $_TFUI_LINE_STATUS
  [ "$_TFUI_UI_LINES" = "10" ]
}

@test "is_line_enabled returns 1 when disabled" {
  _TFUI_UI_LINES="10"
  run _tfui_ui_is_line_enabled $_TFUI_LINE_STATUS
  [ "$status" -eq 1 ]
}
