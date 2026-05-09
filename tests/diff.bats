#!/usr/bin/env bats

setup() {
  load 'helpers/common-setup'
  _common_setup
  CLI="$PROJECT_ROOT/bin/tfui"
}

# -- Save summary --

@test "diff save: creates .tfui/plans directory" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  local work_dir="$BATS_TEST_TMPDIR/workspace"
  mkdir -p "$work_dir"

  cat > "$plan_file" <<'JSON'
{"resource_changes":[
  {"address":"aws_instance.web","change":{"actions":["create"]}},
  {"address":"aws_iam_role.old","change":{"actions":["delete"]}}
]}
JSON

  _tfui_diff_save_summary "$plan_file" "$work_dir"
  assert [ -d "$work_dir/.tfui/plans" ]
}

@test "diff save: extracts creates, updates, destroys, replaces" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  local work_dir="$BATS_TEST_TMPDIR/workspace"
  mkdir -p "$work_dir"

  cat > "$plan_file" <<'JSON'
{"resource_changes":[
  {"address":"aws_instance.web","change":{"actions":["create"]}},
  {"address":"aws_s3_bucket.data","change":{"actions":["update"]}},
  {"address":"aws_iam_role.old","change":{"actions":["delete"]}},
  {"address":"aws_instance.replace","change":{"actions":["delete","create"]}},
  {"address":"data.source","change":{"actions":["read"]}},
  {"address":"aws_vpc.main","change":{"actions":["no-op"]}}
]}
JSON

  _tfui_diff_save_summary "$plan_file" "$work_dir"

  local summary_file
  summary_file=$(ls -1t "$work_dir/.tfui/plans/"*.json | head -1)

  run jq -r '.creates[]' "$summary_file"
  assert_output "aws_instance.web"

  run jq -r '.updates[]' "$summary_file"
  assert_output "aws_s3_bucket.data"

  run jq -r '.destroys[]' "$summary_file"
  assert_output "aws_iam_role.old"

  run jq -r '.replaces[]' "$summary_file"
  assert_output "aws_instance.replace"
}

@test "diff save: no-op and read actions are excluded" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  local work_dir="$BATS_TEST_TMPDIR/workspace"
  mkdir -p "$work_dir"

  cat > "$plan_file" <<'JSON'
{"resource_changes":[
  {"address":"data.source","change":{"actions":["read"]}},
  {"address":"aws_vpc.main","change":{"actions":["no-op"]}}
]}
JSON

  _tfui_diff_save_summary "$plan_file" "$work_dir"

  local summary_file
  summary_file=$(ls -1t "$work_dir/.tfui/plans/"*.json | head -1)

  run jq '.creates | length' "$summary_file"
  assert_output "0"
  run jq '.updates | length' "$summary_file"
  assert_output "0"
  run jq '.destroys | length' "$summary_file"
  assert_output "0"
  run jq '.replaces | length' "$summary_file"
  assert_output "0"
}

# -- Cleanup --

@test "diff cleanup: keeps only HISTORY_LIMIT files" {
  local plans_dir="$BATS_TEST_TMPDIR/plans"
  mkdir -p "$plans_dir"

  # Create 12 files with different timestamps
  for i in $(seq -w 1 12); do
    echo '{"timestamp":"T'$i'","creates":[],"updates":[],"destroys":[],"replaces":[]}' \
      > "$plans_dir/202401${i}T120000.json"
    sleep 0.01
  done

  _TFUI_DIFF_HISTORY_LIMIT=10
  _tfui_diff_cleanup "$plans_dir"

  local count
  count=$(ls -1 "$plans_dir"/*.json | wc -l)
  [ "$count" -eq 10 ]
}

# -- Load latest --

@test "diff load: returns null when no plans directory exists" {
  run _tfui_diff_load_latest "$BATS_TEST_TMPDIR/nonexistent"
  local current previous
  current=$(jq -r '.current' <<< "$output")
  previous=$(jq -r '.previous' <<< "$output")
  [ "$current" = "null" ]
  [ "$previous" = "null" ]
}

@test "diff load: returns current but null previous with single plan" {
  local plans_dir="$BATS_TEST_TMPDIR/plans"
  mkdir -p "$plans_dir"
  echo '{"timestamp":"T1","creates":["a"],"updates":[],"destroys":[],"replaces":[]}' \
    > "$plans_dir/20240101T120000.json"

  run _tfui_diff_load_latest "$plans_dir"
  local current previous
  current=$(jq -r '.current.timestamp' <<< "$output")
  previous=$(jq -r '.previous' <<< "$output")
  [ "$current" = "T1" ]
  [ "$previous" = "null" ]
}

@test "diff load: returns current and previous with two plans" {
  local plans_dir="$BATS_TEST_TMPDIR/plans"
  mkdir -p "$plans_dir"
  echo '{"timestamp":"T1","creates":["a"],"updates":[],"destroys":[],"replaces":[]}' \
    > "$plans_dir/20240101T120000.json"
  echo '{"timestamp":"T2","creates":["a","b"],"updates":[],"destroys":[],"replaces":[]}' \
    > "$plans_dir/20240102T120000.json"

  run _tfui_diff_load_latest "$plans_dir"
  local current previous
  current=$(jq -r '.current.timestamp' <<< "$output")
  previous=$(jq -r '.previous.timestamp' <<< "$output")
  [ "$current" = "T2" ]
  [ "$previous" = "T1" ]
}

# -- Compute diff --

@test "diff compute: detects new creates" {
  local current='{"timestamp":"T2","creates":["a","b"],"updates":[],"destroys":[],"replaces":[]}'
  local previous='{"timestamp":"T1","creates":["a"],"updates":[],"destroys":[],"replaces":[]}'

  run _tfui_diff_compute "$current" "$previous"
  local new_creates
  new_creates=$(jq -r '.delta.new_creates[]' <<< "$output")
  [ "$new_creates" = "b" ]
}

@test "diff compute: detects resolved destroys" {
  local current='{"timestamp":"T2","creates":[],"updates":[],"destroys":[],"replaces":[]}'
  local previous='{"timestamp":"T1","creates":[],"updates":[],"destroys":["aws_iam_role.old"],"replaces":[]}'

  run _tfui_diff_compute "$current" "$previous"
  local resolved
  resolved=$(jq -r '.delta.resolved_destroys[]' <<< "$output")
  [ "$resolved" = "aws_iam_role.old" ]
}

@test "diff compute: detects new updates" {
  local current='{"timestamp":"T2","creates":[],"updates":["aws_sg.tls"],"destroys":[],"replaces":[]}'
  local previous='{"timestamp":"T1","creates":[],"updates":[],"destroys":[],"replaces":[]}'

  run _tfui_diff_compute "$current" "$previous"
  local new_updates
  new_updates=$(jq -r '.delta.new_updates[]' <<< "$output")
  [ "$new_updates" = "aws_sg.tls" ]
}

@test "diff compute: risk trend improving when destroys removed" {
  local current='{"timestamp":"T2","creates":["a"],"updates":[],"destroys":[],"replaces":[]}'
  local previous='{"timestamp":"T1","creates":[],"updates":[],"destroys":["b"],"replaces":[]}'

  run _tfui_diff_compute "$current" "$previous"
  local trend
  trend=$(jq -r '.risk_trend' <<< "$output")
  [ "$trend" = "improving" ]
}

@test "diff compute: risk trend worsening when destroys added" {
  local current='{"timestamp":"T2","creates":[],"updates":[],"destroys":["a","b"],"replaces":[]}'
  local previous='{"timestamp":"T1","creates":[],"updates":[],"destroys":[],"replaces":[]}'

  run _tfui_diff_compute "$current" "$previous"
  local trend
  trend=$(jq -r '.risk_trend' <<< "$output")
  [ "$trend" = "worsening" ]
}

@test "diff compute: risk trend unchanged when same risk profile" {
  local current='{"timestamp":"T2","creates":["a"],"updates":["b"],"destroys":[],"replaces":[]}'
  local previous='{"timestamp":"T1","creates":["c"],"updates":["d"],"destroys":[],"replaces":[]}'

  run _tfui_diff_compute "$current" "$previous"
  local trend
  trend=$(jq -r '.risk_trend' <<< "$output")
  [ "$trend" = "unchanged" ]
}

@test "diff compute: summary counts are correct" {
  local current='{"timestamp":"T2","creates":["a","b"],"updates":["c"],"destroys":[],"replaces":["d"]}'
  local previous='{"timestamp":"T1","creates":["a"],"updates":[],"destroys":["e"],"replaces":[]}'

  run _tfui_diff_compute "$current" "$previous"
  [ "$(jq '.summary.before.add' <<< "$output")" = "1" ]
  [ "$(jq '.summary.before.change' <<< "$output")" = "0" ]
  [ "$(jq '.summary.before.destroy' <<< "$output")" = "1" ]
  [ "$(jq '.summary.before.replace' <<< "$output")" = "0" ]
  [ "$(jq '.summary.after.add' <<< "$output")" = "2" ]
  [ "$(jq '.summary.after.change' <<< "$output")" = "1" ]
  [ "$(jq '.summary.after.destroy' <<< "$output")" = "0" ]
  [ "$(jq '.summary.after.replace' <<< "$output")" = "1" ]
}

# -- Render human --

@test "diff render human: shows new creates with ++ prefix" {
  local diff_json='{"previous_plan":"T1","current_plan":"T2","delta":{"new_creates":["aws_instance.worker"],"resolved_creates":[],"new_updates":[],"resolved_updates":[],"new_destroys":[],"resolved_destroys":[],"new_replaces":[],"resolved_replaces":[]},"summary":{"before":{"add":0,"change":0,"destroy":0,"replace":0},"after":{"add":1,"change":0,"destroy":0,"replace":0}},"risk_trend":"unchanged"}'

  run _tfui_diff_render_human "$diff_json"
  assert_output --partial "++ aws_instance.worker (new create)"
}

@test "diff render human: shows resolved destroys with -- prefix" {
  local diff_json='{"previous_plan":"T1","current_plan":"T2","delta":{"new_creates":[],"resolved_creates":[],"new_updates":[],"resolved_updates":[],"new_destroys":[],"resolved_destroys":["aws_iam_role.old"],"new_replaces":[],"resolved_replaces":[]},"summary":{"before":{"add":0,"change":0,"destroy":1,"replace":0},"after":{"add":0,"change":0,"destroy":0,"replace":0}},"risk_trend":"improving"}'

  run _tfui_diff_render_human "$diff_json"
  assert_output --partial "-- aws_iam_role.old (destroy resolved)"
  assert_output --partial "Risk: improving"
}

@test "diff render human: shows no changes message when delta is empty" {
  local diff_json='{"previous_plan":"T1","current_plan":"T2","delta":{"new_creates":[],"resolved_creates":[],"new_updates":[],"resolved_updates":[],"new_destroys":[],"resolved_destroys":[],"new_replaces":[],"resolved_replaces":[]},"summary":{"before":{"add":1,"change":0,"destroy":0,"replace":0},"after":{"add":1,"change":0,"destroy":0,"replace":0}},"risk_trend":"unchanged"}'

  run _tfui_diff_render_human "$diff_json"
  assert_output --partial "(no changes between plans)"
}

# -- Public API: tfui_diff --

@test "tfui_diff: returns error when no plan history exists" {
  local work_dir="$BATS_TEST_TMPDIR/empty-workspace"
  mkdir -p "$work_dir"

  run tfui_diff "$work_dir" "json"
  [ "$status" -eq 1 ]
  assert_output --partial "no_plan_history"
}

@test "tfui_diff: returns error when only one plan exists" {
  local work_dir="$BATS_TEST_TMPDIR/workspace"
  mkdir -p "$work_dir/.tfui/plans"
  echo '{"timestamp":"T1","creates":["a"],"updates":[],"destroys":[],"replaces":[]}' \
    > "$work_dir/.tfui/plans/20240101T120000.json"

  run tfui_diff "$work_dir" "json"
  [ "$status" -eq 1 ]
  assert_output --partial "no_previous_plan"
}

@test "tfui_diff: returns error in human format when no history" {
  local work_dir="$BATS_TEST_TMPDIR/empty-workspace"
  mkdir -p "$work_dir"

  run tfui_diff "$work_dir" "human"
  [ "$status" -eq 1 ]
  assert_output --partial "No plan history found"
}

@test "tfui_diff: computes diff with two plans in json format" {
  local work_dir="$BATS_TEST_TMPDIR/workspace"
  mkdir -p "$work_dir/.tfui/plans"
  echo '{"timestamp":"T1","creates":["a"],"updates":[],"destroys":["b"],"replaces":[]}' \
    > "$work_dir/.tfui/plans/20240101T120000.json"
  echo '{"timestamp":"T2","creates":["a","c"],"updates":[],"destroys":[],"replaces":[]}' \
    > "$work_dir/.tfui/plans/20240102T120000.json"

  run tfui_diff "$work_dir" "json"
  [ "$status" -eq 0 ]
  [ "$(jq -r '.risk_trend' <<< "$output")" = "improving" ]
  [ "$(jq -r '.delta.new_creates[0]' <<< "$output")" = "c" ]
  [ "$(jq -r '.delta.resolved_destroys[0]' <<< "$output")" = "b" ]
}

@test "tfui_diff: computes diff in human format" {
  local work_dir="$BATS_TEST_TMPDIR/workspace"
  mkdir -p "$work_dir/.tfui/plans"
  echo '{"timestamp":"T1","creates":[],"updates":[],"destroys":["aws_iam_role.old"],"replaces":[]}' \
    > "$work_dir/.tfui/plans/20240101T120000.json"
  echo '{"timestamp":"T2","creates":["aws_instance.new"],"updates":[],"destroys":[],"replaces":[]}' \
    > "$work_dir/.tfui/plans/20240102T120000.json"

  run tfui_diff "$work_dir" "human"
  [ "$status" -eq 0 ]
  assert_output --partial "++ aws_instance.new (new create)"
  assert_output --partial "-- aws_iam_role.old (destroy resolved)"
  assert_output --partial "Risk: improving"
}

# -- CLI: tfui diff --

@test "cli diff: returns error when no plan history" {
  local work_dir="$BATS_TEST_TMPDIR/empty-workspace"
  mkdir -p "$work_dir"

  run "$CLI" diff --dir "$work_dir"
  [ "$status" -eq 1 ]
  assert_output --partial "no_plan_history"
}

@test "cli diff: with nonexistent dir exits 1" {
  run "$CLI" diff --dir /nonexistent
  [ "$status" -eq 1 ]
  assert_output --partial "directory not found"
}

@test "cli diff: unknown option exits 1" {
  run "$CLI" diff --nope
  [ "$status" -eq 1 ]
  assert_output --partial "unknown option: --nope"
}

@test "cli diff: outputs json format by default" {
  local work_dir="$BATS_TEST_TMPDIR/workspace"
  mkdir -p "$work_dir/.tfui/plans"
  echo '{"timestamp":"T1","creates":["a"],"updates":[],"destroys":[],"replaces":[]}' \
    > "$work_dir/.tfui/plans/20240101T120000.json"
  echo '{"timestamp":"T2","creates":["a","b"],"updates":[],"destroys":[],"replaces":[]}' \
    > "$work_dir/.tfui/plans/20240102T120000.json"

  run "$CLI" diff --dir "$work_dir"
  [ "$status" -eq 0 ]
  [ "$(jq -r '.delta.new_creates[0]' <<< "$output")" = "b" ]
}

@test "cli diff: outputs human format with --format human" {
  local work_dir="$BATS_TEST_TMPDIR/workspace"
  mkdir -p "$work_dir/.tfui/plans"
  echo '{"timestamp":"T1","creates":[],"updates":[],"destroys":["x"],"replaces":[]}' \
    > "$work_dir/.tfui/plans/20240101T120000.json"
  echo '{"timestamp":"T2","creates":[],"updates":[],"destroys":[],"replaces":[]}' \
    > "$work_dir/.tfui/plans/20240102T120000.json"

  run "$CLI" diff --dir "$work_dir" --format human
  [ "$status" -eq 0 ]
  assert_output --partial "-- x (destroy resolved)"
  assert_output --partial "Risk: improving"
}
