#!/usr/bin/env bats

setup() {
  load 'helpers/common-setup'
  _common_setup
}

# -- _tfui_analyze_blast_radius --

@test "blast radius: no destructive changes returns empty result" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  cat > "$plan_file" <<'JSON'
{
  "resource_changes": [
    {"address": "local_file.one", "change": {"actions": ["create"]}},
    {"address": "local_file.two", "change": {"actions": ["update"]}}
  ],
  "configuration": {
    "root_module": {
      "resources": [
        {"address": "local_file.one", "dependencies": []},
        {"address": "local_file.two", "dependencies": ["local_file.one"]}
      ]
    }
  }
}
JSON
  run _tfui_analyze_blast_radius "$plan_file"
  assert_success
  # Should have zero affected
  local total
  total=$(echo "$output" | jq -r '.total_affected')
  [ "$total" -eq 0 ]
}

@test "blast radius: delete with one dependent finds it" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  cat > "$plan_file" <<'JSON'
{
  "resource_changes": [
    {"address": "local_file.database", "change": {"actions": ["delete"]}}
  ],
  "configuration": {
    "root_module": {
      "resources": [
        {"address": "local_file.database", "dependencies": []},
        {"address": "local_file.app_config", "dependencies": ["local_file.database"]}
      ]
    }
  }
}
JSON
  run _tfui_analyze_blast_radius "$plan_file"
  assert_success
  local total
  total=$(echo "$output" | jq -r '.total_affected')
  [ "$total" -eq 1 ]
  # Verify the affected resource
  local affected
  affected=$(echo "$output" | jq -r '.blast_radius[0].affected_resources[0]')
  [ "$affected" = "local_file.app_config" ]
}

@test "blast radius: delete with transitive chain finds all dependents" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  cat > "$plan_file" <<'JSON'
{
  "resource_changes": [
    {"address": "local_file.database", "change": {"actions": ["delete"]}}
  ],
  "configuration": {
    "root_module": {
      "resources": [
        {"address": "local_file.database", "dependencies": []},
        {"address": "local_file.app_config", "dependencies": ["local_file.database"]},
        {"address": "local_file.backup", "dependencies": ["local_file.database"]},
        {"address": "local_file.web_server", "dependencies": ["local_file.app_config"]},
        {"address": "local_file.independent", "dependencies": []}
      ]
    }
  }
}
JSON
  run _tfui_analyze_blast_radius "$plan_file"
  assert_success
  # Should find 3 affected (app_config, backup, web_server) but NOT independent
  local total
  total=$(echo "$output" | jq -r '.total_affected')
  [ "$total" -eq 3 ]
  # Cascade depth should be 2
  local depth
  depth=$(echo "$output" | jq -r '.max_cascade_depth')
  [ "$depth" -eq 2 ]
}

@test "blast radius: replace action is treated as destructive" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  cat > "$plan_file" <<'JSON'
{
  "resource_changes": [
    {"address": "local_file.database", "change": {"actions": ["delete", "create"]}}
  ],
  "configuration": {
    "root_module": {
      "resources": [
        {"address": "local_file.database", "dependencies": []},
        {"address": "local_file.app_config", "dependencies": ["local_file.database"]}
      ]
    }
  }
}
JSON
  run _tfui_analyze_blast_radius "$plan_file"
  assert_success
  local action
  action=$(echo "$output" | jq -r '.blast_radius[0].action')
  [ "$action" = "replace" ]
  local total
  total=$(echo "$output" | jq -r '.total_affected')
  [ "$total" -eq 1 ]
}

@test "blast radius: risk is low for 1-2 affected" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  cat > "$plan_file" <<'JSON'
{
  "resource_changes": [
    {"address": "local_file.a", "change": {"actions": ["delete"]}}
  ],
  "configuration": {
    "root_module": {
      "resources": [
        {"address": "local_file.a", "dependencies": []},
        {"address": "local_file.b", "dependencies": ["local_file.a"]},
        {"address": "local_file.c", "dependencies": ["local_file.a"]}
      ]
    }
  }
}
JSON
  run _tfui_analyze_blast_radius "$plan_file"
  assert_success
  local risk
  risk=$(echo "$output" | jq -r '.blast_radius[0].risk')
  [ "$risk" = "low" ]
}

@test "blast radius: risk is moderate for 3-5 affected" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  cat > "$plan_file" <<'JSON'
{
  "resource_changes": [
    {"address": "local_file.a", "change": {"actions": ["delete"]}}
  ],
  "configuration": {
    "root_module": {
      "resources": [
        {"address": "local_file.a", "dependencies": []},
        {"address": "local_file.b", "dependencies": ["local_file.a"]},
        {"address": "local_file.c", "dependencies": ["local_file.a"]},
        {"address": "local_file.d", "dependencies": ["local_file.a"]}
      ]
    }
  }
}
JSON
  run _tfui_analyze_blast_radius "$plan_file"
  assert_success
  local risk
  risk=$(echo "$output" | jq -r '.blast_radius[0].risk')
  [ "$risk" = "moderate" ]
}

@test "blast radius: risk is critical for delete with 6+ affected" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  cat > "$plan_file" <<'JSON'
{
  "resource_changes": [
    {"address": "local_file.a", "change": {"actions": ["delete"]}}
  ],
  "configuration": {
    "root_module": {
      "resources": [
        {"address": "local_file.a", "dependencies": []},
        {"address": "local_file.b", "dependencies": ["local_file.a"]},
        {"address": "local_file.c", "dependencies": ["local_file.a"]},
        {"address": "local_file.d", "dependencies": ["local_file.a"]},
        {"address": "local_file.e", "dependencies": ["local_file.a"]},
        {"address": "local_file.f", "dependencies": ["local_file.a"]},
        {"address": "local_file.g", "dependencies": ["local_file.a"]}
      ]
    }
  }
}
JSON
  run _tfui_analyze_blast_radius "$plan_file"
  assert_success
  local risk
  risk=$(echo "$output" | jq -r '.blast_radius[0].risk')
  [ "$risk" = "critical" ]
}

@test "blast radius: risk is high for replace with 6+ affected" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  cat > "$plan_file" <<'JSON'
{
  "resource_changes": [
    {"address": "local_file.a", "change": {"actions": ["delete", "create"]}}
  ],
  "configuration": {
    "root_module": {
      "resources": [
        {"address": "local_file.a", "dependencies": []},
        {"address": "local_file.b", "dependencies": ["local_file.a"]},
        {"address": "local_file.c", "dependencies": ["local_file.a"]},
        {"address": "local_file.d", "dependencies": ["local_file.a"]},
        {"address": "local_file.e", "dependencies": ["local_file.a"]},
        {"address": "local_file.f", "dependencies": ["local_file.a"]},
        {"address": "local_file.g", "dependencies": ["local_file.a"]}
      ]
    }
  }
}
JSON
  run _tfui_analyze_blast_radius "$plan_file"
  assert_success
  local risk
  risk=$(echo "$output" | jq -r '.blast_radius[0].risk')
  [ "$risk" = "high" ]
}

@test "blast radius: max_depth limits traversal" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  cat > "$plan_file" <<'JSON'
{
  "resource_changes": [
    {"address": "local_file.a", "change": {"actions": ["delete"]}}
  ],
  "configuration": {
    "root_module": {
      "resources": [
        {"address": "local_file.a", "dependencies": []},
        {"address": "local_file.b", "dependencies": ["local_file.a"]},
        {"address": "local_file.c", "dependencies": ["local_file.b"]},
        {"address": "local_file.d", "dependencies": ["local_file.c"]}
      ]
    }
  }
}
JSON
  # With max_depth=1, should only find direct dependents
  run _tfui_analyze_blast_radius "$plan_file" 1
  assert_success
  local total
  total=$(echo "$output" | jq -r '.total_affected')
  [ "$total" -eq 1 ]
}

@test "blast radius: handles missing configuration gracefully" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  cat > "$plan_file" <<'JSON'
{
  "resource_changes": [
    {"address": "local_file.a", "change": {"actions": ["delete"]}}
  ],
  "configuration": {
    "root_module": {}
  }
}
JSON
  run _tfui_analyze_blast_radius "$plan_file"
  assert_success
  local total
  total=$(echo "$output" | jq -r '.total_affected')
  [ "$total" -eq 0 ]
}

@test "blast radius: handles circular dependencies without infinite loop" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  cat > "$plan_file" <<'JSON'
{
  "resource_changes": [
    {"address": "local_file.a", "change": {"actions": ["delete"]}}
  ],
  "configuration": {
    "root_module": {
      "resources": [
        {"address": "local_file.a", "dependencies": ["local_file.b"]},
        {"address": "local_file.b", "dependencies": ["local_file.a"]}
      ]
    }
  }
}
JSON
  run _tfui_analyze_blast_radius "$plan_file"
  assert_success
  local total
  total=$(echo "$output" | jq -r '.total_affected')
  [ "$total" -eq 1 ]
}

# -- _tfui_render_blast_radius --

@test "blast radius render: no affected resources produces no output" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  cat > "$plan_file" <<'JSON'
{
  "resource_changes": [
    {"address": "local_file.a", "change": {"actions": ["create"]}}
  ],
  "configuration": {
    "root_module": {
      "resources": [
        {"address": "local_file.a", "dependencies": []}
      ]
    }
  }
}
JSON
  run _tfui_render_blast_radius "$plan_file"
  assert_success
  assert_output ""
}

@test "blast radius render: shows tree for destructive change with dependents" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  cat > "$plan_file" <<'JSON'
{
  "resource_changes": [
    {"address": "local_file.database", "change": {"actions": ["delete"]}}
  ],
  "configuration": {
    "root_module": {
      "resources": [
        {"address": "local_file.database", "dependencies": []},
        {"address": "local_file.app_config", "dependencies": ["local_file.database"]}
      ]
    }
  }
}
JSON
  run _tfui_render_blast_radius "$plan_file"
  assert_success
  assert_line --partial "Blast Radius:"
  assert_line --partial "- local_file.database (LOW)"
  assert_line --partial "local_file.app_config"
  assert_line --partial "Total cascade: 1 additional resource(s) affected"
}

@test "blast radius render: replace shows -/+ prefix" {
  local plan_file="$BATS_TEST_TMPDIR/plan.json"
  cat > "$plan_file" <<'JSON'
{
  "resource_changes": [
    {"address": "local_file.database", "change": {"actions": ["delete", "create"]}}
  ],
  "configuration": {
    "root_module": {
      "resources": [
        {"address": "local_file.database", "dependencies": []},
        {"address": "local_file.app_config", "dependencies": ["local_file.database"]}
      ]
    }
  }
}
JSON
  run _tfui_render_blast_radius "$plan_file"
  assert_success
  assert_line --partial "-/+ local_file.database (LOW)"
}
