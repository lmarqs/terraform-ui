# Scenario: Var-file flag passthrough to apply command
#
# Verifies that when -var-file is provided via CLI, it is threaded
# through to the emitted terraform commands. When the user triggers
# an apply, the recorded command should include -var-file=<path>.
#
# Pre-condition: launched with -var-file=prod.tfvars (once CLI flag exists).
# For now this tests the existing plan + apply flow and verifies
# the command string includes var-file when PlanOptions/ApplyOptions
# carry VarFiles.
#
# Expected:
#   - Navigate to plan view, verify it loads
#   - Trigger apply flow through both confirmations
#   - The stdout command output includes -var-file=prod.tfvars
#
# Note: The -var-file CLI flag does not yet exist. This tape
# validates the apply flow works; the integration test will add
# the -var-file assertion once the feature is implemented.

wait ready
key p
wait view aws_instance.web
key a
wait view Apply plan
key y
wait view Are you sure
key y
