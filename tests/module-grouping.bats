#!/usr/bin/env bats

setup() {
  load 'helpers/common-setup'
  _common_setup
}

# -- Module path extraction (via group_by_module JSON output) --

@test "group by module: root resources grouped under root" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  cat > "$plan_file" <<'JSON'
{"resource_changes":[
  {"address":"aws_instance.web","change":{"actions":["create"]}},
  {"address":"aws_iam_role.old","change":{"actions":["delete"]}}
]}
JSON
  run _tfui_group_by_module "$plan_file"
  [ "$status" -eq 0 ]
  echo "$output" | jq -e '.by_module.root.summary.add == 1'
  echo "$output" | jq -e '.by_module.root.summary.destroy == 1'
  echo "$output" | jq -e '.by_module.root.summary.change == 0'
}

@test "group by module: single module prefix extracted" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  cat > "$plan_file" <<'JSON'
{"resource_changes":[
  {"address":"module.vpc.aws_subnet.private","change":{"actions":["update"]}},
  {"address":"module.vpc.aws_route_table.private","change":{"actions":["update"]}},
  {"address":"module.vpc.aws_nat_gateway.main","change":{"actions":["create"]}}
]}
JSON
  run _tfui_group_by_module "$plan_file"
  [ "$status" -eq 0 ]
  echo "$output" | jq -e '.by_module["module.vpc"].summary.add == 1'
  echo "$output" | jq -e '.by_module["module.vpc"].summary.change == 2'
  echo "$output" | jq -e '.by_module["module.vpc"].summary.destroy == 0'
  echo "$output" | jq -e '.by_module["module.vpc"].changes | length == 3'
}

@test "group by module: nested modules fully preserved" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  cat > "$plan_file" <<'JSON'
{"resource_changes":[
  {"address":"module.vpc.module.subnets.aws_subnet.private[0]","change":{"actions":["update"]}},
  {"address":"module.vpc.module.subnets.aws_subnet.private[1]","change":{"actions":["update"]}},
  {"address":"module.vpc.aws_vpc.main","change":{"actions":["update"]}}
]}
JSON
  run _tfui_group_by_module "$plan_file"
  [ "$status" -eq 0 ]
  echo "$output" | jq -e '.by_module["module.vpc.module.subnets"].summary.change == 2'
  echo "$output" | jq -e '.by_module["module.vpc"].summary.change == 1'
}

@test "group by module: mixed root and module resources" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  cat > "$plan_file" <<'JSON'
{"resource_changes":[
  {"address":"aws_instance.web","change":{"actions":["create"]}},
  {"address":"module.vpc.aws_subnet.a","change":{"actions":["update"]}},
  {"address":"module.vpc.aws_subnet.b","change":{"actions":["update"]}},
  {"address":"aws_iam_role.old","change":{"actions":["delete"]}}
]}
JSON
  run _tfui_group_by_module "$plan_file"
  [ "$status" -eq 0 ]
  echo "$output" | jq -e '.by_module | keys | sort == ["module.vpc", "root"]'
  echo "$output" | jq -e '.by_module.root.summary.add == 1'
  echo "$output" | jq -e '.by_module.root.summary.destroy == 1'
  echo "$output" | jq -e '.by_module["module.vpc"].summary.change == 2'
}

@test "group by module: no-op and read actions excluded" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  cat > "$plan_file" <<'JSON'
{"resource_changes":[
  {"address":"module.vpc.aws_vpc.main","change":{"actions":["no-op"]}},
  {"address":"data.aws_ami.latest","change":{"actions":["read"]}},
  {"address":"module.vpc.aws_subnet.a","change":{"actions":["create"]}}
]}
JSON
  run _tfui_group_by_module "$plan_file"
  [ "$status" -eq 0 ]
  echo "$output" | jq -e '.by_module | keys == ["module.vpc"]'
  echo "$output" | jq -e '.by_module["module.vpc"].summary.add == 1'
}

@test "group by module: empty plan returns empty object" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  echo '{"resource_changes":[]}' > "$plan_file"
  run _tfui_group_by_module "$plan_file"
  [ "$status" -eq 0 ]
  echo "$output" | jq -e '.by_module == {}'
}

@test "group by module: indexed resources grouped with parent module" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  cat > "$plan_file" <<'JSON'
{"resource_changes":[
  {"address":"module.ecs.aws_ecs_service.svc[0]","change":{"actions":["update"]}},
  {"address":"module.ecs.aws_ecs_service.svc[1]","change":{"actions":["update"]}},
  {"address":"module.ecs.aws_ecs_task_definition.task","change":{"actions":["create"]}}
]}
JSON
  run _tfui_group_by_module "$plan_file"
  [ "$status" -eq 0 ]
  echo "$output" | jq -e '.by_module["module.ecs"].changes | length == 3'
  echo "$output" | jq -e '.by_module["module.ecs"].summary.change == 2'
  echo "$output" | jq -e '.by_module["module.ecs"].summary.add == 1'
}

# -- Grouped human-readable output --

@test "grouped tree: renders module headers with counts" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  cat > "$plan_file" <<'JSON'
{"resource_changes":[
  {"address":"module.vpc.aws_subnet.a","change":{"actions":["update"]}},
  {"address":"module.vpc.aws_subnet.b","change":{"actions":["update"]}},
  {"address":"aws_instance.web","change":{"actions":["create"]}}
]}
JSON
  run _tfui_render_grouped_plan_tree "$plan_file"
  [ "$status" -eq 0 ]
  assert_line --index 0 "module.vpc (2 to change)"
  assert_line --index 1 "  ~ module.vpc.aws_subnet.a"
  assert_line --index 2 "  ~ module.vpc.aws_subnet.b"
  assert_line --index 3 "root (1 to add)"
  assert_line --index 4 "  + aws_instance.web"
}

@test "grouped tree: multiple action types in summary" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  cat > "$plan_file" <<'JSON'
{"resource_changes":[
  {"address":"aws_instance.new","change":{"actions":["create"]}},
  {"address":"aws_instance.old","change":{"actions":["delete"]}},
  {"address":"aws_instance.existing","change":{"actions":["update"]}}
]}
JSON
  run _tfui_render_grouped_plan_tree "$plan_file"
  [ "$status" -eq 0 ]
  assert_line --index 0 "root (1 to add, 1 to change, 1 to destroy)"
}

@test "grouped tree: no changes shows up-to-date message" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  echo '{"resource_changes":[{"address":"a","change":{"actions":["no-op"]}}]}' > "$plan_file"
  run _tfui_render_grouped_plan_tree "$plan_file"
  [ "$status" -eq 0 ]
  assert_output "No changes. Infrastructure is up-to-date."
}

@test "grouped tree: replace action renders -/+" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  echo '{"resource_changes":[{"address":"module.x.aws_instance.y","change":{"actions":["delete","create"]}}]}' > "$plan_file"
  run _tfui_render_grouped_plan_tree "$plan_file"
  [ "$status" -eq 0 ]
  assert_line --index 1 "  -/+ module.x.aws_instance.y"
}
