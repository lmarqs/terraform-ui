#!/usr/bin/env bats

setup() {
  load 'helpers/common-setup'
  _common_setup
}

# -- Pattern matching --

@test "env_matches_patterns: exact match returns 0" {
  run _tfui_env_matches_patterns "prod" "prod production prd"
  [ "$status" -eq 0 ]
}

@test "env_matches_patterns: no match returns 1" {
  run _tfui_env_matches_patterns "productive" "prod production prd"
  [ "$status" -eq 1 ]
}

@test "env_matches_patterns: matches second pattern" {
  run _tfui_env_matches_patterns "production" "prod production prd"
  [ "$status" -eq 0 ]
}

@test "env_matches_patterns: empty value returns 1" {
  run _tfui_env_matches_patterns "" "prod production prd"
  [ "$status" -eq 1 ]
}

# -- Value classification --

@test "env_classify_value: prod classifies as production" {
  run _tfui_env_classify_value "prod"
  assert_output "production"
}

@test "env_classify_value: PRODUCTION (uppercase) classifies as production" {
  run _tfui_env_classify_value "PRODUCTION"
  assert_output "production"
}

@test "env_classify_value: staging classifies as staging" {
  run _tfui_env_classify_value "staging"
  assert_output "staging"
}

@test "env_classify_value: stg classifies as staging" {
  run _tfui_env_classify_value "stg"
  assert_output "staging"
}

@test "env_classify_value: dev classifies as development" {
  run _tfui_env_classify_value "dev"
  assert_output "development"
}

@test "env_classify_value: sandbox classifies as development" {
  run _tfui_env_classify_value "sandbox"
  assert_output "development"
}

@test "env_classify_value: unknown value returns empty" {
  run _tfui_env_classify_value "feature-branch"
  assert_output ""
}

@test "env_classify_value: partial match is rejected (productive)" {
  run _tfui_env_classify_value "productive"
  assert_output ""
}

# -- Detection from env var --

@test "env_detect_from_env_var: TFUI_ENVIRONMENT=production returns production" {
  TFUI_ENVIRONMENT="production" run _tfui_env_detect_from_env_var
  assert_output "production"
}

@test "env_detect_from_env_var: TFUI_ENVIRONMENT=PROD returns production" {
  TFUI_ENVIRONMENT="PROD" run _tfui_env_detect_from_env_var
  assert_output "production"
}

@test "env_detect_from_env_var: unset returns empty" {
  unset TFUI_ENVIRONMENT
  run _tfui_env_detect_from_env_var
  assert_output ""
}

@test "env_detect_from_env_var: unrecognized value returns empty" {
  TFUI_ENVIRONMENT="custom-env" run _tfui_env_detect_from_env_var
  assert_output ""
}

# -- Detection from .tfui-env file --

@test "env_detect_from_file: reads production from .tfui-env" {
  local dir="$BATS_TEST_TMPDIR/project"
  mkdir -p "$dir"
  echo "production" > "$dir/.tfui-env"
  run _tfui_env_detect_from_file "$dir"
  assert_output "production"
}

@test "env_detect_from_file: handles whitespace in file" {
  local dir="$BATS_TEST_TMPDIR/project"
  mkdir -p "$dir"
  printf "  staging  \n" > "$dir/.tfui-env"
  run _tfui_env_detect_from_file "$dir"
  assert_output "staging"
}

@test "env_detect_from_file: missing file returns empty" {
  local dir="$BATS_TEST_TMPDIR/project"
  mkdir -p "$dir"
  run _tfui_env_detect_from_file "$dir"
  assert_output ""
}

@test "env_detect_from_file: empty file returns empty" {
  local dir="$BATS_TEST_TMPDIR/project"
  mkdir -p "$dir"
  echo "" > "$dir/.tfui-env"
  run _tfui_env_detect_from_file "$dir"
  assert_output ""
}

# -- Detection from workspace --

@test "env_detect_from_workspace: reads prod workspace" {
  local dir="$BATS_TEST_TMPDIR/project"
  mkdir -p "$dir/.terraform"
  echo "prod" > "$dir/.terraform/environment"
  run _tfui_env_detect_from_workspace "$dir"
  assert_output "production"
}

@test "env_detect_from_workspace: default workspace returns empty" {
  local dir="$BATS_TEST_TMPDIR/project"
  mkdir -p "$dir/.terraform"
  echo "default" > "$dir/.terraform/environment"
  run _tfui_env_detect_from_workspace "$dir"
  assert_output ""
}

@test "env_detect_from_workspace: no .terraform dir returns empty" {
  local dir="$BATS_TEST_TMPDIR/project"
  mkdir -p "$dir"
  run _tfui_env_detect_from_workspace "$dir"
  assert_output ""
}

@test "env_detect_from_workspace: staging workspace" {
  local dir="$BATS_TEST_TMPDIR/project"
  mkdir -p "$dir/.terraform"
  echo "stg" > "$dir/.terraform/environment"
  run _tfui_env_detect_from_workspace "$dir"
  assert_output "staging"
}

# -- Detection from plan variables --

@test "env_detect_from_plan_vars: environment variable in plan" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  echo '{"variables":{"environment":{"value":"production"}}}' > "$plan_file"
  run _tfui_env_detect_from_plan_vars "$plan_file"
  assert_output "production"
}

@test "env_detect_from_plan_vars: env variable in plan" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  echo '{"variables":{"env":{"value":"staging"}}}' > "$plan_file"
  run _tfui_env_detect_from_plan_vars "$plan_file"
  assert_output "staging"
}

@test "env_detect_from_plan_vars: no env variable returns empty" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  echo '{"variables":{"region":{"value":"us-east-1"}}}' > "$plan_file"
  run _tfui_env_detect_from_plan_vars "$plan_file"
  assert_output ""
}

@test "env_detect_from_plan_vars: missing plan file returns empty" {
  run _tfui_env_detect_from_plan_vars "/nonexistent/plan.json"
  assert_output ""
}

@test "env_detect_from_plan_vars: environment takes precedence over env" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  echo '{"variables":{"environment":{"value":"prod"},"env":{"value":"dev"}}}' > "$plan_file"
  run _tfui_env_detect_from_plan_vars "$plan_file"
  assert_output "production"
}

# -- Detection from directory path --

@test "env_detect_from_directory: /infra/prod/ detects production" {
  run _tfui_env_detect_from_directory "/home/user/infra/prod/vpc"
  assert_output "production"
}

@test "env_detect_from_directory: /envs/staging/ detects staging" {
  run _tfui_env_detect_from_directory "/home/user/envs/staging/app"
  assert_output "staging"
}

@test "env_detect_from_directory: /environments/dev/ detects development" {
  run _tfui_env_detect_from_directory "/home/user/environments/dev/networking"
  assert_output "development"
}

@test "env_detect_from_directory: no env segments returns empty" {
  run _tfui_env_detect_from_directory "/home/user/infra/networking/vpc"
  assert_output ""
}

@test "env_detect_from_directory: productive is not matched" {
  run _tfui_env_detect_from_directory "/home/user/productive/app"
  assert_output ""
}

# -- Signal precedence --

@test "detect_environment: env var takes precedence over file" {
  local dir="$BATS_TEST_TMPDIR/project"
  mkdir -p "$dir"
  echo "staging" > "$dir/.tfui-env"
  TFUI_ENVIRONMENT="production" run _tfui_detect_environment "$dir"
  local detected
  detected=$(echo "$output" | jq -r '.environment.detected')
  [ "$detected" = "production" ]
}

@test "detect_environment: file takes precedence over workspace" {
  local dir="$BATS_TEST_TMPDIR/project"
  mkdir -p "$dir/.terraform"
  echo "prod" > "$dir/.terraform/environment"
  echo "staging" > "$dir/.tfui-env"
  unset TFUI_ENVIRONMENT
  run _tfui_detect_environment "$dir"
  local detected
  detected=$(echo "$output" | jq -r '.environment.detected')
  [ "$detected" = "staging" ]
}

@test "detect_environment: workspace takes precedence over plan vars" {
  local dir="$BATS_TEST_TMPDIR/project"
  mkdir -p "$dir/.terraform"
  echo "prod" > "$dir/.terraform/environment"
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  echo '{"variables":{"environment":{"value":"dev"}}}' > "$plan_file"
  unset TFUI_ENVIRONMENT
  run _tfui_detect_environment "$dir" "$plan_file"
  local detected
  detected=$(echo "$output" | jq -r '.environment.detected')
  [ "$detected" = "production" ]
}

@test "detect_environment: returns unknown when no signals match" {
  local dir="$BATS_TEST_TMPDIR/project"
  mkdir -p "$dir"
  unset TFUI_ENVIRONMENT
  run _tfui_detect_environment "$dir"
  local detected
  detected=$(echo "$output" | jq -r '.environment.detected')
  [ "$detected" = "unknown" ]
}

@test "detect_environment: reports confidence level" {
  local dir="$BATS_TEST_TMPDIR/project"
  mkdir -p "$dir"
  echo "prod" > "$dir/.tfui-env"
  unset TFUI_ENVIRONMENT
  run _tfui_detect_environment "$dir"
  local confidence
  confidence=$(echo "$output" | jq -r '.environment.confidence')
  [ "$confidence" = "high" ]
}

@test "detect_environment: reports signal source" {
  local dir="$BATS_TEST_TMPDIR/project"
  mkdir -p "$dir"
  echo "prod" > "$dir/.tfui-env"
  unset TFUI_ENVIRONMENT
  run _tfui_detect_environment "$dir"
  local signal
  signal=$(echo "$output" | jq -r '.environment.signals[0]')
  [ "$signal" = "file: .tfui-env" ]
}

# -- Risk multiplier --

@test "env_risk_multiplier: production returns 3.0" {
  run _tfui_env_risk_multiplier "production"
  assert_output "3.0"
}

@test "env_risk_multiplier: staging returns 1.5" {
  run _tfui_env_risk_multiplier "staging"
  assert_output "1.5"
}

@test "env_risk_multiplier: development returns 0.5" {
  run _tfui_env_risk_multiplier "development"
  assert_output "0.5"
}

@test "env_risk_multiplier: unknown returns 1.0" {
  run _tfui_env_risk_multiplier "unknown"
  assert_output "1.0"
}

# -- Base risk computation --

@test "compute_base_risk: creates only is low" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  echo '{"resource_changes":[{"address":"a","change":{"actions":["create"]}}]}' > "$plan_file"
  run _tfui_compute_base_risk "$plan_file"
  assert_output "low"
}

@test "compute_base_risk: updates is medium" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  echo '{"resource_changes":[{"address":"a","change":{"actions":["update"]}}]}' > "$plan_file"
  run _tfui_compute_base_risk "$plan_file"
  assert_output "medium"
}

@test "compute_base_risk: deletes is high" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  echo '{"resource_changes":[{"address":"a","change":{"actions":["delete"]}}]}' > "$plan_file"
  run _tfui_compute_base_risk "$plan_file"
  assert_output "high"
}

@test "compute_base_risk: replace (delete+create) is high" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  echo '{"resource_changes":[{"address":"a","change":{"actions":["delete","create"]}}]}' > "$plan_file"
  run _tfui_compute_base_risk "$plan_file"
  assert_output "high"
}

@test "compute_base_risk: no changes is none" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  echo '{"resource_changes":[{"address":"a","change":{"actions":["no-op"]}}]}' > "$plan_file"
  run _tfui_compute_base_risk "$plan_file"
  assert_output "none"
}

@test "compute_base_risk: missing file is none" {
  run _tfui_compute_base_risk "/nonexistent/plan.json"
  assert_output "none"
}

# -- Adjusted risk computation --

@test "compute_adjusted_risk: low * 3.0 (production) = medium" {
  run _tfui_compute_adjusted_risk "low" "3.0"
  assert_output "medium"
}

@test "compute_adjusted_risk: medium * 3.0 (production) = critical" {
  run _tfui_compute_adjusted_risk "medium" "3.0"
  assert_output "critical"
}

@test "compute_adjusted_risk: high * 3.0 (production) = critical" {
  run _tfui_compute_adjusted_risk "high" "3.0"
  assert_output "critical"
}

@test "compute_adjusted_risk: high * 0.5 (development) = low" {
  run _tfui_compute_adjusted_risk "high" "0.5"
  assert_output "low"
}

@test "compute_adjusted_risk: medium * 1.5 (staging) = medium" {
  run _tfui_compute_adjusted_risk "medium" "1.5"
  assert_output "medium"
}

@test "compute_adjusted_risk: high * 1.5 (staging) = high" {
  run _tfui_compute_adjusted_risk "high" "1.5"
  assert_output "high"
}

@test "compute_adjusted_risk: none stays none regardless of multiplier" {
  run _tfui_compute_adjusted_risk "none" "3.0"
  assert_output "none"
}

# -- Config loading --

@test "env_load_config: loads custom patterns from config file" {
  local config="$BATS_TEST_TMPDIR/config.json"
  cat > "$config" <<'JSON'
{
  "environments": {
    "production": { "patterns": ["live", "prod-us"], "risk_multiplier": 5.0 }
  }
}
JSON
  _tfui_env_load_config "$config"
  [ "$_TFUI_ENV_PATTERNS_PRODUCTION" = "live prod-us" ]
  [ "$_TFUI_ENV_MULTIPLIER_PRODUCTION" = "5.0" ]
}

@test "env_load_config: missing file preserves defaults" {
  local original="$_TFUI_ENV_PATTERNS_PRODUCTION"
  _tfui_env_load_config "/nonexistent/config.json"
  [ "$_TFUI_ENV_PATTERNS_PRODUCTION" = "$original" ]
}

# -- Full report --

@test "tfui_env_report: produces valid JSON with all fields" {
  local dir="$BATS_TEST_TMPDIR/project"
  mkdir -p "$dir"
  echo "prod" > "$dir/.tfui-env"

  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  echo '{"resource_changes":[{"address":"a","change":{"actions":["delete"]}}]}' > "$plan_file"

  unset TFUI_ENVIRONMENT
  run tfui_env_report "$dir" "$plan_file"

  # Validate JSON structure
  echo "$output" | jq -e '.environment.detected' >/dev/null
  echo "$output" | jq -e '.environment.confidence' >/dev/null
  echo "$output" | jq -e '.environment.signals' >/dev/null
  echo "$output" | jq -e '.environment.risk_multiplier' >/dev/null
  echo "$output" | jq -e '.base_risk_level' >/dev/null
  echo "$output" | jq -e '.adjusted_risk_level' >/dev/null
}

@test "tfui_env_report: production with deletes shows critical" {
  local dir="$BATS_TEST_TMPDIR/project"
  mkdir -p "$dir"
  echo "prod" > "$dir/.tfui-env"

  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  echo '{"resource_changes":[{"address":"a","change":{"actions":["delete"]}}]}' > "$plan_file"

  unset TFUI_ENVIRONMENT
  run tfui_env_report "$dir" "$plan_file"

  local adjusted
  adjusted=$(echo "$output" | jq -r '.adjusted_risk_level')
  [ "$adjusted" = "critical" ]
}

@test "tfui_env_report: development with deletes shows low risk" {
  local dir="$BATS_TEST_TMPDIR/project"
  mkdir -p "$dir"
  echo "dev" > "$dir/.tfui-env"

  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  echo '{"resource_changes":[{"address":"a","change":{"actions":["delete"]}}]}' > "$plan_file"

  unset TFUI_ENVIRONMENT
  run tfui_env_report "$dir" "$plan_file"

  local adjusted
  adjusted=$(echo "$output" | jq -r '.adjusted_risk_level')
  [ "$adjusted" = "low" ]
}
