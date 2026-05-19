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

Shared via `Context.Options` as a pointer — plugins store the pointer at `Init()` and read fields at call time. The app mutates `VarFiles` and `Vars` in-place on workspace/chdir changes (via `resolveOptions`). `ExtraArgs` is immutable after boot (CLI `--` passthrough).

Used by `BuildPlanOptions` / `BuildApplyOptions`.

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

Optional interfaces: `Activatable`, `Busy`, `Cancellable`, `Countable`, `Hintable`, `KeyCapturer`, `Pinnable`, `Stackable`, `Outputter`, `ExitCoder`, `ActivateWithArgs`.

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

Central routing with four private methods:

```go
navigateTo(p)           // checks NavBehaviorFor(p.ID()), saves returnTo if NavPush
navigateBack()          // restores returnTo as activePlugin, logs transition
popIfPushed(cmd)        // called by event handlers; pops NavPush plugin if active
resolveOptions(ws)      // re-runs config.Resolve and mutates shared *ResolvedOptions
```

`returnTo sdk.Plugin` — single-level return address. Set by `NavPush` transitions AND workflow transitions (e.g., plan→apply). Consumed by:
- Event handlers (`ChdirChangedEvent`, `WorkspaceChangedEvent`) via `popIfPushed`
- `DeactivateMsg` handler (esc cancel path)

Config propagation: App stores `rootCfg` and `childCfg`. On `WorkspaceChangedEvent`/`WorkspaceCreatedEvent`: calls `resolveOptions(name)`. On `ChdirChangedEvent`: reloads `childCfg` from new dir, then calls `resolveOptions(activeWorkspace)`. This keeps `*ResolvedOptions` in sync without new event types — plugins read fresh values on their next terraform call.

Workflow transitions (plan→apply): The app sets `returnTo` manually when a plugin triggers a workflow to another plugin. This is distinct from `NavPush` metadata — it's a runtime decision. Example: `ApplyRequestMsg` handler sets `returnTo = plan` before activating apply.


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

## Verb-First Action Plugins (taint, untaint, import)

Action plugins are transient — arrive with context, confirm, execute, return. They are NavPush, hidden from menu, reachable via contextual keys or `:command`.

Navigation: State/Plan `t`/`T`/`n` keys emit `TaintRequestMsg`/`UntaintRequestMsg`/`ImportRequestMsg`. App handles these by setting targets on the plugin and navigating with NavPush.

Events emitted on success:
- Taint/Untaint: `PlanInvalidatedEvent`
- Import: `StateRefreshedEvent` + `PlanInvalidatedEvent`

State plugin auto-refreshes on `PlanInvalidatedEvent` (implements `PlanInvalidatedHandler`).

## Apply Plugin Navigation Model

Apply is NOT on the home menu (`MenuVisible: false`). It's reachable through plan's `a` key (confirm) or `A` key (auto-approve).

Flow (no targets): Plan → `ApplyRequestMsg` → app pushes navStack, activates apply with `RequestApply()` → apply shows confirmation → user confirms → apply runs → `PlanInvalidatedEvent` emitted on success.

Flow (with targets): Plan → `ApplyRequestMsg` → apply enters `StatusReplanning` → runs `terraform plan -target=X` → shows targeted plan summary → user confirms → apply runs saved plan → `PlanInvalidatedEvent`.

Auto-approve: Plan → `AutoApplyRequestMsg` → apply calls `AutoApply()` → skips confirmation (replans if targets present, then applies immediately).

All `esc`/cancel paths in apply emit `DeactivateMsg`, which the app handles by checking navStack and navigating back to plan.

Apply resets to idle on `Activate()` if in a terminal state (error/done), preventing stale state when re-entered.

## Targeted Apply (terraform constraint)

Terraform does NOT support `-target` with a saved plan file. Apply handles this via replan:
- When targets are present: apply plugin enters `StatusReplanning`, runs `terraform plan -target=X -target=Y` to produce a new targeted plan file, then applies that plan file
- When no targets: applies the saved plan file via `terraform apply tfplan.out`

This ensures the user always reviews exactly what will be applied — the targeted plan may differ from the full plan due to dependency resolution.

## Service Interface (`pkg/sdk/service.go`)

All terraform operations: `Plan`, `Apply`, `StateList`, `Show`, `StateRm`, `StateMove`, `Import`, `Taint`, `Untaint`, `Validate`, `Output`, `Refresh`, `Init`, `Workspace*`, `ForceUnlock`, `Version`, `WithDir`.

Two implementations:
- `ExecService` — wraps terraform-exec, uses ServiceCache for reads (service.go, state_ops.go, workspace_ops.go)
- `MacroService` — records commands as sdk.Command, reads from ServiceCache, never executes (macro_service.go)

`ServiceCache` (service_cache.go) is a typed, source-aware cache pre-seeded from `-plan`/`-state` flags at startup. Three source kinds: file (re-reads on invalidate), stdin (immutable), exec (cleared on invalidate).

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

Programmatic TUI driver + tape DSL for automated testing and demo recording.

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

Recorder (`recorder.go`): Wraps `tea.Model` to capture ANSI frames + generate tape during interactive sessions. Used by `-record` flag. In headless mode, the Runner captures frames via `CaptureView()`.

Key files:
- `driver.go` — synchronous model driver (no terminal needed)
- `runner.go` — executes parsed tape commands against the driver
- `tape.go` — tape DSL parser
- `recorder.go` — recording middleware (frames + tape generation)
- `key_string.go` — reverse mapping (`tea.KeyMsg` → tape DSL string)

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
