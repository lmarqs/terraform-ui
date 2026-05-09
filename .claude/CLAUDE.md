# CLAUDE.md

## Overview

terraform-ui (tfui) is a k9s-style interactive TUI for Terraform operations. Single Go binary, plugin architecture, BubbleTea framework.

## Architecture

```
cmd/tfui/              — CLI entry point (cobra commands, plugin registration)
pkg/sdk/               — Public SDK: Plugin interface, Service interface, types, UI primitives
internal/
  config/              — Config loading (tfui.yaml with dot-notation access)
  terraform/           — TerraformService implementation (terraform-exec wrapper)
  ui/                  — App model, input handling, layout components
  editor/              — Editor integration ($EDITOR at file:line)
  ai/                  — AI provider (Claude via Bedrock, auto-detection)
  plugin/              — Registry (factory pattern, config-driven enablement)
  logging/             — Structured logger setup
plugins/               — All features as plugins (one dir per plugin)
  context/             — Project scope picker (monorepo support)
  state/               — State browser (list, inspect, pin, delete, move, edit)
  plan/                — Plan review (diff view, expand attributes, risk)
  apply/               — Apply executor
  workspaces/          — Workspace management
  repl/                — Terraform console (REPL)
  output/              — Terraform outputs viewer
  validate/            — Terraform validate
  risk/                — Risk classification (decorates plan)
  phantom/             — Phantom change detection (decorates plan)
  blastradius/         — Blast radius visualization
  init/                — Config generator (CLI only)
tests/
  integration/         — Integration tests with real terraform
  fixtures/            — Real terraform projects for testing
```

## Core Abstractions

### AppContext (`pkg/sdk/app_context.go`)

Single source of truth for the application, partitioned by domain:

| Field | Type | Purpose |
|-------|------|---------|
| `Project` | `ProjectContext` | Immutable: dir, discovered contexts, active context |
| `Config` | `*ConfigContext` | Read-only: dot-notation access to tfui.yaml |
| `Terraform` | `*TerraformContext` | Mutable: workspace, pinned targets, cached state/plan |
| `UI` | `*UIContext` | Mutable: window size, active plugin, input mode |
| `Cache` | `*CacheContext` | Generic TTL cache |
| `AI` | `AIProvider` | AI service (nil if disabled) |
| `Logger` | `*slog.Logger` | Structured logger |

### Plugin Interface (`pkg/sdk/plugin.go`)

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

Optional interfaces: `Activatable` (work on navigation).

### Service Interface (`pkg/sdk/service.go`)

All terraform operations: `Plan`, `Apply`, `StateList`, `Show`, `StateRm`, `StateMove`, `Import`, `Taint`, `Untaint`, `Validate`, `Output`, `Refresh`, `Init`, `Workspace*`, `WithDir`.

### Config (`tfui.yaml`)

```yaml
terraform:
  bin: tofu                    # auto-detects if empty (tofu → terraform)
cache:
  staleness_threshold: 5m      # prompt before destructive ops on stale data
ai:
  enabled: true
  provider: ""                 # auto-detect (bedrock if AWS creds, anthropic if API key)
  model: ""                    # auto-detect per provider
  region: us-east-1            # for Bedrock
context:
  paths: ["modules/*"]         # glob patterns for monorepo discovery
plugins:
  risk:
    enabled: true
```

Access via `ConfigContext.GetString("terraform.bin", "")`, `GetBool("ai.enabled", false)`, etc.

## Conventions

### Commits

Conventional commits: `feat:`, `fix:`, `test:`, `ci:`, `refactor:`, `docs:`, `chore:`

### Go Package Layout

| Package | Purpose | Import rule |
|---------|---------|-------------|
| `pkg/sdk` | Public contract for plugins | Plugins import ONLY this |
| `internal/*` | App internals | Not importable by plugins |
| `plugins/*` | Feature implementations | Import `pkg/sdk` only |
| `cmd/tfui` | CLI glue | Imports everything |

### Naming

| Pattern | Example | Rule |
|---------|---------|------|
| Plugin IDs | `"state"`, `"plan"` | lowercase, single word |
| Plugin packages | `plugins/state/` | match the ID |
| Messages | `StateListMsg`, `PlanResultMsg` | `{Subject}{Verb}Msg` |
| Session keys | `"terraform.pinned"` | dot-separated namespace |
| Config keys | `"ai.model"` | dot-separated, yaml structure |

### Imports

- ALL plugin imports use `tfui` prefix: `tfuistate`, `tfuiplan`, `tfuiapply`, etc.
- BubbleTea always aliased as `tea`
- Enforced by `golangci-lint` with `importas` linter (`.golangci.yaml`)
- Import block order: stdlib, external, internal
- Run `golangci-lint run` to check (or `mise run lint`)

### Testing

- 100% coverage on all packages excluding `cmd/` glue
- Table-driven tests preferred
- Mock services implement `sdk.Service` with no-op methods
- Use `t.TempDir()` for filesystem tests, `t.Setenv()` for env var tests
- Test file naming: `*_test.go` in same package (white-box)
- Integration tests in `tests/integration/`

### Plugin Patterns

Every plugin follows the same shape:

```go
type Status int
const (StatusIdle, StatusLoading, StatusDone, StatusError)

type Plugin struct { svc sdk.Service; log *slog.Logger; session *sdk.Session; status Status; ... }
func New(svc sdk.Service) sdk.Plugin { ... }
func (p *Plugin) Activate() tea.Cmd { /* respect context, load data */ }
func (p *Plugin) Update(msg tea.Msg) (sdk.Plugin, tea.Cmd) { /* handle result msgs + keys */ }
func (p *Plugin) View(w, h int) string { /* switch on status */ }
```

### UX Model (k9s-inspired)

- **`:` command mode**: type plugin name to switch views. Tab autocomplete.
- **`/` filter mode**: fzf-style fuzzy filter. `esc` exits.
- **`space` pin**: toggle pin on selected resource. Pinned shown with `*`.
- **`enter` inspect**: show detail view with expanded values.
- **`d` delete**: destructive — triggers confirmation prompt.
- **`e` edit**: opens $EDITOR at resource's .tf file:line.
- **`r` refresh**: reload data from terraform.
- **`ctrl+w` / `w` wrap**: toggle line wrapping.
- **`←→` pan**: horizontal scroll (10 chars/press).
- **`esc`** exits current level. **`q`** exits to home.

### Detail/Inspect View

- Shows expanded attribute values (JSON)
- Context actions remain available: `space` pin, `d` delete, `e` edit
- Scroll indicator `[n/total]` when content overflows
- `[pinned]` indicator shown if resource is pinned

### Staleness Guard

Before destructive operations (apply, state rm, state mv, import), check data freshness:
- Threshold: `cache.staleness_threshold` config (default 5m)
- If stale: prompt user "State is Xm old. Refresh first? (y/n)"
- If nil: prompt "No state loaded. Load first?"

### AI Integration

- Provider auto-detection: `ANTHROPIC_API_KEY` → direct API, AWS creds → Bedrock
- Default model: `us.anthropic.claude-sonnet-4-6-v1` (Bedrock) or `claude-sonnet-4-6-20250514` (direct)
- `?` key triggers AI explain on selected item (streaming response)
- AI features gracefully degrade if no credentials available

## Development Workflow

```bash
go build ./...           # Build everything
go test ./...            # Run all tests
go test -race ./...      # Test with race detector
mise run build           # Build binary with version
mise run test            # Run tests via mise
mise run run             # Run TUI in dev mode
```

## Key Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/charmbracelet/bubbletea` | TUI framework |
| `github.com/charmbracelet/lipgloss` | Terminal styling |
| `github.com/hashicorp/terraform-exec` | Terraform CLI wrapper |
| `github.com/hashicorp/terraform-json` | Terraform JSON types |
| `github.com/junegunn/fzf` | Fuzzy search algorithm |
| `github.com/spf13/cobra` | CLI framework |
| `github.com/anthropics/anthropic-sdk-go` | Claude AI (Bedrock + direct) |
| `gopkg.in/yaml.v3` | YAML config parsing |

## Important Rules

- Plugins import ONLY `pkg/sdk` — never `internal/`
- All state mutations go through `TerraformContext` (thread-safe)
- Destructive ops require staleness check + user confirmation
- AI features check `ctx.AI != nil` before offering
- Config getters ALWAYS take a default value — no nil panics
- Session keys use dot-notation namespacing
- Editor integration uses `tea.ExecProcess` for proper terminal handoff
