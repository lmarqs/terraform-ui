# Scenario: CLI flag normalization — plan with -target flag
#
# Verifies that when the user launches tfui with target flags,
# the plan view loads correctly and shows the targeted resource.
# This validates that PlanOptions.Targets is properly threaded
# from CLI flags through to the plan plugin.
#
# Expected: plan view shows the targeted resource in the list.

wait ready
key p
wait view aws_instance.web
assert view aws_instance.web
