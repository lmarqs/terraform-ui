# CLAUDE.md

## Overview

terraform-ui (tfui) is a k9s-style interactive TUI for Terraform operations. Single Go binary, plugin architecture, BubbleTea framework.

## Terminology

| Term | Definition | Example | Where shown |
|------|-----------|---------|-------------|
| **Project** | Root directory where tfui.yaml lives | `../medprev-cloud-iac` | Header line 1 |
| **Scope** | Selected subdirectory within a project (terraform root module) | `modules/sa-east-1` | Header line 2 |
| **Workspace** | Terraform workspace within a scope | `default`, `staging` | Header line 3 |
| **Context** | The full working state: Project + Scope + Workspace combined | — | Plugin name (umbrella) |

Rules:
- "Context" is ONLY used as the umbrella concept (the plugin managing all three selections)
- Code referring to subdirectory selection uses "scope" (never "context")
- Config YAML key: `scope:` (not `context:`)
- SDK fields: `Scopes`, `ActiveScope`, `ActiveScopeAbs` (in `ProjectContext`)
- Session keys: `scope.active`, `scope.active_abs`, `scope.count`

## Architecture

```
cmd/tfui/              — CLI entry point (cobra commands, plugin registration)
pkg/sdk/               — Public SDK: Plugin interface, Service interface, types, UI primitives
internal/
  config/              — Config loading (tfui.yaml with dot-notation access)
  terraform/           — TerraformService + StaticService (read-only mode)
  source/              — Universal source abstraction (URI resolution, providers)
  macro/               — Macro engine (Driver, tape DSL parser)
  ui/                  — App model, input handling, layout components
  editor/              — Editor integration ($EDITOR at file:line)
  ai/                  — AI provider (Claude via Bedrock, auto-detection)
  plugin/              — Registry (factory pattern, config-driven enablement)
  logging/             — Structured logger setup
plugins/               — All features as plugins (one dir per plugin)
  context/             — Context dashboard: shows Project + Scope + Workspace (FormFrame)
  scope/               — Scope picker: select subdirectory within project
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
| `Project` | `ProjectContext` | Immutable: dir, discovered scopes, active scope |
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
    Init(ctx *Context) tea.Cmd
    Update(msg tea.Msg) (Plugin, tea.Cmd)
    View(width, height int) string
    Configure(cfg map[string]interface{}) error
    Ready() bool
}
```

Optional interfaces: `Activatable` (work on navigation), `Countable` (item counts for border title), `Hintable` (state-aware key hints for status bar).

### Plugin Routing (`internal/plugin/registry.go`)

Plugins are **invocation-agnostic** — they don't know their keybinding, menu position, or how they're reached. Routing metadata is external:

```go
type PluginMeta struct {
    Keybinding  string // single key, empty = not in home menu
    MenuVisible bool   // whether to show in home menu
}
```

Registration happens at the entry point (`cmd/tfui/main.go`):
```go
registry.RegisterFactory("state", tfuistate.New, plugin.PluginMeta{
    Keybinding: "s", MenuVisible: true,
})
```

The home menu and command bar are independent consumers of registry metadata. This keeps plugins focused on their domain logic with zero coupling to navigation.

### Service Interface (`pkg/sdk/service.go`)

All terraform operations: `Plan`, `Apply`, `StateList`, `Show`, `StateRm`, `StateMove`, `Import`, `Taint`, `Untaint`, `Validate`, `Output`, `Refresh`, `Init`, `Workspace*`, `WithDir`.

Two implementations:
- `TerraformService` — wraps terraform-exec, calls real terraform binary
- `StaticService` — pre-loaded data, read-only (mutating methods return `ErrReadOnly`)

### Source Abstraction (`internal/source/`)

Universal I/O layer for loading external data (plan, state, macros). All external inputs resolve through the same pipeline.

```
Consumer (LoadPlan, LoadState, tape parser)
    ↓
Resolver (URI dispatch)
    ↓
Provider (LocalProvider, StdinProvider, future: HTTP, S3)
```

**URI resolution rules (strict, no heuristics):**
- `-` → stdin (only one flag per invocation)
- `/path` → absolute local path
- `./path` or `../path` → relative local path (relative to CWD)
- `scheme://...` → dispatches to matching provider (RFC 3986 scheme validation)
- `file://...` → normalized to local path
- Anything else → **error** with actionable suggestion

**Providers implement:**
```go
type Provider interface {
    Scheme() string
    Read(ctx context.Context, uri string) ([]byte, error)
}
```

**Extending:** register new providers (HTTP, S3, GCS) without changing consumers or existing providers (Open/Closed).

### Macro Engine (`internal/macro/`)

Programmatic TUI driver + tape DSL for automated testing and CI.

**Driver** — synchronous BubbleTea model controller:
```go
d := macro.NewDriver(app, 80, 24)
d.Init()
d.SendKey("p")
d.WaitUntil(func(v string) bool { return strings.Contains(v, "create") }, 5*time.Second)
```

**Tape DSL** — line-oriented commands:
```
key p
wait ready
wait view to add
assert view create
screenshot /tmp/plan.txt
resize 120 40
sleep 500ms
```

### CLI: Read-Only Mode (`--plan`, `--state`)

```bash
tfui --plan ./plan.json                      # local file
tfui --plan /absolute/path/plan.json         # absolute
tfui --state ../terraform.tfstate            # relative
terraform show -json tfplan.out | tfui --plan -   # stdin pipe
tfui --plan ./plan.json --state ./state.json      # both
```

When `--plan` or `--state` provided:
- `StaticService` replaces `TerraformService`
- All mutating operations return `ErrReadOnly`
- Header shows `[read-only]` badge
- Mutating hints hidden from status bar

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
scope:
  paths: ["modules/*"]         # glob patterns for monorepo scope discovery
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

- **TDD workflow**: write a failing test first, then implement the fix/feature
- 100% coverage on all packages excluding `cmd/` glue
- Table-driven tests preferred
- Mock services implement `sdk.Service` with no-op methods
- Use `t.TempDir()` for filesystem tests, `t.Setenv()` for env var tests
- Test file naming: `*_test.go` in same package (white-box)
- Integration tests in `tests/integration/`

### Plugin Patterns

Every plugin follows the same shape:

```go
type Plugin struct { svc sdk.Service; log *slog.Logger; guard *sdk.ScopeGuard; pins *sdk.PinService; ... }
func New(svc sdk.Service) sdk.Plugin { ... }
func (p *Plugin) Init(ctx *sdk.Context) tea.Cmd { p.guard = sdk.NewScopeGuard(ctx.Session, ctx.Service); ... }
func (p *Plugin) Activate() tea.Cmd { /* use guard.Check(), load data */ }
func (p *Plugin) Update(msg tea.Msg) (sdk.Plugin, tea.Cmd) { /* handle result msgs + keys */ }
func (p *Plugin) View(w, h int) string { /* switch on status */ }
```

Plugins are registered with external metadata — they never declare their own keybinding or menu visibility.

### SDK Utilities (pkg/sdk/ and pkg/sdk/ui/)

Use these instead of reimplementing common patterns:

| Utility | Location | Purpose | Used by |
|---------|----------|---------|---------|
| `ScopeGuard` | `pkg/sdk/scope_guard.go` | Detect scope changes in Activate(), auto-rescope service | state, plan, output, validate, workspaces, apply |
| `PinService` | `pkg/sdk/pin_service.go` | Toggle/query/bulk-set pinned resource addresses | state, plan |
| `Status` | `pkg/sdk/status.go` | Shared enum (Idle/Loading/Done/Error) with predicates | all plugins |
| `Cursor` | `pkg/sdk/ui/cursor.go` | Index selection + bounds + viewport windowing | plan, output, validate, workspaces, scope |
| `ExpandSet` | `pkg/sdk/ui/expand.go` | Track expanded indices in lists | plan, validate, phantom, blastradius |
| `FuzzyFilter[T]` | `pkg/sdk/ui/filter.go` | fzf matching + score-sorted results | state, output |

**Rules:**
- Use `ScopeGuard` instead of reading `SessionKeyActiveScopeAbs` manually
- Use `PinService` instead of raw `session.Set("terraform.pinned", ...)`
- Use `Cursor.VisibleWindow(h)` instead of manual startIdx/endIdx calculation
- Use `FuzzyFilter[T]` instead of importing `fzf/src/algo` directly
- Reference implementation: `plugins/state/` demonstrates all SDK primitives

### Navigation Stack (Android-style)

Plugins use a nested navigation stack instead of boolean state flags. Input always routes to the topmost frame — no key leakage between modes.

```
App Stack: [Home] → [State Plugin]
                      └── Plugin Stack: [List] → [Filter]
                                                → [Inspect] → [Confirm]
```

**Rules:**
- Input goes to the deepest leaf frame only
- `esc` always pops the innermost frame (universal "back")
- `q` pops to app root (deactivate plugin)
- `:` side-navigates at app level (replaces plugin)
- Each frame declares its own `Hints() []KeyHint` — rendered automatically

**SDK types** (`pkg/sdk/`):
- `Frame` interface: `ID()`, `Update(msg) (Frame, Cmd)`, `View(w,h)`, `Hints()`
- `Stack`: LIFO container with `Push`, `Pop`, `Update`, `View`, `Hints`
- `Stackable` interface: optional on plugins, returns their internal `*Stack`

**Reusable frames** (`pkg/sdk/frames/`):
- `FilterFrame`: consumes ALL printable keys as text input; only esc/enter/arrows escape
- `InspectFrame`: scrollable detail + configurable action keys
- `ConfirmFrame`: blocks all input except y/n/esc

**Frame lifecycle:**
- Return `nil` from `Update` → frame is popped (back navigation)
- Return a different `Frame` → in-place replacement
- Return self → no change

**Migration:** plugins implement `Stackable` to opt in. Legacy plugins continue using direct `Update` delegation unchanged.

### UX Model (k9s-inspired)

- **`:` command mode**: type plugin name to switch views. Tab autocomplete.
- **`/` filter mode**: fzf-style fuzzy filter. `esc` exits.
- **`space` pin**: toggle pin on selected resource. Pinned = apply/plan target.
- **`enter` / `i` inspect**: show detail view with expanded values.
- **`d` delete**: remove from state — triggers confirmation prompt.
- **`e` edit**: opens $EDITOR at resource's .tf file:line.
- **`m` move**: rename resource address in state — text input + confirmation.
- **`t` taint**: mark for recreation — triggers confirmation.
- **`T` untaint**: remove taint mark — triggers confirmation.
- **`n` import**: import existing resource — text input for ID + confirmation.
- **`!` batch**: open batch action palette (only when pins > 0).
- **`r` refresh**: reload data from terraform.
- **`ctrl+w` wrap**: toggle line wrapping.
- **`←→` pan**: horizontal scroll (10 chars/press, when wrap is off).
- **`q`** exits plugin to home. **`esc`** exits current sub-state (scoped).

### Keybinding Ergonomics

**Convention:**
- Capital letter = non-terraform feature (Context `C`, Risk `R`, Phantom `P`, Blast Radius `B`)
- Lowercase = terraform operation (state `s`, plan `p`, apply `a`, workspaces `w`)
- `ctrl+char` = modifier actions within a view (ctrl+w wrap, ctrl+s screen capture)
- Punctuation = mode/overlay triggers (`/` filter, `!` actions, `?` AI explain, `:` command)

Redundant keybindings exist for keyboard layout accessibility, but hints show only the primary key:

| Action | Primary (shown in hint) | Alias (not shown) | Scope |
|--------|------------------------|-------------------|-------|
| Inspect/expand | `Enter` | `i` | All list views |
| Back to home | `q` | `esc` (when no sub-state) | Global |
| Exit sub-state | `Esc` | — | Filter, detail, confirm |
| Pin (apply target) | `Space` | — | Resources/changes |

Rules:
- `enter`/`i` always means inspect — never overloaded for other actions (e.g., not used as pin toggle)
- `space` always means pin — never overloaded for expand/inspect
- `q` shown in hints at plugin top-level; `esc` shown only in sub-state hints
- Plugins must NOT start in filter mode by default — user opts in with `/`

### Action Model (cursor vs batch)

**Core rule: direct keys always act on the cursor item. Batch operations go through `!` palette only.**

| Layer | Keys | Scope | Mental model |
|-------|------|-------|--------------|
| Navigate | `↑↓`, `Enter`, `/`, `q`, `Esc` | — | Move around, look at things |
| Act (single) | `d`, `e`, `t`, `T`, `m`, `n` | Cursor | Do something to this one thing |
| Batch | `!` (palette) | Pinned set | Do something to all pinned items |

**Rules:**
- Direct action keys NEVER read the pinned set — always cursor only
- `!` is hidden from hint bar when no pins exist (nothing to batch)
- `!` replaces the hint bar with batch action keys, list stays visible
- Detail/inspect frame has no `!` — single resource only
- Destructive batch ops always show confirmation with resource count

**Pin semantics:**
- PRIMARY purpose: scoping `plan` and `apply` to specific resources
- SECONDARY: enabling batch state actions via `!` palette
- Pins are persistent (survive view switches and sessions) — NOT ephemeral marks

**Design rationale (benchmarked against k9s, vim, ranger, lazygit, mutt):**
- k9s marks are ephemeral (cleared after action) — ours persist, so the k9s "marks supersede cursor" model is dangerous here
- This app follows the lazygit model: batch operations are separate, named commands — never an implicit side-effect of a single-item keybinding
- Prevents "forgot I had pins, accidentally batch-deleted" scenarios

### Hint Bar Design

**Rules:**
- Always 1 line. Never 2 lines.
- One key per entry: `d delete`, `t taint`, `T untaint` (no grouping, no slash notation)
- Context-sensitive: content changes per frame/state, line count stays at 1
- Show only the most relevant keys for the current state
- No novel notation patterns (no `d/D`, no `(t/T) taint/untaint`)
- All notation must have precedent in established TUI apps (k9s, vim, ranger)
- Display preferences (`^w` wrap) shown only in detail/inspect frame — not in list frame
- List frame hints = navigation + actions that change state
- Detail frame hints = display controls + single-item actions
- Dynamic hints show the target state (what pressing the key switches TO): `^t tree` means "press to enter tree mode"

### UX Anti-patterns (do NOT introduce)

- Shift+letter = batch version of same action (zero precedent in TUI apps)
- Implicit batch based on pin state (dangerous with persistent pins)
- Novel hint bar notation (slash grouping, parenthetical pairs)
- 2-line hint bars (costs screen real estate, no established precedent)
- Auto-batch for non-destructive actions (creates inconsistency)

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
mise run dev              # Run TUI in development mode
mise run fmt              # Format source files (gofmt)
mise run check:lint       # Full lint suite (golangci-lint v2)
mise run check:vet        # Quick go vet
mise run test:unit        # Unit tests (produces reports/junit.xml)
mise run test:coverage    # Coverage enforcement (90% threshold)
mise run test:integration # Integration tests (requires terraform)
mise run build            # Cross-platform binaries (goreleaser snapshot)
mise run test:macro       # Macro tapes against built binary
mise run setup            # Install CI deps (npm + gotestsum)
```

### Mise Task Convention

Tasks are namespaced by pipeline stage:

| Namespace | Purpose | Examples |
|-----------|---------|----------|
| `check:*` | Static analysis (no build) | `check:lint`, `check:vet` |
| `build` | Produce artifacts | `build` (goreleaser snapshot) |
| `test:*` | Verify correctness | `test:unit`, `test:coverage`, `test:integration`, `test:macro` |
| `release` | Publish (CI only) | `release` (semantic-release) |
| _(top-level)_ | Developer tools | `run`, `fmt`, `setup` |

Rules:
- Each task does ONE thing — no implicit dependencies between stages
- CI orchestrates execution order, not mise
- All tasks are callable standalone (no hidden prerequisites)
- Task names map directly to what CI workflows call

### Toolchain (managed by mise)

| Tool | Version | Purpose |
|------|---------|---------|
| `go` | 1.25 | Build + test |
| `golangci-lint` | 2.12 | Lint (6 linters: importas, govet, errcheck, staticcheck, unused + goimports formatter) |
| `goreleaser` | 2.15 | Cross-platform builds + release archives |
| `terraform` | 1.14 | Integration tests |
| `node` | 22 | semantic-release (CI only) |

## CI/CD Pipeline

### Flow

```
PR / push to main → main.yaml
  ├── build.yaml    lint → unit tests (ubuntu+macos) → coverage → binaries
  ├── test.yaml     macro tapes + integration tests (against built artifacts)
  └── release.yaml  semantic-release → goreleaser (if new version)
```

### Stage Responsibilities

| Stage | File | What it does | What it produces |
|-------|------|-------------|-----------------|
| **Build** | `build.yaml` | Lint, unit tests, coverage, compile | `dist/` artifact (4 binaries) |
| **Test** | `test.yaml` | Blackbox: macro tapes + integration | Pass/fail (no artifacts) |
| **Release** | `release.yaml` | Version + publish binaries | Git tag, CHANGELOG.md, GitHub release with archives |

### How semantic-release and goreleaser work together

goreleaser is invoked by semantic-release via `@semantic-release/exec.publishCmd`:

1. semantic-release analyzes conventional commits since last release
2. If releasable: bumps version, writes CHANGELOG.md + VERSION, creates git tag + GitHub release
3. `publishCmd: "goreleaser release --clean"` runs — builds 4 binaries and uploads archives to the GitHub release

Single orchestrator (semantic-release), single config (`.releaserc`), no detection logic.

Key config:
- `.releaserc`: `publishCmd` invokes goreleaser (it has the tag context already)
- `.goreleaser.yaml`: `release.mode: append` (SR owns the release), `changelog.disable: true` (SR owns the changelog)

### Versioning

| Context | Version source | Example |
|---------|---------------|---------|
| goreleaser build (CI) | git tag via ldflags | `0.40.0` |
| `go install ...@v0.40.0` | module metadata (ReadBuildInfo) | `v0.40.0` |
| `go run ./cmd/tfui` (dev) | fallback | `0.0.0-SNAPSHOT` |
| goreleaser snapshot | git describe | `0.39.0-SNAPSHOT-2d0d9dc` |

Resolution chain in `cmd/tfui/main.go`:
```
ldflags (-X main.version=...) → debug.ReadBuildInfo().Main.Version → "0.0.0-SNAPSHOT"
```

### Adding a new CI check

1. Create a mise task in `mise.toml` under the appropriate namespace
2. Call it from the relevant workflow stage (`build.yaml` for fast checks, `test.yaml` for artifact-dependent tests)
3. Workflow files stay thin — just `mise run <task>`

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

## Agents (`.claude/agents/`)

| Agent | Purpose | When to use |
|-------|---------|-------------|
| `test-writer` | Generate table-driven tests | **MUST be invoked BEFORE any implementation edit** — failing test first, always |
| `code-checker` | Audit CLAUDE.md code conventions | Before commits, during PR review, after large refactors |
| `ux-checker` | Validate hint placement and UX rules | Changes to `View()`, `Hints()`, frames, or new plugins |
| `macro-runner` | Run macro tapes to verify UI rendering | After modifying `View()`, layout, or plugin navigation |
| `architect` | Design implementation plans | New plugins or cross-cutting features (before coding) |
| `security-checker` | Terraform-specific security audit | PRs touching terraform service, state display, or AI integration |

Agents run in isolation and can be spawned in parallel. Unlike commands, they don't need conversation context and produce self-contained reports.

## Important Rules

- **TDD is non-negotiable**: spawn `test-writer` agent to produce a failing test BEFORE writing any implementation code. Never edit production files without a failing test already in place.
- Plugins import ONLY `pkg/sdk` — never `internal/`
- All state mutations go through `TerraformContext` (thread-safe)
- Destructive ops require staleness check + user confirmation
- AI features check `ctx.AI != nil` before offering
- Config getters ALWAYS take a default value — no nil panics
- Session keys use dot-notation namespacing
- Editor integration uses `tea.ExecProcess` for proper terminal handoff
