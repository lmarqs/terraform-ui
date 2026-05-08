#!/usr/bin/env bats

setup() {
  load 'test_helper/common-setup'
  _common_setup
}

# -- Strategy selection --

@test "plain mode selects silent strategy" {
  _tfui_choose_strategy "plain"
  [ "$_TFUI_STRATEGY" = "_tfui_strategy_silent" ]
}

@test "simple mode selects spinner strategy" {
  _tfui_choose_strategy "simple"
  [ "$_TFUI_STRATEGY" = "_tfui_strategy_spinner" ]
}

@test "rich mode selects progress strategy" {
  _tfui_choose_strategy "rich"
  [ "$_TFUI_STRATEGY" = "_tfui_strategy_progress" ]
}

@test "auto mode selects progress when tty is present" {
  _tfui_choose_strategy "auto"
  [ "$_TFUI_STRATEGY" = "_tfui_strategy_progress" ]
}

# -- Strategy resolution --

@test "patterns provided keeps progress strategy" {
  _TFUI_STRATEGY="_tfui_strategy_progress"
  _tfui_resolve_strategy "some pattern"
  [ "$_TFUI_RESOLVED_STRATEGY" = "_tfui_strategy_progress" ]
}

@test "empty patterns downgrades progress to spinner" {
  _TFUI_STRATEGY="_tfui_strategy_progress"
  _tfui_resolve_strategy ""
  [ "$_TFUI_RESOLVED_STRATEGY" = "_tfui_strategy_spinner" ]
}

@test "empty patterns keeps spinner as spinner" {
  _TFUI_STRATEGY="_tfui_strategy_spinner"
  _tfui_resolve_strategy ""
  [ "$_TFUI_RESOLVED_STRATEGY" = "_tfui_strategy_spinner" ]
}

@test "empty patterns keeps silent as silent" {
  _TFUI_STRATEGY="_tfui_strategy_silent"
  _tfui_resolve_strategy ""
  [ "$_TFUI_RESOLVED_STRATEGY" = "_tfui_strategy_silent" ]
}
