---
layout: default
title: Architecture
description: Internal architecture of terraform-ui
---

# Architecture

terraform-ui is a Go application built with [Bubbletea](https://github.com/charmbracelet/bubbletea) (TUI framework) and [terraform-exec](https://github.com/hashicorp/terraform-exec) (terraform SDK).

## Project Structure

```
cmd/tfui/main.go           — Entry point, CLI flag parsing, program setup
internal/
  config/config.go         — tfui.yaml loading and CLI config merging
  terraform/
    service.go             — Service interface + TerraformService (terraform-exec)
    parser.go              — Plan JSON parsing into typed structs
    risk.go                — Risk classification engine
    grouping.go            — Module-level change grouping
    phantom.go             — Phantom change detection
  ui/
    app.go                 — Root Bubbletea model, view routing
    views/
      home.go              — Home screen (key dispatch)
      plan.go              — Plan review (tree navigation)
      apply.go             — Live apply progress
      state.go             — State browser
      workspaces.go        — Workspace management
      modules.go           — Monorepo project picker
    components/
      header.go            — Top bar (title, workspace)
      statusbar.go         — Bottom bar (keybindings)
    styles/
      theme.go             — Lipgloss style definitions
```

## How terraform-exec Is Used

`internal/terraform/service.go` defines a `Service` interface wrapping terraform operations. `TerraformService` implements it using `hashicorp/terraform-exec`:

- `Plan()` — runs `terraform plan -json`, streams output, parses resource changes
- `Apply()` — runs `terraform apply -auto-approve -json`, streams progress events
- `StateList()` — runs `terraform state list`
- `Show()` — runs `terraform state show <address>`
- `Workspace()` — runs `terraform workspace show`

The service layer is injected into the UI, keeping terraform logic decoupled from rendering.

## Bubbletea Model/Update/View

The TUI follows Bubbletea's Elm architecture:

```
┌────────────────────────────────────────┐
│  App (root model)                      │
│  - routes messages to active view      │
│  - handles global keys (q, Esc)        │
│  - manages window resize               │
├────────────────────────────────────────┤
│  Views (home, plan, apply, state, ...) │
│  - each implements tea.Model           │
│  - own Init/Update/View methods        │
│  - emit commands for async work        │
├────────────────────────────────────────┤
│  Components (header, statusbar)        │
│  - shared UI elements                  │
│  - rendered by views via View()        │
└────────────────────────────────────────┘
```

**Model** — `App` holds the active view, config, window dimensions, and shared state (current plan, workspace).

**Update** — Messages flow through `App.Update()` which delegates to the active view. View transitions happen via custom messages (e.g., `NavigateMsg{View: ViewPlan}`).

**View** — `App.View()` composes the active view's output between the header and status bar using lipgloss for layout.

## How Styles Are Organized

`internal/ui/styles/theme.go` defines all lipgloss styles in one place:

- Color palette constants (adaptive for light/dark terminals)
- Component styles (borders, padding, alignment)
- Semantic styles (risk level colors, action colors, dimmed text for phantom changes)

Views import `styles` and reference named styles rather than defining inline styles, keeping visual consistency centralized.

## Development

All commands go through [mise](https://mise.jdx.dev/):

```bash
mise install              # Install Go, terraform, and other tools
mise run go:build         # Build binary to dist/tfui
mise run go:test          # Run unit tests
mise run go:lint          # Run go vet
mise run go:coverage      # Test with coverage report
mise run go:run           # Launch TUI in dev mode
```

No Makefile — mise tasks are the single entry point for both local development and CI.
