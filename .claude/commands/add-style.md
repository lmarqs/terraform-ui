---
allowed-tools: Read, Edit, Bash(mise run:*)
description: Add or modify styles in the theme
---

## Add or modify styles

All colors and styles live in `pkg/sdk/styles.go` (Dracula theme).
Actions bar chip styles live in `pkg/sdk/ui/actionsbar.go`.

Steps:
1. Read `pkg/sdk/styles.go`
2. Add new color constants or style variables following existing patterns
3. Use descriptive names: `StyleRisk<Level>`, `StyleAction<Type>`, `Style<Purpose>`
4. Run `mise run check:build` to verify

Rules:
- Never define inline lipgloss.NewStyle() in view Render() methods
- Colors use Dracula hex values: `lipgloss.Color("#bd93f9")`
- Styles are package-level vars (allocated once, not per frame)
- Group by purpose: colors first, then styles
- Theme: Dracula (`#bd93f9` purple primary, `#44475a` selection bg, `#f8f8f2` fg, `#6272a4` faint)
