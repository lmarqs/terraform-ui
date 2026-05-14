# Scenario: Workspace navigation in preloaded mode
#
# Verifies that the workspace plugin is accessible in the TUI
# when loaded with preloaded data.
#
# Pre-condition: launched with --plan (preloaded data mode).
#
# Expected:
#   - Home menu loads
#   - The TUI is functional (can navigate to workspace)
#   - Plugin view renders (shows workspace list or error)

wait ready
key w
wait view Workspaces
