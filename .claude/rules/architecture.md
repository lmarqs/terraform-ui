---
description: "Core SDK abstractions, service interface, source layer, and macro engine internals"
globs: ["pkg/sdk/**", "internal/**"]
---

# Architecture Details

## Plugin Context (`pkg/sdk/context.go`)

Passed to `Init()` — gives each plugin its dependencies:

```go
type Context struct {
    WorkingDir string
    Workspace  string
    Service    Service
    Logger     *slog.Logger
    Pins       *PinService
    Options    *ResolvedOptions
}
```

## Event Bus (`pkg/sdk/bus.go`, `pkg/sdk/events.go`)

Typed pub/sub. Plugins subscribe by implementing handler interfaces.

Events: `ChdirChangedEvent`, `WorkspaceChangedEvent`, `WorkspaceCreatedEvent`, `PlanCompletedEvent`, `PinsChangedEvent`, `PlanInvalidatedEvent`, `LockDetectedEvent`, `LockClearedEvent`, `StateRefreshedEvent`

Handler interfaces: `ChdirHandler`, `WorkspaceHandler`, `PlanCompletedHandler`, `PinsHandler`, `PlanInvalidatedHandler`, `LockDetectedHandler`, `LockClearedHandler`, `StateRefreshedHandler`

Note: `WorkspaceCreatedEvent` has no handler interface — the app converts it to `WorkspaceChangedEvent` internally.

Flow: App dispatches events to all plugins implementing the matching handler interface.

## ResolvedOptions (`pkg/sdk/options.go`)

```go
type ResolvedOptions struct {
    VarFiles  []string
    Vars      map[string]string
    ExtraArgs []string
}
```

Shared via `Context.Options`. Used by `BuildPlanOptions` / `BuildApplyOptions`.

## Plugin Interface (`pkg/sdk/plugin.go`)

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

Optional interfaces: `Activatable`, `Busy`, `Countable`, `Hintable`, `Pinnable`, `Stackable`.

## Plugin Routing (`internal/plugin/registry.go`)

Plugins are invocation-agnostic. Routing metadata is external:

```go
type NavBehavior int
const (
    NavReplace NavBehavior = iota  // lateral switch, no history
    NavPush                        // preserves return context
)

type PluginMeta struct {
    Keybinding  string
    MenuVisible bool
    Nav         NavBehavior
}
```

Registration in `cmd/tfui/main.go`:
```go
registry.RegisterFactory("state", tfuistate.New, plugin.PluginMeta{
    Keybinding: "s", MenuVisible: true,
})
registry.RegisterFactory("workspaces", tfuiworkspaces.New, plugin.PluginMeta{
    Keybinding: "w", MenuVisible: true, Nav: plugin.NavPush,
})
```

## App Navigation (`internal/ui/app.go`)

Central routing with three private methods:

```go
navigateTo(p)    // checks NavBehaviorFor(p.ID()), saves returnTo if NavPush
navigateBack()   // restores returnTo as activePlugin, logs transition
popIfPushed(cmd) // called by event handlers; pops NavPush plugin if active
```

`returnTo sdk.Plugin` — single-level return address. Set by `NavPush` transitions AND workflow transitions (e.g., plan→apply). Consumed by:
- Event handlers (`ChdirChangedEvent`, `WorkspaceChangedEvent`) via `popIfPushed`
- `DeactivateMsg` handler (esc cancel path)

Workflow transitions (plan→apply): The app sets `returnTo` manually when a plugin triggers a workflow to another plugin. This is distinct from `NavPush` metadata — it's a runtime decision. Example: `ApplyRequestMsg` handler sets `returnTo = plan` before activating apply.

Naming rationale (benchmarked against React Router, iOS UIKit, Flutter Navigator, lazygit, k9s):
- `NavBehavior` over "NavMode" — "behavior" fits static metadata; "mode" implies runtime toggle
- `NavPush`/`NavReplace` — universal terms across all frameworks studied
- `returnTo` over "previousPlugin" — communicates intent (destination), not time (temporal)

## Inter-Plugin Navigation (`pkg/sdk/plugin.go`)

```go
type NavigateMsg struct { PluginID string }  // request app navigate to plugin
type DeactivateMsg struct{}                   // request app deactivate current plugin
```

Plugins emit `NavigateMsg` to delegate to another plugin (e.g., context → workspaces). The app applies the target's `NavBehavior`. This keeps plugins decoupled — they never hold references to each other.

## View Delegation for Stackable Plugins

The app calls `plugin.View(width, height)` directly — NOT `plugin.Stack().View()`. Stackable plugins must handle frame delegation themselves:

```go
func (e *Plugin) View(width, height int) string {
    if top := e.stack.Peek(); top != nil && top.ID() != "list" {
        return top.View(width, height)
    }
    // ... default list rendering
}
```

The frame stack routes **input** (via `Stack.Update()`) but NOT rendering. Each plugin's `View()` must check which frame is active and delegate accordingly.

## Apply Plugin Navigation Model

Apply is NOT on the home menu (`MenuVisible: false`). It's only reachable through plan's `a` key.

Flow: Plan → `ApplyRequestMsg` → app sets `returnTo=plan`, activates apply with `RequestApply()` → apply shows confirmation → user confirms → apply runs → `PlanInvalidatedEvent` emitted on success.

All `esc`/cancel paths in apply emit `DeactivateMsg`, which the app handles by checking `returnTo` and navigating back to plan.

Apply resets to idle on `Activate()` if in a terminal state (error/done), preventing stale state when re-entered.

## Targeted Apply (terraform constraint)

Terraform does NOT support `-target` with a saved plan file. The `ExecService.Apply()` handles this:
- When targets are present: runs `terraform apply -target=X` directly (no plan file)
- When no targets: applies the saved plan file via `terraform apply tfplan.out`

This is because the plan file already encodes ALL changes — you cannot subset them at apply time.

## Service Interface (`pkg/sdk/service.go`)

All terraform operations: `Plan`, `Apply`, `StateList`, `Show`, `StateRm`, `StateMove`, `Import`, `Taint`, `Untaint`, `Validate`, `Output`, `Refresh`, `Init`, `Workspace*`, `ForceUnlock`, `Version`, `WithDir`.

Two implementations:
- `ExecService` — wraps terraform-exec, uses ServiceCache for reads (service.go, state_ops.go, workspace_ops.go)
- `MacroService` — records commands as sdk.Command, reads from ServiceCache, never executes (macro_service.go)

`ServiceCache` (service_cache.go) is a typed, source-aware cache pre-seeded from `--plan`/`--state` flags at startup. Three source kinds: file (re-reads on invalidate), stdin (immutable), exec (cleared on invalidate).

Cache invalidation rules:
- State-mutating operations (`StateRm`, `StateMove`, `Import`, `Taint`, `Untaint`) auto-invalidate state cache internally
- `Refresh()` and `WorkspaceSelect()` invalidate all cached data
- Plugins bypass cache via `StateList(ctx, sdk.SkipCache())` for explicit refresh (ctrl+r)
- `StateListOption` uses a variadic functional-options pattern — keeps cache semantics out of the interface contract

## Source Abstraction (`internal/source/`)

Pure byte-resolution layer. Resolves URIs to raw bytes.

URI rules (strict):
- `-` → stdin
- `/path` → absolute local path
- `./path` or `../path` → relative local path
- `scheme://...` → dispatches to matching provider
- `file://...` → normalized to local path
- Anything else → error with actionable suggestion

```go
type Provider interface {
    Scheme() string
    Read(ctx context.Context, uri string) ([]byte, error)
}
```

Domain parsing in `internal/terraform/loader.go`: `LoadPlan([]byte)` and `LoadState([]byte)`.

## Macro Engine (`internal/macro/`)

Programmatic TUI driver + tape DSL for automated testing.

Driver:
```go
d := macro.NewDriver(app, 80, 24)
d.Init()
d.SendKey("p")
d.WaitUntil(func(v string) bool { return strings.Contains(v, "create") }, 5*time.Second)
```

Tape DSL:
```
key p
wait ready
wait view to add
assert view create
screenshot /tmp/plan.txt
resize 120 40
sleep 500ms
```

## SDK Utilities (pkg/sdk/ and pkg/sdk/ui/)

| Utility | Location | Purpose |
|---------|----------|---------|
| `EventBus` | `pkg/sdk/bus.go` | Typed event dispatch |
| `PinService` | `pkg/sdk/pin_service.go` | Shared via `Context.Pins` |
| `ResolvedOptions` | `pkg/sdk/options.go` | Var-files, vars, extra-args |
| `Status` | `pkg/sdk/status.go` | Idle/Loading/Done/Error enum |
| `Cursor` | `pkg/sdk/ui/cursor.go` | Index selection + viewport windowing |
| `ExpandSet` | `pkg/sdk/ui/expand.go` | Track expanded indices |
| `FuzzyFilter[T]` | `pkg/sdk/ui/filter.go` | fzf matching + score-sorted results |
| `Timer` | `pkg/sdk/ui/timer.go` | Elapsed time tracking with tick integration |
| `Tree` | `pkg/sdk/ui/tree/` | Hierarchical rendering with expand/collapse |

Rules:
- Implement `ChdirHandler` to react to chdir changes
- Use `Cursor.VisibleWindow(h)` instead of manual calculation
- Use `FuzzyFilter[T]` instead of importing fzf directly
- Use `Timer` for elapsed time display during long operations
- Reference implementation: `plugins/state/` (list/filter/tree), `plugins/plan/` (tree + inspect frame)

## AppContext (`pkg/sdk/app_context.go`)

Root application state container, partitioned by domain:

```go
type AppContext struct {
    Project   ProjectContext    // immutable project info (Dir, Members, Chdir, ChdirAbs)
    Config    *ConfigContext    // dot-notation config access
    Terraform *TerraformContext // workspace, pinned targets, cached state/plan, service
    UI        *UIContext        // dimensions, active plugin, input mode
    Cache     *CacheContext     // generic TTL cache
    AI        AIProvider        // nil if disabled
    Logger    *slog.Logger
}
```

`TerraformContext` provides thread-safe pin management (`Pin`, `Unpin`, `IsPinned`, `PinnedTargets`) and cached `TerraformState`/`TerraformPlan` with loading/error metadata.

## Overlay + Input System (`pkg/sdk/overlay.go`, `pkg/sdk/input.go`)

Modal interaction patterns for user prompts and confirmations:

```go
type Overlay interface {
    ID() string
    Open() tea.Cmd
    Update(msg tea.Msg) (Overlay, tea.Cmd)
    View(width, height int) string
    Hints() []KeyHint
}
```

Input request/response protocol for plugins needing user input:
- `InputRequest` — specifies mode (Text/Bool/Select/Filter), prompt, callback
- `RequestInputMsg` — wraps request as `tea.Msg` for dispatch
- `InputResponseMsg` — delivers answer back to plugin
- Helpers: `InputConfirm(prompt, onYes)`, `InputText(prompt, default, onSubmit)`, `InputSelect(prompt, options, onSelect)`

## PluginAction System (`pkg/sdk/action.go`)

CLI/REPL-callable operations exposed by plugins (e.g., `tfui state mv`, `:state mv`):

```go
type PluginAction struct {
    Name        string
    Description string
    Args        []ArgDef   // positional arguments
    Flags       []FlagDef  // flag parameters
    Run         func(ctx *AppContext, args ActionArgs) error
}
```

`ActionArgs` provides `GetArg(index)`, `GetFlag(name, default)`, `HasFlag(name)`.

## Additional Frames (`pkg/sdk/frames/`)

Beyond `FilterFrame`, `InspectFrame`, `ConfirmFrame`:

- `ActionFrame` — displays running operation progress with cancel support
- `FormFrame` — multi-field form with validation and submit/cancel
