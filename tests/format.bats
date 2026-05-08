#!/usr/bin/env bats

setup() {
  load 'test_helper/common-setup'
  _common_setup
}

# -- Format header --

@test "format_header: first frame with zero elapsed" {
  run _tfui_ui_format_header 0 "Planning" 0
  assert_output "⠋ Planning (0s)"
}

@test "format_header: third frame with elapsed time" {
  run _tfui_ui_format_header 2 "Applying" 42
  assert_output "⠹ Applying (42s)"
}

@test "format_header: message with special characters" {
  run _tfui_ui_format_header 0 "Planning module: sa-east-1" 7
  assert_output "⠋ Planning module: sa-east-1 (7s)"
}

@test "format_header: large elapsed time" {
  run _tfui_ui_format_header 5 "Waiting" 999
  assert_output "⠴ Waiting (999s)"
}

# -- Format progress --

@test "format_progress: zero progress" {
  run _tfui_ui_format_progress 0 100
  assert_output "  Progress: 0/100 [░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░] 0%"
}

@test "format_progress: full progress" {
  run _tfui_ui_format_progress 50 50
  assert_output "  Progress: 50/50 [██████████████████████████████] 100%"
}

@test "format_progress: partial progress" {
  run _tfui_ui_format_progress 25 100
  assert_output "  Progress: 25/100 [███████░░░░░░░░░░░░░░░░░░░░░░░] 25%"
}

@test "format_progress: zero total" {
  run _tfui_ui_format_progress 0 0
  assert_output "  Progress: 0/0 [░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░] 0%"
}

# -- Format status --

@test "format_status: label with dots" {
  run _tfui_ui_format_status "Calculating" "..."
  assert_output "  Calculating..."
}

@test "format_status: empty dots" {
  run _tfui_ui_format_status "Rendering" ""
  assert_output "  Rendering"
}

@test "format_status: label with max dots" {
  run _tfui_ui_format_status "Working" "....."
  assert_output "  Working....."
}

# -- Build bar --

@test "build_bar: empty progress bar" {
  run _tfui_ui_build_bar 0 100
  assert_output "░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░"
}

@test "build_bar: full progress bar" {
  run _tfui_ui_build_bar 100 100
  assert_output "██████████████████████████████"
}

@test "build_bar: half progress bar" {
  run _tfui_ui_build_bar 15 30
  assert_output "███████████████░░░░░░░░░░░░░░░"
}

@test "build_bar: zero total" {
  run _tfui_ui_build_bar 0 0
  assert_output "░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░"
}

# -- Percentage calculation --

@test "calc_percent: 0/100 is 0" {
  run _tfui_ui_calc_percent 0 100
  assert_output "0"
}

@test "calc_percent: 50/100 is 50" {
  run _tfui_ui_calc_percent 50 100
  assert_output "50"
}

@test "calc_percent: 100/100 is 100" {
  run _tfui_ui_calc_percent 100 100
  assert_output "100"
}

@test "calc_percent: 0/0 is 0" {
  run _tfui_ui_calc_percent 0 0
  assert_output "0"
}

@test "calc_percent: 33/100 is 33" {
  run _tfui_ui_calc_percent 33 100
  assert_output "33"
}
