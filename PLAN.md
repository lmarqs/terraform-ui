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
| Release | semantic-release + goreleaser | Automatic versioning + cross-platform binaries |
| Features | Plugin system | Each feature is a plugin under `plugins/` |
| Plugin SDK | `pkg/sdk/` | Public contract — plugins never import internal/ |

## CLI Interface

```bash
tfui                              # → interactive TUI (default)
tfui --dir ./infra                # → TUI scoped to directory
tfui --debug                      # → TUI with debug logging to ~/.tfui/logs/
tfui init                         # → generate tfui.yaml interactively
tfui plan --dir ./infra --mode progress
tfui plan --dir ./infra --mode agent | jq .
tfui apply --dir ./infra --mode spinner
tfui version
```

## Project Structure

```
terraform-ui/
├── cmd/tfui/main.go              — CLI entry point, plugin registration (thin glue)
├── pkg/sdk/                      — Public SDK (the only dependency for plugins)
│   ├── plugin.go                 — Plugin + Activatable interfaces
│   ├── context.go                — Shared context (service, logger, session)
│   ├── session.go                — Inter-plugin session cache
│   ├── keys.go                   — Well-known session keys
│   ├── types.go                  — Domain types (Resource, PlanChange, Action, RiskLevel)
│   ├── service.go                — Service interface
│   └── styles.go                 — Style constants
├── internal/
│   ├── config/config.go          — tfui.yaml, project discovery, OpenTofu detection
│   ├── logging/logging.go        — slog debug logger (JSON lines)
│   ├── plugin/registry.go        — Plugin registry (host-side only)
│   ├── terraform/                — TerraformService (implements sdk.Service)
│   │   ├── service.go            — terraform-exec wrapper
│   │   ├── risk.go               — Risk classification (shared logic)
│   │   ├── phantom.go            — Phantom detection (shared logic)
│   │   └── grouping.go           — Module grouping (shared logic)
│   └── ui/
│       ├── app.go                — Root model, plugin routing via registry
│       ├── styles/theme.go       — Lipgloss styles
│       ├── components/           — Header, statusbar
│       └── views/home.go         — Home (auto-generated from plugins)
├── plugins/                      — Each plugin imports only pkg/sdk
│   ├── plan/                     — Plan review (Activatable)
│   ├── risk/                     — Risk analysis
│   ├── phantom/                  — Phantom change detection
│   ├── blastradius/              — Blast radius visualization
│   ├── state/                    — State browser (Activatable)
│   ├── apply/                    — Apply with session-cached plan data
│   ├── workspaces/               — Workspace management (Activatable)
│   ├── projects/                 — Monorepo project picker (Activatable)
│   └── init/                     — Init wizard (Activatable)
├── tests/
│   ├── integration/              — CLI integration tests (build tag)
│   └── fixtures/                 — Real terraform projects
├── docs/                         — Jekyll site
│   ├── plugins/                  — Per-plugin docs
│   └── *.md                      — Architecture, config, CLI ref, getting-started
├── .claude/commands/             — Slash commands
├── .github/workflows/go.yaml     — CI: terraform+tofu × ubuntu+macos
├── .goreleaser.yaml
├── .releaserc                    — semantic-release config
├── tfui.example.yaml
├── go.mod / go.sum
└── mise.toml
```

## Plugin System

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

// Optional — plugins implement this to do work when activated
type Activatable interface {
    Activate() tea.Cmd
}
```

Configured via `tfui.yaml`:

```yaml
terraform_binary: terraform  # auto-detects tofu if omitted

projects:
  paths:
    - "modules/*"

plugins:
  risk:
    enabled: true
  apply:
    enabled: true
```

## Status: v1.0 Complete

### All Done

- [x] Go project scaffold (cobra CLI, config loading)
- [x] Terraform service (terraform-exec: Plan, Apply, StateList, Show, Workspace)
- [x] Risk classification, phantom detection, module grouping
- [x] Plugin system (interface, registry, factory, config, Activatable)
- [x] SDK extraction (`pkg/sdk/` — plugins never import internal/)
- [x] Session cache (inter-plugin data sharing)
- [x] 9 plugins (plan, risk, phantom, blastradius, state, apply, workspaces, projects, init)
- [x] Plugin routing (app.go delegates to active plugin via registry)
- [x] Non-interactive CLI modes (silent, spinner, progress, agent)
- [x] `tfui init` wizard (detect monorepo patterns, generate tfui.yaml)
- [x] OpenTofu support (auto-detection, configurable)
- [x] Structured debug logging (`--debug`, JSON lines to ~/.tfui/logs/)
- [x] `/debug-review` slash command for AI-assisted session analysis
- [x] Theme system (centralized styles in pkg/sdk)
- [x] Unit tests (92%+ coverage, 100% on plugins)
- [x] Integration tests (31 tests with real terraform fixtures)
- [x] CI (terraform+tofu × ubuntu+macos, coverage enforcement, goreleaser)
- [x] GoReleaser (linux/darwin × amd64/arm64, homebrew tap)
- [x] semantic-release (automatic versioning from conventional commits)
- [x] Documentation (Jekyll site, per-plugin docs, architecture)
- [x] Mise tasks (fmt, build, test, lint, coverage, run, test:integration)
- [x] Slash commands (/build, /test, /lint, /fmt, /coverage, /run, /add-plugin, /debug-review)
- [x] Godoc comments on all exported functions
- [x] Dependency injection (logger, service, session via context)
- [x] Legacy bash codebase removed

### Remaining Polish (post-merge)

| Task | Priority | Effort |
|------|----------|--------|
| Help overlay (`?` key) | Medium | 2 days |
| Fuzzy search (upgrade from substring) | Low | 1 week |
| Error notifications on home screen | Low | 2 days |
| Update CLAUDE.md for new structure | Low | 1 hour |

### Post-v1.0 (future plugins)

| Plugin | Description |
|--------|-------------|
| `drift` | Periodic drift detection |
| `cost` | Infracost integration |
| `import` | Interactive resource import wizard |
| `graph` | Module dependency graph visualization |
| `history` | Local operation history (SQLite) |
| External (gRPC) | Third-party plugins via hashicorp/go-plugin |

## Build Pipeline

```
mise run fmt        → gofmt -w cmd/ internal/ plugins/
mise run lint       → go vet ./...
mise run build      → go build (depends on fmt + lint)
mise run test       → go test ./...
mise run coverage   → 90%+ enforcement (excludes cmd/)
mise run test:integration → integration tests (need terraform)
mise run run        → go run ./cmd/tfui (dev mode)
mise run release    → goreleaser release --clean
```

## Release Strategy

1. Push to `main` with conventional commit (feat:/fix:)
2. semantic-release determines version, creates tag `v1.0.0`
3. Tag triggers goreleaser via CI
4. Goreleaser builds linux/darwin × amd64/arm64
5. Publishes to GitHub Releases + homebrew tap (`lmarqs/tap/tfui`)
6. Install: `brew install lmarqs/tap/tfui` or `go install github.com/lmarqs/terraform-ui/cmd/tfui@latest`

## Decisions Made

1. **Bash library mode** — Fully deprecated. No compatibility shim.
2. **OpenTofu** — Supported from day one. Auto-detects `tofu` on PATH.
3. **Plugin system** — All features are modular plugins. `pkg/sdk` is the public contract.
4. **Naming** — "plugins" everywhere. Matches k9s, terraform providers, vim.
5. **Activatable pattern** — Plugins do async work only when user navigates to them (not on startup).
6. **Session cache** — Plugins share data (plan results) via thread-safe session store.
7. **Debug logging** — `--debug` writes structured JSON to ~/.tfui/logs/ for AI-assisted review.
8. **No auto-plan** — Plan only runs when explicitly triggered. Critical for monorepos.
9. **DI everywhere** — Service, logger, session injected via context. Enables testing.
10. **Coverage** — 90%+ enforced in CI. Untestable layer (cmd/ glue) excluded.
