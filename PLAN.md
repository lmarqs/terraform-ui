# terraform-ui v1.0

## Summary

terraform-ui is a Go-based interactive TUI for Terraform, inspired by k9s. It provides a keyboard-driven full-screen interface for plan review, risk analysis, blast radius visualization, state browsing, and apply — plus backward-compatible non-interactive CLI modes.

## Motivation

- **Interactive UX**: k9s proved that keyboard-driven TUIs dramatically improve infrastructure workflows. No equivalent exists for Terraform.
- **Single binary**: No jq dependency. Install via homebrew, go install, or download.
- **Plugin architecture**: Every feature is a self-contained plugin. Enable, disable, or configure individually.
- **Local-first**: No SaaS, no server, no account. Works offline.

## Architecture

| Choice | Decision | Rationale |
|--------|----------|-----------|
| Language | Go 1.22+ | Terraform ecosystem is Go; terraform-exec SDK; target audience knows Go |
| TUI framework | charmbracelet/bubbletea | Elm Architecture, dominant Go TUI framework |
| Terraform SDK | hashicorp/terraform-exec | Official Go SDK, structured output |
| CLI | spf13/cobra | Standard Go CLI framework |
| Config | YAML (tfui.yaml) | Human-readable, pnpm-workspace.yaml style |
| Tooling | mise | Single entry point, no Makefile |
| Release | goreleaser | Cross-compile, homebrew tap, GitHub Releases |
| Features | Plugin system | Each feature is a plugin under `plugins/` |

## CLI Interface

```bash
tfui                              # → interactive TUI (default)
tfui --dir ./infra                # → TUI scoped to directory
tfui plan --dir ./infra --mode progress
tfui plan --dir ./infra --mode agent | jq .
tfui apply --dir ./infra --mode spinner
tfui version
```

## Project Structure

```
terraform-ui/
├── cmd/tfui/main.go              — CLI entry point, plugin registration
├── internal/
│   ├── config/config.go          — tfui.yaml loading, project discovery, OpenTofu detection
│   ├── plugin/
│   │   ├── plugin.go             — Plugin interface
│   │   ├── registry.go           — Registry (factory pattern, config-driven build)
│   │   └── context.go            — Shared context for plugins
│   ├── terraform/
│   │   ├── service.go            — Service interface + terraform-exec implementation
│   │   ├── parser.go             — Domain types (Resource, PlanChange, Action, RiskLevel)
│   │   ├── risk.go               — Risk classification engine (shared)
│   │   ├── phantom.go            — Phantom change detection (shared)
│   │   ├── grouping.go           — Module-level change grouping (shared)
│   │   └── *_test.go             — Unit tests
│   └── ui/
│       ├── app.go                — Root bubbletea model, plugin routing
│       ├── styles/theme.go       — Centralized lipgloss styles
│       ├── components/           — Header, statusbar
│       └── views/home.go         — Home screen (auto-generated from plugins)
├── plugins/
│   ├── plan/                     — Plan review (risk badges, phantom highlighting, diffs)
│   ├── risk/                     — Risk analysis (grouped by level)
│   ├── phantom/                  — Phantom change detection + explanation
│   ├── blastradius/              — Blast radius (module-grouped, impact scores)
│   ├── state/                    — State browser (filterable, detail view)
│   ├── apply/                    — Apply with confirmation + progress
│   ├── workspaces/               — Workspace management
│   └── projects/                 — Monorepo project picker
├── docs/                         — Jekyll site (17 pages)
│   ├── plugins/                  — Per-plugin documentation
│   └── *.md                      — Getting started, architecture, config, CLI ref
├── .claude/commands/             — Slash commands (11)
├── .github/workflows/go.yaml     — CI: terraform+tofu × ubuntu+macos
├── .goreleaser.yaml              — Cross-platform release
├── tfui.example.yaml             — Example config
├── go.mod / go.sum
└── mise.toml                     — Task runner
```

## Plugin System

Every feature is a plugin implementing the `Plugin` interface:

```go
type Plugin interface {
    ID() string
    Name() string
    Description() string
    KeyBinding() string
    Init(ctx *Context) tea.Cmd
    Update(msg tea.Msg) (Plugin, tea.Cmd)
    View(width, height int) string
    Configure(cfg map[string]interface{}) error
    Ready() bool
}
```

Plugins are configured via `tfui.yaml`:

```yaml
terraform_binary: terraform  # auto-detects tofu if omitted

projects:
  paths:
    - "modules/*"

plugins:
  risk:
    enabled: true
    # custom_rules:
    #   - type: "aws_lambda_function"
    #     level: critical
  apply:
    enabled: true
  # blastradius:
  #   enabled: false
```

## Metrics

| Metric | Value |
|--------|-------|
| Go source files | 31 |
| Go source lines | 6,329 |
| Test lines | 926 |
| Plugins | 8 |
| Docs pages | 17 |
| Slash commands | 11 |
| Commits on branch | 22+ |

## Current Status

### Completed

- [x] Go project scaffold (cobra CLI, config loading)
- [x] Terraform service (Plan, Apply, StateList, Show, Workspace via terraform-exec)
- [x] Risk classification (3-tier resource types × action severity)
- [x] Phantom change detection (JSON normalization, deep equality)
- [x] Module grouping (path extraction, alphabetical sorting)
- [x] Plugin system (interface, registry, factory pattern, per-plugin config)
- [x] All 8 plugins (plan, risk, phantom, blastradius, state, apply, workspaces, projects)
- [x] Non-interactive CLI modes (silent, spinner, progress, agent)
- [x] OpenTofu support (auto-detection, config)
- [x] Theme system (centralized styles, no inline lipgloss)
- [x] Unit tests (table-driven for risk/phantom/grouping)
- [x] CI pipeline (terraform+tofu × ubuntu+macos, coverage, goreleaser)
- [x] GoReleaser (linux/darwin × amd64/arm64, homebrew tap)
- [x] Documentation (Jekyll site, per-plugin docs, architecture)
- [x] Mise tasks (fmt, build, test, lint, coverage, run)
- [x] Slash commands (/build, /test, /lint, /fmt, /coverage, /run, /add-plugin, /add-command, /add-style, /add-terraform-feature)
- [x] Godoc comments on all exported functions
- [x] Legacy bash codebase removed

### In Progress

- [ ] Wire plugin registry into app.go (replace hardcoded views with registry dispatch)

### Remaining for v1.0 Release

| Task | Priority | Effort |
|------|----------|--------|
| Wire plugins into app.go | Critical | 1 day |
| Help overlay (`?` key) | Medium | 2 days |
| Integration test with real terraform fixture | Medium | 2 days |
| Update CLAUDE.md for new structure | Medium | 1 hour |
| Fuzzy search (upgrade from substring) | Low | 1 week |

### Post-v1.0 (future plugins)

| Plugin | Description |
|--------|-------------|
| `drift` | Periodic drift detection |
| `cost` | Infracost integration |
| `import` | Interactive resource import wizard |
| `graph` | Module dependency graph visualization |
| `history` | Local operation history (SQLite) |

## Build Pipeline

```
mise run fmt        → gofmt -w cmd/ internal/ plugins/
mise run lint       → go vet ./...
mise run build      → go build (depends on fmt + lint)
mise run test       → go test ./...
mise run coverage   → go test -coverprofile + go tool cover
mise run run        → go run ./cmd/tfui (dev mode)
mise run release    → goreleaser release --clean
```

## Release Strategy

1. Complete plugin wiring (in progress)
2. Merge `feat/go-rewrite` to `main`
3. Tag `v1.0.0` → triggers goreleaser via CI
4. Publishes to GitHub Releases + homebrew tap (`lmarqs/tap/tfui`)
5. Users: `brew install lmarqs/tap/tfui` or `go install github.com/lmarqs/terraform-ui/cmd/tfui@latest`

## Decisions Made

1. **Bash library mode (`source lib/tfui.sh`)** — Fully deprecated. No compatibility shim.
2. **OpenTofu** — Supported from day one. Auto-detects `tofu` on PATH, falls back to `terraform`.
3. **Plugin system** — Implemented now. All features are modular plugins. Third-party plugins possible in future via Go plugin interface or gRPC.
4. **Naming** — "plugins" everywhere (not "extensions"). Matches k9s, terraform providers, vim.
