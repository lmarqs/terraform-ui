# Scenario: Raw passthrough flags (-- separator / ExtraArgs)
#
# Verifies that extra arguments passed via -- separator are
# included in the emitted terraform commands without tfui
# interpreting them.
#
# Pre-condition: launched with ExtraArgs like "-no-color" or
# "-compact-warnings" configured.
#
# Expected:
#   - Plan and apply commands include the extra args
#   - The TUI behaves normally (doesn't crash or misinterpret)
#
# Note: This tape verifies the TUI loads and plan works normally.
# The actual ExtraArgs passthrough is verified by checking stdout
# in the integration test harness.

wait ready
key p
wait view aws_instance.web
assert view aws_instance.web
assert view aws_s3_bucket.data
