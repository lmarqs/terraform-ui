#!/usr/bin/env bash
# =============================================================================
# tfui — Terraform UI library
# =============================================================================
#
# Provides animated terminal feedback for terraform plan/apply operations:
# spinner, elapsed timer, progress bar, and tree-view diff output.
#
# Designed to be sourced by any terraform task runner, independent of
# project structure or module organization.
#
# -- Public API ---------------------------------------------------------------
#
#   tfui_init <working_dir> [ui_mode]
#   tfui_plan <message> [extra_args] --out <plan_file>
#   tfui_confirm <plan_file> [--auto-approve]
#   tfui_apply <plan_file> <message> [extra_args]
#
# -- Usage example -------------------------------------------------------------
#
#   plan_file=$(mktemp)
#   tfui_init "$MODULE_DIR" "$ui_mode"
#   tfui_plan "Planning module: $MODULE" "$TARGET" --out "$plan_file"
#   if tfui_confirm "$plan_file" --auto-approve; then
#     tfui_apply "$plan_file" "Applying module: $MODULE" "$TARGET"
#   fi
#
# -- Environment variables (set by caller) ------------------------------------
#
#   TF_CLI_ARGS_plan   — passed to `terraform plan` (native terraform env var)
#   TF_CLI_ARGS_apply  — passed to `terraform apply` (native terraform env var)
#
# -- File descriptors ---------------------------------------------------------
#
#   &1 (stdout)  — final output (tree view)
#   &2 (stderr)  — error messages
#   &3           — terminal UI (animations, progress bar)
#
# -- Naming conventions -------------------------------------------------------
#
#   Functions:
#     tfui_*                — public API (called by task scripts)
#     _tfui_run*            — orchestration (delegates to strategy)
#     _tfui_state_*         — state mutators (set message, timer, etc.)
#     _tfui_lifecycle_*     — process lifecycle (die, on_exit)
#     _tfui_ui_render_*     — render content at current cursor position (fd3)
#     _tfui_ui_format_*     — pure formatters (return string via stdout)
#     _tfui_ui_*            — layout and animation control (cursor, open, close)
#     _tfui_strategy_*      — execution strategies (silent, spinner, progress)
#     _tfui_render_*        — output formatting (tree view)
#
#   Variables:
#     _TFUI_*             — internal state (all prefixed to avoid collisions)
#
# -- Strategies ---------------------------------------------------------------
#
#   silent   — no UI, captures output silently (plain mode)
#   spinner  — one-line animated spinner with elapsed time (simple mode)
#   progress — two-line UI: spinner + progress bar tracking resource count (rich mode)
#
# -- Terminal layout ----------------------------------------------------------
#
#   Line 0 (HEADER):    ⠋ <message> (<elapsed>s)
#   Line 1 (STATUS):      Progress: 47/120 [████████████░░░░░░░░] 39%
#                       — or —
#                         Calculating...
#
# -- Rendering architecture ---------------------------------------------------
#
#   Render functions write content to fd3 at the current cursor position.
#   They never move the cursor — that responsibility belongs to the caller
#   (animation loop or strategy). This separation allows render functions
#   to be tested by redirecting fd3 to a file.
#
# =============================================================================

# -- Terminal constants -------------------------------------------------------

_TFUI_ESC_HIDE_CURSOR="\033[?25l"
_TFUI_ESC_SHOW_CURSOR="\033[?25h"
_TFUI_ESC_MOVE_UP="\033[A"
_TFUI_ESC_ERASE_REST="\033[K"
_TFUI_ESC_CLEAR_LINE="\r\033[K"
_TFUI_SPINNER='⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏'

# -- Animation constants -------------------------------------------------------

_TFUI_DOTS_TICK_INTERVAL=8
_TFUI_DOTS_MAX_LENGTH=5
_TFUI_BAR_WIDTH=30

# -- Line identifiers (bitmask positions) ------------------------------------

_TFUI_LINE_HEADER=0
_TFUI_LINE_STATUS=1

# -- State --------------------------------------------------------------------

_TFUI_WORKING_DIR=""
_TFUI_OUTPUT_FILE=""
_TFUI_MESSAGE=""
_TFUI_START_TIME=0
_TFUI_STRATEGY=""
_TFUI_ANIMATION_PID=""
_TFUI_UI_LINES="10"

# -- Public -------------------------------------------------------------------

# @description Initialize working directory, temp files, terminal fd, and choose strategy.
# @param $1 {string} working_dir - Directory where terraform commands run
# @param $2 {string} ui_mode - Output mode: auto, rich, simple, plain
# @return void
# @side-effect _TFUI_*, fd3, EXIT trap
tfui_init() {
  local working_dir="$1"
  local ui_mode="${2:-auto}"

  _TFUI_WORKING_DIR="$working_dir"
  _TFUI_OUTPUT_FILE=$(mktemp)
  trap "_tfui_lifecycle_on_exit" EXIT

  _tfui_open_ui_channel
  _tfui_choose_strategy "$ui_mode"
}

# @description Plan, convert to JSON, render tree view.
# @param $1 {string} message - Status message for the spinner
# @param $2 {string} [extra_args] - Additional arguments for terraform plan
# @param --out {string} plan_file - Path to write the plan JSON
# @return stdout - Tree view of planned changes
# @side-effect fd3 (UI), writes plan_file
# @requires tfui_init, TF_CLI_ARGS_plan
tfui_plan() {
  local message="" extra_args="" plan_file=""

  while [ $# -gt 0 ]; do
    case "$1" in
      --out) plan_file="$2"; shift 2 ;;
      *)     if [ -z "$message" ]; then message="$1"; else extra_args="$1"; fi; shift ;;
    esac
  done

  if [ -z "$plan_file" ]; then
    echo "tfui_plan: --out is required" >&2; return 1
  fi

  _tfui_run "$message" ": Refreshing state\.\.\.|: Reading\.\.\." "terraform plan -out=tfplan.out $extra_args"
  _tfui_run_sub "Rendering" "terraform show -json tfplan.out > '$plan_file'"

  if [ "$_TFUI_STRATEGY" = "_tfui_strategy_agent" ]; then
    _tfui_render_plan_json "$plan_file"
  else
    _tfui_render_plan_tree "$plan_file"
  fi
}

# @description Check if plan has changes and optionally prompt user for confirmation.
# @param $1 {string} plan_file - Path to the plan JSON file
# @param --auto-approve {flag} - Skip user prompt, just check for changes
# @return exit_code - 0 if confirmed (or auto-approved with changes), 1 otherwise
# @requires tfui_plan
tfui_confirm() {
  local plan_file="$1"
  local auto_approve=false

  local i; for i in "$@"; do
    case "$i" in --auto-approve) auto_approve=true ;; esac
  done

  local query='[.resource_changes // [] | .[] | select(.change.actions != ["no-op"] and .change.actions != ["read"])] | length > 0'
  if [ "$(jq "$query" "$plan_file")" != "true" ]; then
    return 1
  fi

  if [ "$auto_approve" = "true" ]; then
    return 0
  fi

  echo ""
  local confirm
  read -p "Do you want to apply these changes? (yes/no): " confirm
  [ "$confirm" = "yes" ]
}

# @description Apply the saved plan file.
# @param $1 {string} plan_file - Path to the plan JSON file (plan dir contains tfplan.out)
# @param $2 {string} message - Status message for the spinner
# @param $3 {string} [extra_args] - Additional arguments for terraform apply
# @return void
# @side-effect fd3 (UI), terraform state
# @requires tfui_init, tfui_plan, TF_CLI_ARGS_apply
tfui_apply() {
  local plan_file="$1"
  local message="$2"
  local extra_args="${3:-}"
  _tfui_run "$message" ": Creating\.\.\.|: Modifying\.\.\.|: Destroying\.\.\.|: Creation complete|: Modifications complete|: Destruction complete" "terraform apply tfplan.out $extra_args"
}

# -- Init helpers -------------------------------------------------------------

# @description Open fd 3 for UI output, targeting the terminal regardless of stdout redirection.
# @return void
# @side-effect fd3
_tfui_open_ui_channel() {
  if (printf '' >&3) 2>/dev/null; then
    return
  fi

  if [ -t 1 ]; then
    exec 3>&1
  elif (echo "" > /dev/tty) 2>/dev/null; then
    exec 3>/dev/tty
  else
    exec 3>/dev/null
  fi
}

# @description Select execution strategy based on UI mode.
# @param $1 {string} mode - UI mode: auto, rich, simple, plain
# @return void
# @side-effect _TFUI_STRATEGY
_tfui_choose_strategy() {
  local mode="$1"

  if [ "$mode" = "auto" ]; then
    if [ -e /dev/tty ]; then
      mode="rich"
    else
      mode="plain"
    fi
  fi

  case "$mode" in
    rich)    _TFUI_STRATEGY="_tfui_strategy_progress" ;;
    simple)  _TFUI_STRATEGY="_tfui_strategy_spinner" ;;
    plain)   _TFUI_STRATEGY="_tfui_strategy_silent" ;;
    agent)   _TFUI_STRATEGY="_tfui_strategy_agent" ;;
  esac
}

# -- Execution ----------------------------------------------------------------

# @description Execute a command in the working directory, capturing output to file.
# @param $1 {string} command - Shell command to evaluate
# @return exit_code - From the executed command
# @side-effect writes _TFUI_OUTPUT_FILE
# @requires _TFUI_WORKING_DIR, _TFUI_OUTPUT_FILE
_tfui_exec() {
  local command="$1"

  local exit_code=0
  (cd "$_TFUI_WORKING_DIR"; eval "$command") > "$_TFUI_OUTPUT_FILE" 2>&1 || exit_code=$?
  return $exit_code
}

# -- Run (orchestration) ------------------------------------------------------

# @description Prepare context and delegate to the active strategy.
# @param $1 {string} message - Status message for the spinner
# @param $2 {regex} patterns - Grep patterns to match for progress counting
# @param $3 {string} command - Shell command to evaluate
# @return void
# @side-effect _TFUI_*, fd3, files
# @requires tfui_init
_tfui_run() {
  _tfui_state_set_message "$1"
  _tfui_state_reset_timer
  _tfui_state_clear_output

  _tfui_resolve_strategy "$2"

  local exit_code=0
  set +e
  $_TFUI_RESOLVED_STRATEGY "$2" "$3"
  exit_code=$?
  set -e

  if [ $exit_code -ne 0 ]; then
    _tfui_lifecycle_die
  fi
}

# @description Resolve which strategy to use based on patterns.
# @param $1 {string} patterns - Empty patterns downgrade progress to spinner
# @return void
# @side-effect _TFUI_RESOLVED_STRATEGY
_tfui_resolve_strategy() {
  local patterns="$1"

  if [ -z "$patterns" ] && [ "$_TFUI_STRATEGY" = "_tfui_strategy_progress" ]; then
    _TFUI_RESOLVED_STRATEGY="_tfui_strategy_spinner"
  else
    _TFUI_RESOLVED_STRATEGY="$_TFUI_STRATEGY"
  fi
}

# @description Run a sub-phase command, keeping the original header message.
# @param $1 {string} label - Status text shown as animated dots on line 2
# @param $2 {string} command - Shell command to evaluate
# @return void
# @side-effect fd3, _TFUI_OUTPUT_FILE
# @requires _tfui_run (preserves _TFUI_MESSAGE from prior run)
_tfui_run_sub() {
  local label="$1"
  local command="$2"

  _tfui_state_clear_output

  if [ "$_TFUI_STRATEGY" = "_tfui_strategy_agent" ] || [ "$_TFUI_STRATEGY" = "_tfui_strategy_silent" ]; then
    local exit_code=0
    _tfui_exec "$command" || exit_code=$?
    if [ $exit_code -ne 0 ]; then
      _tfui_lifecycle_die
    fi
    return
  fi

  _tfui_ui_enable_line $_TFUI_LINE_STATUS
  _tfui_ui_open
  _tfui_ui_animate "$label"

  local exit_code=0
  _tfui_exec "$command" || exit_code=$?

  _tfui_ui_stop_animation
  _tfui_ui_close
  _tfui_ui_disable_line $_TFUI_LINE_STATUS

  if [ $exit_code -ne 0 ]; then
    _tfui_lifecycle_die
  fi
}

# -- State mutators -----------------------------------------------------------

# @description Set the message displayed in the header line.
# @param $1 {string} message
# @return void
# @side-effect _TFUI_MESSAGE
_tfui_state_set_message() {
  _TFUI_MESSAGE="$1"
}

# @description Reset the elapsed time counter to now.
# @return void
# @side-effect _TFUI_START_TIME
_tfui_state_reset_timer() {
  _TFUI_START_TIME=$SECONDS
}

# @description Truncate the output capture file.
# @return void
# @side-effect writes _TFUI_OUTPUT_FILE
# @requires tfui_init
_tfui_state_clear_output() {
  : > "$_TFUI_OUTPUT_FILE"
}

# -- Line management (bitmask) ------------------------------------------------

# @description Enable a UI line by setting its bit to 1.
# @param $1 {int} position - Line identifier (_TFUI_LINE_HEADER or _TFUI_LINE_STATUS)
# @return void
# @side-effect _TFUI_UI_LINES
_tfui_ui_enable_line() {
  local position="$1"
  _TFUI_UI_LINES="${_TFUI_UI_LINES:0:$position}1${_TFUI_UI_LINES:$((position+1))}"
}

# @description Disable a UI line by setting its bit to 0.
# @param $1 {int} position - Line identifier (_TFUI_LINE_HEADER or _TFUI_LINE_STATUS)
# @return void
# @side-effect _TFUI_UI_LINES
_tfui_ui_disable_line() {
  local position="$1"
  _TFUI_UI_LINES="${_TFUI_UI_LINES:0:$position}0${_TFUI_UI_LINES:$((position+1))}"
}

# @description Check if a UI line is enabled.
# @param $1 {int} position - Line identifier
# @return exit_code - 0 if enabled, 1 if disabled
_tfui_ui_is_line_enabled() {
  local position="$1"
  [ "${_TFUI_UI_LINES:$position:1}" = "1" ]
}

# -- Lifecycle ----------------------------------------------------------------

# @description Stop UI, print captured error output to stderr, and exit.
# @return never (exits with code 1)
# @side-effect fd3, stderr, process exit
_tfui_lifecycle_die() {
  _tfui_ui_stop_animation
  _tfui_ui_close
  cat "$_TFUI_OUTPUT_FILE" >&2
  exit 1
}

# @description Trap handler: restore cursor and clean up temp files.
# @return void
# @side-effect fd3, removes temp files
_tfui_lifecycle_on_exit() {
  _tfui_ui_stop_animation
  printf "$_TFUI_ESC_SHOW_CURSOR" >&3 2>/dev/null
  rm -f "$_TFUI_OUTPUT_FILE" "$_TFUI_WORKING_DIR/tfplan.out"
}

# -- UI: pure formatters ------------------------------------------------------

# @description Format the header line text.
# @param $1 {int} frame_index - Index into the spinner character sequence
# @param $2 {string} message - Status message
# @param $3 {int} elapsed - Elapsed seconds
# @return stdout - Formatted header line (no escape sequences)
_tfui_ui_format_header() {
  local frame_index="$1"
  local message="$2"
  local elapsed="$3"

  printf "%s %s (%ds)" "${_TFUI_SPINNER:$frame_index:1}" "$message" "$elapsed"
}

# @description Format the progress bar text.
# @param $1 {int} current - Current count
# @param $2 {int} total - Total count
# @return stdout - Formatted progress line
_tfui_ui_format_progress() {
  local current="$1"
  local total="$2"
  local bar
  local percent

  bar=$(_tfui_ui_build_bar "$current" "$total")
  percent=$(_tfui_ui_calc_percent "$current" "$total")

  printf "  Progress: %d/%d [%s] %d%%" "$current" "$total" "$bar" "$percent"
}

# @description Format the status dots text.
# @param $1 {string} label - Status label
# @param $2 {string} dots - Current dots string
# @return stdout - Formatted status line
_tfui_ui_format_status() {
  local label="$1"
  local dots="$2"

  printf "  %s%s" "$label" "$dots"
}

# @description Build the visual bar string (█░).
# @param $1 {int} current
# @param $2 {int} total
# @return stdout - Bar string of fixed width
_tfui_ui_build_bar() {
  local current="$1"
  local total="$2"
  local filled=0

  if [ "$total" -gt 0 ]; then
    filled=$(( current * _TFUI_BAR_WIDTH / total ))
  fi

  local empty=$(( _TFUI_BAR_WIDTH - filled ))
  local bar=""
  for ((i=0; i<filled; i++)); do
    bar+="█"
  done
  for ((i=0; i<empty; i++)); do
    bar+="░"
  done
  echo "$bar"
}

# @description Calculate percentage (0-100).
# @param $1 {int} current
# @param $2 {int} total
# @return stdout - Percentage integer
_tfui_ui_calc_percent() {
  local current="$1"
  local total="$2"

  if [ "$total" -gt 0 ]; then
    echo $(( current * 100 / total ))
  else
    echo 0
  fi
}

# @description Advance the dots animation state.
# @param $1 {string} current_dots - Current dots string
# @param $2 {int} tick - Current tick counter
# @return stdout - New dots string (may be unchanged if tick isn't on interval)
_tfui_ui_advance_dots() {
  local current_dots="$1"
  local tick="$2"

  if [ $(( tick % _TFUI_DOTS_TICK_INTERVAL )) -eq 0 ]; then
    current_dots="${current_dots}."
    if [ ${#current_dots} -gt $_TFUI_DOTS_MAX_LENGTH ]; then
      current_dots="."
    fi
  fi
  printf "%s" "$current_dots"
}

# @description Check if a line matches the progress patterns.
# @param $1 {string} line - Line of output to check
# @param $2 {regex} patterns - Extended grep pattern
# @return exit_code - 0 if matches, 1 if not
_tfui_progress_line_matches() {
  local line="$1"
  local patterns="$2"

  echo "$line" | grep -qE "$patterns"
}

# -- UI: render (write content at current cursor position) --------------------

# @description Render the header line at the current cursor position.
# @param $1 {int} frame_index - Spinner frame index
# @param $2 {string} message - Status message
# @param $3 {int} elapsed - Elapsed seconds
# @return void
# @side-effect fd3
_tfui_ui_render_header() {
  local frame_index="$1"
  local message="$2"
  local elapsed="$3"

  printf "\r%s${_TFUI_ESC_ERASE_REST}" "$(_tfui_ui_format_header "$frame_index" "$message" "$elapsed")" >&3
}

# @description Render the progress bar at the current cursor position.
# @param $1 {int} current - Current count
# @param $2 {int} total - Total count
# @return void
# @side-effect fd3
_tfui_ui_render_progress() {
  local current="$1"
  local total="$2"

  printf "\r%s${_TFUI_ESC_ERASE_REST}" "$(_tfui_ui_format_progress "$current" "$total")" >&3
}

# @description Render the status dots at the current cursor position.
# @param $1 {string} label - Status label
# @param $2 {string} dots - Current dots string
# @return void
# @side-effect fd3
_tfui_ui_render_status() {
  local label="$1"
  local dots="$2"

  printf "\r%s${_TFUI_ESC_ERASE_REST}" "$(_tfui_ui_format_status "$label" "$dots")" >&3
}

# -- UI: layout (cursor management) ------------------------------------------

# @description Move cursor up to line 0 (header). No-op if status line is disabled.
# @return void
# @side-effect cursor position (fd3)
_tfui_ui_cursor_to_header() {
  if _tfui_ui_is_line_enabled $_TFUI_LINE_STATUS; then
    printf "$_TFUI_ESC_MOVE_UP" >&3
  fi
}

# @description Move cursor down to line 1 (status). No-op if status line is disabled.
# @return void
# @side-effect cursor position (fd3)
_tfui_ui_cursor_to_status() {
  if _tfui_ui_is_line_enabled $_TFUI_LINE_STATUS; then
    printf "\n" >&3
  fi
}

# @description Open the UI frame: hide cursor and render the initial state.
# @return void
# @side-effect fd3, cursor visibility
# @requires _TFUI_MESSAGE
_tfui_ui_open() {
  printf "${_TFUI_ESC_HIDE_CURSOR}" >&3
  _tfui_ui_render_header 0 "$_TFUI_MESSAGE" 0
  _tfui_ui_cursor_to_status
}

# @description Close the UI frame: clear all active lines and restore cursor.
# @return void
# @side-effect fd3, cursor visibility
_tfui_ui_close() {
  printf "$_TFUI_ESC_CLEAR_LINE" >&3
  if _tfui_ui_is_line_enabled $_TFUI_LINE_STATUS; then
    printf "${_TFUI_ESC_MOVE_UP}${_TFUI_ESC_CLEAR_LINE}" >&3
  fi
  printf "${_TFUI_ESC_SHOW_CURSOR}" >&3
}

# -- UI: animation ------------------------------------------------------------

# @description Start background animation loop.
# @param $1 {string} status_label - Label for status line (empty = no status rendering)
# @return void
# @side-effect _TFUI_ANIMATION_PID, fd3 (background writes)
# @requires _TFUI_MESSAGE, _TFUI_START_TIME
_tfui_ui_animate() {
  local status_label="$1"
  local dots=""
  local frame=0
  local tick=0

  while true; do
    frame=$(( (frame + 1) % ${#_TFUI_SPINNER} ))
    tick=$(( tick + 1 ))

    _tfui_ui_cursor_to_header
    _tfui_ui_render_header "$frame" "$_TFUI_MESSAGE" "$(( SECONDS - _TFUI_START_TIME ))"

    if [ -n "$status_label" ] && _tfui_ui_is_line_enabled $_TFUI_LINE_STATUS; then
      _tfui_ui_cursor_to_status
      dots=$(_tfui_ui_advance_dots "$dots" "$tick")
      _tfui_ui_render_status "$status_label" "$dots"
    else
      _tfui_ui_cursor_to_status
    fi

    sleep 0.1
  done &

  _TFUI_ANIMATION_PID=$!
}

# @description Stop the background animation process.
# @return void
# @side-effect kills _TFUI_ANIMATION_PID, resets to ""
_tfui_ui_stop_animation() {
  if [ -z "$_TFUI_ANIMATION_PID" ]; then
    return
  fi
  kill "$_TFUI_ANIMATION_PID" 2>/dev/null
  wait "$_TFUI_ANIMATION_PID" 2>/dev/null || true
  _TFUI_ANIMATION_PID=""
}

# -- Strategies ---------------------------------------------------------------

# @description Silent strategy: run command with no UI, capture output.
# @param $1 {regex} patterns - Unused (interface compatibility)
# @param $2 {string} command - Shell command to evaluate
# @return exit_code - From the executed command
# @side-effect writes _TFUI_OUTPUT_FILE
_tfui_strategy_silent() {
  local patterns="$1"
  local command="$2"

  _tfui_exec "$command"
}

# @description Spinner strategy: one-line animated spinner while command runs.
# @param $1 {regex} patterns - Unused (interface compatibility)
# @param $2 {string} command - Shell command to evaluate
# @return exit_code - From the executed command
# @side-effect fd3, _TFUI_OUTPUT_FILE, _TFUI_ANIMATION_PID
_tfui_strategy_spinner() {
  local patterns="$1"
  local command="$2"

  _tfui_ui_open
  _tfui_ui_animate ""

  local exit_code=0
  _tfui_exec "$command" || exit_code=$?

  _tfui_ui_stop_animation
  _tfui_ui_close
  return $exit_code
}

# @description Progress strategy: two-line UI with spinner + progress bar.
# @param $1 {regex} patterns - Grep patterns to count for progress
# @param $2 {string} command - Shell command to evaluate
# @return exit_code - From the executed command
# @side-effect fd3, _TFUI_OUTPUT_FILE, _TFUI_ANIMATION_PID, _TFUI_UI_LINES
_tfui_strategy_progress() {
  local patterns="$1"
  local command="$2"

  _tfui_ui_enable_line $_TFUI_LINE_STATUS

  # Phase 1: count total resources
  _tfui_ui_open
  _tfui_ui_animate "Calculating"

  local total
  total=$(cd "$_TFUI_WORKING_DIR"; terraform state list 2>/dev/null | wc -l)

  _tfui_ui_stop_animation

  # Phase 2: run command with progress tracking
  _tfui_ui_cursor_to_header
  _tfui_ui_render_header 0 "$_TFUI_MESSAGE" "$(( SECONDS - _TFUI_START_TIME ))"
  _tfui_ui_cursor_to_status
  _tfui_ui_render_progress 0 "$total"

  _tfui_ui_animate ""

  local current=0
  local exit_code=0

  set -o pipefail
  (cd "$_TFUI_WORKING_DIR"; eval "$command") 2>&1 | while IFS= read -r line; do
    if _tfui_progress_line_matches "$line" "$patterns"; then
      current=$(( current + 1 ))
      _tfui_ui_render_progress "$current" "$total"
    fi
    echo "$line" >> "$_TFUI_OUTPUT_FILE"
  done || exit_code=$?
  set +o pipefail

  _tfui_ui_stop_animation
  _tfui_ui_close
  _tfui_ui_disable_line $_TFUI_LINE_STATUS
  return $exit_code
}

# @description Agent strategy: run command silently, no UI output.
# @param $1 {regex} patterns - Unused (interface compatibility)
# @param $2 {string} command - Shell command to evaluate
# @return exit_code - From the executed command
# @side-effect writes _TFUI_OUTPUT_FILE
_tfui_strategy_agent() {
  local patterns="$1"
  local command="$2"

  _tfui_exec "$command"
}

# -- Renderer -----------------------------------------------------------------

# @description Parse plan JSON and print a tree view of changes.
# @param $1 {string} plan_file - Path to the plan JSON file
# @return stdout - Formatted tree view with +/~/- icons and summary
_tfui_render_plan_tree() {
  local plan_file="$1"
  jq -r '
    [.resource_changes // [] | .[] | select(.change.actions != ["no-op"] and .change.actions != ["read"])] |
    if length == 0 then
      "No changes. Infrastructure is up-to-date."
    else
      (map(
        (if .change.actions == ["create"] then "+"
         elif .change.actions == ["delete"] then "-"
         elif .change.actions == ["update"] then "~"
         elif (.change.actions | sort) == ["create", "delete"] then "-/+"
         else "?"
         end) + " " + .address
      ) | sort | join("\n")) + "\n\n" +
      "Plan: \(map(select(.change.actions == ["create"])) | length) to add, \(map(select(.change.actions == ["update"])) | length) to change, \(map(select(.change.actions == ["delete"])) | length) to destroy."
    end
  ' "$plan_file"
}

# @description Parse plan JSON and output structured JSON summary for agent consumption.
# @param $1 {string} plan_file - Path to the plan JSON file
# @return stdout - JSON object with has_changes, summary, changes, risk_level, destructive
_tfui_render_plan_json() {
  local plan_file="$1"
  jq '
    # Risk classification patterns
    def critical_types:
      ["aws_db_instance", "aws_rds_cluster", "aws_rds_cluster_instance",
       "aws_dynamodb_table", "aws_s3_bucket", "aws_efs_file_system",
       "aws_fsx_lustre_file_system", "aws_fsx_windows_file_system",
       "aws_redshift_cluster", "aws_elasticache_cluster",
       "aws_elasticache_replication_group", "aws_docdb_cluster",
       "aws_neptune_cluster", "aws_kms_key", "aws_kms_alias",
       "google_sql_database_instance", "google_storage_bucket",
       "google_kms_key_ring", "google_kms_crypto_key",
       "azurerm_sql_server", "azurerm_cosmosdb_account",
       "azurerm_storage_account", "azurerm_key_vault"];

    def high_risk_types:
      ["aws_iam_role", "aws_iam_policy", "aws_iam_user",
       "aws_iam_group", "aws_iam_instance_profile",
       "aws_vpc", "aws_subnet", "aws_route_table",
       "aws_nat_gateway", "aws_internet_gateway",
       "aws_eip", "aws_lb", "aws_alb", "aws_elb",
       "aws_ecs_cluster", "aws_eks_cluster",
       "aws_lambda_function", "aws_cloudfront_distribution",
       "google_compute_network", "google_compute_subnetwork",
       "google_container_cluster", "google_project_iam_member",
       "azurerm_virtual_network", "azurerm_kubernetes_cluster",
       "azurerm_role_assignment"];

    def medium_risk_types:
      ["aws_security_group", "aws_security_group_rule",
       "aws_network_acl", "aws_route53_record",
       "aws_cloudwatch_log_group", "aws_sns_topic",
       "aws_sqs_queue", "aws_ecr_repository",
       "google_compute_firewall", "google_dns_record_set",
       "azurerm_network_security_group", "azurerm_dns_zone"];

    def assess_risk(action):
      .type as $type |
      if action == "delete" or action == "replace" then
        if (critical_types | any(. == $type)) then "critical"
        elif (high_risk_types | any(. == $type)) then "critical"
        elif (medium_risk_types | any(. == $type)) then "high"
        else "high"
        end
      elif action == "update" then
        if (critical_types | any(. == $type)) then "high"
        elif (high_risk_types | any(. == $type)) then "high"
        elif (medium_risk_types | any(. == $type)) then "medium"
        else "medium"
        end
      elif action == "create" then
        if (critical_types | any(. == $type)) then "medium"
        elif (high_risk_types | any(. == $type)) then "medium"
        else "low"
        end
      else "low"
      end;

    def risk_priority:
      if . == "critical" then 4
      elif . == "high" then 3
      elif . == "medium" then 2
      elif . == "low" then 1
      else 0
      end;

    def action_label:
      if . == ["create"] then "create"
      elif . == ["delete"] then "delete"
      elif . == ["update"] then "update"
      elif (. | sort) == ["create", "delete"] then "replace"
      else "unknown"
      end;

    [.resource_changes // [] | .[] | select(.change.actions != ["no-op"] and .change.actions != ["read"])] |
    if length == 0 then
      {
        has_changes: false,
        summary: { add: 0, change: 0, destroy: 0, replace: 0 },
        changes: [],
        risk_level: "low",
        destructive: false
      }
    else
      . as $changes |
      ($changes | map(select(.change.actions == ["create"])) | length) as $add |
      ($changes | map(select(.change.actions == ["update"])) | length) as $change |
      ($changes | map(select(.change.actions == ["delete"])) | length) as $destroy |
      ($changes | map(select((.change.actions | sort) == ["create", "delete"])) | length) as $replace |
      ($changes | map(
        (.change.actions | action_label) as $action |
        {
          action: $action,
          address: .address,
          risk: assess_risk($action)
        }
      )) as $classified |
      ($classified | map(.risk | risk_priority) | max) as $max_risk |
      (if $max_risk >= 4 then "critical"
       elif $max_risk >= 3 then "high"
       elif $max_risk >= 2 then "medium"
       else "low"
       end) as $overall_risk |
      {
        has_changes: true,
        summary: { add: $add, change: $change, destroy: $destroy, replace: $replace },
        changes: $classified,
        risk_level: $overall_risk,
        destructive: ($destroy > 0 or $replace > 0)
      }
    end
  ' "$plan_file"
}
