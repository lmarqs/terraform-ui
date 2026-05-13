---
layout: default
title: Testing Strategy
description: Fixture conventions, test patterns, and how to add new tests
---

# Testing Strategy

## Principle: State-Based Assertions, Path-Agnostic

Don't test "does the UI show the right text." Test "does the terraform state end up correct."

```
INITIAL STATE → [operation via any path] → FINAL STATE
     ↓                                          ↓
  fixtures/                              assertions check
  (committed)                            terraform.tfstate
```

## Equivalence Guarantee

> For any terraform operation, CLI and TUI must produce **identical final terraform state**.

This is the core invariant. Every operation has multiple paths (CLI, TUI, macro) — all must converge on the same outcome.

## Test Layers

| Layer | What it tests | Where | Speed |
|-------|--------------|-------|-------|
| **Unit tests** | Plugin logic, rendering, message handling | `plugins/*/_test.go` | Milliseconds |
| **Golden tests** | Visual output snapshots | `plugins/*/testdata/golden/` | Milliseconds |
| **Integration tests** | Real terraform execution | `tests/integration/` | Seconds |
| **Macro tests** | TUI navigation and assertions | `tests/integration/macro_test.go` | Seconds |

## Running Tests

```bash
# Unit tests (fast, no dependencies)
go test ./...
mise run test

# Integration tests (requires terraform on PATH)
go test -tags integration -timeout 120s ./tests/integration/
mise run test:integration

# Coverage report (90% gate, excludes cmd/)
mise run coverage
```

## Fixtures

### Location

```
tests/fixtures/
├── create/              # 2 resources to add (no prior state)
├── delete/              # 1 resource to remove (has .tfstate)
├── update/              # 1 resource to change in-place (has .tfstate)
├── replace/             # 1 resource force-replacement (has .tfstate)
├── multi-resource/      # 5 resources (batch scenarios)
├── no-changes/          # Idempotent — nothing to do (has .tfstate)
├── dependency-chain/    # Complex dependencies (blast radius)
├── plan.json            # Static plan fixture (AWS, for macros)
├── state.json           # Static state fixture (AWS, for macros)
└── tapes/               # Macro tape files
```

### Conventions

1. **One concern per fixture.** Each fixture tests exactly one scenario.
2. **Local provider only.** No cloud credentials required. Use `hashicorp/local` for files.
3. **Self-contained.** Each fixture has its own `terraform {}` block with provider requirements.
4. **Committed state.** Fixtures testing delete/update/replace include a pre-applied `terraform.tfstate`.
5. **Naming:** kebab-case, grouped by feature prefix (`apply-create`, `state-ops`).

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

If the fixture needs pre-existing state (for delete/update/replace tests):
```bash
cd tests/fixtures/my-scenario
terraform init
terraform apply -auto-approve
# terraform.tfstate is created — commit it
git add terraform.tfstate
```

### Static Fixtures (for Macros)

Static JSON fixtures (`plan.json`, `state.json`) are hand-written and represent AWS resources for realistic TUI testing. They're loaded via `--plan` / `--state` flags in read-only mode.

## Integration Test Patterns

### Plan Test

```go
func TestPlan_MyScenario_AgentMode(t *testing.T) {
    result := runPlanAgent(t, "my-scenario")
    
    if result.Summary.Add != 1 {
        t.Errorf("expected 1 add, got %d", result.Summary.Add)
    }
}
```

### Apply Test (requires temp directory)

```go
func TestApply_MyScenario(t *testing.T) {
    dir := copyFixture(t, "my-scenario")  // isolate from other tests
    
    // Plan first
    runTfui("plan", "--project", dir, "--ci")
    
    // Apply
    stdout, _, err := runTfui("apply", "--project", dir, "--ci")
    if err != nil {
        t.Fatalf("apply failed: %v", err)
    }
    
    // Verify outcome
    assertFileExists(t, filepath.Join(dir, "out/example.txt"))
}
```

### State Mutation Test (requires temp directory)

```go
func TestState_Rm(t *testing.T) {
    dir := copyFixture(t, "state-ops")  // has terraform.tfstate with 2 resources
    
    _, _, err := runTfui("state", "rm", "local_file.one", "--project", dir)
    if err != nil {
        t.Fatalf("state rm failed: %v", err)
    }
    
    // Verify resource removed from state
    assertStateNotContains(t, dir, "local_file.one")
    assertStateContains(t, dir, "local_file.two")
}
```

### Equivalence Test

```go
func TestEquivalence_Apply(t *testing.T) {
    // Path 1: CLI
    dir1 := copyFixture(t, "apply-create")
    runTfui("plan", "--project", dir1, "--ci")
    runTfui("apply", "--project", dir1, "--ci")
    
    // Path 2: Macro (TUI simulation)
    dir2 := copyFixture(t, "apply-create")
    tape := "wait ready\nkey p\nwait view local_file\nkey a\nkey y\nwait view complete"
    runTfuiMacro(t, dir2, tape)
    
    // Both produce identical state
    state1 := readState(t, dir1)
    state2 := readState(t, dir2)
    assertStatesEqual(t, state1, state2)
}
```

## Macro Tape Tests

### What Macros Test

Macros test **UI behavior** — navigation, rendering, keyboard interactions. They run against static fixtures in read-only mode.

```tape
# Verify plan view shows resources
wait ready
key p
assert view aws_instance.web
assert view aws_s3_bucket.data
```

### Adding a New Tape

1. Create `tests/fixtures/tapes/my-test.tape`
2. Add test case in `tests/integration/macro_test.go`:

```go
{
    name:     "my test description",
    tape:     "wait ready\nkey p\nassert view expected_content",
    args:     []string{"--plan", planFixture},
    wantExit: 0,
},
```

### Macro Commands Reference

| Command | Description |
|---------|-------------|
| `key <key>` | Send keypress (p, enter, esc, ctrl+c, space, etc.) |
| `wait ready` | Wait for model to be initialized and not loading |
| `wait view <text>` | Wait for text to appear in current view |
| `assert view <text>` | Fail (exit 1) if text not in current view |
| `screenshot <path>` | Write current view to file |
| `resize <w> <h>` | Change terminal dimensions |
| `sleep <duration>` | Pause (100ms, 2s) |

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | All assertions passed |
| 1 | Assertion failure |
| 2 | Syntax error in tape |
| 3 | Timeout waiting for condition |

## Unit Test Conventions

- **Table-driven:** prefer test cases in slice of structs
- **Mock service:** implements `sdk.Service` with configurable results
- **Temp files:** use `t.TempDir()` for filesystem tests
- **Env vars:** use `t.Setenv()` for environment-dependent tests
- **Same package:** test files in same package (white-box access)
- **Coverage target:** 100% on packages excluding `cmd/` glue
