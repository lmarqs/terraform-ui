---
title: Plugin Architecture Refactoring
status: planned
priority: medium
created: 2026-05-11
effort: medium
tags: [debt, refactor, plugins, sdk]
depends_on: []
---

## Summary

12 plugins duplicate the same patterns (Status enum, ScopeGuard init, logger setup) instead of using shared SDK primitives. Several plugin files exceed 800 lines mixing rendering, logic, and actions. Error handling is inconsistent (60% of errors returned bare without context wrapping).

## Need

1. **Duplicate Status enum in 12 plugins** — Each plugin defines its own `type Status int` with `StatusIdle`, `StatusLoading`, `StatusDone`, etc. The SDK already has a shared `Status` type at `pkg/sdk/status.go` with predicates, but plugins don't use it.

2. **Repeated ScopeGuard boilerplate** — `apply`, `plan`, and `state` plugins all duplicate the same 15-line pattern in `Init()` + `Activate()` for scope change detection and service rescoping. This is error-prone to maintain across plugins.

3. **Monolithic plugin files** — `plugins/state/state.go` is 862 lines mixing filtering, rendering, and state operations. `internal/terraform/service.go` is 746 lines with multiple concerns.

4. **Inconsistent logger initialization** — Some plugins create `slog.New(slog.NewTextHandler(io.Discard, nil))`, others use `ctx.Logger`, and `apply` doesn't initialize a logger at all. No standard pattern.

5. **Bare error returns** — Only 58 of 236 `err != nil` checks wrap with `fmt.Errorf(... %w)`. The rest return bare errors losing call-site context, making debugging harder.

## Expected UX

No user-visible changes. Internal quality improvement that reduces maintenance burden and bug surface.

## Advantages

- Single Status enum eliminates drift (new states automatically available to all plugins)
- ScopeGuard mixin prevents subtle scope bugs when copy-pasting patterns
- Split files improve code review and navigation
- Consistent logging makes debug sessions productive
- Wrapped errors produce actionable stack traces

## Effort Justification

Medium effort (2-5 days) because:
- Status migration requires touching 12 plugin files but changes are mechanical
- File splits need careful preservation of git history (git mv workflow)
- Error wrapping audit spans 236 locations but most are simple additions
- ScopedPlugin helper is new code but straightforward given existing ScopeGuard
- All changes are backwards-compatible with existing plugin API

## Design

### 1. Status Enum Migration

Replace per-plugin Status types with `sdk.Status`:

```go
// Before (in each plugin)
type Status int
const (
    StatusIdle Status = iota
    StatusLoading
    StatusDone
    StatusError
)

// After (plugins import from sdk)
import "github.com/lmarqs/terraform-ui/pkg/sdk"
// Use sdk.Status directly
```

All 12 plugins use identical enum values, so migration is a straightforward find-replace per plugin.

### 2. ScopeGuard Helper

Create `sdk.ScopedPlugin` mixin to encapsulate repeated pattern:

```go
// pkg/sdk/scoped_plugin.go
type ScopedPlugin struct {
    guard   *ScopeGuard
    service Service
}

func NewScopedPlugin(service Service) *ScopedPlugin {
    return &ScopedPlugin{service: service}
}

func (s *ScopedPlugin) InitScope(session Session, svc Service) {
    s.guard = NewScopeGuard(session, svc)
    s.service = svc
}

func (s *ScopedPlugin) ActivateScope() (Service, bool) {
    return s.guard.Check()
}
```

Plugins embed this and call `InitScope` in `Init()`, `ActivateScope` in `Activate()`.

### 3. File Splits

Split large files by concern:

**plugins/state/** (862 lines → 3 files)
- `state.go` — Plugin struct, Init, Activate, core Update routing
- `view.go` — View(), rendering logic, styling
- `filter.go` — Filter frame, fuzzy matching

**internal/terraform/** (746 lines → 3 files)
- `service.go` — Interface definition, constructor, shared methods
- `state_ops.go` — StateList, StateRm, StateMove, StateShow, Import
- `workspace_ops.go` — WorkspaceList, WorkspaceNew, WorkspaceSelect, WorkspaceDelete

Use `git mv` where possible to preserve blame history.

### 4. Logger Standardization

All plugins initialize logger identically in `Init()`:

```go
func (p *Plugin) Init(ctx *sdk.Context) tea.Cmd {
    p.log = ctx.Logger.With("plugin", p.ID())
    // ... rest of init
}
```

Remove all `slog.New(slog.NewTextHandler(io.Discard, nil))` fallback patterns.

### 5. Error Wrapping

Add context at plugin boundaries and operation call sites:

```go
// Before
if err != nil {
    return err
}

// After
if err != nil {
    return fmt.Errorf("list state resources: %w", err)
}
```

Focus on:
- Plugin Update message handlers (user-visible errors)
- Service method calls (operation context)
- Config/session reads (data source context)

### 6. Linter Integration

Add to `.golangci.yaml`:

```yaml
linters:
  enable:
    - errcheck  # enforce error handling
```

This prevents future bare error returns.

## Open Questions

- Should ScopedPlugin be a base struct (embedding) or an interface adapter (composition)?
- Do we want a `sdk.BasePlugin` that bundles logger + status + scope, or keep them separate?
- Should error wrapping use a custom error type with structured fields instead of fmt.Errorf?

## Tasks

- [ ] Migrate all 12 plugins from local Status enum to `sdk.Status` (apply, blastradius, init, output, phantom, plan, repl, risk, scope, state, validate, workspaces)
- [ ] Create `sdk.ScopedPlugin` helper or mixin that encapsulates ScopeGuard init + Activate pattern
- [ ] Split `plugins/state/state.go` (862 lines) into: `state.go` (core), `view.go` (rendering), `filter.go` (filtering logic)
- [ ] Split `internal/terraform/service.go` (746 lines) into: `service.go` (core), `state_ops.go` (state mutations), `workspace_ops.go` (workspace management)
- [ ] Standardize logger injection: all plugins use `ctx.Logger.With("plugin", p.ID())`
- [ ] Audit and wrap bare error returns with call-site context (`fmt.Errorf("operation: %w", err)`)
- [ ] Add errcheck to enforced linters (depends on linting-enforcement roadmap item)

## Notes

- Status migration can be done plugin-by-plugin (incremental, low risk)
- File splits should preserve git blame via `git mv` + edit where possible
- Error wrapping should focus on plugin boundaries first (where users see errors)
- The ScopedPlugin helper must not break plugins that don't need scope awareness
