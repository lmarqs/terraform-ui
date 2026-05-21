---
layout: default
title: "ADR-0016: Non-reentrant directory lock for terraform CLI serialization"
grand_parent: Development
parent: Architecture
nav_order: 0016
description: Decision to use a non-reentrant per-directory mutex in ExecService
---

# Non-reentrant directory lock for terraform CLI serialization

Terraform's `.terraform/` directory is not safe for concurrent access. Multiple terraform processes operating on the same working directory corrupt provider binaries, lock files, and plan outputs. This is a hard constraint of the terraform CLI — not a tfui design choice.

BubbleTea dispatches `tea.Cmd` functions as concurrent goroutines. Any two commands that call terraform methods (e.g., `loadWorkspace` + plugin activation, `ctrl+r` refresh + plan) race at the filesystem level. ExecService must serialize all terraform CLI calls per working directory.

The lock is a non-reentrant `sync.Mutex` per directory path, shared across all `WithDir` children of the same root service. Non-reentrant means: a locked method must never call another locked method. The call graph must be flat — public methods acquire the lock and call only unlocked helpers (like `newTerraform`). Violating this deadlocks deterministically, not flakily.

We chose non-reentrant over reentrant because reentrant locks hide call-graph problems — they let you accidentally nest terraform calls without surfacing the architectural violation at development time.

## Considered Options

### Reentrant (recursive) mutex

Would allow locked methods to call other locked methods freely. Rejected — hides call-graph problems. A nested terraform call would silently re-acquire the lock instead of deadlocking, masking the fact that two terraform processes are being composed in ways that may not be safe (e.g., reading state mid-apply).

### Skip loadWorkspace in standalone mode

Simpler fix for the immediate bug (3-line conditional in `app.Init()`). Rejected as sole solution — only patches the specific init race. Does not protect against future concurrent operations (e.g., ctrl+r refresh while plan is running, chdir switch during apply).

### Context-aware acquire (channel-based semaphore)

Would allow a queued goroutine to bail out if its context is cancelled while waiting for the lock. Rejected — unnecessary given the cancel flow. When a user cancels an in-flight operation, the plugin calls `cancelFn()`, which kills the terraform process, which returns an error, which triggers `defer Release()`. The queued goroutine proceeds immediately. There is no scenario where a goroutine blocks indefinitely on `Acquire` without the user being able to unblock it via cancellation or `:q!`.

### No lock, rely on BubbleTea's single-threaded model

BubbleTea's `Update()` loop is single-threaded, but `tea.Cmd` functions (returned from `Update`) run as concurrent goroutines. Any two `tea.Cmd` closures that call terraform methods can race. The single-threaded model only protects state mutations, not side effects.

## Consequences

- Adding a new ExecService method that shells out to terraform requires one `Acquire`/`Release` pair and must not call any other locked method.
- `loadState` is locked internally; callers (`StateList`, `Show`) must not add their own lock.
- If future refactoring composes locked methods, the deadlock will be immediate and deterministic — not a flaky race.

## Known Risks

- **Stale cache reads outside the lock**: `loadState` checks the cache before acquiring the lock. A concurrent `StateRm` (which holds the lock) could invalidate the cache between the check and the lock acquisition. Worst case: a stale read that the user refreshes. Not a crash or corruption — ServiceCache is internally thread-safe.
- **Long holds block the UI**: If `terraform plan` takes 5 minutes, all other terraform operations for that directory (workspace show, state list) queue behind it. The user sees loading spinners. Acceptable because these operations would conflict at the filesystem level anyway.
- **Map grows unboundedly**: Each unique directory path allocates one `*sync.Mutex` that is never reclaimed. Acceptable — a project has a bounded number of members, and each entry is a single pointer.
- **Context cancellation waits for lock**: If a user cancels an operation (`Ctrl+C`), the cancellation propagates via `context.Context` to the terraform process, but the next queued operation still has to wait for the lock to be released after the cancelled process exits. This is correct — terraform needs to clean up before the next operation can safely proceed.
