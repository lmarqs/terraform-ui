---
description: "Core SDK abstractions, service interface, source layer, and macro engine internals"
globs: ["pkg/sdk/**", "internal/**"]
---

# Architecture Details

## Plugin Dependencies (`pkg/sdk/plugin_deps.go`) and Context (`pkg/sdk/context.go`)

Two distinct types live side by side:

`PluginDeps` — DI container handed to `Init()`:

```go
type PluginDeps struct {
    Logger    *slog.Logger
    Service   Service              // unscoped; use Context().Service for chdir-scoped
    Context   func() *Context      // live getter; returns current immutable snapshot
    Pin       func(string) tea.Cmd // toggle pin → triggers Context replacement
    ClearPins func() tea.Cmd       // remove all pins
}
```

`Context` — immutable per-chdir/workspace snapshot, replaced atomically by the app on any chdir/workspace/pin change (ADR-0018):

```go
type Context struct {
    Chdir                 string         // relative member path within project
    WorkingDir, Workspace string
    Service               Service        // chdir-scoped via WithDir
    Pins                  []string       // pinned addresses (UI concept)
    VarFiles              []string
    Vars                  map[string]string
    ExtraArgs             []string
    Parallelism           int
    Lock                  *bool
    LockTimeout           string
}

func (c *Context) PlanOptions() PlanOptions      // Pins become PlanOptions.Targets
func (c *Context) ApplyOptions() ApplyOptions    // never includes Targets (ADR-0019)
func (c *Context) WithPins([]string) *Context    // returns a fresh Context
func (c *Context) TogglePin(string) *Context     // add if absent, remove if present
```

Naming: Pins = UI selection (user picks resources). Targets = terraform `-target=` flags. The boundary is `PlanOptions()` where `opts.Targets = ctx.Pins`.

Rule: anything that affects terraform commands lives on Context. Plugins must NEVER mutate it — they react via `ContextChangedHandler`.

## Event Bus (`pkg/sdk/bus.go`, `pkg/sdk/events.go`)

Typed pub/sub. Plugins subscribe by implementing handler interfaces.

Events: `ContextChangedEvent`, `PlanCompletedEvent`, `PlanInvalidatedEvent`, `LockDetectedEvent`, `LockClearedEvent`, `StateRefreshedEvent`

Handler interfaces: `ContextChangedHandler`, `PlanCompletedHandler`, `PlanInvalidatedHandler`, `LockDetectedHandler`, `LockClearedHandler`, `StateRefreshedHandler`

`ContextChangedEvent.OnlyPinsChanged()` lets plan distinguish a pin toggle (preserve UI state, mark stale) from a chdir/workspace switch (full reset).

Flow: App dispatches events to all plugins implementing the matching handler interface.

## Plugin Interface (`pkg/sdk/plugin.go`)

```go
type Plugin interface {
    ID() string
    Name() string
    Description() string
    Init(deps *PluginDeps) tea.Cmd
    Update(msg tea.Msg) (Plugin, tea.Cmd)
    View(width, height int) string
    Configure(cfg map[string]interface{}) error
    Ready() bool
}
```

Optional interfaces: `Activatable`, `Busy`, `Cancellable`, `Countable`, `Hintable`, `KeyCapturer`, `Pinnable`, `Positionable`, `Stackable`, `Outputter`, `ExitCoder`, `ActivateWithArgs`.

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

Central routing methods:

```go
navigateTo(p)           // requireIdle gate; NavBehaviorFor(p.ID()); saves returnTo if NavPush
navigateBack()          // restores returnTo as activePlugin, logs transition
popIfPushed(cmd)        // called by event handlers; pops NavPush plugin if active
rebuildContext(...)     // builds a fresh sdk.Context snapshot from rootCfg+childCfg
replaceContext(next)    // atomically swaps a.current; returns Cmd that emits ContextChangedEvent
requireIdle(reason)     // universal busy-guard: rejects with commandError if any plugin is Busy()
```

`returnTo sdk.Plugin` — single-level return address. Set by `NavPush` transitions AND workflow transitions (e.g., plan→apply). Consumed by:
- Event handler for `ContextChangedEvent` via `popIfPushed`
- `DeactivateMsg` handler (esc cancel path)

Context propagation: App stores `rootCfg` and `childCfg`. On `ContextSwitchRequestMsg` (emitted by chdir/workspace plugins): reloads `childCfg`, calls `rebuildContext(chdir, workspace)` to produce a fresh immutable `sdk.Context`, then `replaceContext(next)` swaps `a.current` and dispatches a single `ContextChangedEvent{Prev, Next}`. Plugins react via `ContextChangedHandler` — same shape for chdir, workspace, AND pin changes (pins live on `Context.Pins`, replaced by `Context.WithPins` or `Context.TogglePin`).

Universal busy-guard: every action that may mutate terraform inputs or start a new terraform call routes through `requireIdle(reason)`. While any registered plugin is `Busy()` (holds DirLock per ADR-0016), the action is rejected with a uniform message instructing the user to escape via `:q!`. Chokepoints: `ContextSwitchRequestMsg`, `navigateTo`, `cmdQuit`. `:q!` (`cmdForceQuit`) bypasses the guard and calls `Cancel()` on every cancellable plugin.

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

## Apply Plugin Navigation Model (ADR-0019)

Apply is NOT on the home menu (`MenuVisible: false`). It's reachable through plan's `a` key (confirm) or `A` key (auto-approve).

Plan owns plan-file generation; apply consumes the file. Apply has no awareness of targets, var-files, or any other plan-time inputs — those are baked into the saved plan file by terraform.

Flow: Plan → `ApplyRequestMsg{PlanFile, AutoApprove}` → app pushes navStack, activates apply with `RequestApply()` (or `AutoApply()`) → apply shows confirmation (skipped on AutoApprove) → user confirms → apply runs `terraform apply <PlanFile>` → `PlanInvalidatedEvent` emitted on success.

All `esc`/cancel paths in apply emit `DeactivateMsg`, which the app handles by checking navStack and navigating back to plan.

Apply resets to idle on `Activate()` if in a terminal state (error/done), preventing stale state when re-entered.

## Targeted Apply (terraform constraint, ADR-0019)

In the TUI pipeline, targets belong on **plan**, never on apply (ADR-0019):
- Plan includes `-target=X` flags when pinned addresses are present, producing a plan file scoped to those resources
- Apply reads only the plan file: `terraform apply <planfile>`
- Terraform does NOT support `-target` with a saved plan file

The standalone CLI path (`tfui apply --target=X`) is independent — it passes targets directly to terraform's one-shot plan-and-apply mode. No plan file is involved.

Pin toggles are routed through `replaceContext` so they reach the plan plugin via the next `ContextChangedEvent`.

## Service Interface (`pkg/sdk/service.go`)

All terraform operations: `Plan`, `Apply`, `StateList`, `Show`, `StateRm`, `StateMove`, `Import`, `Taint`, `Untaint`, `Validate`, `Output`, `Refresh`, `Init`, `WorkspaceList`, `WorkspaceSelect`, `WorkspaceNew`, `WorkspaceDelete`, `ForceUnlock`, `Version`, `WithDir`.

Two implementations:
- `ExecService` — wraps terraform-exec, uses ServiceCache for reads. All terraform CLI calls are serialized per working directory via DirLock (see ADR-0016). (service.go, state_ops.go, workspace_ops.go, dir_lock.go)
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
| `Context` | `pkg/sdk/context.go` | Immutable per-chdir snapshot (Pins, VarFiles, …) |
| `PluginDeps` | `pkg/sdk/plugin_deps.go` | DI container handed to `Init()` |
| `Status` | `pkg/sdk/status.go` | Idle/Loading/Done/Error enum |
| `Cursor` | `pkg/sdk/ui/cursor.go` | Index selection + viewport windowing |
| `ExpandSet` | `pkg/sdk/ui/expand.go` | Track expanded indices |
| `FuzzyFilter[T]` | `pkg/sdk/ui/filter.go` | fzf matching + score-sorted results |
| `Timer` | `pkg/sdk/ui/timer.go` | Elapsed time tracking with tick integration |
| `Tree` | `pkg/sdk/ui/tree/` | Hierarchical rendering with expand/collapse |

Rules:
- Implement `ContextChangedHandler` to react to chdir/workspace/pin changes
- Use `Cursor.VisibleWindow(h)` instead of manual calculation
- Use `FuzzyFilter[T]` instead of importing fzf directly
- Use `Timer` for elapsed time display during long operations
- Reference implementation: `plugins/state/` (list/filter/tree), `plugins/plan/` (tree + inspect frame)

## Plugin Test Harness (`pkg/sdk/sdktest/testdeps.go`)

`PluginDepsHarness` is the canonical test DI container. Every plugin test MUST use it — never construct `PluginDeps` manually or call `New(svc)` without `Init`.

```go
h := sdktest.NewDeps(svc)
p := plan.New(svc)
p.Init(h.Deps)
```

The harness provides:
- `h.Ctx` — mutable `*sdk.Context` (the live snapshot returned by `deps.Context()`)
- `h.PinRequests` — captures every address passed to `deps.Pin()`
- `h.ClearPinsCount` — counts `deps.ClearPins()` invocations

Convention: plugin test helpers follow `newTestPlugin(svc) → (*Plugin, *PluginDepsHarness)`:
```go
func newTestPlugin(svc sdk.Service) (*Plugin, *sdktest.PluginDepsHarness) {
    h := sdktest.NewDeps(svc)
    p := New(svc)
    p.Init(h.Deps)
    return p, h
}
```

Testing patterns:
- Set `h.Ctx.Pins` to simulate pre-existing pins
- Check `h.PinRequests` to verify a pin toggle was requested
- Execute returned `tea.Cmd` (lazy) before asserting side effects
- The harness does NOT replay pin requests onto `Ctx` — tests that need the next snapshot to reflect a pin must mutate `h.Ctx.Pins` explicitly

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

## Additional Frames (`pkg/sdk/frames/`)

Beyond `FilterFrame`, `InspectFrame`, `ConfirmFrame`:

- `ActionFrame` — displays running operation progress with cancel support
- `FormFrame` — multi-field form with validation and submit/cancel
- `StreamFrame` — real-time streaming output for long operations (plan, apply, init)
