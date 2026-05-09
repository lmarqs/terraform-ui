#!/usr/bin/env bats

setup() {
  load 'helpers/common-setup'
  _common_setup
}

@test "plan tree: no changes shows up-to-date message" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  echo '{"resource_changes":[{"address":"a","change":{"actions":["no-op"]}}]}' > "$plan_file"
  run _tfui_render_plan_tree "$plan_file"
  assert_output "No changes. Infrastructure is up-to-date."
}

@test "plan tree: mixed actions render correct symbols" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  cat > "$plan_file" <<'JSON'
{"resource_changes":[
  {"address":"module.a.resource_b","change":{"actions":["create"]}},
  {"address":"module.a.resource_a","change":{"actions":["update"]}},
  {"address":"module.b.resource_c","change":{"actions":["delete"]}},
  {"address":"data.source","change":{"actions":["read"]}}
]}
JSON
  run _tfui_render_plan_tree "$plan_file"
  expected="+ module.a.resource_b
- module.b.resource_c
~ module.a.resource_a

Plan: 1 to add, 1 to change, 1 to destroy."
  assert_output "$expected"
}

@test "plan tree: replace action shows -/+" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  echo '{"resource_changes":[{"address":"module.x.y","change":{"actions":["delete","create"]}}]}' > "$plan_file"
  run _tfui_render_plan_tree "$plan_file"
  assert_line --index 0 "-/+ module.x.y"
}

# -- Change detection --

@test "confirm: plan with only no-op returns 1" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  echo '{"resource_changes":[{"address":"a","change":{"actions":["no-op"]}}]}' > "$plan_file"
  run tfui_confirm "$plan_file" --auto-approve
  [ "$status" -eq 1 ]
}

@test "confirm: plan with create returns 0" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  echo '{"resource_changes":[{"address":"a","change":{"actions":["create"]}}]}' > "$plan_file"
  tfui_confirm "$plan_file" --auto-approve
}

@test "confirm: plan with only read returns 1" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  echo '{"resource_changes":[{"address":"a","change":{"actions":["read"]}}]}' > "$plan_file"
  run tfui_confirm "$plan_file" --auto-approve
  [ "$status" -eq 1 ]
}
