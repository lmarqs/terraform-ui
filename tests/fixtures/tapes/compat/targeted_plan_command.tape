# Scenario: Target flag recorded in plan command (ADR-0019)
#
# Verifies that when targets are configured (via pinning a resource),
# the **plan** command emitted to stdout includes the -target= flag,
# and the subsequent apply consumes only the saved plan file (no
# -target= on apply, since terraform forbids it with a plan file).
#
# Pre-condition: launched with -plan and -state. User pins a
# resource, then triggers apply.
#
# Expected:
#   - Pin aws_instance.web (first resource in list)
#   - Press 'a' to initiate apply
#   - Single confirmation: "Are you sure"
#   - stdout includes "terraform plan -target=aws_instance.web"
#   - stdout includes "terraform apply <planfile>" (no -target=)

wait ready
key p
wait view aws_instance.web
key space
key a
wait view Are you sure
key y
wait view Apply complete
