---
allowed-tools: Read, Write, Edit, Bash(go build:*)
description: Add a new CLI subcommand (cobra)
---

## Add a new CLI subcommand

Add a new subcommand to the cobra CLI in `cmd/tfui/main.go`.

Steps:
1. Read `cmd/tfui/main.go` for existing command patterns
2. Add the new `&cobra.Command{}` with Use, Short, and RunE
3. Register it with `rootCmd.AddCommand()`
4. Add flags specific to the subcommand
5. Implement the run function
6. Run `go build ./...` to verify

Key patterns:
- Subcommands preserve the existing CLI interface
- Mode flags: --mode silent|spinner|progress|agent
- Use the terraform service for operations
- Spinner/progress output goes to stderr, data to stdout
- Agent mode always outputs JSON
