#!/usr/bin/env bats

setup() {
  load 'helpers/common-setup'
  _common_setup
}

# -- Policy defaults --

@test "policy defaults: returns valid JSON with rules array" {
  run _tfui_policy_defaults
  echo "$output" | jq -e '.rules | length > 0'
}

@test "policy defaults: each rule has required fields" {
  run _tfui_policy_defaults
  local count
  count=$(echo "$output" | jq '[.rules[] | select(.id and .description and .severity and .match.resource_type and .match.action)] | length')
  local total
  total=$(echo "$output" | jq '.rules | length')
  [ "$count" -eq "$total" ]
}

@test "policy defaults: severities are valid values" {
  run _tfui_policy_defaults
  local invalid
  invalid=$(echo "$output" | jq '[.rules[] | .severity | select(. != "critical" and . != "high" and . != "medium" and . != "low")] | length')
  [ "$invalid" -eq 0 ]
}

# -- Policy discovery --

@test "policy discover: finds file in current directory" {
  local dir="$BATS_TEST_TMPDIR/project"
  mkdir -p "$dir"
  echo '{"rules":[]}' > "$dir/.tfui-policy.json"

  run _tfui_policy_discover "$dir"
  assert_success
  assert_output "$dir/.tfui-policy.json"
}

@test "policy discover: finds file in parent directory" {
  local parent="$BATS_TEST_TMPDIR/parent"
  local child="$parent/child/grandchild"
  mkdir -p "$child"
  echo '{"rules":[]}' > "$parent/.tfui-policy.json"

  run _tfui_policy_discover "$child"
  assert_success
  assert_output "$parent/.tfui-policy.json"
}

@test "policy discover: returns failure when no file found" {
  local dir="$BATS_TEST_TMPDIR/empty"
  mkdir -p "$dir"

  run _tfui_policy_discover "$dir"
  assert_failure
  assert_output ""
}

@test "policy discover: closest file wins over parent" {
  local parent="$BATS_TEST_TMPDIR/parent2"
  local child="$parent/child"
  mkdir -p "$child"
  echo '{"rules":[{"id":"parent"}]}' > "$parent/.tfui-policy.json"
  echo '{"rules":[{"id":"child"}]}' > "$child/.tfui-policy.json"

  run _tfui_policy_discover "$child"
  assert_success
  assert_output "$child/.tfui-policy.json"
}

# -- Policy load --

@test "policy load: uses discovered file when present" {
  local dir="$BATS_TEST_TMPDIR/with-policy"
  mkdir -p "$dir"
  echo '{"rules":[{"id":"custom","description":"Custom rule","severity":"low","match":{"resource_type":"null_resource","action":["create"]}}]}' > "$dir/.tfui-policy.json"

  run _tfui_policy_load "$dir"
  assert_success
  local rule_id
  rule_id=$(echo "$output" | jq -r '.rules[0].id')
  [ "$rule_id" = "custom" ]
}

@test "policy load: falls back to defaults when no file found" {
  local dir="$BATS_TEST_TMPDIR/no-policy"
  mkdir -p "$dir"

  run _tfui_policy_load "$dir"
  assert_success
  local count
  count=$(echo "$output" | jq '.rules | length')
  [ "$count" -gt 0 ]
}

# -- Policy evaluate --

@test "policy evaluate: no warnings when no rules match" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  local policy_dir="$BATS_TEST_TMPDIR/eval-no-match"
  mkdir -p "$policy_dir"

  cat > "$plan_file" <<'JSON'
{"resource_changes":[
  {"address":"local_file.foo","type":"local_file","change":{"actions":["create"]}}
]}
JSON

  cat > "$policy_dir/.tfui-policy.json" <<'JSON'
{"rules":[
  {"id":"db-delete","description":"DB destroyed","severity":"critical","match":{"resource_type":"aws_db_instance","action":["delete"]}}
]}
JSON

  run _tfui_policy_evaluate "$plan_file" "$policy_dir"
  assert_success
  local warning_count
  warning_count=$(echo "$output" | jq '.warnings | length')
  [ "$warning_count" -eq 0 ]
}

@test "policy evaluate: matches exact resource type and action" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  local policy_dir="$BATS_TEST_TMPDIR/eval-exact"
  mkdir -p "$policy_dir"

  cat > "$plan_file" <<'JSON'
{"resource_changes":[
  {"address":"aws_db_instance.main","type":"aws_db_instance","change":{"actions":["delete"]}}
]}
JSON

  cat > "$policy_dir/.tfui-policy.json" <<'JSON'
{"rules":[
  {"id":"db-delete","description":"Database destroyed","severity":"critical","match":{"resource_type":"aws_db_instance","action":["delete"]}}
]}
JSON

  run _tfui_policy_evaluate "$plan_file" "$policy_dir"
  assert_success
  local warning_count
  warning_count=$(echo "$output" | jq '.warnings | length')
  [ "$warning_count" -eq 1 ]

  local rule_id
  rule_id=$(echo "$output" | jq -r '.warnings[0].rule_id')
  [ "$rule_id" = "db-delete" ]

  local severity
  severity=$(echo "$output" | jq -r '.warnings[0].severity')
  [ "$severity" = "critical" ]

  local resource
  resource=$(echo "$output" | jq -r '.warnings[0].resource')
  [ "$resource" = "aws_db_instance.main" ]
}

@test "policy evaluate: regex pattern matches resource type" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  local policy_dir="$BATS_TEST_TMPDIR/eval-regex"
  mkdir -p "$policy_dir"

  cat > "$plan_file" <<'JSON'
{"resource_changes":[
  {"address":"aws_security_group_rule.allow_ssh","type":"aws_security_group_rule","change":{"actions":["create"]}}
]}
JSON

  cat > "$policy_dir/.tfui-policy.json" <<'JSON'
{"rules":[
  {"id":"sg-create","description":"Security group resource created","severity":"medium","match":{"resource_type":"aws_security_group.*","action":["create"]}}
]}
JSON

  run _tfui_policy_evaluate "$plan_file" "$policy_dir"
  assert_success
  local warning_count
  warning_count=$(echo "$output" | jq '.warnings | length')
  [ "$warning_count" -eq 1 ]
}

@test "policy evaluate: alternation pattern matches multiple types" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  local policy_dir="$BATS_TEST_TMPDIR/eval-alt"
  mkdir -p "$policy_dir"

  cat > "$plan_file" <<'JSON'
{"resource_changes":[
  {"address":"aws_rds_cluster.main","type":"aws_rds_cluster","change":{"actions":["delete"]}},
  {"address":"aws_db_instance.replica","type":"aws_db_instance","change":{"actions":["delete"]}}
]}
JSON

  cat > "$policy_dir/.tfui-policy.json" <<'JSON'
{"rules":[
  {"id":"db-delete","description":"Database destroyed","severity":"critical","match":{"resource_type":"aws_db_instance|aws_rds_cluster","action":["delete"]}}
]}
JSON

  run _tfui_policy_evaluate "$plan_file" "$policy_dir"
  assert_success
  local warning_count
  warning_count=$(echo "$output" | jq '.warnings | length')
  [ "$warning_count" -eq 2 ]
}

@test "policy evaluate: replace action matches delete+create" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  local policy_dir="$BATS_TEST_TMPDIR/eval-replace"
  mkdir -p "$policy_dir"

  cat > "$plan_file" <<'JSON'
{"resource_changes":[
  {"address":"aws_db_instance.main","type":"aws_db_instance","change":{"actions":["delete","create"]}}
]}
JSON

  cat > "$policy_dir/.tfui-policy.json" <<'JSON'
{"rules":[
  {"id":"db-replace","description":"Database replaced","severity":"critical","match":{"resource_type":"aws_db_instance","action":["replace"]}}
]}
JSON

  run _tfui_policy_evaluate "$plan_file" "$policy_dir"
  assert_success
  local warning_count
  warning_count=$(echo "$output" | jq '.warnings | length')
  [ "$warning_count" -eq 1 ]
  local rule_id
  rule_id=$(echo "$output" | jq -r '.warnings[0].rule_id')
  [ "$rule_id" = "db-replace" ]
}

@test "policy evaluate: no-op and read actions are ignored" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  local policy_dir="$BATS_TEST_TMPDIR/eval-noop"
  mkdir -p "$policy_dir"

  cat > "$plan_file" <<'JSON'
{"resource_changes":[
  {"address":"aws_db_instance.existing","type":"aws_db_instance","change":{"actions":["no-op"]}},
  {"address":"aws_db_instance.data","type":"aws_db_instance","change":{"actions":["read"]}}
]}
JSON

  cat > "$policy_dir/.tfui-policy.json" <<'JSON'
{"rules":[
  {"id":"db-any","description":"Database touched","severity":"high","match":{"resource_type":"aws_db_instance","action":["create","update","delete","replace"]}}
]}
JSON

  run _tfui_policy_evaluate "$plan_file" "$policy_dir"
  assert_success
  local warning_count
  warning_count=$(echo "$output" | jq '.warnings | length')
  [ "$warning_count" -eq 0 ]
}

@test "policy evaluate: action mismatch produces no warning" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  local policy_dir="$BATS_TEST_TMPDIR/eval-action-mismatch"
  mkdir -p "$policy_dir"

  cat > "$plan_file" <<'JSON'
{"resource_changes":[
  {"address":"aws_db_instance.main","type":"aws_db_instance","change":{"actions":["create"]}}
]}
JSON

  cat > "$policy_dir/.tfui-policy.json" <<'JSON'
{"rules":[
  {"id":"db-delete","description":"Database destroyed","severity":"critical","match":{"resource_type":"aws_db_instance","action":["delete"]}}
]}
JSON

  run _tfui_policy_evaluate "$plan_file" "$policy_dir"
  assert_success
  local warning_count
  warning_count=$(echo "$output" | jq '.warnings | length')
  [ "$warning_count" -eq 0 ]
}

@test "policy evaluate: type mismatch produces no warning" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  local policy_dir="$BATS_TEST_TMPDIR/eval-type-mismatch"
  mkdir -p "$policy_dir"

  cat > "$plan_file" <<'JSON'
{"resource_changes":[
  {"address":"aws_s3_bucket.logs","type":"aws_s3_bucket","change":{"actions":["delete"]}}
]}
JSON

  cat > "$policy_dir/.tfui-policy.json" <<'JSON'
{"rules":[
  {"id":"db-delete","description":"Database destroyed","severity":"critical","match":{"resource_type":"aws_db_instance","action":["delete"]}}
]}
JSON

  run _tfui_policy_evaluate "$plan_file" "$policy_dir"
  assert_success
  local warning_count
  warning_count=$(echo "$output" | jq '.warnings | length')
  [ "$warning_count" -eq 0 ]
}

# -- Policy summary --

@test "policy evaluate: summary counts severities correctly" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  local policy_dir="$BATS_TEST_TMPDIR/eval-summary"
  mkdir -p "$policy_dir"

  cat > "$plan_file" <<'JSON'
{"resource_changes":[
  {"address":"aws_db_instance.main","type":"aws_db_instance","change":{"actions":["delete"]}},
  {"address":"aws_s3_bucket.logs","type":"aws_s3_bucket","change":{"actions":["delete"]}},
  {"address":"aws_iam_role.admin","type":"aws_iam_role","change":{"actions":["create"]}}
]}
JSON

  cat > "$policy_dir/.tfui-policy.json" <<'JSON'
{"rules":[
  {"id":"db-delete","description":"Database destroyed","severity":"critical","match":{"resource_type":"aws_db_instance","action":["delete"]}},
  {"id":"storage-delete","description":"Storage destroyed","severity":"high","match":{"resource_type":"aws_s3_bucket","action":["delete"]}},
  {"id":"iam-change","description":"IAM modified","severity":"medium","match":{"resource_type":"aws_iam_role","action":["create","update","delete"]}},
  {"id":"unused-rule","description":"Never triggers","severity":"low","match":{"resource_type":"aws_lambda_function","action":["delete"]}}
]}
JSON

  run _tfui_policy_evaluate "$plan_file" "$policy_dir"
  assert_success

  local critical high medium low passed
  critical=$(echo "$output" | jq '.policy_summary.critical')
  high=$(echo "$output" | jq '.policy_summary.high')
  medium=$(echo "$output" | jq '.policy_summary.medium')
  low=$(echo "$output" | jq '.policy_summary.low')
  passed=$(echo "$output" | jq '.policy_summary.passed')

  [ "$critical" -eq 1 ]
  [ "$high" -eq 1 ]
  [ "$medium" -eq 1 ]
  [ "$low" -eq 0 ]
  [ "$passed" -eq 1 ]
}

@test "policy evaluate: multiple rules can match same resource" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  local policy_dir="$BATS_TEST_TMPDIR/eval-multi-rule"
  mkdir -p "$policy_dir"

  cat > "$plan_file" <<'JSON'
{"resource_changes":[
  {"address":"aws_db_instance.main","type":"aws_db_instance","change":{"actions":["delete"]}}
]}
JSON

  cat > "$policy_dir/.tfui-policy.json" <<'JSON'
{"rules":[
  {"id":"db-delete","description":"Database destroyed","severity":"critical","match":{"resource_type":"aws_db_instance","action":["delete"]}},
  {"id":"any-delete","description":"Any resource destroyed","severity":"low","match":{"resource_type":".*","action":["delete"]}}
]}
JSON

  run _tfui_policy_evaluate "$plan_file" "$policy_dir"
  assert_success
  local warning_count
  warning_count=$(echo "$output" | jq '.warnings | length')
  [ "$warning_count" -eq 2 ]
}

@test "policy evaluate: empty plan produces no warnings" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  local policy_dir="$BATS_TEST_TMPDIR/eval-empty"
  mkdir -p "$policy_dir"

  echo '{"resource_changes":[]}' > "$plan_file"

  cat > "$policy_dir/.tfui-policy.json" <<'JSON'
{"rules":[
  {"id":"db-delete","description":"Database destroyed","severity":"critical","match":{"resource_type":"aws_db_instance","action":["delete"]}}
]}
JSON

  run _tfui_policy_evaluate "$plan_file" "$policy_dir"
  assert_success
  local warning_count
  warning_count=$(echo "$output" | jq '.warnings | length')
  [ "$warning_count" -eq 0 ]
}

@test "policy evaluate: null resource_changes produces no warnings" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  local policy_dir="$BATS_TEST_TMPDIR/eval-null"
  mkdir -p "$policy_dir"

  echo '{}' > "$plan_file"

  cat > "$policy_dir/.tfui-policy.json" <<'JSON'
{"rules":[
  {"id":"db-delete","description":"Database destroyed","severity":"critical","match":{"resource_type":"aws_db_instance","action":["delete"]}}
]}
JSON

  run _tfui_policy_evaluate "$plan_file" "$policy_dir"
  assert_success
  local warning_count
  warning_count=$(echo "$output" | jq '.warnings | length')
  [ "$warning_count" -eq 0 ]
}

# -- Default rules integration --

@test "policy evaluate: default rules catch database deletion" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  local policy_dir="$BATS_TEST_TMPDIR/eval-defaults"
  mkdir -p "$policy_dir"
  # No policy file — will use defaults

  cat > "$plan_file" <<'JSON'
{"resource_changes":[
  {"address":"aws_db_instance.production","type":"aws_db_instance","change":{"actions":["delete"]}}
]}
JSON

  run _tfui_policy_evaluate "$plan_file" "$policy_dir"
  assert_success
  local warning_count
  warning_count=$(echo "$output" | jq '.warnings | length')
  [ "$warning_count" -ge 1 ]

  local has_db_delete
  has_db_delete=$(echo "$output" | jq '[.warnings[] | select(.rule_id == "database-delete")] | length')
  [ "$has_db_delete" -eq 1 ]
}

@test "policy evaluate: default rules catch S3 bucket deletion" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  local policy_dir="$BATS_TEST_TMPDIR/eval-defaults-s3"
  mkdir -p "$policy_dir"

  cat > "$plan_file" <<'JSON'
{"resource_changes":[
  {"address":"aws_s3_bucket.backups","type":"aws_s3_bucket","change":{"actions":["delete"]}}
]}
JSON

  run _tfui_policy_evaluate "$plan_file" "$policy_dir"
  assert_success
  local has_storage
  has_storage=$(echo "$output" | jq '[.warnings[] | select(.rule_id == "storage-delete")] | length')
  [ "$has_storage" -eq 1 ]
}

@test "policy evaluate: default rules catch IAM changes" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  local policy_dir="$BATS_TEST_TMPDIR/eval-defaults-iam"
  mkdir -p "$policy_dir"

  cat > "$plan_file" <<'JSON'
{"resource_changes":[
  {"address":"aws_iam_role.admin","type":"aws_iam_role","change":{"actions":["update"]}}
]}
JSON

  run _tfui_policy_evaluate "$plan_file" "$policy_dir"
  assert_success
  local has_iam
  has_iam=$(echo "$output" | jq '[.warnings[] | select(.rule_id == "iam-change")] | length')
  [ "$has_iam" -eq 1 ]
}

@test "policy evaluate: default rules ignore safe operations" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  local policy_dir="$BATS_TEST_TMPDIR/eval-defaults-safe"
  mkdir -p "$policy_dir"

  cat > "$plan_file" <<'JSON'
{"resource_changes":[
  {"address":"aws_instance.web","type":"aws_instance","change":{"actions":["create"]}},
  {"address":"aws_cloudwatch_log_group.app","type":"aws_cloudwatch_log_group","change":{"actions":["update"]}}
]}
JSON

  run _tfui_policy_evaluate "$plan_file" "$policy_dir"
  assert_success
  local warning_count
  warning_count=$(echo "$output" | jq '.warnings | length')
  [ "$warning_count" -eq 0 ]
}

# -- Extends --

@test "policy extends: resolves tfui:aws pack" {
  local dir="$BATS_TEST_TMPDIR/extends-aws"
  mkdir -p "$dir"

  cat > "$dir/.tfui-policy.json" <<'JSON'
{"extends": ["tfui:aws"], "rules": []}
JSON

  run _tfui_policy_load "$dir"
  assert_success
  local count
  count=$(echo "$output" | jq '.rules | length')
  [ "$count" -gt 0 ]
  local has_aws
  has_aws=$(echo "$output" | jq '[.rules[] | select(.id | startswith("aws-"))] | length')
  [ "$has_aws" -gt 0 ]
}

@test "policy extends: resolves tfui:gcp pack" {
  local dir="$BATS_TEST_TMPDIR/extends-gcp"
  mkdir -p "$dir"

  cat > "$dir/.tfui-policy.json" <<'JSON'
{"extends": ["tfui:gcp"], "rules": []}
JSON

  run _tfui_policy_load "$dir"
  assert_success
  local has_gcp
  has_gcp=$(echo "$output" | jq '[.rules[] | select(.id | startswith("gcp-"))] | length')
  [ "$has_gcp" -gt 0 ]
}

@test "policy extends: resolves tfui:azure pack" {
  local dir="$BATS_TEST_TMPDIR/extends-azure"
  mkdir -p "$dir"

  cat > "$dir/.tfui-policy.json" <<'JSON'
{"extends": ["tfui:azure"], "rules": []}
JSON

  run _tfui_policy_load "$dir"
  assert_success
  local has_azure
  has_azure=$(echo "$output" | jq '[.rules[] | select(.id | startswith("azure-"))] | length')
  [ "$has_azure" -gt 0 ]
}

@test "policy extends: merges multiple packs" {
  local dir="$BATS_TEST_TMPDIR/extends-multi"
  mkdir -p "$dir"

  cat > "$dir/.tfui-policy.json" <<'JSON'
{"extends": ["tfui:aws", "tfui:gcp"], "rules": []}
JSON

  run _tfui_policy_load "$dir"
  assert_success
  local has_aws has_gcp
  has_aws=$(echo "$output" | jq '[.rules[] | select(.id | startswith("aws-"))] | length')
  has_gcp=$(echo "$output" | jq '[.rules[] | select(.id | startswith("gcp-"))] | length')
  [ "$has_aws" -gt 0 ]
  [ "$has_gcp" -gt 0 ]
}

@test "policy extends: user rules appended after pack rules" {
  local dir="$BATS_TEST_TMPDIR/extends-user"
  mkdir -p "$dir"

  cat > "$dir/.tfui-policy.json" <<'JSON'
{"extends": ["tfui:aws"], "rules": [
  {"id":"my-custom","description":"Custom rule","severity":"low","match":{"resource_type":"null_resource","action":["create"]}}
]}
JSON

  run _tfui_policy_load "$dir"
  assert_success
  local last_id
  last_id=$(echo "$output" | jq -r '.rules[-1].id')
  [ "$last_id" = "my-custom" ]
}

@test "policy extends: ignores unknown pack names" {
  local dir="$BATS_TEST_TMPDIR/extends-unknown"
  mkdir -p "$dir"

  cat > "$dir/.tfui-policy.json" <<'JSON'
{"extends": ["tfui:nonexistent"], "rules": [
  {"id":"my-rule","description":"Only rule","severity":"low","match":{"resource_type":"null_resource","action":["create"]}}
]}
JSON

  run _tfui_policy_load "$dir"
  assert_success
  local count
  count=$(echo "$output" | jq '.rules | length')
  [ "$count" -eq 1 ]
}

@test "policy extends: resolves relative path" {
  local dir="$BATS_TEST_TMPDIR/extends-relative"
  mkdir -p "$dir"

  cat > "$dir/my-policies.json" <<'JSON'
{"rules": [{"id":"relative-rule","description":"From relative file","severity":"medium","match":{"resource_type":"aws_instance","action":["delete"]}}]}
JSON

  cat > "$dir/.tfui-policy.json" <<'JSON'
{"extends": ["my-policies.json"], "rules": []}
JSON

  run _tfui_policy_load "$dir"
  assert_success
  local has_relative
  has_relative=$(echo "$output" | jq '[.rules[] | select(.id == "relative-rule")] | length')
  [ "$has_relative" -eq 1 ]
}

@test "policy extends: no extends field works as before" {
  local dir="$BATS_TEST_TMPDIR/extends-none"
  mkdir -p "$dir"

  cat > "$dir/.tfui-policy.json" <<'JSON'
{"rules": [{"id":"standalone","description":"Standalone rule","severity":"low","match":{"resource_type":"null_resource","action":["create"]}}]}
JSON

  run _tfui_policy_load "$dir"
  assert_success
  local rule_id
  rule_id=$(echo "$output" | jq -r '.rules[0].id')
  [ "$rule_id" = "standalone" ]
}
