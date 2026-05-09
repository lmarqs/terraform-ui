#!/usr/bin/env bats

setup() {
  load 'helpers/common-setup'
  load 'helpers/fixtures'
  _common_setup
  CLI="$PROJECT_ROOT/bin/tfui"
}

# -- Strategy selection --

@test "agent mode selects agent strategy" {
  _tfui_choose_strategy "agent"
  [ "$_TFUI_STRATEGY" = "_tfui_strategy_agent" ]
}

@test "agent strategy resolves unchanged with empty patterns" {
  _TFUI_STRATEGY="_tfui_strategy_agent"
  _tfui_resolve_strategy ""
  [ "$_TFUI_RESOLVED_STRATEGY" = "_tfui_strategy_agent" ]
}

@test "agent strategy resolves unchanged with patterns" {
  _TFUI_STRATEGY="_tfui_strategy_agent"
  _tfui_resolve_strategy "some pattern"
  [ "$_TFUI_RESOLVED_STRATEGY" = "_tfui_strategy_agent" ]
}

# -- JSON output: no changes --

@test "agent mode: no changes outputs has_changes false" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  echo '{"resource_changes":[{"address":"a","change":{"actions":["no-op"]}}]}' > "$plan_file"
  run _tfui_render_plan_json "$plan_file"
  [ "$status" -eq 0 ]
  result=$(echo "$output" | jq -r '.has_changes')
  [ "$result" = "false" ]
}

@test "agent mode: no changes outputs zero summary" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  echo '{"resource_changes":[{"address":"a","change":{"actions":["no-op"]}}]}' > "$plan_file"
  run _tfui_render_plan_json "$plan_file"
  result=$(echo "$output" | jq '.summary')
  [ "$(echo "$result" | jq '.add')" = "0" ]
  [ "$(echo "$result" | jq '.change')" = "0" ]
  [ "$(echo "$result" | jq '.destroy')" = "0" ]
  [ "$(echo "$result" | jq '.replace')" = "0" ]
}

@test "agent mode: no changes outputs empty changes array" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  echo '{"resource_changes":[{"address":"a","change":{"actions":["no-op"]}}]}' > "$plan_file"
  run _tfui_render_plan_json "$plan_file"
  result=$(echo "$output" | jq '.changes | length')
  [ "$result" = "0" ]
}

@test "agent mode: no changes outputs low risk" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  echo '{"resource_changes":[{"address":"a","change":{"actions":["no-op"]}}]}' > "$plan_file"
  run _tfui_render_plan_json "$plan_file"
  [ "$(echo "$output" | jq -r '.risk_level')" = "low" ]
  [ "$(echo "$output" | jq -r '.destructive')" = "false" ]
}

# -- JSON output: create actions --

@test "agent mode: create actions produce correct summary" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  cat > "$plan_file" <<'JSON'
{"resource_changes":[
  {"address":"local_file.alpha","type":"local_file","change":{"actions":["create"]}},
  {"address":"local_file.beta","type":"local_file","change":{"actions":["create"]}}
]}
JSON
  run _tfui_render_plan_json "$plan_file"
  [ "$status" -eq 0 ]
  [ "$(echo "$output" | jq '.has_changes')" = "true" ]
  [ "$(echo "$output" | jq '.summary.add')" = "2" ]
  [ "$(echo "$output" | jq '.summary.change')" = "0" ]
  [ "$(echo "$output" | jq '.summary.destroy')" = "0" ]
  [ "$(echo "$output" | jq '.summary.replace')" = "0" ]
}

@test "agent mode: create of simple resource is low risk" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  cat > "$plan_file" <<'JSON'
{"resource_changes":[
  {"address":"local_file.alpha","type":"local_file","change":{"actions":["create"]}}
]}
JSON
  run _tfui_render_plan_json "$plan_file"
  [ "$(echo "$output" | jq -r '.changes[0].action')" = "create" ]
  [ "$(echo "$output" | jq -r '.changes[0].risk')" = "low" ]
  [ "$(echo "$output" | jq -r '.risk_level')" = "low" ]
  [ "$(echo "$output" | jq -r '.destructive')" = "false" ]
}

# -- JSON output: update actions --

@test "agent mode: update actions produce correct summary" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  cat > "$plan_file" <<'JSON'
{"resource_changes":[
  {"address":"aws_security_group.allow_tls","type":"aws_security_group","change":{"actions":["update"]}}
]}
JSON
  run _tfui_render_plan_json "$plan_file"
  [ "$(echo "$output" | jq '.summary.change')" = "1" ]
  [ "$(echo "$output" | jq -r '.changes[0].action')" = "update" ]
}

@test "agent mode: update of security group is medium risk" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  cat > "$plan_file" <<'JSON'
{"resource_changes":[
  {"address":"aws_security_group.allow_tls","type":"aws_security_group","change":{"actions":["update"]}}
]}
JSON
  run _tfui_render_plan_json "$plan_file"
  [ "$(echo "$output" | jq -r '.changes[0].risk')" = "medium" ]
  [ "$(echo "$output" | jq -r '.risk_level')" = "medium" ]
}

# -- JSON output: delete actions --

@test "agent mode: delete actions produce correct summary and destructive flag" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  cat > "$plan_file" <<'JSON'
{"resource_changes":[
  {"address":"aws_iam_role.old_role","type":"aws_iam_role","change":{"actions":["delete"]}}
]}
JSON
  run _tfui_render_plan_json "$plan_file"
  [ "$(echo "$output" | jq '.summary.destroy')" = "1" ]
  [ "$(echo "$output" | jq -r '.destructive')" = "true" ]
}

@test "agent mode: delete of IAM role is critical risk" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  cat > "$plan_file" <<'JSON'
{"resource_changes":[
  {"address":"aws_iam_role.old_role","type":"aws_iam_role","change":{"actions":["delete"]}}
]}
JSON
  run _tfui_render_plan_json "$plan_file"
  [ "$(echo "$output" | jq -r '.changes[0].risk')" = "critical" ]
  [ "$(echo "$output" | jq -r '.risk_level')" = "critical" ]
}

# -- JSON output: replace actions --

@test "agent mode: replace actions produce correct summary" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  cat > "$plan_file" <<'JSON'
{"resource_changes":[
  {"address":"aws_instance.web","type":"aws_instance","change":{"actions":["delete","create"]}}
]}
JSON
  run _tfui_render_plan_json "$plan_file"
  [ "$(echo "$output" | jq '.summary.replace')" = "1" ]
  [ "$(echo "$output" | jq -r '.changes[0].action')" = "replace" ]
  [ "$(echo "$output" | jq -r '.destructive')" = "true" ]
}

# -- Risk assessment: critical resources --

@test "agent mode: delete of RDS instance is critical risk" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  cat > "$plan_file" <<'JSON'
{"resource_changes":[
  {"address":"aws_db_instance.main","type":"aws_db_instance","change":{"actions":["delete"]}}
]}
JSON
  run _tfui_render_plan_json "$plan_file"
  [ "$(echo "$output" | jq -r '.changes[0].risk')" = "critical" ]
  [ "$(echo "$output" | jq -r '.risk_level')" = "critical" ]
}

@test "agent mode: delete of S3 bucket is critical risk" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  cat > "$plan_file" <<'JSON'
{"resource_changes":[
  {"address":"aws_s3_bucket.data","type":"aws_s3_bucket","change":{"actions":["delete"]}}
]}
JSON
  run _tfui_render_plan_json "$plan_file"
  [ "$(echo "$output" | jq -r '.changes[0].risk')" = "critical" ]
}

@test "agent mode: update of RDS instance is high risk" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  cat > "$plan_file" <<'JSON'
{"resource_changes":[
  {"address":"aws_db_instance.main","type":"aws_db_instance","change":{"actions":["update"]}}
]}
JSON
  run _tfui_render_plan_json "$plan_file"
  [ "$(echo "$output" | jq -r '.changes[0].risk')" = "high" ]
}

@test "agent mode: create of RDS instance is medium risk" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  cat > "$plan_file" <<'JSON'
{"resource_changes":[
  {"address":"aws_db_instance.main","type":"aws_db_instance","change":{"actions":["create"]}}
]}
JSON
  run _tfui_render_plan_json "$plan_file"
  [ "$(echo "$output" | jq -r '.changes[0].risk')" = "medium" ]
}

# -- Risk assessment: high risk resources --

@test "agent mode: update of IAM role is high risk" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  cat > "$plan_file" <<'JSON'
{"resource_changes":[
  {"address":"aws_iam_role.app","type":"aws_iam_role","change":{"actions":["update"]}}
]}
JSON
  run _tfui_render_plan_json "$plan_file"
  [ "$(echo "$output" | jq -r '.changes[0].risk')" = "high" ]
}

@test "agent mode: create of IAM role is medium risk" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  cat > "$plan_file" <<'JSON'
{"resource_changes":[
  {"address":"aws_iam_role.app","type":"aws_iam_role","change":{"actions":["create"]}}
]}
JSON
  run _tfui_render_plan_json "$plan_file"
  [ "$(echo "$output" | jq -r '.changes[0].risk')" = "medium" ]
}

# -- Risk assessment: overall risk level --

@test "agent mode: overall risk is the highest among all changes" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  cat > "$plan_file" <<'JSON'
{"resource_changes":[
  {"address":"local_file.a","type":"local_file","change":{"actions":["create"]}},
  {"address":"aws_security_group.b","type":"aws_security_group","change":{"actions":["update"]}},
  {"address":"aws_db_instance.c","type":"aws_db_instance","change":{"actions":["delete"]}}
]}
JSON
  run _tfui_render_plan_json "$plan_file"
  [ "$(echo "$output" | jq -r '.risk_level')" = "critical" ]
}

# -- JSON output: mixed actions --

@test "agent mode: mixed actions produce correct summary and changes count" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  cat > "$plan_file" <<'JSON'
{"resource_changes":[
  {"address":"local_file.a","type":"local_file","change":{"actions":["create"]}},
  {"address":"local_file.b","type":"local_file","change":{"actions":["update"]}},
  {"address":"local_file.c","type":"local_file","change":{"actions":["delete"]}},
  {"address":"local_file.d","type":"local_file","change":{"actions":["delete","create"]}},
  {"address":"data.source","type":"data","change":{"actions":["read"]}}
]}
JSON
  run _tfui_render_plan_json "$plan_file"
  [ "$(echo "$output" | jq '.summary.add')" = "1" ]
  [ "$(echo "$output" | jq '.summary.change')" = "1" ]
  [ "$(echo "$output" | jq '.summary.destroy')" = "1" ]
  [ "$(echo "$output" | jq '.summary.replace')" = "1" ]
  [ "$(echo "$output" | jq '.changes | length')" = "4" ]
}

@test "agent mode: data source reads are excluded from changes" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  cat > "$plan_file" <<'JSON'
{"resource_changes":[
  {"address":"data.aws_ami.latest","type":"aws_ami","change":{"actions":["read"]}}
]}
JSON
  run _tfui_render_plan_json "$plan_file"
  [ "$(echo "$output" | jq '.has_changes')" = "false" ]
  [ "$(echo "$output" | jq '.changes | length')" = "0" ]
}

# -- Edge cases --

@test "agent mode: null resource_changes produces no changes" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  echo '{}' > "$plan_file"
  run _tfui_render_plan_json "$plan_file"
  [ "$status" -eq 0 ]
  [ "$(echo "$output" | jq '.has_changes')" = "false" ]
}

@test "agent mode: output is valid JSON" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  cat > "$plan_file" <<'JSON'
{"resource_changes":[
  {"address":"aws_instance.web","type":"aws_instance","change":{"actions":["create"]}},
  {"address":"aws_security_group.sg","type":"aws_security_group","change":{"actions":["update"]}}
]}
JSON
  run _tfui_render_plan_json "$plan_file"
  [ "$status" -eq 0 ]
  echo "$output" | jq empty
}

# -- CLI integration --

@test "CLI: plan --mode agent produces valid JSON output" {
  _fixture_prepare "create"
  run "$CLI" plan --dir "$FIXTURE_DIR" --mode agent
  [ "$status" -eq 0 ]
  echo "$output" | jq empty
  [ "$(echo "$output" | jq '.has_changes')" = "true" ]
  [ "$(echo "$output" | jq '.summary.add')" = "2" ]
}

@test "CLI: plan --mode agent with no changes" {
  _fixture_prepare "no-changes"
  run "$CLI" plan --dir "$FIXTURE_DIR" --mode agent
  [ "$status" -eq 0 ]
  [ "$(echo "$output" | jq '.has_changes')" = "false" ]
}

@test "CLI: invalid mode exits with error" {
  run "$CLI" plan --mode bogus --dir .
  [ "$status" -eq 1 ]
  [[ "$output" == *"unknown mode: bogus"* ]]
}

@test "CLI: plan --mode agent with delete fixture shows destructive" {
  _fixture_prepare "delete"
  run "$CLI" plan --dir "$FIXTURE_DIR" --mode agent
  [ "$status" -eq 0 ]
  [ "$(echo "$output" | jq '.destructive')" = "true" ]
  [ "$(echo "$output" | jq '.summary.destroy')" = "1" ]
}

@test "CLI: plan --mode agent with replace fixture" {
  _fixture_prepare "replace"
  run "$CLI" plan --dir "$FIXTURE_DIR" --mode agent
  [ "$status" -eq 0 ]
  [ "$(echo "$output" | jq '.destructive')" = "true" ]
  [ "$(echo "$output" | jq '.summary.replace')" = "1" ]
  [ "$(echo "$output" | jq -r '.changes[0].action')" = "replace" ]
}
