# Scenario: Standalone CLI apply with --target (auto-plan mode)
#
# Verifies that `tfui apply --target=X --auto-approve` emits
# `terraform apply -target=X` (no plan file positional argument).
# This is the CLI standalone path, independent of the TUI pipeline.
#
# Pre-condition: launched with `apply --target=aws_instance.web --auto-approve -state fixture`
# Auto-approve skips TUI confirmation; apply runs immediately.

wait ready
wait view Apply complete
