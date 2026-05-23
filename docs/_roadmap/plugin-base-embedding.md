---
title: Plugin Base Embedding — Eliminate Shared Boilerplate
status: planned
priority: medium
created: 2026-05-23
effort: medium
tags: [arch, sdk, plugins]
depends_on: []
---

## Summary

Every plugin repeats ~150 lines of identical lifecycle code: Init field assignments, HandleContextChanged reset logic, and metadata one-liners (ID/Name/Description). An embedded `sdk.PluginBase` struct would eliminate this duplication while keeping Go's composition model explicit.

## Problem

| Boilerplate | Occurrences | Lines saved |
|------------|-------------|-------------|
| Init() field assignments (svc, log, getCtx, pinFn, clearPinsFn) | 18 plugins | ~5 lines each = 90 |
| HandleContextChanged null-check + service rebind + reset | 14 plugins | ~8 lines each = 112 |
| ID()/Name()/Description() one-liners | 18 plugins | ~3 lines each = 54 |
| Ready()/Status()/Busy() one-liners | 18 plugins | ~3 lines each = 54 |

Every new plugin copy-pastes this. Forgetting a field assignment causes a nil-panic at runtime, not a compile error.

## Design

### Embedded PluginBase struct

```go
// pkg/sdk/plugin_base.go
type PluginBase struct {
    id          string
    name        string
    description string
    Svc         Service
    Log         *slog.Logger
    GetCtx      func() *Context
    PinFn       func(string) tea.Cmd
    ClearPinsFn func() tea.Cmd
    Status_     Status
}

func NewPluginBase(id, name, desc string) PluginBase {
    return PluginBase{id: id, name: name, description: desc}
}

func (b *PluginBase) ID() string          { return b.id }
func (b *PluginBase) Name() string        { return b.name }
func (b *PluginBase) Description() string { return b.description }

func (b *PluginBase) InitBase(deps *PluginDeps) {
    b.Svc = deps.Service
    b.Log = deps.Logger
    b.GetCtx = deps.Context
    b.PinFn = deps.Pin
    b.ClearPinsFn = deps.ClearPins
}

func (b *PluginBase) PinnedAddresses() []string {
    return PinnedAddresses(b.GetCtx)
}
```

### Plugin usage

```go
type Plugin struct {
    sdk.PluginBase
    // plugin-specific fields only
    stack   *sdk.Stack
    summary *sdk.PlanSummary
    // ...
}

func New(svc sdk.Service) sdk.Plugin {
    return &Plugin{
        PluginBase: sdk.NewPluginBase("plan", "Plan", "Review terraform plan changes"),
        // ...
    }
}

func (e *Plugin) Init(deps *sdk.PluginDeps) tea.Cmd {
    e.InitBase(deps)
    e.reset()
    return nil
}
```

### HandleContextChanged default

Provide a helper, not a default method (Go embedding doesn't do virtual dispatch):

```go
// pkg/sdk/plugin_base.go
func (b *PluginBase) HandleContextChangedDefault(ev ContextChangedEvent) bool {
    if ev.Next == nil {
        return false // caller should return nil
    }
    if ev.Next.Service != nil {
        b.Svc = ev.Next.Service
    }
    return true // caller should proceed with reset
}
```

Plugins:
```go
func (e *Plugin) HandleContextChanged(ev sdk.ContextChangedEvent) tea.Cmd {
    if !e.HandleContextChangedDefault(ev) {
        return nil
    }
    e.reset()
    return nil
}
```

Plan plugin keeps its special `OnlyPinsChanged()` logic by not calling the default.

## Constraints

- Go embedding is composition, not inheritance — no virtual dispatch
- Plugins that DON'T want the default behavior (plan with OnlyPinsChanged) must be able to opt out cleanly
- Exported field names on PluginBase must not collide with plugin-specific methods
- The test harness (`sdktest.PluginDepsHarness`) stays unchanged — it feeds into `Init(deps)` which calls `InitBase`

## Migration Strategy

1. Introduce `PluginBase` + `InitBase` + helpers (no consumers yet)
2. Migrate plugins one-by-one (mechanical: embed, delete duplicated fields/methods)
3. Start with simple plugins (output, validate, console) as proof
4. End with complex plugins (plan, state) that have custom overrides

## Verification

- All 18 plugins embed PluginBase
- `go vet ./...` clean
- Coverage gate holds (100%)
- No runtime behavior change — purely structural refactor
