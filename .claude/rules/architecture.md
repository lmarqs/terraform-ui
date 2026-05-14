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
type PluginMeta struct {
    Keybinding  string
    MenuVisible bool
}
```

Registration in `cmd/tfui/main.go`:
```go
registry.RegisterFactory("state", tfuistate.New, plugin.PluginMeta{
    Keybinding: "s", MenuVisible: true,
})
```

## Service Interface (`pkg/sdk/service.go`)

All terraform operations: `Plan`, `Apply`, `StateList`, `Show`, `StateRm`, `StateMove`, `Import`, `Taint`, `Untaint`, `Validate`, `Output`, `Refresh`, `Init`, `Workspace*`, `ForceUnlock`, `WithDir`.

Three implementations:
- `TerraformService` — wraps terraform-exec (service.go, state_ops.go, workspace_ops.go)
- `RecordingService` — pre-loaded data, builds command flags from options
- `CompositeService` — hybrid: pre-loaded for reads, delegates writes to TerraformService

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
