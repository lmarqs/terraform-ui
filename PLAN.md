# terraform-ui v1.0 Migration Plan

## Summary

terraform-ui is being rewritten from a pure-bash CLI tool (v0.39.0) to a Go-based interactive TUI (v1.0.0), inspired by k9s. The rewrite preserves the existing CLI interface while adding a full-screen interactive mode as the new default experience.

## Motivation

- **Interactive UX**: k9s proved that keyboard-driven TUIs dramatically improve infrastructure workflows. No equivalent exists for Terraform.
- **Single binary**: Go eliminates the jq dependency and simplifies distribution (homebrew, goreleaser, go install).
- **Richer features**: State browsing, inline attribute diffs, monorepo project selection — these require a real UI framework.
- **Local-first**: No SaaS, no server, no account. Works offline.

## Architecture Decision

| Choice | Decision | Rationale |
|--------|----------|-----------|
| Language | Go 1.22+ | Terraform ecosystem is Go; terraform-exec SDK; target audience knows Go |
| TUI framework | charmbracelet/bubbletea | Elm Architecture, dominant Go TUI framework, active ecosystem |
| Terraform SDK | hashicorp/terraform-exec | Official Go SDK, structured output, version detection |
| CLI | spf13/cobra | Standard Go CLI framework, subcommand routing |
| Config | YAML (tfui.yaml) | Human-readable, familiar from pnpm-workspace.yaml |
| Tooling | mise (tasks + slash commands) | Single entry point, no Makefile, consistent dev/CI |
| Release | goreleaser | Cross-compile, homebrew tap, GitHub Releases |

## CLI Interface (preserved from v0.x)

```bash
# New behavior — bare invocation launches TUI
tfui                              # → interactive TUI
tfui --dir ./infra                # → TUI scoped to directory

# Preserved — non-interactive modes (backward compatible)
tfui plan --dir ./infra --mode progress
tfui plan --dir ./infra --mode agent | jq .
tfui apply --dir ./infra --mode spinner
tfui version
```

## Current State (branch: feat/go-rewrite)

### Completed

| Component | Status | Details |
|-----------|--------|---------|
| Go project scaffold | ✅ Done | Module, cobra CLI, config loading |
| Terraform service | ✅ Done | Plan, Apply, StateList, Show, Workspace, WorkspaceList via terraform-exec |
| Risk classification | ✅ Done | 3-tier resource types × action severity, ported from bash |
| Phantom detection | ✅ Done | JSON normalization (strip nulls, sort arrays), deep equality |
| Module grouping | ✅ Done | Extract module paths, group changes, sort alphabetically |
| TUI home screen | ✅ Done | Action-oriented dashboard: Plan, Risk, Blast Radius, Apply, State, Workspaces, Projects |
| Plan view | ✅ Done | Navigable change list, risk badges, phantom highlighting, scroll |
| State view | ✅ Done | Resource browser with substring filtering |
| Apply view | ✅ Done | Running/success/error status feedback |
| Workspaces view | ✅ Done | List with active indicator, navigation |
| Projects view | ✅ Done | Monorepo project picker from tfui.yaml |
| Non-interactive modes | ✅ Done | silent, spinner, progress, agent — all functional |
| Theme system | ✅ Done | Centralized styles package, no inline lipgloss |
| Unit tests | ✅ Done | 926 lines, table-driven tests for risk/phantom/grouping |
| CI pipeline | ✅ Done | GitHub Actions: build, test, vet, coverage, goreleaser release |
| GoReleaser | ✅ Done | linux/darwin × amd64/arm64, homebrew tap |
| Documentation | ✅ Done | Jekyll site: 8 pages (index, getting-started, config, architecture, risk, phantom, blast-radius, CLI ref) |
| Mise tasks | ✅ Done | go:fmt, go:build (depends fmt+lint), go:test, go:lint, go:coverage, go:run |
| Slash commands | ✅ Done | /go-fmt, /go-build, /go-test, /go-lint, /go-coverage, /go-run, /add-view, /add-terraform-feature, /add-style, /add-command |

### Metrics

| Metric | Value |
|--------|-------|
| Go source files | 20 |
| Go source lines | 3,419 |
| Test lines | 926 |
| Docs pages | 8 |
| Mise tasks (Go) | 5 |
| Slash commands | 16 |
| Commits on branch | 14 |

### Not Yet Implemented

| Feature | Priority | Effort |
|---------|----------|--------|
| Risk analysis view (dedicated) | High | 1 week |
| Blast radius visualization | High | 2 weeks |
| Workspace switching (live) | Medium | 3 days |
| Resource detail view (Enter to expand) | Medium | 1 week |
| Attribute-level diffs in plan | Medium | 1 week |
| Help overlay (?) | Medium | 2 days |
| Fuzzy search (fzf-style) | Medium | 1 week |
| Drift detection | Low | 2 weeks |
| Cost estimation (infracost) | Low | 2 weeks |
| Import wizard | Low | 3 weeks |
| Module dependency graph | Low | 3 weeks |
| History/audit log (SQLite) | Low | 2 weeks |
| OpenTofu support | Low | 2 days |

## Project Structure

```
terraform-ui/
├── cmd/tfui/main.go              — CLI entry point, cobra, TUI launch
├── internal/
│   ├── config/config.go          — tfui.yaml loading, project discovery
│   ├── terraform/
│   │   ├── service.go            — Service interface + terraform-exec impl
│   │   ├── parser.go             — Domain types (Resource, PlanChange, Action, RiskLevel)
│   │   ├── risk.go               — Risk classification engine
│   │   ├── phantom.go            — Phantom change detection
│   │   ├── grouping.go           — Module-level change grouping
│   │   ├── risk_test.go          — Risk classification tests
│   │   ├── phantom_test.go       — Phantom detection tests
│   │   └── grouping_test.go      — Module grouping tests
│   └── ui/
│       ├── app.go                — Root bubbletea model, routing, async commands
│       ├── styles/theme.go       — Centralized lipgloss styles
│       ├── components/
│       │   ├── header.go         — Top bar (workspace, dir, count)
│       │   └── statusbar.go      — Bottom bar (keybindings)
│       └── views/
│           ├── home.go           — Action dashboard
│           ├── plan.go           — Plan review (navigable, risk badges)
│           ├── state.go          — State browser (filterable)
│           ├── apply.go          — Apply progress
│           ├── workspaces.go     — Workspace list
│           └── modules.go        — Project picker
├── docs/                         — Jekyll site (8 pages)
├── .claude/commands/             — Slash commands (16)
├── .github/workflows/go.yaml     — CI: build, test, coverage, release
├── .goreleaser.yaml              — Cross-platform release config
├── tfui.example.yaml             — Example monorepo config
├── go.mod / go.sum               — Go dependencies
└── mise.toml                     — Task runner (Go + legacy bash tasks)
```

## Configuration (tfui.yaml)

```yaml
terraform_binary: terraform    # or "tofu" for OpenTofu

projects:
  paths:
    - "modules/*"              # glob patterns for project discovery
    - "envs/**"
```

Placed at repo root. tfui walks up the directory tree to find it.

## Build Pipeline

```
mise run go:fmt       → gofmt -w cmd/ internal/
mise run go:lint      → go vet ./...
mise run go:build     → go build (depends on fmt + lint)
mise run go:test      → go test ./...
mise run go:coverage  → go test -coverprofile + go tool cover
mise run go:run       → go run ./cmd/tfui (dev mode)
```

## Release Strategy

1. Merge `feat/go-rewrite` to `main` as v1.0.0
2. Tag `v1.0.0` triggers goreleaser via CI
3. Goreleaser builds linux/darwin × amd64/arm64 binaries
4. Publishes to GitHub Releases + homebrew tap
5. Old bash tool remains in git history, no longer maintained

## Migration Path for Users

| Current (v0.x bash) | New (v1.0 Go) |
|---------------------|---------------|
| `source lib/tfui.sh` | Deprecated (no library mode) |
| `tfui plan --dir X` | Same — works identically |
| `tfui apply --dir X` | Same — works identically |
| `tfui plan --mode agent` | Same — JSON output preserved |
| Requires: bash + jq | Requires: single binary |
| Install: tarball + PATH | Install: brew / go install / binary |

## Open Questions

1. Should the bash library embedding use case (`source lib/tfui.sh`) get a compatibility shim, or is it fully deprecated?
2. OpenTofu support from day one? (just a `terraform_binary: tofu` config, mostly works already)
3. Plugin/extension system in the future? (custom risk rules, provider-specific views)
