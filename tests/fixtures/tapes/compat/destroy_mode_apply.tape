# Scenario: Destroy mode apply confirmation
#
# Verifies that when applying a destroy plan, the apply confirmation
# communicates that resources will be destroyed. The confirmation
# flow should warn the user about infrastructure modification.
#
# Pre-condition: launched with -plan pointing to plan_destroy.json
# and -state pointing to state.json.
#
# Expected:
#   - Navigate to plan, then trigger apply
#   - First confirmation asks "Apply plan (2 changes)?"
#   - Second confirmation warns "Are you sure you want to apply these changes?"
#   - The emitted command should be "terraform apply"

wait ready
key p
wait view aws_instance.web
key a
wait view Apply plan
key y
wait view Are you sure
key y
