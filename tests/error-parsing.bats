#!/usr/bin/env bats

setup() {
  load 'helpers/common-setup'
  _common_setup
  ERROR_FIXTURES="$PROJECT_ROOT/tests/fixtures/errors"
}

# -- Category detection -------------------------------------------------------

@test "given missing argument error, categorizes as missing_argument" {
  run bash -c '
    source "'"$PROJECT_ROOT"'/lib/tfui.sh"
    _tfui_error_categorize "Missing required argument" "The argument \"ami\" is required, but no definition was found."
  '
  assert_output "missing_argument"
}

@test "given invalid value error, categorizes as invalid_value" {
  run bash -c '
    source "'"$PROJECT_ROOT"'/lib/tfui.sh"
    _tfui_error_categorize "Invalid value for variable" "expected a number but got string"
  '
  assert_output "invalid_value"
}

@test "given syntax error, categorizes as syntax_error" {
  run bash -c '
    source "'"$PROJECT_ROOT"'/lib/tfui.sh"
    _tfui_error_categorize "Invalid expression" "no expression was found"
  '
  assert_output "syntax_error"
}

@test "given provider error, categorizes as provider_error" {
  run bash -c '
    source "'"$PROJECT_ROOT"'/lib/tfui.sh"
    _tfui_error_categorize "Error configuring provider" "API error: AuthFailure"
  '
  assert_output "provider_error"
}

@test "given state lock error, categorizes as state_error" {
  run bash -c '
    source "'"$PROJECT_ROOT"'/lib/tfui.sh"
    _tfui_error_categorize "Error acquiring the state lock" "Another terraform process is already running"
  '
  assert_output "state_error"
}

@test "given cycle error, categorizes as dependency_error" {
  run bash -c '
    source "'"$PROJECT_ROOT"'/lib/tfui.sh"
    _tfui_error_categorize "Cycle detected" "Cycle: aws_security_group.a, aws_security_group.b"
  '
  assert_output "dependency_error"
}

@test "given unknown error, categorizes as unknown" {
  run bash -c '
    source "'"$PROJECT_ROOT"'/lib/tfui.sh"
    _tfui_error_categorize "Something unexpected happened" "no further details"
  '
  assert_output "unknown"
}

# -- Suggestion generation ----------------------------------------------------

@test "given missing_argument with attribute and resource, suggests adding attribute" {
  run bash -c '
    source "'"$PROJECT_ROOT"'/lib/tfui.sh"
    _tfui_error_suggest "missing_argument" "Missing required argument" "ami" "aws_instance.web"
  '
  assert_output "Add 'ami' attribute to the aws_instance.web resource block"
}

@test "given syntax_error, suggests fixing HCL syntax" {
  run bash -c '
    source "'"$PROJECT_ROOT"'/lib/tfui.sh"
    _tfui_error_suggest "syntax_error" "Invalid expression" "" ""
  '
  assert_output "Fix the HCL syntax error — check for unclosed braces, quotes, or invalid expressions"
}

@test "given state_error with lock, suggests force-unlock" {
  run bash -c '
    source "'"$PROJECT_ROOT"'/lib/tfui.sh"
    _tfui_error_suggest "state_error" "Error acquiring the state lock" "" ""
  '
  assert_output "Release the state lock (terraform force-unlock) or wait for the other operation to complete"
}

@test "given unknown category, returns empty suggestion" {
  run bash -c '
    source "'"$PROJECT_ROOT"'/lib/tfui.sh"
    _tfui_error_suggest "unknown" "Something happened" "" ""
  '
  assert_output ""
}

# -- Full parsing (integration with fixture files) ----------------------------

@test "given missing_argument fixture, parses resource and category" {
  run bash -c '
    source "'"$PROJECT_ROOT"'/lib/tfui.sh"
    _tfui_parse_error "'"$ERROR_FIXTURES"'/missing_argument.txt" | jq -r ".errors[0].category"
  '
  assert_output "missing_argument"
}

@test "given missing_argument fixture, extracts attribute from body" {
  run bash -c '
    source "'"$PROJECT_ROOT"'/lib/tfui.sh"
    _tfui_parse_error "'"$ERROR_FIXTURES"'/missing_argument.txt" | jq -r ".errors[0].attribute"
  '
  assert_output "ami"
}

@test "given missing_argument fixture, extracts file location" {
  run bash -c '
    source "'"$PROJECT_ROOT"'/lib/tfui.sh"
    _tfui_parse_error "'"$ERROR_FIXTURES"'/missing_argument.txt" | jq -r ".errors[0].file"
  '
  assert_output "main.tf:12"
}

@test "given provider_error fixture, extracts resource address" {
  run bash -c '
    source "'"$PROJECT_ROOT"'/lib/tfui.sh"
    _tfui_parse_error "'"$ERROR_FIXTURES"'/provider_error.txt" | jq -r ".errors[0].resource"
  '
  assert_output "aws_instance.web"
}

@test "given provider_error fixture, categorizes as provider_error" {
  run bash -c '
    source "'"$PROJECT_ROOT"'/lib/tfui.sh"
    _tfui_parse_error "'"$ERROR_FIXTURES"'/provider_error.txt" | jq -r ".errors[0].category"
  '
  assert_output "provider_error"
}

@test "given syntax_error fixture, categorizes correctly" {
  run bash -c '
    source "'"$PROJECT_ROOT"'/lib/tfui.sh"
    _tfui_parse_error "'"$ERROR_FIXTURES"'/syntax_error.txt" | jq -r ".errors[0].category"
  '
  assert_output "syntax_error"
}

@test "given state_error fixture, categorizes correctly" {
  run bash -c '
    source "'"$PROJECT_ROOT"'/lib/tfui.sh"
    _tfui_parse_error "'"$ERROR_FIXTURES"'/state_error.txt" | jq -r ".errors[0].category"
  '
  assert_output "state_error"
}

@test "given dependency_error fixture, categorizes correctly" {
  run bash -c '
    source "'"$PROJECT_ROOT"'/lib/tfui.sh"
    _tfui_parse_error "'"$ERROR_FIXTURES"'/dependency_error.txt" | jq -r ".errors[0].category"
  '
  assert_output "dependency_error"
}

@test "given multi_error fixture, parses all three diagnostics" {
  run bash -c '
    source "'"$PROJECT_ROOT"'/lib/tfui.sh"
    _tfui_parse_error "'"$ERROR_FIXTURES"'/multi_error.txt" | jq ".errors | length"
  '
  assert_output "3"
}

@test "given multi_error fixture, first error is missing_argument" {
  run bash -c '
    source "'"$PROJECT_ROOT"'/lib/tfui.sh"
    _tfui_parse_error "'"$ERROR_FIXTURES"'/multi_error.txt" | jq -r ".errors[0].category"
  '
  assert_output "missing_argument"
}

@test "given multi_error fixture, second error is invalid_value" {
  run bash -c '
    source "'"$PROJECT_ROOT"'/lib/tfui.sh"
    _tfui_parse_error "'"$ERROR_FIXTURES"'/multi_error.txt" | jq -r ".errors[1].category"
  '
  assert_output "invalid_value"
}

@test "given multi_error fixture, third entry is a warning" {
  run bash -c '
    source "'"$PROJECT_ROOT"'/lib/tfui.sh"
    _tfui_parse_error "'"$ERROR_FIXTURES"'/multi_error.txt" | jq -r ".errors[2].severity"
  '
  assert_output "warning"
}

@test "given unstructured error, falls back to unknown category" {
  run bash -c '
    source "'"$PROJECT_ROOT"'/lib/tfui.sh"
    _tfui_parse_error "'"$ERROR_FIXTURES"'/unstructured.txt" | jq -r ".errors[0].category"
  '
  assert_output "unknown"
}

@test "given unstructured error, captures raw message" {
  run bash -c '
    source "'"$PROJECT_ROOT"'/lib/tfui.sh"
    _tfui_parse_error "'"$ERROR_FIXTURES"'/unstructured.txt" | jq -r ".errors[0].message"
  '
  assert_output "terraform: command not found"
}

@test "parse_error always sets success to false" {
  run bash -c '
    source "'"$PROJECT_ROOT"'/lib/tfui.sh"
    _tfui_parse_error "'"$ERROR_FIXTURES"'/missing_argument.txt" | jq -r ".success"
  '
  assert_output "false"
}

@test "parse_error includes raw_output field" {
  run bash -c '
    source "'"$PROJECT_ROOT"'/lib/tfui.sh"
    _tfui_parse_error "'"$ERROR_FIXTURES"'/missing_argument.txt" | jq -r ".raw_output" | grep -c "Missing required argument"
  '
  assert_output "1"
}

@test "parse_error generates suggestion for missing_argument" {
  run bash -c '
    source "'"$PROJECT_ROOT"'/lib/tfui.sh"
    _tfui_parse_error "'"$ERROR_FIXTURES"'/missing_argument.txt" | jq -r ".errors[0].suggestion"
  '
  assert_output --partial "ami"
}

# -- Stdin input mode ---------------------------------------------------------

@test "parse_error reads from stdin when no file argument given" {
  run bash -c '
    source "'"$PROJECT_ROOT"'/lib/tfui.sh"
    echo "Error: Invalid expression

  on main.tf:3:

Unexpected token." | _tfui_parse_error | jq -r ".errors[0].category"
  '
  assert_output "syntax_error"
}
