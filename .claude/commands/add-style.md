---
allowed-tools: Read, Edit, Bash(go build:*)
description: Add or modify styles in the theme
---

## Add or modify styles

All styles live in `internal/ui/styles/theme.go`.

Steps:
1. Read `internal/ui/styles/theme.go`
2. Add new color constants or style variables following existing patterns
3. Use descriptive names: `StyleRisk<Level>`, `StyleAction<Type>`, `Style<Purpose>`
4. Run `go build ./...` to verify

Rules:
- Never define inline lipgloss.NewStyle() in view Render() methods
- Colors are lipgloss.Color("N") where N is an ANSI 256-color code
- Styles are package-level vars (allocated once, not per frame)
- Group by purpose: colors first, then styles
