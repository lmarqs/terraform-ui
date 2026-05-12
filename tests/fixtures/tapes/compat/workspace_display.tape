# Scenario: Workspace selection via flag
#
# Verifies that the workspace is displayed correctly in the TUI
# header when loaded via StaticService. In read-only mode the
# workspace is shown as "readonly" by default.
#
# Pre-condition: launched with --plan (read-only mode).
#
# Expected:
#   - Home menu loads
#   - The TUI is functional (can navigate)
#   - Workspace shows in workspaces plugin view

wait ready
key w
wait view readonly
assert view readonly
