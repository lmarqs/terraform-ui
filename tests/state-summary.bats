#!/usr/bin/env bats

setup() {
  load 'helpers/common-setup'
  load 'helpers/fixtures'
  _common_setup
  CLI="$PROJECT_ROOT/bin/tfui"
}

# -- _tfui_render_state_json (unit tests with raw state JSON) -----------------

@test "render state json: empty state returns zero counts" {
  local state='{"resources":[]}'
  run _tfui_render_state_json "$state"
  [ "$status" -eq 0 ]
  [[ "$output" == *'"total_resources": 0'* ]]
  [[ "$output" == *'"by_type": {}'* ]]
  [[ "$output" == *'"by_module": {}'* ]]
}

@test "render state json: single root resource" {
  local state='{"resources":[{"mode":"managed","type":"terraform_data","name":"alpha","provider":"provider[\"terraform.io/builtin/terraform\"]","instances":[]}]}'
  run _tfui_render_state_json "$state"
  [ "$status" -eq 0 ]
  [[ "$output" == *'"total_resources": 1'* ]]
  [[ "$output" == *'"terraform_data": 1'* ]]
  [[ "$output" == *'"root": 1'* ]]
  [[ "$output" == *'"address": "terraform_data.alpha"'* ]]
}

@test "render state json: data sources are excluded" {
  local state='{"resources":[{"mode":"data","type":"aws_ami","name":"latest","provider":"provider[\"registry.terraform.io/hashicorp/aws\"]","instances":[]},{"mode":"managed","type":"terraform_data","name":"alpha","provider":"provider[\"terraform.io/builtin/terraform\"]","instances":[]}]}'
  run _tfui_render_state_json "$state"
  [ "$status" -eq 0 ]
  [[ "$output" == *'"total_resources": 1'* ]]
}

@test "render state json: module resources grouped correctly" {
  local state='{"resources":[{"mode":"managed","type":"terraform_data","name":"alpha","provider":"provider[\"terraform.io/builtin/terraform\"]","instances":[]},{"module":"module.child","mode":"managed","type":"terraform_data","name":"one","provider":"provider[\"terraform.io/builtin/terraform\"]","instances":[]}]}'
  run _tfui_render_state_json "$state"
  [ "$status" -eq 0 ]
  [[ "$output" == *'"total_resources": 2'* ]]
  [[ "$output" == *'"root": 1'* ]]
  [[ "$output" == *'"module.child": 1'* ]]
  [[ "$output" == *'"address": "module.child.terraform_data.one"'* ]]
}

# -- _tfui_render_state_text (unit tests with raw state JSON) -----------------

@test "render state text: empty state shows empty message" {
  local state='{"resources":[]}'
  run _tfui_render_state_text "$state"
  [ "$status" -eq 0 ]
  [[ "$output" == *"empty"* ]]
}

@test "render state text: shows resource count" {
  local state='{"resources":[{"mode":"managed","type":"terraform_data","name":"a","provider":"provider[\"terraform.io/builtin/terraform\"]","instances":[]},{"mode":"managed","type":"terraform_data","name":"b","provider":"provider[\"terraform.io/builtin/terraform\"]","instances":[]}]}'
  run _tfui_render_state_text "$state"
  [ "$status" -eq 0 ]
  [[ "$output" == *"State: 2 resources"* ]]
  [[ "$output" == *"terraform_data 2"* ]]
  [[ "$output" == *"root (2)"* ]]
}

@test "render state text: shows module grouping" {
  local state='{"resources":[{"mode":"managed","type":"terraform_data","name":"a","provider":"provider[\"terraform.io/builtin/terraform\"]","instances":[]},{"module":"module.net","mode":"managed","type":"terraform_data","name":"b","provider":"provider[\"terraform.io/builtin/terraform\"]","instances":[]}]}'
  run _tfui_render_state_text "$state"
  [ "$status" -eq 0 ]
  [[ "$output" == *"module.net (1)"* ]]
  [[ "$output" == *"root (1)"* ]]
}

# -- tfui_state (integration with fixture) ------------------------------------

@test "given state-multi fixture, tfui_state text shows 5 resources" {
  _fixture_prepare "state-multi"
  run tfui_state "text"
  [ "$status" -eq 0 ]
  [[ "$output" == *"State: 5 resources"* ]]
  [[ "$output" == *"terraform_data 5"* ]]
}

@test "given state-multi fixture, tfui_state text shows module grouping" {
  _fixture_prepare "state-multi"
  run tfui_state "text"
  [ "$status" -eq 0 ]
  [[ "$output" == *"module.child (2)"* ]]
  [[ "$output" == *"root (3)"* ]]
}

@test "given state-multi fixture, tfui_state agent returns valid JSON" {
  _fixture_prepare "state-multi"
  run tfui_state "agent"
  [ "$status" -eq 0 ]
  echo "$output" | jq . >/dev/null 2>&1
}

@test "given state-multi fixture, tfui_state agent has correct totals" {
  _fixture_prepare "state-multi"
  run tfui_state "agent"
  [ "$status" -eq 0 ]
  [ "$(echo "$output" | jq '.total_resources')" = "5" ]
  [ "$(echo "$output" | jq '.by_type.terraform_data')" = "5" ]
  [ "$(echo "$output" | jq '.by_module.root')" = "3" ]
  [ "$(echo "$output" | jq '.by_module["module.child"]')" = "2" ]
}

@test "given state-multi fixture, tfui_state agent lists all resources" {
  _fixture_prepare "state-multi"
  run tfui_state "agent"
  [ "$status" -eq 0 ]
  [ "$(echo "$output" | jq '.resources | length')" = "5" ]
  [[ "$output" == *"terraform_data.alpha"* ]]
  [[ "$output" == *"module.child.terraform_data.one"* ]]
}

@test "given no-changes fixture, tfui_state shows 1 resource" {
  _fixture_prepare "no-changes"
  run tfui_state "text"
  [ "$status" -eq 0 ]
  [[ "$output" == *"State: 1 resources"* ]]
  [[ "$output" == *"local_file 1"* ]]
}

# -- CLI integration ----------------------------------------------------------

@test "state with unknown option exits 1" {
  run "$CLI" state --nope
  [ "$status" -eq 1 ]
  [[ "$output" == *"unknown option: --nope"* ]]
}

@test "state with nonexistent dir exits 1" {
  run "$CLI" state --dir /nonexistent
  [ "$status" -eq 1 ]
  [[ "$output" == *"directory not found"* ]]
}

@test "given state-multi fixture, CLI state produces text summary" {
  _fixture_prepare "state-multi"
  run "$CLI" state --dir "$FIXTURE_DIR"
  [ "$status" -eq 0 ]
  [[ "$output" == *"State: 5 resources"* ]]
}

@test "given state-multi fixture, CLI state --mode agent produces JSON" {
  _fixture_prepare "state-multi"
  run "$CLI" state --dir "$FIXTURE_DIR" --mode agent
  [ "$status" -eq 0 ]
  echo "$output" | jq . >/dev/null 2>&1
  [ "$(echo "$output" | jq '.total_resources')" = "5" ]
}
