#!/usr/bin/env bash
# =============================================================================
# tfui test suite (BDD style)
# =============================================================================
# Run: bash tests/tfui-test.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/../lib/tfui.sh"

_PASS=0
_FAIL=0

# -- Assertions ---------------------------------------------------------------

_assert_equals() {
  local description="$1"
  local expected="$2"
  local actual="$3"

  if [ "$expected" = "$actual" ]; then
    echo "    PASS: $description"
    _PASS=$(( _PASS + 1 ))
  else
    echo "    FAIL: $description"
    echo "      expected: '$expected'"
    echo "      actual:   '$actual'"
    _FAIL=$(( _FAIL + 1 ))
  fi
}

_assert_exit_code() {
  local description="$1"
  local expected="$2"
  local actual="$3"

  if [ "$expected" = "$actual" ]; then
    echo "    PASS: $description"
    _PASS=$(( _PASS + 1 ))
  else
    echo "    FAIL: $description"
    echo "      expected exit: $expected"
    echo "      actual exit:   $actual"
    _FAIL=$(( _FAIL + 1 ))
  fi
}

_assert_contains() {
  local description="$1"
  local needle="$2"
  local haystack="$3"

  if echo "$haystack" | grep -qF "$needle"; then
    echo "    PASS: $description"
    _PASS=$(( _PASS + 1 ))
  else
    echo "    FAIL: $description"
    echo "      expected to contain: '$needle'"
    echo "      actual: '$haystack'"
    _FAIL=$(( _FAIL + 1 ))
  fi
}

_test_summary() {
  echo ""
  echo "Results: $_PASS passed, $_FAIL failed"
  if [ $_FAIL -gt 0 ]; then
    exit 1
  fi
}

# -- Hooks --------------------------------------------------------------------

_setup() {
  exec 3>/dev/null
  _TFUI_UI_LINES="10"
  _TFUI_MESSAGE=""
  _TFUI_START_TIME=0
  _TFUI_ANIMATION_PID=""
}

_cleanup() {
  rm -f "$_TFUI_OUTPUT_FILE" 2>/dev/null
  exec 3>/dev/null
}

# -- Mocks --------------------------------------------------------------------

_MOCK_DIR=""

_mock_terraform_setup() {
  _MOCK_DIR=$(mktemp -d)

  cat > "$_MOCK_DIR/terraform" <<'MOCK'
#!/usr/bin/env bash
case "$1" in
  plan)
    echo "module.a.resource_b: Refreshing state..."
    echo "module.a.resource_a: Refreshing state..."
    echo "module.b.resource_c: Reading..."
    for arg in "$@"; do
      if [[ "$arg" == -out=* ]]; then
        touch "${arg#-out=}"
      fi
    done
    ;;
  show)
    cat <<'JSON'
{"resource_changes":[
  {"address":"module.a.resource_b","change":{"actions":["create"]}},
  {"address":"module.a.resource_a","change":{"actions":["update"]}},
  {"address":"module.b.resource_c","change":{"actions":["delete"]}}
]}
JSON
    ;;
  apply)
    echo "module.a.resource_b: Creating..."
    echo "module.a.resource_b: Creation complete after 1s"
    echo "module.a.resource_a: Modifying..."
    echo "module.a.resource_a: Modifications complete after 1s"
    ;;
  state)
    echo "module.a.resource_a"
    echo "module.a.resource_b"
    echo "module.b.resource_c"
    ;;
esac
MOCK
  chmod +x "$_MOCK_DIR/terraform"
  export PATH="$_MOCK_DIR:$PATH"
}

_mock_terraform_teardown() {
  rm -rf "$_MOCK_DIR"
}

# -- Initial setup ------------------------------------------------------------

_setup

# =============================================================================
# Feature: State management
# =============================================================================

echo ""
echo "Feature: State management"

echo "  Scenario: Setting a message"
_tfui_state_set_message "hello"
_assert_equals "should store the message" "hello" "$_TFUI_MESSAGE"

echo "  Scenario: Setting a message with special characters"
_tfui_state_set_message "Planning module: sa-east-1"
_assert_equals "should handle complex strings" "Planning module: sa-east-1" "$_TFUI_MESSAGE"

echo "  Scenario: Resetting the timer"
_tfui_state_reset_timer
_assert_equals "should set start time to now" "$SECONDS" "$_TFUI_START_TIME"

echo "  Scenario: Clearing the output file"
_TFUI_OUTPUT_FILE=$(mktemp)
echo "some content" > "$_TFUI_OUTPUT_FILE"
_tfui_state_clear_output
_assert_equals "should truncate to 0 bytes" "0" "$(wc -c < "$_TFUI_OUTPUT_FILE")"
rm -f "$_TFUI_OUTPUT_FILE"

# =============================================================================
# Feature: Line management (bitmask)
# =============================================================================

echo ""
echo "Feature: Line management (bitmask)"

echo "  Scenario: Default line state"
_TFUI_UI_LINES="10"
_assert_equals "should have header=1" "1" "${_TFUI_UI_LINES:0:1}"
_assert_equals "should have status=0" "0" "${_TFUI_UI_LINES:1:1}"

echo "  Scenario: Enabling a line"
_TFUI_UI_LINES="10"
_tfui_ui_enable_line $_TFUI_LINE_STATUS
_assert_equals "should set status bit to 1" "11" "$_TFUI_UI_LINES"

echo "  Scenario: Checking if a line is enabled"
_TFUI_UI_LINES="11"
_tfui_ui_is_line_enabled $_TFUI_LINE_STATUS
_assert_exit_code "should return 0 (true)" "0" "$?"

echo "  Scenario: Disabling a line"
_TFUI_UI_LINES="11"
_tfui_ui_disable_line $_TFUI_LINE_STATUS
_assert_equals "should set status bit to 0" "10" "$_TFUI_UI_LINES"

echo "  Scenario: Checking if a disabled line is enabled"
_TFUI_UI_LINES="10"
rc=0; _tfui_ui_is_line_enabled $_TFUI_LINE_STATUS || rc=$?
_assert_exit_code "should return 1 (false)" "1" "$rc"

# =============================================================================
# Feature: Strategy selection
# =============================================================================

echo ""
echo "Feature: Strategy selection"

echo "  Scenario: Plain mode"
_tfui_choose_strategy "plain"
_assert_equals "should select silent" "_tfui_strategy_silent" "$_TFUI_STRATEGY"

echo "  Scenario: Simple mode"
_tfui_choose_strategy "simple"
_assert_equals "should select spinner" "_tfui_strategy_spinner" "$_TFUI_STRATEGY"

echo "  Scenario: Rich mode"
_tfui_choose_strategy "rich"
_assert_equals "should select progress" "_tfui_strategy_progress" "$_TFUI_STRATEGY"

echo "  Scenario: Auto mode with tty"
_tfui_choose_strategy "auto"
_assert_equals "should select progress (tty present)" "_tfui_strategy_progress" "$_TFUI_STRATEGY"

# =============================================================================
# Feature: Strategy resolution
# =============================================================================

echo ""
echo "Feature: Strategy resolution"

echo "  Scenario: Patterns provided with progress strategy"
_TFUI_STRATEGY="_tfui_strategy_progress"
_tfui_resolve_strategy "some pattern"
_assert_equals "should keep progress" "_tfui_strategy_progress" "$_TFUI_RESOLVED_STRATEGY"

echo "  Scenario: Empty patterns with progress strategy"
_TFUI_STRATEGY="_tfui_strategy_progress"
_tfui_resolve_strategy ""
_assert_equals "should downgrade to spinner" "_tfui_strategy_spinner" "$_TFUI_RESOLVED_STRATEGY"

echo "  Scenario: Empty patterns with spinner strategy"
_TFUI_STRATEGY="_tfui_strategy_spinner"
_tfui_resolve_strategy ""
_assert_equals "should keep spinner" "_tfui_strategy_spinner" "$_TFUI_RESOLVED_STRATEGY"

echo "  Scenario: Empty patterns with silent strategy"
_TFUI_STRATEGY="_tfui_strategy_silent"
_tfui_resolve_strategy ""
_assert_equals "should keep silent" "_tfui_strategy_silent" "$_TFUI_RESOLVED_STRATEGY"

# =============================================================================
# Feature: Format header
# =============================================================================

echo ""
echo "Feature: Format header"

echo "  Scenario: First frame with zero elapsed"
result=$(_tfui_ui_format_header 0 "Planning" 0)
_assert_equals "should format first frame" "⠋ Planning (0s)" "$result"

echo "  Scenario: Third frame with elapsed time"
result=$(_tfui_ui_format_header 2 "Applying" 42)
_assert_equals "should format third frame" "⠹ Applying (42s)" "$result"

echo "  Scenario: Message with special characters"
result=$(_tfui_ui_format_header 0 "Planning module: sa-east-1" 7)
_assert_equals "should handle special chars" "⠋ Planning module: sa-east-1 (7s)" "$result"

echo "  Scenario: Large elapsed time"
result=$(_tfui_ui_format_header 5 "Waiting" 999)
_assert_equals "should handle large numbers" "⠴ Waiting (999s)" "$result"

# =============================================================================
# Feature: Format progress
# =============================================================================

echo ""
echo "Feature: Format progress"

echo "  Scenario: Zero progress"
result=$(_tfui_ui_format_progress 0 100)
_assert_equals "should format zero progress" "  Progress: 0/100 [░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░] 0%" "$result"

echo "  Scenario: Full progress"
result=$(_tfui_ui_format_progress 50 50)
_assert_equals "should format full progress" "  Progress: 50/50 [██████████████████████████████] 100%" "$result"

echo "  Scenario: Partial progress"
result=$(_tfui_ui_format_progress 25 100)
_assert_equals "should format partial progress" "  Progress: 25/100 [███████░░░░░░░░░░░░░░░░░░░░░░░] 25%" "$result"

echo "  Scenario: Zero total"
result=$(_tfui_ui_format_progress 0 0)
_assert_equals "should handle zero total" "  Progress: 0/0 [░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░] 0%" "$result"

# =============================================================================
# Feature: Format status
# =============================================================================

echo ""
echo "Feature: Format status"

echo "  Scenario: Label with dots"
result=$(_tfui_ui_format_status "Calculating" "...")
_assert_equals "should format label and dots" "  Calculating..." "$result"

echo "  Scenario: Empty dots"
result=$(_tfui_ui_format_status "Rendering" "")
_assert_equals "should format label without dots" "  Rendering" "$result"

echo "  Scenario: Label with max dots"
result=$(_tfui_ui_format_status "Working" ".....")
_assert_equals "should format max dots" "  Working....." "$result"

# =============================================================================
# Feature: Progress bar building
# =============================================================================

echo ""
echo "Feature: Progress bar building"

echo "  Scenario: Empty progress bar"
bar=$(_tfui_ui_build_bar 0 100)
_assert_equals "should show 30 empty chars" "░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░" "$bar"

echo "  Scenario: Full progress bar"
bar=$(_tfui_ui_build_bar 100 100)
_assert_equals "should show 30 filled chars" "██████████████████████████████" "$bar"

echo "  Scenario: Half progress bar"
bar=$(_tfui_ui_build_bar 15 30)
_assert_equals "should show 15 filled + 15 empty" "███████████████░░░░░░░░░░░░░░░" "$bar"

echo "  Scenario: Zero total"
bar=$(_tfui_ui_build_bar 0 0)
_assert_equals "should handle 0/0 gracefully" "░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░" "$bar"

# =============================================================================
# Feature: Percentage calculation
# =============================================================================

echo ""
echo "Feature: Percentage calculation"

_assert_equals "0/100 should be 0%" "0" "$(_tfui_ui_calc_percent 0 100)"
_assert_equals "50/100 should be 50%" "50" "$(_tfui_ui_calc_percent 50 100)"
_assert_equals "100/100 should be 100%" "100" "$(_tfui_ui_calc_percent 100 100)"
_assert_equals "0/0 should be 0%" "0" "$(_tfui_ui_calc_percent 0 0)"
_assert_equals "33/100 should be 33%" "33" "$(_tfui_ui_calc_percent 33 100)"

# =============================================================================
# Feature: Dots animation advancement
# =============================================================================

echo ""
echo "Feature: Dots animation advancement"

echo "  Scenario: Tick not on interval"
result=$(_tfui_ui_advance_dots "." 3)
_assert_equals "should not advance" "." "$result"

echo "  Scenario: Tick on interval"
result=$(_tfui_ui_advance_dots "." 8)
_assert_equals "should add a dot" ".." "$result"

echo "  Scenario: Tick on interval with empty dots"
result=$(_tfui_ui_advance_dots "" 16)
_assert_equals "should start with one dot" "." "$result"

echo "  Scenario: Dots at max length resets"
result=$(_tfui_ui_advance_dots "....." 24)
_assert_equals "should reset after max" "." "$result"

echo "  Scenario: Dots just below max"
result=$(_tfui_ui_advance_dots "...." 32)
_assert_equals "should grow to max" "....." "$result"

echo "  Scenario: Non-interval tick preserves state"
result=$(_tfui_ui_advance_dots "..." 7)
_assert_equals "should preserve existing dots" "..." "$result"

# =============================================================================
# Feature: Progress line pattern matching
# =============================================================================

echo ""
echo "Feature: Progress line pattern matching"

echo "  Scenario: Line matches pattern"
rc=0; _tfui_progress_line_matches "module.a.resource_b: Refreshing state..." ": Refreshing state\.\.\." || rc=$?
_assert_exit_code "should match refresh line" "0" "$rc"

echo "  Scenario: Line does not match pattern"
rc=0; _tfui_progress_line_matches "Terraform initialized" ": Refreshing state\.\.\." || rc=$?
_assert_exit_code "should not match unrelated line" "1" "$rc"

echo "  Scenario: Multi-pattern matching first alternative"
rc=0; _tfui_progress_line_matches "module.x: Creating..." ": Creating\.\.\.|: Modifying\.\.\." || rc=$?
_assert_exit_code "should match first pattern" "0" "$rc"

echo "  Scenario: Multi-pattern matching second alternative"
rc=0; _tfui_progress_line_matches "module.x: Modifying..." ": Creating\.\.\.|: Modifying\.\.\." || rc=$?
_assert_exit_code "should match second pattern" "0" "$rc"

echo "  Scenario: Empty line does not match"
rc=0; _tfui_progress_line_matches "" ": Creating\.\.\." || rc=$?
_assert_exit_code "should not match empty line" "1" "$rc"

# =============================================================================
# Feature: Render functions write to fd3
# =============================================================================

echo ""
echo "Feature: Render functions write to fd3"

_render_output=$(mktemp)

echo "  Scenario: render_header writes formatted header to fd3"
exec 3>"$_render_output"
_tfui_ui_render_header 0 "Planning" 5
exec 3>/dev/null
_assert_contains "should contain spinner char" "⠋" "$(cat "$_render_output")"
_assert_contains "should contain message" "Planning" "$(cat "$_render_output")"
_assert_contains "should contain elapsed" "(5s)" "$(cat "$_render_output")"

echo "  Scenario: render_progress writes formatted bar to fd3"
exec 3>"$_render_output"
_tfui_ui_render_progress 10 100
exec 3>/dev/null
_assert_contains "should contain counts" "10/100" "$(cat "$_render_output")"
_assert_contains "should contain percentage" "10%" "$(cat "$_render_output")"

echo "  Scenario: render_status writes label and dots to fd3"
exec 3>"$_render_output"
_tfui_ui_render_status "Calculating" ".."
exec 3>/dev/null
_assert_contains "should contain label" "Calculating" "$(cat "$_render_output")"
_assert_contains "should contain dots" ".." "$(cat "$_render_output")"

rm -f "$_render_output"

# =============================================================================
# Feature: UI open/close lifecycle
# =============================================================================

echo ""
echo "Feature: UI open/close lifecycle"

_ui_output=$(mktemp)

echo "  Scenario: open renders initial header"
exec 3>"$_ui_output"
_TFUI_UI_LINES="10"
_TFUI_MESSAGE="Testing"
_tfui_ui_open
exec 3>/dev/null
_assert_contains "should contain message in open" "Testing" "$(cat "$_ui_output")"
_assert_contains "should contain 0s elapsed" "(0s)" "$(cat "$_ui_output")"

echo "  Scenario: close clears lines"
exec 3>"$_ui_output"
_TFUI_UI_LINES="10"
_tfui_ui_close
exec 3>/dev/null
output=$(cat "$_ui_output")
_assert_contains "should contain show cursor" $'\033[?25h' "$output"

rm -f "$_ui_output"

# =============================================================================
# Feature: Plan tree rendering
# =============================================================================

echo ""
echo "Feature: Plan tree rendering"

plan_file=$(mktemp)

echo "  Scenario: No changes"
echo '{"resource_changes":[{"address":"a","change":{"actions":["no-op"]}}]}' > "$plan_file"
output=$(_tfui_render_plan_tree "$plan_file")
_assert_equals "should show no-changes message" "No changes. Infrastructure is up-to-date." "$output"

echo "  Scenario: Mixed actions (create, update, delete)"
cat > "$plan_file" <<'JSON'
{"resource_changes":[
  {"address":"module.a.resource_b","change":{"actions":["create"]}},
  {"address":"module.a.resource_a","change":{"actions":["update"]}},
  {"address":"module.b.resource_c","change":{"actions":["delete"]}},
  {"address":"data.source","change":{"actions":["read"]}}
]}
JSON
output=$(_tfui_render_plan_tree "$plan_file")
expected="+ module.a.resource_b
- module.b.resource_c
~ module.a.resource_a

Plan: 1 to add, 1 to change, 1 to destroy."
_assert_equals "should render correct tree" "$expected" "$output"

echo "  Scenario: Replace action"
echo '{"resource_changes":[{"address":"module.x.y","change":{"actions":["delete","create"]}}]}' > "$plan_file"
output=$(_tfui_render_plan_tree "$plan_file")
_assert_equals "should show replace icon" "-/+ module.x.y" "$(echo "$output" | head -1)"

rm -f "$plan_file"

# =============================================================================
# Feature: Change detection
# =============================================================================

echo ""
echo "Feature: Change detection"

plan_file=$(mktemp)

echo "  Scenario: Plan with only no-op resources"
echo '{"resource_changes":[{"address":"a","change":{"actions":["no-op"]}}]}' > "$plan_file"
rc=0; tfui_confirm "$plan_file" --auto-approve || rc=$?
_assert_exit_code "should detect no changes" "1" "$rc"

echo "  Scenario: Plan with a create action"
echo '{"resource_changes":[{"address":"a","change":{"actions":["create"]}}]}' > "$plan_file"
tfui_confirm "$plan_file" --auto-approve
_assert_exit_code "should detect changes" "0" "$?"

echo "  Scenario: Plan with only read actions"
echo '{"resource_changes":[{"address":"a","change":{"actions":["read"]}}]}' > "$plan_file"
rc=0; tfui_confirm "$plan_file" --auto-approve || rc=$?
_assert_exit_code "should ignore read-only" "1" "$rc"

rm -f "$plan_file"

# =============================================================================
# Feature: Command execution
# =============================================================================

echo ""
echo "Feature: Command execution"

_TFUI_WORKING_DIR=$(mktemp -d)
_TFUI_OUTPUT_FILE=$(mktemp)

echo "  Scenario: Capturing stdout"
_tfui_exec "echo hello"
_assert_equals "should capture stdout" "hello" "$(cat "$_TFUI_OUTPUT_FILE")"

echo "  Scenario: Capturing stderr"
_tfui_exec "echo err >&2"
_assert_equals "should capture stderr" "err" "$(cat "$_TFUI_OUTPUT_FILE")"

echo "  Scenario: Propagating exit code"
exit_code=0
_tfui_exec "exit 42" || exit_code=$?
_assert_exit_code "should propagate exit code" "42" "$exit_code"

echo "  Scenario: Running in working directory"
touch "$_TFUI_WORKING_DIR/marker.txt"
_tfui_exec "ls marker.txt"
_assert_equals "should run in working dir" "marker.txt" "$(cat "$_TFUI_OUTPUT_FILE")"

echo "  Scenario: Silent strategy delegates to exec"
exit_code=0
_tfui_strategy_silent "" "echo delegated"
_assert_equals "should capture via exec" "delegated" "$(cat "$_TFUI_OUTPUT_FILE")"

rm -f "$_TFUI_OUTPUT_FILE"
rm -rf "$_TFUI_WORKING_DIR"

# =============================================================================
# Feature: Rendering preserves original message
# =============================================================================

echo ""
echo "Feature: Rendering preserves original message"

_mock_terraform_setup
_TFUI_WORKING_DIR=$(mktemp -d)
_TFUI_OUTPUT_FILE=$(mktemp)
plan_file=$(mktemp)
_TFUI_STRATEGY="_tfui_strategy_silent"
_TFUI_UI_LINES="10"

echo "  Scenario: After plan completes, rendering preserves original message"
tfui_plan "Planning module: sa-east-1" --out "$plan_file" > /dev/null 2>&1
_assert_equals "message should be preserved from plan step" "Planning module: sa-east-1" "$_TFUI_MESSAGE"

_mock_terraform_teardown
rm -f "$_TFUI_OUTPUT_FILE" "$plan_file"
rm -rf "$_TFUI_WORKING_DIR"

# =============================================================================
# Feature: _tfui_run uses spinner for empty patterns
# =============================================================================

echo ""
echo "Feature: _tfui_run uses spinner for empty patterns"

_TFUI_WORKING_DIR=$(mktemp -d)
_TFUI_OUTPUT_FILE=$(mktemp)

echo "  Scenario: Progress strategy with empty patterns"
_TFUI_STRATEGY="_tfui_strategy_progress"
_TFUI_UI_LINES="10"

set +e
_tfui_run "Rendering" "" "echo done" 2>/dev/null
set -e

_assert_equals "should capture command output" "done" "$(cat "$_TFUI_OUTPUT_FILE")"
_assert_equals "status line should be disabled after" "0" "${_TFUI_UI_LINES:$_TFUI_LINE_STATUS:1}"

rm -f "$_TFUI_OUTPUT_FILE"
rm -rf "$_TFUI_WORKING_DIR"

# =============================================================================
# Feature: Full plan flow (mocked terraform)
# =============================================================================

echo ""
echo "Feature: Full plan flow (mocked terraform)"

_mock_terraform_setup
_TFUI_WORKING_DIR=$(mktemp -d)
_TFUI_OUTPUT_FILE=$(mktemp)
plan_file=$(mktemp)
_TFUI_STRATEGY="_tfui_strategy_silent"
_TFUI_UI_LINES="10"

echo "  Scenario: tfui_plan produces tree view"
output=$(tfui_plan "Planning" --out "$plan_file" 2>/dev/null)
_assert_equals "should show create" "+ module.a.resource_b" "$(echo "$output" | grep "^+")"
_assert_equals "should show update" "~ module.a.resource_a" "$(echo "$output" | grep "^~")"
_assert_equals "should show delete" "- module.b.resource_c" "$(echo "$output" | grep "^-")"
_assert_equals "should show summary" "Plan: 1 to add, 1 to change, 1 to destroy." "$(echo "$output" | grep "^Plan:")"

echo "  Scenario: tfui_confirm detects changes"
tfui_confirm "$plan_file" --auto-approve
_assert_exit_code "should detect changes" "0" "$?"

rm -f "$_TFUI_OUTPUT_FILE" "$plan_file" "$_TFUI_WORKING_DIR/tfplan.out"
rm -rf "$_TFUI_WORKING_DIR"
_mock_terraform_teardown

# =============================================================================
# Feature: Full plan flow with progress strategy (mocked terraform)
# =============================================================================

echo ""
echo "Feature: Full plan flow with progress strategy (mocked terraform)"

_mock_terraform_setup
_TFUI_WORKING_DIR=$(mktemp -d)
_TFUI_OUTPUT_FILE=$(mktemp)
plan_file=$(mktemp)
_TFUI_STRATEGY="_tfui_strategy_progress"
_TFUI_UI_LINES="10"

echo "  Scenario: Progress strategy tracks resource refresh lines"
output=$(tfui_plan "Planning" --out "$plan_file" 2>/dev/null)
_assert_equals "should show summary" "Plan: 1 to add, 1 to change, 1 to destroy." "$(echo "$output" | grep "^Plan:")"
_assert_equals "status line should be disabled" "0" "${_TFUI_UI_LINES:$_TFUI_LINE_STATUS:1}"

rm -f "$_TFUI_OUTPUT_FILE" "$plan_file" "$_TFUI_WORKING_DIR/tfplan.out"
rm -rf "$_TFUI_WORKING_DIR"
_mock_terraform_teardown

# =============================================================================
# Feature: _tfui_run_sub error handling
# =============================================================================

echo ""
echo "Feature: _tfui_run_sub error handling"

echo "  Scenario: Sub-phase command fails"
exit_code=0
(
  set -e
  source "$SCRIPT_DIR/tfui.sh"
  exec 3>/dev/null
  _TFUI_WORKING_DIR=$(mktemp -d)
  _TFUI_OUTPUT_FILE=$(mktemp)
  _TFUI_UI_LINES="10"
  _TFUI_MESSAGE="Test"
  _TFUI_START_TIME=$SECONDS
  _tfui_run_sub "Failing" "exit 99"
) 2>/dev/null || exit_code=$?

_assert_exit_code "should exit with 1 on sub-phase failure" "1" "$exit_code"

# =============================================================================
# Feature: Full apply flow (mocked terraform)
# =============================================================================

echo ""
echo "Feature: Full apply flow (mocked terraform)"

_mock_terraform_setup
_TFUI_WORKING_DIR=$(mktemp -d)
_TFUI_OUTPUT_FILE=$(mktemp)
plan_file=$(mktemp)
_TFUI_STRATEGY="_tfui_strategy_silent"
_TFUI_UI_LINES="10"

echo "  Scenario: tfui_apply completes successfully"
touch "$_TFUI_WORKING_DIR/tfplan.out"
tfui_apply "$plan_file" "Applying" 2>/dev/null
_assert_equals "should capture apply output" "module.a.resource_b: Creating..." "$(head -1 "$_TFUI_OUTPUT_FILE")"

_mock_terraform_teardown
rm -f "$_TFUI_OUTPUT_FILE" "$plan_file"
rm -rf "$_TFUI_WORKING_DIR"

# =============================================================================

_test_summary
