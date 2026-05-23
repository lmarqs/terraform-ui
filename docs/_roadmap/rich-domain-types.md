# Rich domain types (replace primitive obsession)

## Problem

Core domain concepts are bare primitives with implicit behavioral rules that should be encapsulated in types.

## Candidates

| Primitive | Concept | Key invariant | Recommendation |
|-----------|---------|---------------|----------------|
| `PlanFile string` | Plan artifact with lifecycle | Temp (cleanup after apply) vs immutable (user-provided, never delete) | `PlanFile` interface: `Path()`, `Cleanup()` |
| `PlanFile` + `Targets` on ApplyOptions | Two mutually exclusive apply modes | Both set = error | `ApplyMode` sum type (plan-file vs auto-plan) |
| `Refresh *bool` + `RefreshOnly bool` | Three-mode state refresh strategy | Only one path valid | `RefreshMode` enum |
| `Lock *bool` + `LockTimeout string` | Lock strategy (always co-vary) | Timeout only meaningful when locked | `LockConfig` struct |
| `Severity string` | Diagnostic level | Only "error" or "warning" | `DiagnosticSeverity` const (like existing `RiskLevel`) |
| `Workspace string` | Terraform workspace | Magic `"default"` literal scattered | `Workspace` type + `DefaultWorkspace()` |
| `Chdir` + `WorkingDir` on Context | Member directory (relative + absolute) | Always travel together | `WorkingDir` pair struct |
| Resource addresses (`Pins`, `Targets`, `Address`) | Terraform resource address | Valid format, used as keys | `ResourceAddress` type |
| `Backend *bool` on InitOptions | Backend init strategy (nil/true/false) | Three modes in a pointer | `BackendMode` enum |

## Direction

### PlanFile

```go
type PlanFile interface {
    Path() string
    Cleanup() // temp: removes file; immutable: no-op
}
```

Service layer calls `planFile.Cleanup()` — doesn't need to know the policy.

### ApplyMode (replaces PlanFile + Targets mutual exclusivity)

```go
type ApplyMode interface{ applyMode() }
type PlanFileApply struct{ File PlanFile }
type TargetedApply struct{ Targets []ResourceAddress }
```

Impossible to set both — compile-time safety instead of runtime validation.

### LockConfig

```go
type LockConfig struct {
    Enabled bool
    Timeout time.Duration // zero = terraform default
}
```

### Workspace

```go
type Workspace string
func DefaultWorkspace() Workspace { return "default" }
```

### WorkingDir

```go
type WorkingDir struct {
    Relative string // "modules/vpc"
    Absolute string // "/home/user/project/modules/vpc"
}
```

### Event/Message builders

Events and messages (e.g., `ContextSwitchRequestMsg`, `PlanCompletedEvent`) are currently bare struct literals. Builders enforce required fields, validate invariants at construction time, and guide through IDE autocomplete:

```go
// Usage: sdk.NewContextSwitch().Chdir("modules/vpc").Workspace("default").Build()
type ContextSwitchBuilder struct { ... }
func NewContextSwitch() *ContextSwitchBuilder
func (b *ContextSwitchBuilder) Chdir(path string) *ContextSwitchBuilder
func (b *ContextSwitchBuilder) Workspace(name string) *ContextSwitchBuilder
func (b *ContextSwitchBuilder) Build() ContextSwitchRequestMsg // panics if required fields missing
```

Benefits:
- Required fields enforced (no partial construction)
- Validation at construction (e.g., Chdir non-empty, Workspace non-empty)
- IDE discoverability — autocomplete shows what's needed
- Single point to add future fields without breaking call sites

Apply to: `ContextSwitchRequestMsg`, `PlanCompletedEvent`, `ApplyRequestMsg`, `PinToggleRequestMsg`.

### CLI seed flags (`--plan`, `--state`)

The remaining `--plan` and `--state` flags are bare URI strings threaded through `Session.WithPlan(uri)` / `Session.WithState(uri)` → `seedCache()`. They should be domain types:

```go
type SeedSource interface {
    Load(ctx context.Context) ([]byte, error)
    Path() string // empty for stdin
}
```

Implementations: `StdinSource`, `FileSource` (with format detection for plan: JSON vs binary `.tfplan`). Eliminates the `if uri == "-"` branching in `seedCache`.

## Impact

Touches `pkg/sdk/` types, `internal/terraform/exec/service.go`, `internal/ui/app.go`, all plugins. Large effort but eliminates entire classes of bugs by making invalid states unrepresentable.
