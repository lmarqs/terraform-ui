---
layout: default
title: Testing Strategy
parent: Development
nav_order: 2
description: Behavioral testing standard, conventions, and layered test architecture
---

# Testing Strategy

## Philosophy

Tests describe **behavior**, not implementation. A test that exercises a code path without asserting user-visible outcomes is worse than no test — it creates false confidence.

This project suffered bugs that passed 100% code coverage but failed behavioral correctness. The root cause: tests focused on exercising code paths (coverage-driven) rather than specifying behavior. For example, the workspace plugin had 100% coverage but tests never verified that:

- The header preserves chdir after workspace changes
- Operations are guarded during loading state
- Delete requires confirmation before execution
- Creating a workspace stays in the list (does not pop back)

The standard is now clear:

1. **BDD/TDD is non-negotiable** — tests describe BEHAVIOR, not implementation
2. **Coverage is a side-effect** of good behavior tests, never the goal
3. **Tests are the living spec** — if a test does not describe a user-visible behavior, it is wrong
4. **One test suite** — there is no separation between "behavior tests" and "coverage tests"

## Test Naming Convention

Use `Test{Subject}_{Given/When}_{ExpectedBehavior}` format. The name should read as a specification:

```go
// Good: describes a scenario and expected outcome
func TestSwitchToSelected_GivenDifferentWorkspace_ShouldSetLoadingAndDispatch(t *testing.T)
func TestFrame_GivenLoading_EnterKey_ShouldBeIgnored(t *testing.T)
func TestUpdate_GivenCreateSuccess_ShouldRefreshAndEmitCreatedEvent(t *testing.T)
func TestHints_GivenDone_CursorOnDeletable_ShouldShowDeleteAndSelect(t *testing.T)
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
2. **Run it** — confirm it fails for the RIGHT reason (not a compile error or wrong assertion)
3. **Implement** the minimum code to pass
4. **Refactor** while tests stay green
5. **Coverage should naturally be 100%** — if not, you missed a behavior

The `test-writer` agent MUST be invoked before any implementation edit. This enforces the "failing test first" discipline.

## Running Tests

```bash
# Unit tests (fast, no external dependencies)
mise run test:unit

# Coverage report (100% gate, excludes cmd/ glue)
mise run test:coverage

# Macro tests (TUI flows against built binary)
mise run test:macro

# Integration tests (requires terraform/tofu on PATH)
mise run 'test:integration:*'

# Full verification before commit
mise run check:lint && mise run test:unit
```

## Test Layers

### Unit Tests (`*_test.go` in same package)

**What they test:** Behavior of a single plugin or component — state transitions, event emissions, command dispatch, loading guards, UI feedback.

**When to write:** For every plugin operation, every frame interaction, every event handler.

**Location:** Same package as the code under test (white-box access).

```go
func TestDeleteWorkspace_ShouldSetLoadingAndDispatch(t *testing.T) {
    svc := &mockService{}
    p := New(svc).(*Plugin)
    p.svc = svc
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

**Mock services:** Implement `sdk.Service` with configurable return values. Keep them minimal — only the methods your test exercises need real logic.

```go
type mockService struct {
    workspaceList      []string
    workspaceListErr   error
    workspaceSelectErr error
    // ... only what's needed
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

### App-Level Integration Tests (`internal/ui/app_test.go`)

**What they test:** Cross-cutting flows that span multiple components — navigation between plugins, event dispatch causing header updates, `returnTo` behavior after push/pop.

**When to write:** When a behavior depends on how the app routes messages between plugins (e.g., "workspace change should update header chdir display").

These tests verify the contract between the app shell and plugins — things no single plugin test can catch.

### Macro Tests (`tests/macro/`)

**What they test:** End-to-end TUI flows against the real compiled binary. Tape DSL drives keystrokes and asserts on rendered output.

**When to write:** After modifying `View()`, layout, navigation flow, or any visual behavior.

```tape
# Verify workspace list renders after activation
wait ready
key w
wait view default
assert view staging
assert view production
```

**Adding a macro test:**

1. Create tape file in `tests/fixtures/tapes/`
2. Add test case in `tests/integration/macro_test.go`
3. Run with `mise run test:macro`

**Tape commands:**

| Command | Description |
|---------|-------------|
| `key <key>` | Send keypress |
| `wait ready` | Wait for initialization complete |
| `wait view <text>` | Wait for text to appear in current view |
| `assert view <text>` | Fail if text not in current view |
| `screenshot <path>` | Write current view to file |
| `resize <w> <h>` | Change terminal dimensions |
| `sleep <duration>` | Pause execution |

### Integration Tests (`tests/integration/`)

**What they test:** Real terraform/tofu/terragrunt execution. Service layer correctness against actual CLI.

**When to write:** When changing service implementations, adding new terraform operations, or modifying CLI argument handling.

**Location:** `tests/integration/` with fixtures in `tests/fixtures/`.

```go
func TestPlan_CreateFixture_ShouldReport1Addition(t *testing.T) {
    result := runPlanAgent(t, "create")
    if result.Summary.Add != 1 {
        t.Errorf("expected 1 add, got %d", result.Summary.Add)
    }
}
```

## What Makes a Good Behavioral Test

A test should answer: **"What does the USER see or experience?"**

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
    // Given a plugin with stale data and a non-zero selection
    svc := &mockService{workspaceList: []string{"default"}, workspace: "default"}
    p := New(svc).(*Plugin)
    p.svc = svc
    p.status = sdk.StatusDone
    p.selected = 5

    // When refresh is triggered
    cmd := p.Refresh()

    // Then loading begins and selection is reset
    if p.status != sdk.StatusLoading {
        t.Errorf("status = %v, want Loading", p.status)
    }
    if cmd == nil {
        t.Fatal("Refresh() should return a fetch command")
    }
}
```

This test specifies: "After refresh, the plugin should be in loading state." If someone accidentally removes the status reset, this test catches it.

### Another Example: Testing Command Results

```go
// Bad: only checks cmd is non-nil
func TestActivate(t *testing.T) {
    p := setup()
    cmd := p.Activate()
    assert(cmd != nil)
}

// Good: executes the command and verifies its result
func TestActivate_ShouldFetchWorkspaceList(t *testing.T) {
    svc := &mockService{workspaceList: []string{"default", "staging"}, workspace: "staging"}
    p := New(svc).(*Plugin)
    p.Init(&sdk.Context{Service: svc})

    cmd := p.Activate()
    msg := cmd()

    result, ok := msg.(WorkspaceListMsg)
    if !ok {
        t.Fatalf("Activate cmd returned %T, want WorkspaceListMsg", msg)
    }
    if len(result.Workspaces) != 2 {
        t.Errorf("len(Workspaces) = %d, want 2", len(result.Workspaces))
    }
    if result.Current != "staging" {
        t.Errorf("Current = %q, want %q", result.Current, "staging")
    }
}
```

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
func TestHints_GivenDone_CursorOnDeletable_ShouldShowDeleteAndSelect(t *testing.T) { ... }
func TestHints_GivenDone_CursorOnCurrent_ShouldHideDelete(t *testing.T) { ... }
```

Each test name reads as a specification. Together, they form the complete behavioral contract.

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

// GOOD: executes the command and verifies behavior
func TestActivate_ShouldFetchWorkspaceList(t *testing.T) {
    cmd := p.Activate()
    msg := cmd()
    result := msg.(WorkspaceListMsg)
    if len(result.Workspaces) != 2 { ... }
}
```

### 3. Testing mock service calls without verifying the plugin's reaction

```go
// BAD: only proves the mock was called
func TestSwitch(t *testing.T) {
    p.SwitchToSelected()
    if !svc.selectCalled { t.Error("not called") }
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

There is ONE test suite that serves both purposes. If you need a separate file to hit coverage, you missed a behavior. Never create `coverage_test.go` or `coverage_gaps_test.go` files — they signal coverage-driven testing, which this project explicitly rejects.

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
- **Each test file name** describes WHAT it tests, not WHY it was written

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

### Mock Services

- Implement `sdk.Service` with no-op defaults
- Only configure the methods your test exercises
- Use descriptive field names: `workspaceListErr`, not `err1`

### Test Utilities

| Utility | Location | Purpose |
|---------|----------|---------|
| `sdktest.AssertGolden` | `pkg/sdk/sdktest/golden.go` | Golden file comparison |
| `sdktest.StripANSI` | `pkg/sdk/sdktest/golden.go` | Remove terminal escape sequences |
| `t.TempDir()` | stdlib | Isolated filesystem for tests |
| `t.Setenv()` | stdlib | Environment variable isolation |

### Coverage Target

100% gate on all packages excluding `cmd/` glue. This is enforced by `mise run test:coverage`. If coverage drops below 100%, it means a behavior is untested — find the behavior, write the test.

## Fixtures

### Location

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

## Summary

| Principle | Rule |
|-----------|------|
| Tests describe behavior | Every test name reads as a spec |
| Coverage is a side-effect | Never write a test "for coverage" |
| TDD is mandatory | Failing test before implementation |
| One suite serves all purposes | No separate "coverage" files |
| Execute commands fully | Never stop at `cmd != nil` |
| Assert user-visible outcomes | Status, messages, events, view content |
| Guard all states | Loading, error, empty, boundary |
| Confirm destructive ops | Push confirmation frame, test y/n/esc |
