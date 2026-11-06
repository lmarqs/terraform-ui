---
name: add-command
allowed-tools: Read, Write, Edit, Bash(mise run:*)
description: Add a new CLI subcommand (cobra)
---

## Add a new CLI subcommand

A subcommand is a thin cobra wrapper that parses flags into a plugin's typed `Input` and runs that plugin through the
session. The plugin owns the behavior; the command owns only parsing. Load `tfui-cli-ux` first for the I/O contract.

Steps:
1. Give the plugin an `Input` DTO in `plugins/<id>/input.go` and an `Activate(Input) tea.Cmd` method, as
   `plugins/taint/` does. The `test-writer` agent writes the failing test first.
2. Add `cmd/tfui/<id>_command.go` with `build<Id>Command(s *Session) *cobra.Command`, copying
   `cmd/tfui/taint_command.go` (positional args) or `cmd/tfui/version_command.go` (no args).
3. In `RunE`, fill the `Input` and call `s.RunPlugin(cmd.Context(), "<id>", ...)`:
   - `input.JSON = s.JSONStdout()` for the root-persistent `--json`
   - `input.AutoConfirm = s.SilentStderr() || s.macro.Active()` for an action that confirms in the TUI
4. Register it in the `rootCmd.AddCommand(...)` list in `cmd/tfui/main.go`.
5. Add `cmd/tfui/<id>_command_test.go` and document the command in `docs/reference/cli-reference.md`.
6. Run `mise run check:build && mise run test:unit ./cmd/...`.

Key patterns:
- `--ci`, `--json`, `--plan`, `--state`, `--chdir` and `--project` are root-persistent (`cmd/tfui/root.go`); a
  subcommand adds only its own flags.
- Data goes to stdout, the TUI and progress to stderr; `docs/reference/cli-io-contract.md` is the contract.
