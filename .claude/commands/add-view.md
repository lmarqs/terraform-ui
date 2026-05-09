---
allowed-tools: Bash(find:*), Read, Write, Edit, Bash(go build:*), Bash(go vet:*)
description: Add a new TUI view to the application
---

## Add a new TUI view

Create a new view in `internal/ui/views/`. Follow the existing pattern:

1. Read `internal/ui/views/plan.go` as a reference for the structure
2. Create the new view file in `internal/ui/views/<name>.go`
3. Wire it into `internal/ui/app.go`:
   - Add a `View<Name>` constant to the View enum
   - Add the view field to the App struct
   - Initialize it in NewApp()
   - Add a case in View() rendering
   - Add keyboard handling in handleKey/updateX method
4. Add it to the home menu in `internal/ui/views/home.go`
5. Run `go build ./...` and `go vet ./...` to verify

Key patterns:
- Views are value types (return copies from Set*/Move* methods)
- Use styles from `internal/ui/styles` (never inline lipgloss)
- Use strings.Builder for render loops
- Async operations return tea.Cmd functions
