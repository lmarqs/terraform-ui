[PRD]
# PRD: Interrupts reach terraform once, and tfui waits for it (#100)

## Overview

tfui has no signal handling of its own. bubbletea's signal handler turns SIGINT into an ordinary quit, so a headless
`tfui plan` interrupted mid-plan exits 0 and prints `Plan: 0 to add, 0 to change, 0 to destroy.` (measured). On
Linux, terraform-exec starts terraform in its own process group with `Pdeathsig: SIGKILL`, so tfui exiting kills
terraform: no graceful stop, no state lock release, no state write. Quitting the TUI during a plan or apply does the
same. A plan interrupted gracefully still writes its `-out` file at the end of its stop, after any cleanup that runs
earlier, and that file holds variable values in plaintext.

The decision is ADR-0023 (`docs/adr/0023-interrupts-reach-terraform-once.md`); the full analysis, measurements and
requirements F1–F7 are the decision document on #100. This PRD implements it: tfui owns SIGINT/SIGTERM, interrupts
each running terraform process once, waits with no timeout for terraform to exit, runs plugin cleanup, writes no
stdout and exits 130 (SIGINT) or 143 (SIGTERM). A second signal, or any signal while tfui waits, exits at once.

## Goals

- G1: an interrupted run is never read as success: exit 130/143, no plan summary on stdout (F1, F2).
- G2: terraform gets exactly one interrupt and finishes its graceful stop before tfui exits (F3, F4, F6).
- G3: plugin cleanup runs after terraform exits, so no temp plan file outlives tfui (F5).
- G4: a second signal, or a signal while waiting, gives the shell back at once (F7).

## Quality Gates

These commands must pass for every user story:
- `mise run check:lint` - Lint
- `mise run test:unit` - Unit tests
- `mise run check:build` - Build
- `mise run test:coverage` - Coverage, which must stay at 100% outside `cmd/` and `internal/terraform/exec`

Project rules that apply to every story (AGENTS.md):
- The `test-writer` agent writes the failing test before any implementation code.
- Tests are table-driven `_test.go` files beside the code.
- Comments record only reasons the code cannot show.
- Conventional commits; each story is pushed to `main` as its own commit with `Refs #100`.

## User Stories

### US-001: Track and interrupt running terraform processes
**Description:** As the tfui session, I want to interrupt every running terraform process and wait for each to exit,
so that shutdown never leaves terraform running or SIGKILLed mid-operation (F3, F4, F6).

**Acceptance Criteria:**
- [ ] New file `internal/terraform/exec/runs.go` defines `Runs`, which tracks the terraform calls the session started:
  - `Track(ctx) (context.Context, func())`: returns a context cancelled when either the caller's context or
    `Interrupt` cancels, and a `done` func.
  - `Interrupt()`: idempotent; cancels every tracked context, including any tracked later.
  - `Wait()`: blocks until every tracked call has called `done`.
  - `Running() bool`: reports whether a tracked call has not yet called `done`.
- [ ] `NewExecService` creates one `*Runs`; `WithDir` copies share it, as they share `dirLock`.
- [ ] `ExecService` exposes the shared tracker to `cmd/tfui` through an accessor, for example `Runs() *Runs`.
- [ ] Every `ExecService` method that starts terraform (`service.go`, `state_ops.go`, `workspace_ops.go`, and
  `runRawJSON`) derives its context from `Track` and defers `done`.
- [ ] `newTerraform()` calls `tf.SetWaitDelay(0)`, so terraform-exec no longer kills terraform 60s after an interrupt.
- [ ] `internal/terraform/exec` defines `TerraformHasOwnProcessGroup` per build target: `true` in a `_linux.go` file,
  `false` in a `!linux` file, with a doc comment saying terraform-exec sets `Setpgid` only on Linux.
- [ ] Unit tests: `Interrupt` cancels contexts tracked before and after it; cancelling the caller's context cancels
  only that tracked context; `Wait` returns only after every `done`; `Running` reflects outstanding calls; a `WithDir`
  copy shares the same tracker.

### US-002: Handle SIGINT/SIGTERM and wait for terraform on exit
**Description:** As a tfui user, I want Ctrl+C, a CI cancel or quitting the TUI to stop terraform gracefully, report
the run as interrupted and clean up, so that my state lock is released and no plaintext plan file is left behind
(F1, F2, F3, F4, F5, F6, F7).

**Acceptance Criteria:**
- [ ] Both `Session.Run` (TUI) and `RunPluginFrom` (verb runs) build the BubbleTea program with
  `tea.WithoutSignalHandler()`, so tfui is the only owner of SIGINT/SIGTERM.
- [ ] `cmd/tfui` adds an interrupt handler per run: `signal.Notify` for SIGINT and SIGTERM, stopped with
  `signal.Stop` when the run returns. It depends on a small interface `*exec.Runs` satisfies (`Interrupt`, `Wait`,
  `Running`), a signal channel, a way to end the program, a stderr writer and an exit func, all injectable.
- [ ] **First signal:** the handler records it, calls `Interrupt()` only when `exec.TerraformHasOwnProcessGroup` is
  true, and ends the program.
- [ ] **Forced exit:** a signal after the first, or any signal while tfui waits, writes
  `terraform was stopped without finishing` to stderr, runs `registry.Shutdown()`, and exits immediately with 130
  (SIGINT) or 143 (SIGTERM).
- [ ] **End of run** (program or headless driver returned, including a TUI quit): if no signal was received, call
  `Interrupt()`; if `Running()`, write `Waiting for terraform to stop… (Ctrl+C again to force)` to stderr and call
  `Wait()`; the deferred `registry.Shutdown()` runs after `Wait()` returns.
- [ ] **After a received signal:** tfui writes no stdout (no plan summary, no recorded macro commands), writes the
  output-port owner's stderr as is, and `RunE` returns `&exitError{code: 130}` for SIGINT or
  `&exitError{code: 143}` for SIGTERM, taking precedence over the plugin's exit code.
- [ ] A macro run (`MacroService`, no terraform processes) prints nothing extra and never waits.
- [ ] Unit tests (table-driven, fake signal channel and fake tracker): first SIGINT with and without
  `TerraformHasOwnProcessGroup`; second signal → stderr note, `Shutdown`, exit 130/143 in that order; a signal while
  waiting → forced exit; normal end with nothing running → no output, no wait; normal end with terraform running →
  `Interrupt`, waiting notice, `Wait`, then `Shutdown`; after SIGINT and SIGTERM → no stdout and the matching
  `exitError`.

### US-003: Document interrupt behaviour and exit codes
**Description:** As a tfui user or contributor, I want the docs to state what Ctrl+C does and which exit codes follow,
so that scripts and CI can rely on them.

**Acceptance Criteria:**
- [ ] `docs/reference/cli-io-contract.md` Exit Codes table lists `130` (interrupted by SIGINT) and `143` (terminated
  by SIGTERM), scope all commands, and a paragraph states: the first signal stops terraform gracefully and tfui waits
  for it with no timeout; a second signal exits at once; no stdout is written after a signal.
- [ ] `docs/adr/0020-planfile-lifecycle-ownership.md`: on a signal or TUI quit, `Shutdown` runs after terraform has
  exited; a forced exit skips the wait and still runs `Shutdown`; the sentence saying a signal skips `Shutdown` goes.
- [ ] `pkg/sdk/plugin.go` `Shutdowner` doc comment matches the ADR-0020 text.
- [ ] The `tfui-cli-ux` skill's exit-code rules (`.agents/skills/`) list 130 and 143.
- [ ] Every identifier and path named in the changed docs exists in the code.

## Functional Requirements

- FR-1: Every terraform process started through `ExecService` is registered with the session's shared tracker until
  its call returns.
- FR-2: `Interrupt()` cancels every tracked context, so terraform-exec sends each process one SIGINT; it is
  idempotent.
- FR-3: terraform-exec's post-interrupt kill is disabled (`SetWaitDelay(0)`); tfui waits for terraform with no timeout.
- FR-4: tfui is the only owner of SIGINT/SIGTERM; bubbletea's signal handler is off.
- FR-5: On the first signal, tfui calls `Interrupt()` only if `TerraformHasOwnProcessGroup`, and ends the program.
- FR-6: On a second signal, or any signal while waiting, tfui writes `terraform was stopped without finishing` to
  stderr, runs `registry.Shutdown()`, and exits immediately with 130 (SIGINT) or 143 (SIGTERM).
- FR-7: When a run ends with no signal, tfui calls `Interrupt()`; if terraform still runs, it writes
  `Waiting for terraform to stop… (Ctrl+C again to force)` to stderr and waits.
- FR-8: `registry.Shutdown()` runs after every tracked call has returned, except on a forced exit (FR-6).
- FR-9: After a received signal, tfui writes no stdout and exits 130 (SIGINT) or 143 (SIGTERM).

## Non-Goals

- Windows: tfui ships only for linux and darwin (`.goreleaser.yaml`).
- Starting terraform outside terraform-exec to forward a real second SIGINT (rejected in ADR-0023).
- A timeout on the wait for terraform (ADR-0023).
- Interrupting terraform on macOS when only tfui is signalled (ADR-0023).
- Killing grandchild processes that ignore SIGINT, such as a provider's external program.
- A latency target for the forced exit: dropped as not worth measuring; a unit test proves it is immediate.
- Integration tests against real terraform.

## Technical Considerations

- terraform-exec v0.25.2 `tfexec/cmd.go`: context cancel → `cmd.Cancel` sends `os.Interrupt`; `WaitDelay` defaults
  to 60s. With `WaitDelay` 0, Go's `exec.Cmd.Wait` never kills after `Cancel`.
- `tfexec/cmd_linux.go` sets `Setpgid: true` and `Pdeathsig: SIGKILL`; `cmd_default.go` (`!linux`) sets neither.
- bubbletea v1.2.4 `handleSignals` turns SIGINT/SIGTERM into `QuitMsg` unless the program is built with
  `tea.WithoutSignalHandler()`.
- In the TUI, Ctrl+C in raw mode is a key (`internal/ui/app.go` binds it to `tea.Quit`), not a signal. Once the
  program exits and the terminal is restored, Ctrl+C becomes SIGINT, which is how the "again to force" path works.
- `runRawJSON` (`show`/`validate`/`output`/`version -json`) uses `exec.CommandContext`, whose default cancel kills;
  these calls are read-only and short, so tracking them for `Wait` is enough.
- Exit codes reach `main` as `exitError` (#99), so deferred cleanup runs; only the forced exit (FR-6) calls the exit
  func directly.
- Plugins import only `pkg/sdk` (ADR-0021): all wiring lives in `cmd/tfui` and `internal/terraform/exec`.
- Testing a signal by hand needs a default SIGINT disposition: a background job (`cmd &`) in a non-interactive shell
  starts with SIGINT ignored.

## Success Metrics

- Rerunning the #100 measurement on Linux (`tfui plan --ci` on a plan with a slow data source, one SIGINT): exit 130,
  no `Plan:` line on stdout, terraform logs one "Interrupt received" and exits before tfui, no `/tmp/tfui-*.tfplan`.
- Coverage stays at 100%, with a table case for every new branch.

## Open Questions

- None. Decisions are in ADR-0023 and on #100.
[/PRD]
