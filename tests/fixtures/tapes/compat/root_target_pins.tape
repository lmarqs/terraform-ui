# Scenario: tfui -target aws_instance.web starts the TUI with that resource pinned
#
# The integration test launches the full TUI with a root -target flag. Opening
# Plan shows the row already pinned and the other row unpinned; the plan run
# carries the pin as its target.

wait ready
key p
wait view aws_instance.web
assert view [*] + aws_instance.web
assert view [ ] ~ aws_s3_bucket.data
