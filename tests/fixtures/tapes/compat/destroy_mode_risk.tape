# Scenario: Destroy mode risk classification
#
# Verifies that destroy operations are classified with appropriate
# risk levels. S3 bucket deletion should be CRITICAL, instance
# deletion should be medium.
#
# Pre-condition: launched with --plan pointing to plan_destroy.json
#
# Expected:
#   - Plan shows risk badges next to each resource
#   - Overall risk shows CRITICAL (highest among changes)
#   - Delete symbols (-) visible for all resources

wait ready
key p
wait view aws_instance.web
assert view CRITICAL
assert view medium
assert view Overall risk: CRITICAL
assert view 2 to destroy
