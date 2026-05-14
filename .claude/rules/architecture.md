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

Events: `ChdirChangedEvent`, `WorkspaceChangedEvent`, `PlanCompletedEvent`, `PinsChangedEvent`, `PlanInvalidatedEvent`

Handler interfaces: `ChdirHandler`, `WorkspaceHandler`, `PlanCompletedHandler`, `PinsHandler`, `PlanInvalidatedHandler`

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

Optional interfaces: `Activatable`, `Countable`, `Hintable`, `Pinnable`, `Stackable`.

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

`returnTo sdk.Plugin` — single-level return address. Set only by `NavPush` transitions. Consumed by:
- Event handlers (`ChdirChangedEvent`, `WorkspaceChangedEvent`) via `popIfPushed`
- `DeactivateMsg` handler (esc cancel path)

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

## Service Interface (`pkg/sdk/service.go`)

All terraform operations: `Plan`, `Apply`, `StateList`, `Show`, `StateRm`, `StateMove`, `Import`, `Taint`, `Untaint`, `Validate`, `Output`, `Refresh`, `Init`, `Workspace*`, `ForceUnlock`, `WithDir`.

Two implementations:
- `ExecService` — wraps terraform-exec, uses ServiceCache for reads (service.go, state_ops.go, workspace_ops.go)
- `MacroService` — records commands as sdk.Command, reads from ServiceCache, never executes (macro_service.go)

`ServiceCache` (service_cache.go) is a typed, source-aware cache pre-seeded from `--plan`/`--state` flags at startup. Three source kinds: file (re-reads on invalidate), stdin (immutable), exec (cleared on invalidate).

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

Rules:
- Implement `ChdirHandler` to react to chdir changes
- Use `Cursor.VisibleWindow(h)` instead of manual calculation
- Use `FuzzyFilter[T]` instead of importing fzf directly
- Reference implementation: `plugins/state/`
