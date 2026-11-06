---
name: tfui-architecture
description: "tfui's core SDK abstractions: PluginDeps and the immutable Context, pins vs targets, event bus, plugin interface and optional interfaces, routing and navigation, action mode, Service and ServiceCache, source layer, macro engine, test harness, overlays and frames. Use before editing pkg/sdk/ or internal/, or when designing a plugin, event or SDK addition."
---

# Architecture

`docs/development/architecture.md` owns these facts; read the section before changing the part it covers. The test
harness is in `docs/development/testing.md` § Plugin Test Harness, the tape language in
`docs/reference/macro-language.md`, the decisions in `docs/adr/`.

## Rules that break the build or the design when missed

- Plugins import only `pkg/sdk`, never `internal/` or another plugin; they talk through `NavigateMsg`, `DeactivateMsg`
  and bus events. (§ Dependency Direction, § Navigation)
- `sdk.Context` is immutable. Anything that shapes a terraform command lives on it; plugins never mutate it and react
  through `ContextChangedHandler`. Pins change only through `deps.Pin` / `deps.ClearPins`. (§ PluginDeps and Context,
  ADR-0018)
- Run terraform through `Context().Service`, scoped to the active chdir; `deps.Service` is the unscoped fallback.
- Pins become `-target` only in `Context.PlanOptions()`; `ApplyOptions()` never carries targets. (ADR-0019)
- Every action that starts a terraform call or changes its inputs goes through the app's `requireIdle`. UI-only
  actions bypass it. (§ Navigation, ADR-0016)
- The frame stack routes input, not rendering: a `Stackable` plugin's `View()` delegates to its top frame.
- A new `Service` method is implemented on `ExecService`, `MacroService` and `sdktest.MockService` (the
  `add-terraform-feature` skill).
- A new subcommand parses flags into the plugin's `Input` and calls `Session.RunPlugin` (the `add-command` skill).
- A plugin test uses `sdktest.NewDeps(svc)` and calls `Init`.

## Where to look

| Part | Code | Section |
|------|------|---------|
| Plugin interface, optional interfaces, ports | `pkg/sdk/plugin.go` | Plugin System |
| PluginDeps, Context, Pins | `pkg/sdk/plugin_deps.go`, `context.go`, `pins.go` | PluginDeps and Context |
| Events and handlers | `pkg/sdk/bus.go`, `events.go` | Event Bus |
| Registration, `PluginMeta` | `cmd/tfui/registry.go`, `internal/plugin/registry.go` | Plugin Loading |
| Navigation, busy-guard, Context ownership | `internal/ui/app.go` | Navigation; Config Resolution |
| Action mode | `pkg/sdk/action_runner.go` | Action Mode |
| Overlays, input, frames | `pkg/sdk/overlay.go`, `input.go`, `frames/` | Overlays, Input and Frames |
| Service, cache, sources | `pkg/sdk/service.go`, `internal/terraform/`, `internal/source/` | Terraform Integration |
| Session pipeline | `cmd/tfui/session.go` | Entry Point |
| Macro engine | `internal/macro/` | Macro Engine |
