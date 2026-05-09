#!/usr/bin/env bats

setup() {
  load 'helpers/common-setup'
  load 'helpers/fixtures'
  _common_setup
  CLI="$PROJECT_ROOT/bin/tfui"
}

# -- help/version -------------------------------------------------------------

@test "no arguments prints usage and exits 1" {
  run "$CLI"
  [ "$status" -eq 1 ]
  [[ "$output" == *"Usage:"* ]]
}

@test "help prints usage and exits 0" {
  run "$CLI" help
  [ "$status" -eq 0 ]
  [[ "$output" == *"Usage:"* ]]
}

@test "version prints version string" {
  run "$CLI" version
  [ "$status" -eq 0 ]
  [[ "$output" == "tfui "* ]]
}

@test "unknown command exits 1 with error" {
  run "$CLI" bogus
  [ "$status" -eq 1 ]
  [[ "$output" == *"unknown command: bogus"* ]]
}

# -- plan ---------------------------------------------------------------------

@test "plan with unknown option exits 1" {
  run "$CLI" plan --nope
  [ "$status" -eq 1 ]
  [[ "$output" == *"unknown option: --nope"* ]]
}

@test "plan with nonexistent dir exits 1" {
  run "$CLI" plan --dir /nonexistent
  [ "$status" -eq 1 ]
  [[ "$output" == *"directory not found"* ]]
}

@test "plan produces tree view output" {
  _fixture_prepare "create"
  run "$CLI" plan --dir "$FIXTURE_DIR" --mode plain
  [ "$status" -eq 0 ]
  [[ "$output" == *"+ local_file."* ]]
  [[ "$output" == *"Plan:"* ]]
}

# -- apply --------------------------------------------------------------------

@test "apply with unknown option exits 1" {
  run "$CLI" apply --nope
  [ "$status" -eq 1 ]
  [[ "$output" == *"unknown option: --nope"* ]]
}

@test "apply --auto-approve runs full lifecycle" {
  _fixture_prepare "create"
  run "$CLI" apply --dir "$FIXTURE_DIR" --mode plain --auto-approve
  [ "$status" -eq 0 ]
  [ -f "$FIXTURE_DIR/out/alpha.txt" ]
  [ -f "$FIXTURE_DIR/out/beta.txt" ]
}

@test "apply with no changes exits cleanly" {
  _fixture_prepare "no-changes"
  run "$CLI" apply --dir "$FIXTURE_DIR" --mode plain --auto-approve
  [ "$status" -eq 0 ]
}
