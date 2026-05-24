---
title: Plugins implement OneShot for headless mode
status: planned
priority: high
created: 2026-05-24
effort: large
tags: [arch, sdk, cli, plugin, refactor]
depends_on: []
---

## Summary

Today, plugins handle headless / one-shot CLI invocations through three
incompatible accretions on the interactive `Plugin` interface:

1. **State** (`plugins/state/cli.go`, `plugins/state/state.go`): a
`cliMode bool` flag, an `ActivateWithArgs` verb dispatcher, an
`ExitCode()` method, and a `Ready()` whose semantics shift between
TUI and CLI mode. Cross-plugin ownership leak: state's CLI calls
`svc.Taint/Untaint/Import` directly, bypassing the dedicated
taint/untaint/import plugins.
2. **Init** (`plugins/init/init.go:85-91`): `ActivateWithArgs` parses
args into form fields and synthesizes an `initSubmitMsg{}`, reusing
the form-submit code path. No `cliMode`; relies on the form
lifecycle to terminate.
3. **Apply** (`plugins/apply/apply.go:141-154`): `ActivateWithArgs`
parses `--auto-approve` / `--target=…` and either calls
`AutoApply()` (a parallel codepath) or transitions to interactive
confirmation. No `ExitCode`. No `Output()` for error cases.

Verb-first plugins (`taint`, `untaint`, `import`) cannot be invoked
headlessly at all. They are reachable only through state's `t`/`T`/`n`
keys (TUI) or through state's CLI shortcut (which bypasses them).

Introduce **`sdk.OneShot`** — a small, BubbleTea-free interface for
"run from args to result and exit." Plugins that need to run headlessly
implement it. The session router prefers it over `ActivateWithArgs`
when the mode is Headless and the plugin implements it. The
interactive `Plugin` interface remains for TUI use.

Outcome: one consistent contract for headless execution, no
`cliMode` flags, no LSP smell on `Ready()`, no cross-plugin
ownership leaks, and `tfui taint <addr>` / `tfui untaint <addr>` /
`tfui import <addr> <id>` become first-class headless commands.

## Problem

### 1. `Ready()` semantics shift between modes (LSP)

`plugins/state/state.go:126-131`:

```go
func (e *Plugin) Ready() bool {
    if e.cliMode {
        return e.status == sdk.StatusDone || e.status == sdk.StatusError
    }
    return e.status == sdk.StatusDone || e.status == StatusShowingDetail
}
```

Two contracts, one method. The headless driver
(`cmd/tfui/session.go:254-258`) polls `Ready()` every 10ms with a
10-minute timeout; the meaning of "ready" depends on which way the
plugin was woken up. A future plugin reader has to chase the flag's
provenance to know what `Ready()` means. ADR-0008's 100% coverage gate
hides this — both branches are tested, but they should not both exist.

### 2. Per-plugin `cliMode` accretion

`plugins/state/state.go:101-102`:

```go
cliMode      bool
cliResult    string
```

`plugins/state/state.go:284-352` then branches on `e.cliMode` in five
distinct `Update` cases (one per typed result message: deleted,
moved, tainted, untainted, imported). Every new headless verb adds
another branch. SRP violation: the same struct represents both an
interactive frame stack and a one-shot CLI runner.

### 3. Cross-plugin ownership leak

`plugins/state/cli.go:107-142` calls `svc.Taint`, `svc.Untaint`,
`svc.Import` directly. The dedicated `plugins/taint`,
`plugins/untaint`, `plugins/import` plugins each own the request
message (`TaintRequestMsg`), the confirmation flow, the result
message, and the `PlanInvalidatedEvent` emission — but the state
CLI shortcuts straight past them and reimplements the success
output (`formatTainted`, `formatUntainted`, `formatImported` in
`plugins/state/cli.go:159-186`).

This is a deliberate pragmatic shortcut, but it means:
- `tfui state taint X` and a hypothetical `tfui taint X` would have
separate codepaths, separate output formatters, separate exit-code
logic, and separate test surface.
- A bug fix in taint plugin's emission (e.g. fixing the
`PlanInvalidatedEvent` payload) does not affect the state-CLI path.
- The state plugin imports `plugins/taint`, `plugins/untaint`,
`plugins/import` for their request-message types
(`plugins/state/actions.go:9-11`) but bypasses their execution.

### 4. Three different headless patterns

| Plugin | Pattern |
|--------|---------|
| State | `cliMode` flag + verb dispatcher + bypass other plugins + free-function executors |
| Init | Synthesize a TUI message (`initSubmitMsg{}`) and reuse the form codepath |
| Apply | Parallel codepath (`AutoApply`) + parse flags into struct fields |

A reader who learns one pattern is not prepared for the others. This
forecloses a uniform contract that consumers (`cmd/tfui/session.go`,
`internal/ui/app.go`) can rely on.

### 5. Output / ExitCode are optional and pluggable in inconsistent ways

`pkg/sdk/plugin.go:99-115` defines `Outputter`, `ExitCoder`, and
`ActivateWithArgs` as three independent optional interfaces. Today:

| Plugin | `Outputter` | `ExitCoder` | `ActivateWithArgs` |
|--------|:-----------:|:-----------:|:------------------:|
| state | yes | yes | yes |
| apply | yes | no | yes |
| init | yes | no | yes |
| plan | yes | yes | no |
| validate | yes | yes | no |
| version | yes | no | no |
| taint / untaint / import | no | no | no |
| forceunlock / chdir / context / workspace / risk / phantom / blastradius / output / console | no | no | no |

`cmd/tfui/session.go:285-298` `emit()` does a pair of type-asserts:

```go
if outputter, ok := p.(sdk.Outputter); ok { ... }
if coder, ok := p.(sdk.ExitCoder); ok { ... }
```

Plugins that *could* run headlessly but don't implement these silently
exit 0 with empty stdout. There is no compile-time enforcement that a
"CLI-capable" plugin reports its result.

### 6. Verb-first plugins have no headless entry

`plugins/taint/taint.go:60-87`, `plugins/untaint/untaint.go`, and
`plugins/import/import.go` implement only `Activatable`. They expect
to be navigated to via `NavPush` from a parent plugin, then drive a
confirmation flow. They cannot be invoked from `cmd/tfui/main.go`
because no cobra command targets them and they don't implement
`ActivateWithArgs`. The CLAUDE.md "Out of scope" line in the prior
plan-mode plan called this out: *"Top-level `tfui taint <addr>` /
`tfui import` cobra commands — reachable via `tfui state taint <addr>`.
Separate roadmap item if ever wanted."* This roadmap is that item.

## Design

### The interface

`pkg/sdk/oneshot.go` (new file):

```go
package sdk

import "context"

// OneShot is the contract for headless plugin execution. A OneShot
// runs from args to terminal Result in a single call. Unlike the
// interactive Plugin lifecycle (Init → Activate → Update loop →
// Ready() → Output() / ExitCode()), OneShot has no BubbleTea
// coupling, no message loop, no polling, and no notion of partial
// state.
//
// A plugin can implement both Plugin and OneShot. The session router
// (cmd/tfui/session.go) chooses OneShot when:
//   - the resolved presentation is Headless,
//   - args are present (or the plugin's default OneShot verb applies),
//   - the plugin implements OneShot.
//
// Otherwise the existing Plugin lifecycle runs.
type OneShot interface {
    // Run executes the plugin's headless action. It must respect ctx
    // cancellation — long terraform calls return when ctx is done.
    //
    // Returns a Result describing the user-visible outcome. A non-nil
    // error indicates an internal failure (a contract violation or a
    // bug); user-facing terraform errors belong in Result.Stderr with
    // Result.ExitCode != 0. The session router treats a non-nil
    // error the same as ExitCode=1 with the error message on stderr.
    Run(ctx context.Context, args []string) (Result, error)
}

// Result is the user-visible outcome of a OneShot.Run call. The
// session router writes Stdout to os.Stdout, Stderr to os.Stderr,
// and exits the process with ExitCode.
type Result struct {
    Stdout   []byte
    Stderr   []byte
    ExitCode int
}
```

### Why this shape

- **No BubbleTea types.** `OneShot.Run` does not return `tea.Cmd`,
does not produce `tea.Msg`. A plugin's TUI lifecycle is irrelevant
in headless mode. A future port of headless mode to a non-BubbleTea
driver (or to `os.Exec`-based subagent harnesses) does not require
reworking the contract.
- **Synchronous, blocking.** No `Ready()` polling. No `done`
channel. The headless driver (today, `macro.NewDriver` polling at
10ms intervals) goes away for OneShot plugins — `session.go`
invokes `oneshot.Run(ctx, args)` directly.
- **Stdout / Stderr / ExitCode are mandatory and typed.** No optional
`Outputter`/`ExitCoder` type-asserts. Either you implement OneShot
and you produce a Result, or you do not.
- **Plugin's interactive state is untouched.** The state plugin's
list/inspect/filter frame stack does not need to know that the
process was launched headlessly. No `cliMode` flag.
- **Args parsing lives in the plugin.** `OneShot.Run` receives
`args []string` as-is. Parsing is the plugin's job. (Cobra in
`cmd/tfui/main.go` already parses tfui's own flags before
forwarding positional args via `WithArgs`; OneShot just receives
the positional tail.)

### Migration scope, by plugin

| Plugin | Migrates to OneShot? | Notes |
|--------|----------------------|-------|
| state | yes | rm, mv, list verbs only. taint/untaint/import verbs route through their own plugins (see below). |
| taint | yes (new) | `tfui taint <addr> [<addr>…]`. Sibling cobra command. |
| untaint | yes (new) | `tfui untaint <addr> [<addr>…]`. Sibling cobra command. |
| import | yes (new) | `tfui import <addr> <id>`. Sibling cobra command. |
| apply | yes | `tfui apply --auto-approve --target=…`. Replaces `AutoApply` parallel codepath. The interactive plan→apply confirmation flow is unchanged. |
| init | yes | Replaces the synthesized-message pattern. Form-driven path is unchanged. |
| plan | yes | Today's `tfui plan` runs the standalone TUI. With `-ci`, it falls through `ActivateWithArgs`-less to a default `Activate()` and produces output via `Outputter`. OneShot makes the headless path explicit and removes the message-loop polling.
|
| validate, output, version | yes | Currently rely on `Activate() → load → Ready() → Outputter`. OneShot collapses this into one synchronous call. |
| forceunlock | optional | Today TUI-only. Could expose `tfui forceunlock <id>` via OneShot. Out of scope for the initial migration; revisit if a user asks. |
| chdir, context, workspace | no | Pure interactive navigation/selection plugins. No headless equivalent. |
| risk, phantom, blastradius | no (initially) | They render against an existing plan's data. Could later expose JSON-only OneShots. Out of scope. |
| console | no | REPL by definition. |

### Cross-plugin ownership: the state shortcut goes away

After migration:

- `tfui state rm <addr>` → state plugin OneShot (rm verb)
- `tfui state mv <src> <dst>` → state plugin OneShot (mv verb)
- `tfui state list` → state plugin OneShot (list verb)
- `tfui taint <addr>` → **taint** plugin OneShot
- `tfui untaint <addr>` → **untaint** plugin OneShot
- `tfui import <addr> <id>` → **import** plugin OneShot

The state plugin no longer dispatches `taint`/`untaint`/`import` verbs.
`plugins/state/cli.go`'s `executeTaint`, `executeUntaint`,
`executeImport`, `formatTainted`, `formatUntainted`, `formatImported`
disappear. The state plugin no longer imports
`plugins/taint`, `plugins/untaint`, `plugins/import` for non-message
types.

The interactive flow inside the state browser (cursor `t`/`T`/`n`
keys → emit `TaintRequestMsg`/`UntaintRequestMsg`/`ImportRequestMsg`)
is **unchanged**. NavPush still routes to the dedicated plugin's
TUI confirmation. The OneShot interface only governs headless entry.

### Signature: `Run` vs. `Run(args, stdin, stdout, stderr)`

The recommended shape is `Run(ctx, args) (Result, error)` (writes
into a Result). Reasons:

- **Buffering simplifies testing.** A test calls `Run` and inspects
`result.Stdout`. No `bytes.Buffer` plumbing per test.
- **The session router owns IO.** `cmd/tfui/session.go` is the only
place that knows about `os.Stdout`/`os.Stderr`/`os.Exit`. Plugins
do not write to global IO.
- **Recording mode (`-macro`) keeps working.** Today MacroService
records commands; OneShot's Result is orthogonal to that.

A streaming variant (e.g. `Run(ctx, args, stdout io.Writer)`) is
worth considering only if a plugin needs to emit progress before
completion. None of the current OneShot candidates do — terraform
state mutations are short, and apply already streams via
`StreamFrame` in interactive mode (a separate concern).

### Wiring: cobra → registry → OneShot

`cmd/tfui/main.go` adds (or migrates) cobra commands. The session
chooses OneShot when applicable. Sketch:

```go
// cmd/tfui/main.go (new commands)
taintCmd := &cobra.Command{
    Use:   "taint <addr> [<addr>...]",
    Short: "Mark resources for recreation",
    RunE: func(cmd *cobra.Command, args []string) error {
        return NewSession(cfg, rootCfg).
            ForPlugin("taint").WithArgs(args).
            WithCI(ciMode).Run()
    },
}
// untaint, import: same shape

// cmd/tfui/session.go: present() Headless branch
case Headless:
    if oneShot, ok := lookupOneShot(registry, s.pluginID); ok {
        // New: synchronous OneShot path
        result, err := oneShot.Run(ctx, s.args)
        return app, s.writeResult(result, err)
    }
    // Existing: macro driver / Ready() polling fallback
    driver := macro.NewDriver(app, 80, 24)
    ...
```

`writeResult` writes Stdout/Stderr and `os.Exit`s with `ExitCode`. The
existing `emit()` function shrinks to handle only the non-OneShot path
(macro recording, `Outputter`/`ExitCoder` for plugins not yet
migrated).

### What `ActivateWithArgs` becomes

It does not disappear. After migration:

- **State** no longer needs `ActivateWithArgs`. State's CLI verbs are
OneShot. `tfui state` (no verb) still routes to the standalone TUI
via the existing `Activate()` path.
- **Apply** keeps `ActivateWithArgs` only if a user wants `tfui apply`
to render the standalone TUI with pre-set targets. The
`--auto-approve` headless path moves to OneShot. To be reviewed —
may be deletable.
- **Init** keeps `ActivateWithArgs` for the same reason as apply (TUI
with form pre-filled). Headless `tfui init --upgrade` moves to
OneShot.

The decision rule for keeping `ActivateWithArgs` after migration:
*"Does pre-filling the standalone TUI from CLI flags make sense?"* If
yes, keep both `ActivateWithArgs` (TUI pre-fill) and `OneShot`
(headless). If no, drop `ActivateWithArgs`. The two are not
redundant — one is for TUI, one is for headless.

### Mode resolution stays put (mostly)

`cmd/tfui/session.go:131-154` resolveAxes() is unchanged. The
distinction between Interactive (TUI) and Headless (no TUI) still
keys off `-ci`, `CI=1`, and stderr TTY presence. What changes is the
**Headless branch** in `present()`: it now prefers
`OneShot.Run` when available, falling back to the macro-driver
polling loop only when the plugin has not migrated yet.

### Result.Stderr semantics

Errors written to Stderr are **the user-facing terraform error**
(e.g. "Invalid target address"). Internal contract violations
(returned `error` from `Run`) become a stderr line with prefix
`tfui: ` and ExitCode=1, written by the session router. This matches
how `cmd/tfui/main.go` already handles macro errors at line 292-295.

### Default verb / no-args behavior

State today: `tfui state` (no verb) and `tfui state list` both
behave as the standalone-TUI list browser. Under OneShot, `state
list` becomes a headless verb (lists addresses to stdout, exit 0)
and `state` (no verb) stays interactive. The session router picks
OneShot when args are present AND args[0] is a registered verb.

Verbs are discovered by the plugin itself; the SDK does not maintain
a verb registry. (A plugin can expose a `Verbs() []string` method
later if discoverability becomes a concern.)

### Test patterns

`plugins/<name>/oneshot_test.go` (new):

```go
func TestStateOneShot_Rm(t *testing.T) {
    svc := &sdktest.MockService{
        StateRmFn: func(_ context.Context, _ string) error { return nil },
    }
    osh := state.NewOneShot(svc) // factory
    res, err := osh.Run(context.Background(), []string{"rm", "local_file.one"})

    if err != nil { t.Fatalf("Run: %v", err) }
    if res.ExitCode != 0 { t.Errorf("ExitCode=%d, want 0", res.ExitCode) }
    if !strings.Contains(string(res.Stdout), "Removed local_file.one") {
        t.Errorf("Stdout=%q, want Removed line", res.Stdout)
    }
    if got := svc.StateRmCalls; len(got) != 1 || got[0] != "local_file.one" {
        t.Errorf("StateRmCalls=%v", got)
    }
}
```

No `Init`, no `Update`, no driveCLIOp helper, no `Ready()` polling.
The whole CLI test surface collapses by ~50% per plugin.

## Migration plan

Phased to keep CI green at every step.

### Phase 1: SDK contract

1. Add `pkg/sdk/oneshot.go` with `OneShot` interface and `Result`
struct.
2. Extend `internal/plugin/registry.go` with `RegisterOneShot` and
`OneShotByID`. Keep the existing `RegisterFactory` API. A plugin
can register both.
3. Update `cmd/tfui/session.go` `present()` Headless branch to prefer
OneShot when available, falling back to macro-driver polling
otherwise.
4. Update `cmd/tfui/session.go` `emit()` to skip the
`Outputter`/`ExitCoder` path when OneShot was used (the Result
already has Stdout/ExitCode).
5. No plugin changes yet. CI passes because no plugin implements
OneShot.

### Phase 2: Verb-first plugins (taint, untaint, import)

Migrate these first because they have no existing headless code to
break.

1. Add `plugins/taint/oneshot.go` with `taint.OneShot` implementing
`sdk.OneShot`.
2. Same for `plugins/untaint/oneshot.go`,
`plugins/import/oneshot.go`.
3. Add cobra commands `tfui taint`, `tfui untaint`, `tfui import`
in `cmd/tfui/main.go`, mirroring the shape of `tfui apply` /
`tfui state`.
4. Tests: `plugins/<name>/oneshot_test.go` for each.
5. Integration tests: `tests/integration/taint_test.go` etc., similar
to existing `state_test.go`.
6. Output format mirrors terraform CLI exactly:
- `tfui taint <addr>` → `Resource instance <addr> has been
    marked as tainted.\n` (exit 0)
- `tfui untaint <addr>` → `Resource instance <addr> has been
    successfully untainted.\n` (exit 0)
- `tfui import <addr> <id>` → `<addr>: Import successful!\n`
    (exit 0)

### Phase 3: State plugin

1. Add `plugins/state/oneshot.go` implementing OneShot for verbs
`rm`, `mv`, `list` (only — taint/untaint/import are owned by
their own plugins now).
2. The free `executeStateRm`, `executeStateMv` functions become
the OneShot's internal helpers (or move into oneshot.go and the
TUI confirm-callbacks call into them).
3. Delete from `plugins/state/cli.go`:
- `executeTaint`, `executeUntaint`, `executeImport`
- `formatTainted`, `formatUntainted`, `formatImported`
- The taint/untaint/import verb cases in `ActivateWithArgs`
4. Delete from `plugins/state/state.go`:
- `cliMode bool`, `cliResult string` fields
- `Ready()`'s cliMode branch (it returns to a single-contract form)
- The `cliMode` branches in 5 `Update` cases
- `StateTaintedMsg`, `StateUntaintedMsg`, `StateImportedMsg`
    and their cases (the dedicated plugins own these now —
    state's TUI cursor `t`/`T`/`n` keys still emit
    `TaintRequestMsg`/`UntaintRequestMsg`/`ImportRequestMsg` and
    refresh on the resulting `PlanInvalidatedEvent`, which it
    already handles via `HandlePlanInvalidated`).
5. Drop `state.Plugin.ExitCode()` and `state.Plugin.Output()`'s
cliMode branch.
6. Decide: keep `ActivateWithArgs` for `tfui state` (no verb) →
standalone-TUI list, or drop it entirely and route no-verb
invocations through `Activate()`. Likely drop.
7. Tests: `plugins/state/oneshot_test.go` replaces
`plugins/state/cli_test.go`. The latter goes away.
8. Verify the 5 `TestState_*` integration tests still pass.

### Phase 4: Apply plugin

1. Add `plugins/apply/oneshot.go` implementing OneShot for
`--auto-approve --target=…` headless apply.
2. Delete `apply.AutoApply()` (the parallel codepath).
3. Decide whether to keep `ActivateWithArgs`:
- If `tfui apply --target=X` (without `--auto-approve`) should
    render the standalone TUI with pre-filled targets, keep it.
- If not, drop it.
4. Add `apply.Plugin.ExitCode()` is no longer needed.
5. Tests update.

### Phase 5: Init plugin

1. Add `plugins/init/oneshot.go` implementing OneShot for
`--upgrade --reconfigure --backend=… --backend-config=…`.
2. Decide on `ActivateWithArgs`: same question as apply. The init
form pre-fill is genuinely useful (a user might want to inspect
defaults before submitting), so likely keep it for TUI-only.
3. The synthesized `initSubmitMsg{}` pattern stays for the
form-submit path; OneShot is a separate, simpler path.
4. Tests update.

### Phase 6: Plan / validate / output / version

These are read-mostly plugins. Their `Activate() → load → Ready() →
Outputter` pattern collapses cleanly to `OneShot.Run`.

1. Add OneShot for each.
2. Plan's exit code 2 ("changes present") is now in `Result.ExitCode`
and no longer requires a separate `ExitCoder` interface.
3. `Outputter` and `ExitCoder` interfaces become unused once all
migrations land — delete them along with the type-asserts in
`cmd/tfui/session.go`.

### Phase 7: Cleanup

1. Delete `pkg/sdk/plugin.go`'s `Outputter` and `ExitCoder` (no
plugin implements them anymore).
2. Delete `cmd/tfui/session.go`'s `emit()` (OneShot handles its own
IO; macro recording is a separate path).
3. Update `docs/reference/cli-ux.md` to describe OneShot as the
contract for headless plugins; remove references to Outputter /
ExitCoder.
4. Update `.claude/rules/architecture.md` and CLAUDE.md.
5. Add an ADR — `docs/adr/0021-oneshot-headless-plugin-contract.md`
— explaining the rationale.

## Alternatives considered

### A. Keep `ActivateWithArgs`, fix `cliMode` in place

Rejected. The patterns are the symptom; the disease is "Plugin is a
god-object that handles both interactive and one-shot lifecycles."
Adding a second `cliMode` flag elsewhere does not fix the LSP smell
or the cross-plugin ownership leak.

### B. `Plugin.OneShot(args) (OneShot, error)` factory method

Plugins return a freshly-constructed OneShot per invocation. This is
slightly cleaner for verb dispatch (each verb returns its own
OneShot type), but adds an indirection that buys little when most
plugins have one OneShot or trivial verb-tables.

Worth keeping in mind for plugins where verbs need very different
state (e.g. state's rm vs. mv), but not the default.

### C. `OneShot` as a function type

```go
type OneShot func(ctx context.Context, args []string) (Result, error)
```

Cleaner for plugins that want to register many small verbs as
free functions. Loses interface-based type-asserts for capability
discovery (no `if oneshot, ok := p.(OneShot); ok`). Considered and
rejected — interface gives better type discovery and parallels
`Activatable`/`Outputter`.

### D. Streaming Result (`Run(ctx, args, stdout io.Writer)`)

Considered for plugins that emit progress (apply). Rejected for the
initial design — apply's streaming is a TUI concern (the
`StreamFrame`); headless apply waits to completion and emits the
terraform-equivalent summary. If a future plugin needs progress on
stdout headlessly, add a `OneShotStreaming` variant then.

### E. Make `Plugin` itself implement OneShot via default method

Go has no default methods. Embedding `sdk.PluginBase` could provide
a no-op OneShot, but that defeats the discovery model and creates
silent failures (a plugin that "implements" OneShot but returns
nothing). Rejected.

### F. Replace `Plugin` entirely with separate `Interactive` and
    `Headless` interfaces

Rejected. The interactive Plugin interface is large, well-tested,
and useful. OneShot is additive; a plugin can implement both. A
big-bang split would be a larger refactor with no additional
correctness benefit.

## Open design questions

These are deferred to implementation time. Each has a tentative
default that the implementer can revisit.

### Q1. Where does Service / Context live for a OneShot?

Options:
1. **Constructor-injected:** `taint.NewOneShot(svc sdk.Service)`.
Simple, mirrors `tfui.RegisterFactory`. Default.
2. **Inherited from the Plugin instance:** if the plugin is already
constructed, the OneShot reuses its `Svc` field. Sleeker but
couples lifetimes.

Default: option 1.

### Q2. How does OneShot get `PluginDeps` (Logger, Pin, Context)?

OneShot does not need `Pin` or `ClearPins` (no UI selection in
headless mode). It does need `Logger` and possibly `Context`.

Options:
1. **A trimmed `OneShotDeps` struct:** `Logger`, `ContextFn`. Skip
`Pin`, `ClearPins`.
2. **Reuse `PluginDeps`:** simpler, has the unused fields. Default.

Default: option 2 — fewer types, parallel structure.

### Q3. Default verb when args is empty

`tfui state` should still drop into the standalone TUI list browser.
The OneShot for state should reject `args == nil` (return an error
that signals "fall back to interactive"), or the session router
should route empty-args to the Plugin path before consulting OneShot.

Default: session router checks `len(args) > 0` before preferring
OneShot.

### Q4. JSON output

`-json` is a flag the cobra layer parses. Today it reaches
`Outputter.Output(json bool)`. OneShot could:
1. Receive json as an arg in `args`.
2. Have a separate method `RunJSON(ctx, args) (Result, error)`.
3. Inspect `args` for a `--json` token.

Default: option 3 — JSON is a flag like any other, plugins inspect
their own args. Cobra layer continues to forward all flags.

### Q5. Cancellation on SIGINT

Today: `tea.WithAltScreen()` and signal handling come from
BubbleTea. Headless mode has neither. The session router needs to
install its own signal handler that cancels the context passed to
`OneShot.Run`.

Default: `cmd/tfui/session.go` adds `signal.NotifyContext(...)` for
the headless OneShot branch. Plugins that wrap terraform calls in
`context.WithCancel(deps.Context())` already get cancellation for
free.

### Q6. Test harness

`pkg/sdk/sdktest/testdeps.go` provides `PluginDepsHarness`. A
parallel `OneShotHarness` may be useful — or `OneShotHarness` may
just be a thin wrapper (`NewOneShotDeps(svc)` returning
`*OneShotDeps` similar to `NewDeps`).

Default: reuse `sdktest.NewDeps(svc).Deps` until the OneShot tests
reveal a real mismatch.

### Q7. Backwards compatibility window

During phases 2-6, both ActivateWithArgs and OneShot coexist. The
session router prefers OneShot when present. No user-visible behavior
changes during the migration. Macro tapes that exercise
`ActivateWithArgs` continue to work because the fallback path remains.

After phase 7, `ActivateWithArgs` is either deleted (if no plugin
needs TUI pre-fill) or kept only for that purpose. The roadmap does
not mandate deletion.

### Q8. Interaction with `-macro <tape.txt>`

Macro mode is a third presentation axis (`Headless + Recording`).
It does not interact with OneShot — macro plays back keypresses
through the BubbleTea model. A plugin in macro mode is interactive,
just driven from a tape instead of a keyboard.

If a user runs `tfui state rm X --ci -macro tape.txt`, the
session router prefers the macro driver (it has a tape). OneShot is
skipped. This is correct: macro mode is for capturing the
recorded-command output via `MacroService`, not for running OneShot.

The session router's precedence: `macroURI != "" → macro driver`,
else `args present + OneShot implemented → OneShot`, else `args
present + ActivateWithArgs → ActivateWithArgs in macro driver`,
else `Activate() in macro driver`.

### Q9. Result.Stderr writes order

If both Stdout and Stderr have content, write Stdout first then
Stderr (consistent with terraform's behavior on partial failure).
If ExitCode != 0, the session router writes Stderr unconditionally;
if ExitCode == 0, Stderr is still written but typically empty.

## Verification

Per-phase, `mise run check:lint && mise run test:unit && mise run
check:build` must pass. Each phase adds tests rather than mutating
existing ones (with the exception of phase 7 cleanup, where deleted
interfaces require deleting their tests).

### End-to-end checks (after phase 7)

```bash
# Verb-first plugins (new)
tfui taint local_file.one -project tests/fixtures/state-ops -ci
echo $?                              # 0
tfui taint nonexistent.thing -ci     # exit 1, error on stderr

tfui untaint local_file.one -ci      # exit 0
tfui import aws_instance.web i-0abc  # exit 0

# State (still works through OneShot)
tfui state rm local_file.one -ci     # exit 0
tfui state mv src.addr dst.addr -ci  # exit 0
tfui state list -ci                  # addresses to stdout

# Apply / init via OneShot
tfui apply --auto-approve -ci        # exit 0 or 1
tfui init --upgrade -ci              # exit 0

# Plan / validate via OneShot
tfui plan -ci                        # exit 0/1/2
tfui validate -ci                    # exit 0/1
tfui version --json                  # JSON to stdout

# Mode interaction
tfui state                           # standalone TUI on stderr, list on quit
tfui state -ci                       # ?: no verb in -ci mode → list to stdout
tfui plan -ci -macro tape.txt        # macro driver, recorded commands to stdout

# Sanity: TUI confirm-flow still works
tfui                                 # full TUI; navigate state, press d, confirm
```

### Coverage

ADR-0008's 100% gate applies to `pkg/sdk` and every `plugins/*`
package. The migration does not relax it. Per phase, new test files
land alongside the new code.

### Documentation parity

- `docs/reference/cli-ux.md` updated to describe OneShot.
- `docs/reference/cli-reference.md` lists `tfui taint`, `tfui
untaint`, `tfui import` as top-level commands.
- `docs/adr/0021-oneshot-headless-plugin-contract.md` records the
decision.
- `CLAUDE.md` "Plugin Interface" reference adjusts.

### Macro tape regression

Run all existing macro tapes (`tests/fixtures/tapes/macro/`,
`tests/fixtures/tapes/smoke/`). None should break — interactive
flows are untouched.

## Effort estimate

**Large** — 5-8 days of focused work, spread across the seven
phases.

| Phase | Scope | Effort |
|-------|-------|--------|
| 1 | SDK contract + session router prefer-OneShot | 1d |
| 2 | taint, untaint, import OneShots + cobra | 1d |
| 3 | State migration + cleanup | 1d |
| 4 | Apply migration | 0.5d |
| 5 | Init migration | 0.5d |
| 6 | Plan, validate, output, version migration | 1.5d |
| 7 | Cleanup, ADR, docs | 0.5-1d |

The unknown is phase 6 — plan in particular has nontrivial state
(diff parsing, tree rendering for the standalone-TUI path) and may
reveal further refactoring needs. Treat the estimate as a band, not
a target.

## Critical files

This section names where the work lands. Patterns that repeat across
plugins are described once.

### New files

- `pkg/sdk/oneshot.go` — interface + Result struct.
- `docs/adr/0021-oneshot-headless-plugin-contract.md`.
- `plugins/<name>/oneshot.go` — one per migrated plugin.
- `plugins/<name>/oneshot_test.go` — one per migrated plugin.
- `tests/integration/taint_test.go`, `untaint_test.go`,
`import_test.go`.

### Edited files

- `cmd/tfui/main.go` — add cobra commands for `taint`, `untaint`,
`import` (mirror existing `state` shape at lines 261-280).
- `cmd/tfui/session.go` — `present()` Headless branch (lines
242-261) prefers OneShot; `emit()` (lines 264-298) shrinks.
- `internal/plugin/registry.go` — `RegisterOneShot`, `OneShotByID`.
- Plugin packages — see migration phases above.

### Deleted (after phase 7)

- `pkg/sdk/plugin.go` Outputter / ExitCoder interfaces (lines
99-109).
- `plugins/state/cli.go` (most of it) — verb dispatch and the
taint/untaint/import paths.
- `plugins/state/cli_test.go` — replaced by oneshot_test.go.
- `plugins/state/state.go` `cliMode`, `cliResult` fields and their
branches.
- `plugins/state/state.go` `StateTaintedMsg`, `StateUntaintedMsg`,
`StateImportedMsg` types and Update cases.
- `plugins/apply/apply.go` `AutoApply()` parallel codepath.
- `plugins/init/init.go` `initSubmitMsg{}` pattern (kept only if
the form-driven TUI path still uses it; likely yes, so it stays).

### Touchstones — read these before implementation

- `plugins/state/cli.go` — current state shortcut, the most complex
case. Migration Phase 3 deletes most of it.
- `plugins/state/state.go:101-131,284-352,788-818` — `cliMode`
branches.
- `plugins/init/init.go:85-91` — `ActivateWithArgs` synthesizing a
TUI message (different, simpler pattern).
- `plugins/apply/apply.go:141-171` — `ActivateWithArgs` +
`AutoApply` parallel codepath.
- `plugins/taint/taint.go`, `untaint/untaint.go`,
`import/import.go` — what verb-first plugins look like
pre-OneShot.
- `cmd/tfui/session.go:131-298` — entire mode-resolution and
Headless flow.
- `internal/ui/app.go:205-224` — `openContextOnStartupMsg` handler.
- `pkg/sdk/plugin.go:99-115` — Outputter, ExitCoder, ActivateWithArgs.
- `pkg/sdk/sdktest/testdeps.go` — test harness.
- `docs/adr/0008-coverage-as-behavioral-gate.md` — coverage rule.
- `docs/adr/0019-unidirectional-data-flow.md` — apply confirmation
ownership.

## Out of scope

- Replacing the macro driver. Macro mode (`-macro <tape>`) is
unrelated to OneShot and stays as-is.
- Plan, risk, phantom, blastradius headless JSON outputs beyond
what they have today.
- A `-yes` / `--auto-approve` flag for state operations — terraform
itself has none.
- External plugins (gRPC). See `docs/_roadmap/external-plugins.md`.
- Top-level `tfui forceunlock <id>` — defer until a user requests
it. Phase 2's pattern makes adding one trivial later.

## Related

- `docs/_roadmap/eliminate-passthrough-args.md` — same architectural
philosophy (typed contracts at boundaries). OneShot's args are
positional; this roadmap does not interact with `--` passthrough.
- `docs/_roadmap/app-decomposition.md` — App's busy-guard and
navigation logic does not touch headless-OneShot mode (no
navigation in OneShot). The two roadmaps are independent.
- ADR-0001 (SDK isolation) — OneShot lives in `pkg/sdk`, plugins
import only that.
- ADR-0010 (command builder, not abstraction) — OneShot's Result
passes terraform's stdout/stderr through verbatim where possible.
- ADR-0008 (coverage gate) — not relaxed.
- ADR-0013 (plugin-owned operation context) — OneShot owns its
context.WithCancel for terraform calls.
- ADR-0019 (unidirectional data flow) — apply's plan-file
consumption pattern is unaffected; headless apply with
`--target=…` runs terraform's plan-and-apply one-shot.