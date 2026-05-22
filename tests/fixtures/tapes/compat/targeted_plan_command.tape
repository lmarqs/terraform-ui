# Scenario: Target flag recorded in plan command
#
# Verifies that when targets are configured (via pinning a resource),
# the apply command emitted to stdout includes the -target= flags.
#
# Pre-condition: launched with -plan and -state. User pins a
# resource, then triggers apply.
#
# Expected:
#   - Pin aws_instance.web (first resource in list)
#   - Press 'a' to initiate apply
#   - First confirmation: "Apply 1 targeted resource(s)?"
#   - Second confirmation: "Are you sure" with "Targeting 1 resource(s)."
#   - stdout includes "terraform apply -target=aws_instance.web"

wait ready
key p
wait view aws_instance.web
key space
key a
wait view Apply targeted plan
key y
wait view Apply complete
