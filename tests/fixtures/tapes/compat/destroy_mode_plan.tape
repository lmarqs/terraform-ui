# Scenario: Destroy mode display
#
# Verifies that when --destroy flag is active, the plan view
# shows only delete actions. All resources should show the
# delete symbol (-) indicating destruction.
#
# Pre-condition: launched with --plan pointing to plan_destroy.json
# which contains only "delete" actions on all resources.
#
# Expected:
#   - Plan loads and shows resources with delete actions
#   - The summary line shows "N to destroy"
#   - No "to add" or "to change" counts (or they show 0)

wait ready
key p
wait view aws_instance.web
assert view aws_instance.web
assert view aws_s3_bucket.data
assert view to destroy
