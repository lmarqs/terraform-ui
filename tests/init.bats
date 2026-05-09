#!/usr/bin/env bats

setup() {
  load 'helpers/common-setup'
  _common_setup
}

@test "tfui_init sets working directory" {
  local tmpdir="$BATS_TEST_TMPDIR/wd"
  mkdir -p "$tmpdir"
  run bash -c '
    source "'"$BATS_TEST_DIRNAME"'/../lib/tfui.sh"
    exec 3>/dev/null
    tfui_init "'"$tmpdir"'" "plain"
    echo "$_TFUI_WORKING_DIR"
  '
  assert_output "$tmpdir"
}

@test "tfui_init creates output file" {
  local tmpdir="$BATS_TEST_TMPDIR/wd"
  mkdir -p "$tmpdir"
  run bash -c '
    source "'"$BATS_TEST_DIRNAME"'/../lib/tfui.sh"
    exec 3>/dev/null
    tfui_init "'"$tmpdir"'" "plain"
    [ -f "$_TFUI_OUTPUT_FILE" ] && echo "exists"
  '
  assert_output "exists"
}

@test "tfui_init selects strategy from mode" {
  local tmpdir="$BATS_TEST_TMPDIR/wd"
  mkdir -p "$tmpdir"
  run bash -c '
    source "'"$BATS_TEST_DIRNAME"'/../lib/tfui.sh"
    exec 3>/dev/null
    tfui_init "'"$tmpdir"'" "simple"
    echo "$_TFUI_STRATEGY"
  '
  assert_output "_tfui_strategy_spinner"
}
