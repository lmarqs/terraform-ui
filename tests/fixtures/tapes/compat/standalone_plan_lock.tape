# Scenario: Standalone CLI plan with per-invocation lock flags
#
# Verifies that `tfui plan --lock=false --lock-timeout=30s` reaches terraform's
# own -lock / -lock-timeout flags on the recorded plan command
# (lmarqs/terraform-ui#58).
#
# Pre-condition: launched with `plan --lock=false --lock-timeout=30s -plan fixture`.

wait ready
