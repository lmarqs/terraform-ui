#!/usr/bin/env bats

setup() {
  load 'helpers/common-setup'
  _common_setup
}

# -- Phantom detection --

@test "phantom filter: identical before/after detected as phantom" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  cat > "$plan_file" <<'JSON'
{"resource_changes":[
  {"address":"aws_security_group.default","change":{"actions":["update"],"before":{"name":"sg","ingress":[{"port":80},{"port":443}]},"after":{"name":"sg","ingress":[{"port":80},{"port":443}]}}}
]}
JSON
  run _tfui_filter_phantom_changes "$plan_file"
  [ "$status" -eq 0 ]
  echo "$output" | jq -e '.phantom_changes == 1'
  echo "$output" | jq -e '.real_changes == 0'
  echo "$output" | jq -e '.phantom_resources == ["aws_security_group.default"]'
}

@test "phantom filter: different before/after detected as real" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  cat > "$plan_file" <<'JSON'
{"resource_changes":[
  {"address":"aws_instance.web","change":{"actions":["update"],"before":{"ami":"ami-old"},"after":{"ami":"ami-new"}}}
]}
JSON
  run _tfui_filter_phantom_changes "$plan_file"
  [ "$status" -eq 0 ]
  echo "$output" | jq -e '.phantom_changes == 0'
  echo "$output" | jq -e '.real_changes == 1'
  echo "$output" | jq -e '.phantom_resources == []'
}

@test "phantom filter: array ordering difference treated as phantom" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  cat > "$plan_file" <<'JSON'
{"resource_changes":[
  {"address":"aws_security_group.web","change":{"actions":["update"],"before":{"tags":[{"key":"Name","value":"web"},{"key":"Env","value":"prod"}]},"after":{"tags":[{"key":"Env","value":"prod"},{"key":"Name","value":"web"}]}}}
]}
JSON
  run _tfui_filter_phantom_changes "$plan_file"
  [ "$status" -eq 0 ]
  echo "$output" | jq -e '.phantom_changes == 1'
  echo "$output" | jq -e '.real_changes == 0'
}

@test "phantom filter: null fields ignored in comparison" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  cat > "$plan_file" <<'JSON'
{"resource_changes":[
  {"address":"aws_route_table.main","change":{"actions":["update"],"before":{"tags":{"Name":"rt"},"propagating_vgws":null},"after":{"tags":{"Name":"rt"}}}}
]}
JSON
  run _tfui_filter_phantom_changes "$plan_file"
  [ "$status" -eq 0 ]
  echo "$output" | jq -e '.phantom_changes == 1'
}

@test "phantom filter: mixed phantom and real changes" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  cat > "$plan_file" <<'JSON'
{"resource_changes":[
  {"address":"aws_security_group.default","change":{"actions":["update"],"before":{"name":"sg"},"after":{"name":"sg"}}},
  {"address":"aws_instance.web","change":{"actions":["update"],"before":{"ami":"ami-old"},"after":{"ami":"ami-new"}}},
  {"address":"aws_route_table.main","change":{"actions":["update"],"before":{"routes":[{"cidr":"10.0.0.0/8"}]},"after":{"routes":[{"cidr":"10.0.0.0/8"}]}}}
]}
JSON
  run _tfui_filter_phantom_changes "$plan_file"
  [ "$status" -eq 0 ]
  echo "$output" | jq -e '.phantom_changes == 2'
  echo "$output" | jq -e '.real_changes == 1'
  echo "$output" | jq -e '.phantom_resources | sort == ["aws_route_table.main", "aws_security_group.default"]'
}

@test "phantom filter: non-update actions are excluded" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  cat > "$plan_file" <<'JSON'
{"resource_changes":[
  {"address":"aws_instance.new","change":{"actions":["create"],"before":null,"after":{"ami":"ami-123"}}},
  {"address":"aws_instance.old","change":{"actions":["delete"],"before":{"ami":"ami-456"},"after":null}},
  {"address":"aws_security_group.default","change":{"actions":["update"],"before":{"name":"sg"},"after":{"name":"sg"}}}
]}
JSON
  run _tfui_filter_phantom_changes "$plan_file"
  [ "$status" -eq 0 ]
  echo "$output" | jq -e '.phantom_changes == 1'
  echo "$output" | jq -e '.real_changes == 0'
}

@test "phantom filter: no resource_changes returns zeros" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  echo '{}' > "$plan_file"
  run _tfui_filter_phantom_changes "$plan_file"
  [ "$status" -eq 0 ]
  echo "$output" | jq -e '.phantom_changes == 0'
  echo "$output" | jq -e '.real_changes == 0'
  echo "$output" | jq -e '.phantom_resources == []'
}

@test "phantom filter: nested object difference detected as real" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  cat > "$plan_file" <<'JSON'
{"resource_changes":[
  {"address":"aws_lambda.fn","change":{"actions":["update"],"before":{"environment":{"variables":{"KEY":"old"}}},"after":{"environment":{"variables":{"KEY":"new"}}}}}
]}
JSON
  run _tfui_filter_phantom_changes "$plan_file"
  [ "$status" -eq 0 ]
  echo "$output" | jq -e '.phantom_changes == 0'
  echo "$output" | jq -e '.real_changes == 1'
}
