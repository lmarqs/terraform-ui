---
layout: default
title: Testing Strategy
parent: Development
nav_order: 2
seo:
  description: Behavioral testing standard, conventions, and layered test architecture
description: "Sets the rules for tests: what to test at which level, naming and the TDD cycle."
audience: "Contributors writing or reviewing tests"
---

# Testing Strategy

Every test asserts what the user sees; coverage follows. Pick the level from what you changed:

| You changed… | Write… |
|---|---|
| Plugin logic: a state transition, an event, a command | A [unit test](#unit-tests-_testgo-in-same-package) built on the [harness](#plugin-test-harness) |
| What a plugin renders in one state | A [golden test](#golden-tests-golden_testgo--testdatagolden) |
| Routing between plugins: navigation, events reaching the header | An [app test](#app-tests-internaluiapp_testgo) in `internal/ui/app_test.go` |
| The terraform command a key records, or a flow through the TUI | A [tape](#macro-tests-testsfixturestapes) |
| Service code or CLI flag handling against real terraform | An [integration test](#integration-tests-testsintegration-build-tag-integration) |

## Rules

1. **Tests describe behavior, not implementation.** A test that runs a code path without asserting a user-visible
   outcome creates false confidence.
2. **Coverage is a side effect, never the goal:** a path run without an assertion passes while the behavior is wrong.
3. **Tests are the living spec.** A test that does not describe a user-visible behavior is wrong.
4. **One test suite.** There is no separation between "behavior tests" and "coverage tests".

## Test Naming Convention

Use `Test{Subject}_{Given/When}_{ExpectedBehavior}` format. The name should read as a specification:

```go
// Good: describes a scenario and expected outcome
func TestSwitchToSelected_GivenDifferentWorkspace_ShouldSetLoadingAndDispatch(t *testing.T)
func TestFrame_GivenLoading_EnterKey_ShouldBeIgnored(t *testing.T)
func TestUpdate_GivenCreateSuccess_ShouldRefreshAndEmitCreatedEvent(t *testing.T)
func TestHints_GivenDone_CursorOnDefault_ShouldNotShowDelete(t *testing.T)
func TestActivate_GivenError_ShouldRetryLoading(t *testing.T)

// Bad: describes mechanics, not behavior
func TestRefresh(t *testing.T)
func TestUpdate(t *testing.T)
func TestHandleMessage(t *testing.T)
```

The format has three parts:

| Part | Meaning | Example |
|------|---------|---------|
| Subject | What is being tested | `SwitchToSelected`, `Frame`, `Hints` |
| Given/When | Precondition or trigger | `GivenDifferentWorkspace`, `GivenLoading_EnterKey` |
| Expected | What should happen | `ShouldSetLoadingAndDispatch`, `ShouldBeIgnored` |

## TDD Workflow

This is the required workflow for every change:

1. **Write a failing test** that describes the behavior
2. **Run it** — confirm it fails for the right reason (not a compile error or wrong assertion)
3. **Implement** the minimum code to pass
4. **Refactor** while tests stay green
5. **Coverage should naturally be 100%** — if not, you missed a behavior

The `test-writer` agent writes the failing test before any implementation edit.

## Running Tests

```bash
# Unit tests (fast, no external dependencies)
mise run test:unit

# Coverage gate (see Coverage Target)
mise run test:coverage

# smoke/ tapes against the binary in dist/ (run mise run build first)
mise run test:macro

# Integration tests and the macro/ and compat/ tape runners (requires terraform on PATH)
mise run test:integration
```

The done check, everything that must pass before a change is done, is in
[AGENTS.md](https://github.com/lmarqs/terraform-ui/blob/main/AGENTS.md#quick-reference).

## Test Layers

### Unit Tests (`*_test.go` in same package)

**What they test:** Behavior of a single plugin or component — state transitions, event emissions, command dispatch, loading guards, UI feedback.

**When to write:** For every plugin operation, every frame interaction, every event handler.

**Location:** Same package as the code under test (white-box access).

```go
func TestDeleteWorkspace_ShouldSetLoadingAndDispatch(t *testing.T) {
    p, _ := newTestPlugin(&sdktest.MockService{})
    p.status = sdk.StatusDone
    p.workspaces = []string{"default", "temp"}
    p.current = "default"

    // When delete is triggered
    cmd := p.deleteWorkspace("temp")

    // Then it dispatches a command and enters loading state
    if cmd == nil {
        t.Fatal("deleteWorkspace should return a command")
    }
    if p.status != sdk.StatusLoading {
        t.Errorf("status = %v, want Loading", p.status)
    }
    if !strings.Contains(p.loadingMsg, "temp") {
        t.Errorf("loadingMsg = %q, should mention workspace name", p.loadingMsg)
    }
}
```

**Mock services:** Use `sdktest.MockService` (`pkg/sdk/sdktest/mock_service.go`) and set only the function fields your
test exercises; unset methods return zero values, and each call is recorded (`WorkspaceSelectCalls`,
`PlanCalls`, …). A per-package mock duplicates it.

```go
svc := &sdktest.MockService{
    WorkspaceListFn: func(context.Context) ([]string, error) { return []string{"default", "staging"}, nil },
    WorkspaceFn:     func(context.Context) (string, error) { return "staging", nil },
}
```

### Golden Tests (`golden_test.go` + `testdata/golden/`)

**What they test:** Visual regression of `View()` output — ensures rendering does not accidentally change.

**When to write:** For every distinct visual state a plugin can display (loading, error, done with data, empty state, contextual messages).

**Location:** `plugins/<name>/golden_test.go` with snapshots in `plugins/<name>/testdata/golden/`.

```go
func TestView_Given_Loading_ShouldRender_LoadingMessage(t *testing.T) {
    p := newGoldenPlugin()
    p.status = sdk.StatusLoading

    sdktest.AssertGolden(t, p.View(80, 18))
}

func TestView_Given_WorkspaceList_WithSelection_ShouldRender_HighlightedRow(t *testing.T) {
    p := newGoldenPlugin()
    p.status = sdk.StatusDone
    p.workspaces = []string{"default", "staging", "production"}
    p.current = "default"
    p.selected = 2

    sdktest.AssertGolden(t, p.View(80, 18))
}
```

**Updating golden files:** Run tests with `-update` flag:

```bash
go test ./plugins/workspace/ -run TestView -update
```

Golden test names map to file paths: `TestView_Given_Loading_ShouldRender_LoadingMessage` becomes `testdata/golden/TestView_Given_Loading_ShouldRender_LoadingMessage.txt`.

### App tests (`internal/ui/app_test.go`)

**What they test:** Cross-cutting flows that span multiple components — navigation between plugins, event dispatch causing header updates, `navStack` behavior after push and pop.

**When to write:** When a behavior depends on how the app routes messages between plugins (e.g., "workspace change should update header chdir display").

These tests verify the contract between the app shell and plugins — things no single plugin test can catch.

### Macro Tests (`tests/fixtures/tapes/`)

Tape-driven TUI tests come in **three layers**. Every layer runs tfui with `--macro`, so every layer uses the
`MacroService` backend: terraform never runs, reads come from plan and state fixtures, and mutations are recorded as
commands. The layers differ in runner and in what a test asserts.

```
tests/fixtures/tapes/
├── compat/   CLI compatibility scenarios (flag normalization)
├── macro/    Key presses and the commands they record
└── smoke/    Flows through the release binary
```

#### Layer comparison

| Layer | Backend | What tapes can wait for | What tests assert | Runner |
|---|---|---|---|---|
| `macro/` | `MacroService`, against the fixtures each case passes | Plugin-rendered text | Recorded command stream via `wantStdout` substring | `TestMacro` in `tests/integration/macro_test.go` (runs in `mise run test:integration`) |
| `smoke/` | `MacroService`, against `tests/fixtures/plan.json` and `state.json` | Plugin-rendered text | Tape's own `assert view` lines (no Go-level assertions) | `mise run test:macro` (requires a built binary in `dist/`) |
| `compat/` | `MacroService`, through CLI subcommands and flags | Plugin-rendered text | Recorded command stream via `Contains` checks in dedicated `TestCompat_*` cases | `tests/integration/compat_test.go` (runs in `mise run test:integration`) |

#### Decision

```
Need to test…                                  Use tapes in…
────────────────────────────────────────────────────────────
"key X records command Y"                      macro/
"key X causes plugin Y to render Z"            macro/
"the release binary renders a flow end to end" smoke/
"flag X is recorded with shape Y"              compat/
```

If you're not sure: prefer `macro/`. It runs in CI on every PR,
needs no binary in `dist/` (the test builds its own), and gives the
strongest signal for key-press → command-recording behavior.

#### Adding a `macro/` test (the common case)

1. Author the tape file in `tests/fixtures/tapes/macro/<slug>.tape`.
2. Add a case in `tests/integration/macro_test.go` whose `tapeFile`
   points at the file. **Never inline the tape body in Go**; the
   table struct has no `tape` field on purpose.
3. Run `mise run test:integration`.

#### Adding a `smoke/` test

1. Author the tape in `tests/fixtures/tapes/smoke/<slug>.tape`.
2. The smoke runner discovers it via glob — no Go test to write.
3. Run `mise run build && mise run test:macro`.
4. Use `assert view <text>` lines inside the tape itself for
   pass/fail — there is no `wantStdout`-style assertion.

#### Adding a `compat/` test

1. Author the tape in `tests/fixtures/tapes/compat/<slug>.tape`.
2. Add a `TestCompat_<Behavior>` function in
   `tests/integration/compat_test.go` that loads the tape and
   asserts on the recorded command stream.
3. Run `mise run test:integration`.

The tape language is in [Macro language](../reference/macro-language.md).

#### Common authoring mistakes

- **Waiting for terraform output in a tape.** MacroService never
  executes terraform; tapes wait for plugin views only.
- **Writing a screenshot to an absolute path in a Go-driven test.**
  Use a relative filename and set `cwdFunc` on the test case so the
  binary runs from `t.TempDir()`.
- **Asserting `wantStdout: "terraform apply\n"` (trailing newline)
  for the apply case.** Apply records `terraform apply <planfile>`;
  use a substring like `"terraform apply "` instead.

### Integration tests (`tests/integration/`, build tag `integration`)

**What they test:** Real-terraform tests of service layer and CLI correctness, plus the runners for `macro/` and
`compat/` tapes.

**When to write:** When changing service implementations, adding new terraform operations, or modifying CLI argument handling.

**Location:** `tests/integration/` with fixtures in `tests/fixtures/`.

```go
func TestPlan_CreateFixture_AgentMode(t *testing.T) {
    result := runPlanAgent(t, "create")
    if s := summarize(result); s.add != 2 {
        t.Errorf("expected add=2, got %d", s.add)
    }
}
```

## What Makes a Good Behavioral Test

A test should answer: **"What does the user see or experience?"**

### Bad: Coverage-Focused

```go
func TestRefresh(t *testing.T) {
    p := setupPlugin()
    cmd := p.Refresh()
    if cmd == nil {
        t.Error("nil")
    }
}
```

This test exercises the code path but proves nothing about behavior. It passes even if `Refresh()` returns a command that does the wrong thing, or fails to reset state.

### Good: Behavior-Focused

```go
func TestRefresh_ShouldResetAndStartLoading(t *testing.T) {
    // Given a plugin showing loaded data
    p, _ := newTestPlugin(&sdktest.MockService{})
    p.status = sdk.StatusDone

    // When refresh is triggered
    cmd := p.Refresh()

    // Then loading begins
    if p.status != sdk.StatusLoading {
        t.Errorf("status = %v, want Loading", p.status)
    }
    if cmd == nil {
        t.Fatal("Refresh() should return a fetch command")
    }
}
```

This test specifies: "After refresh, the plugin should be in loading state." If someone accidentally removes the status reset, this test catches it.

## Test Scenarios to Always Cover

### For Every Plugin Operation

| Scenario | What to verify |
|----------|---------------|
| Happy path | Correct state transition, expected result |
| Error handling | Error status set, error message preserved, graceful recovery |
| Loading guards | Operation is no-op during in-flight loading |
| State transitions | Before and after state fully specified |
| Event emissions | Correct typed event produced with right payload |
| UI feedback | Loading message, error display, success indication |

### For Frames and UI

| Scenario | What to verify |
|----------|---------------|
| Key bindings per state | Each key in each state (done, loading, error, creating) |
| Hints accuracy | Hints match available actions for current state |
| Destructive ops | Confirmation frame pushed, y/n handled correctly |
| Navigation | esc pops, q deactivates, enter inspects |
| Boundary conditions | Empty list, single item, overflow/scroll |

### Case Study: Workspace Plugin

The workspace plugin tests demonstrate the full behavioral pattern:

```go
// Loading guards: operations rejected during in-flight
func TestFrame_GivenLoading_EnterKey_ShouldBeIgnored(t *testing.T) { ... }
func TestFrame_GivenLoading_DeleteKey_ShouldBeIgnored(t *testing.T) { ... }
func TestFrame_GivenLoading_RefreshKey_ShouldBeIgnored(t *testing.T) { ... }
func TestFrame_GivenLoading_NewKey_ShouldBeIgnored(t *testing.T) { ... }

// Delete requires confirmation flow
func TestFrame_GivenDeletableWorkspace_DKey_ShouldPushConfirmFrame(t *testing.T) { ... }
func TestFrame_GivenConfirmFrame_YKey_ShouldTriggerDelete(t *testing.T) { ... }
func TestFrame_GivenConfirmFrame_NKey_ShouldCancelDelete(t *testing.T) { ... }

// Cannot delete current or "default" workspace
func TestFrame_GivenCurrentWorkspace_DKey_ShouldDoNothing(t *testing.T) { ... }
func TestFrame_GivenDefaultWorkspace_DKey_ShouldDoNothing(t *testing.T) { ... }

// Create stays in list (does not pop back)
func TestUpdate_GivenCreateSuccess_ShouldRefreshAndEmitCreatedEvent(t *testing.T) { ... }

// Hints are context-sensitive
func TestHints_GivenLoading_ShouldShowOnlyBack(t *testing.T) { ... }
func TestHints_GivenDone_CursorOnDefault_ShouldNotShowDelete(t *testing.T) { ... }
```

Each test name reads as a specification.

## Anti-Patterns

### 1. Testing internal state without behavioral assertion

```go
// BAD: checks internal field, no behavioral meaning
func TestSetStatus(t *testing.T) {
    p.status = sdk.StatusLoading
    if p.status != sdk.StatusLoading { t.Error("wrong") }
}
```

### 2. Asserting `cmd != nil` without executing it

```go
// BAD: proves nothing about what the command does
func TestActivate(t *testing.T) {
    cmd := p.Activate()
    if cmd == nil { t.Error("nil") }
}

// GOOD: executes the command and verifies its result
func TestActivate_ShouldFetchWorkspaceList(t *testing.T) {
    svc := &sdktest.MockService{
        WorkspaceListFn: func(context.Context) ([]string, error) { return []string{"default", "staging"}, nil },
        WorkspaceFn:     func(context.Context) (string, error) { return "staging", nil },
    }
    p, _ := newTestPlugin(svc)

    batch := p.Activate()().(tea.BatchMsg)

    var result WorkspaceListMsg
    for _, cmd := range batch {
        if msg, ok := cmd().(WorkspaceListMsg); ok {
            result = msg
        }
    }
    if len(result.Workspaces) != 2 {
        t.Errorf("len(Workspaces) = %d, want 2", len(result.Workspaces))
    }
    if result.Current != "staging" {
        t.Errorf("Current = %q, want %q", result.Current, "staging")
    }
}
```

### 3. Testing mock service calls without verifying the plugin's reaction

```go
// BAD: only proves the mock was called
func TestSwitch(t *testing.T) {
    p.SwitchToSelected()
    if len(svc.WorkspaceSelectCalls) != 1 { t.Error("not called") }
}

// GOOD: verifies what the user experiences
func TestSwitchToSelected_GivenDifferentWorkspace_ShouldSetLoadingAndDispatch(t *testing.T) {
    cmd := p.SwitchToSelected()
    if p.status != sdk.StatusLoading { ... }
    if !strings.Contains(p.loadingMsg, "staging") { ... }
    if cmd == nil { ... }
}
```

### 4. Multiple unrelated assertions in one test

```go
// BAD: tests 3 unrelated things, hard to diagnose failures
func TestPlugin(t *testing.T) {
    if p.ID() != "workspace" { t.Error("wrong id") }
    cmd := p.Activate(); if cmd == nil { t.Error("nil") }
    p.MoveDown(); if p.selected != 1 { t.Error("wrong") }
}
```

### 5. Tests that pass regardless of actual behavior

```go
// BAD: this passes even if View returns garbage
func TestView(t *testing.T) {
    view := p.View(80, 24)
    if view == "" { t.Error("empty") }
}

// GOOD: asserts specific content the user would see
func TestView_GivenError_ShouldShowErrorMessage(t *testing.T) {
    p.status = sdk.StatusError
    p.errMsg = "connection failed"
    view := p.View(80, 24)
    if !strings.Contains(view, "connection failed") {
        t.Errorf("view should contain error message")
    }
}
```

### 6. Separate "behavior tests" and "coverage tests" files

There is one test suite that serves both purposes. If you need a separate file to hit coverage, you missed a behavior. Never create `coverage_test.go` or `coverage_gaps_test.go` files — they signal coverage-driven testing, which this project explicitly rejects.

## File Organization

Test files are organized by domain concern, not by coverage goals.

### Splitting Strategy

| Rule | Example |
|------|---------|
| Default: one test file per source file | `loader.go` → `loader_test.go` |
| Split by concern when a source file has distinct responsibilities | `state.go` → `state_test.go` (behavior) + `frames_test.go` (navigation) + `actions_test.go` (batch ops) |
| Golden snapshots get their own file | `golden_test.go` in any package using `sdktest.AssertGolden` |
| Never create files named after testing methodology | No `coverage_test.go`, `unit_test.go`, `integration_test.go` within a package |

### When to split vs merge

- **Split** when a single test file exceeds ~1500 lines AND contains logically separable concerns (different frames, rendering vs behavior, different sub-systems within one source file)
- **Merge** when tests for the same function/type are scattered across multiple files without clear domain separation
- **Each test file name** describes what it tests, not why it was written

### Allowed test file patterns

```
plugins/<name>/
├── <name>_test.go       # Plugin behavior (activation, messages, state transitions)
├── frames_test.go       # Frame-level behavior (key handling, navigation)
├── actions_test.go      # Batch/action-specific behavior
├── output_test.go       # Output formatting behavior
├── rendering_test.go    # View rendering edge cases
└── golden_test.go       # Visual snapshot assertions
```

```
internal/<pkg>/
├── <subject>_test.go    # 1:1 with <subject>.go
└── golden_test.go       # If visual output is tested
```

## Conventions

### Table-Driven Tests

Preferred for validating multiple inputs against the same logic:

```go
func TestIsValidWorkspaceName(t *testing.T) {
    tests := []struct {
        name  string
        valid bool
    }{
        {"default", true},
        {"my-workspace", true},
        {"has space", false},
        {"has/slash", false},
    }
    for _, tt := range tests {
        if got := isValidWorkspaceName(tt.name); got != tt.valid {
            t.Errorf("isValidWorkspaceName(%q) = %v, want %v", tt.name, got, tt.valid)
        }
    }
}
```

### Plugin Test Harness

A plugin test builds its dependencies with `sdktest.NewDeps(svc)` (`pkg/sdk/sdktest/testdeps.go`) and calls `Init`.
`Init` is where a plugin binds its service, logger, Context getter and pin functions (`PluginBase.InitBase`), so a
plugin used without it, or with a `PluginDeps` built by hand, is not the plugin the app runs.

```go
func newTestPlugin(svc sdk.Service) (*Plugin, *sdktest.PluginDepsHarness) {
    h := sdktest.NewDeps(svc)
    p := New(svc).(*Plugin)
    p.Init(h.Deps)
    return p, h
}
```

| Field | Holds |
|-------|-------|
| `h.Ctx` | The `*sdk.Context` that `deps.Context()` returns; set `h.Ctx.Pins` to simulate existing pins |
| `h.PinRequests` | Every address passed to `deps.Pin()` |
| `h.ClearPinsCount` | How many times `deps.ClearPins()` ran |

The harness records pin requests without replaying them onto `h.Ctx`: a test that needs the next snapshot to show a
pin sets `h.Ctx.Pins` itself. Execute a returned `tea.Cmd` before asserting its side effects.

### Test Utilities

| Utility | Location | Purpose |
|---------|----------|---------|
| `sdktest.MockService` | `pkg/sdk/sdktest/mock_service.go` | Shared `sdk.Service` test double |
| `sdktest.AssertGolden` | `pkg/sdk/sdktest/golden.go` | Golden file comparison |
| `sdktest.StripANSI` | `pkg/sdk/sdktest/golden.go` | Remove terminal escape sequences |
| `t.TempDir()` | stdlib | Isolated filesystem for tests |
| `t.Setenv()` | stdlib | Environment variable isolation |

### Coverage Target

100% gate on every package except `cmd/` and `internal/terraform/exec`, enforced by `mise run test:coverage` and in CI
([ADR-0008](../adr/0008-coverage-as-behavioral-gate.md)). If coverage drops below 100%, it means a behavior is untested — find the behavior, write the test.

## Fixtures

### Location

Each scenario is a directory under `tests/fixtures/`; for example:

```
tests/fixtures/
├── create/              # Resources to add (no prior state)
├── delete/              # Resource to remove (has .tfstate)
├── update/              # Resource to change in-place (has .tfstate)
├── multi-resource/      # Batch scenarios
├── no-changes/          # Idempotent — nothing to do
├── plan.json            # Static plan fixture (for macros)
├── state.json           # Static state fixture (for macros)
└── tapes/               # Macro tape files
```

### Conventions

1. **One concern per fixture.** Each tests exactly one scenario.
2. **Local provider only.** No cloud credentials. Use `hashicorp/local`.
3. **Self-contained.** Each has its own `terraform {}` block.
4. **Committed state.** Delete/update fixtures include pre-applied `.tfstate`.

### Creating a New Fixture

```hcl
# tests/fixtures/my-scenario/main.tf
terraform {
  required_providers {
    local = { source = "hashicorp/local", version = "~> 2.5" }
  }
}

resource "local_file" "example" {
  filename = "${path.module}/out/example.txt"
  content  = "example content"
}
```

For fixtures needing pre-existing state:

```bash
cd tests/fixtures/my-scenario
terraform init && terraform apply -auto-approve
git add terraform.tfstate
```

Before you push, run the done check in AGENTS.md; CI runs `mise run test:coverage` at 100%.
