---
layout: default
title: "ADR-0023: Interrupts reach terraform once, and tfui waits for it"
grand_parent: Development
parent: Architecture
nav_order: 0023
seo:
  description: Decision that tfui owns SIGINT/SIGTERM, interrupts terraform once, waits for it to exit and exits 130/143
description: "Records why tfui forwards one interrupt to terraform, waits for terraform with no timeout, and exits 130 or 143."
audience: "Contributors changing signal handling, the exec service or exit codes"
---

# Interrupts reach terraform once, and tfui waits for it

tfui owns SIGINT and SIGTERM in the TUI and in headless and standalone plugin verb runs; `force-unlock` and the
`workspace` commands, which run terraform without a plugin, end by the signal itself. The first signal interrupts each
running terraform process exactly once and ends the program; tfui then waits, with no timeout, until terraform has
exited, runs plugin cleanup, writes no stdout and exits 130 for SIGINT or 143 for SIGTERM. A second signal, or any
signal while tfui waits, ends tfui at once with the stderr note `terraform was stopped without finishing`, after plugin
cleanup. Quitting the TUI while terraform runs interrupts it the same way, prints
`Waiting for terraform to stop… (Ctrl+C again to force)` while tfui waits, and exits 130 as for SIGINT.

Before this decision, bubbletea's own signal handler turned a SIGINT into an ordinary quit. A headless `tfui plan`
interrupted mid-plan exited 0 and printed `Plan: 0 to add, 0 to change, 0 to destroy.`, and terraform disappeared
within a second, though its graceful stop takes as long as its in-flight provider call (16 seconds in the
measurement). On Linux, terraform-exec starts terraform in its own process group with `Pdeathsig: SIGKILL`, so a
terminal's Ctrl+C never reaches terraform and tfui exiting kills it: no graceful stop, no state lock release, no
state write. A plan interrupted gracefully still writes its `-out` file at the end of its stop, after any cleanup
that runs earlier.

The full analysis, with the measurements, requirements and every option weighed, is the decision document on
[issue #100](https://github.com/lmarqs/terraform-ui/issues/100#issuecomment-6880812818).

## Consequences

- A run tracker in the exec service owns every terraform process the session starts. Each exec service call that
  starts terraform registers with it; interrupting cancels the calls' contexts, so terraform-exec sends each process
  one SIGINT. Every `WithDir` copy of the service shares the one tracker. A new exec service method that runs
  terraform must register too.
- terraform-exec's 60-second kill after an interrupt is turned off (`SetWaitDelay(0)`). If terraform never stops,
  tfui waits until the user forces the exit or a CI job timeout ends it.
- tfui forwards the first signal only when terraform runs in its own process group, which terraform-exec arranges on
  Linux only. Elsewhere terraform shares tfui's process group and already receives the terminal's Ctrl+C; forwarding
  it as well would be terraform's second interrupt, which exits it immediately. A CI cancel on macOS that signals
  tfui alone therefore leaves terraform uninterrupted until a second signal, and that forced exit leaves terraform
  running, since only Linux has `Pdeathsig`: terraform can still write its `-out` plan file after cleanup removed it.
- A forced exit on Linux lets the kernel SIGKILL terraform, as every exit did before this decision. terraform-exec
  offers no way to send a second interrupt, so terraform's own "exiting immediately" path is not reached.
- 130 and 143 are the POSIX shell's codes for a process ended by SIGINT and SIGTERM. They join the exit codes in the
  CLI I/O contract and stay distinct from terraform's failure code 1, so a script can tell a cancelled run from a
  failed one.
- Plugin cleanup (ADR-0020) runs after terraform has exited, so the plan file an interrupted plan writes is removed.
- Signal handling lives in `cmd/tfui` and the tracker in `internal/terraform/exec`; no plugin changes (ADR-0021).

## Considered Options

### Do nothing

Rejected. An interrupted run is reported as success, and terraform loses its graceful stop.

### Only turn off bubbletea's signal handler

Rejected. Go then dies by the signal, so the exit code is honest, but terraform is still killed and no cleanup runs.

### Each plugin waits for its own terraform call in `Shutdown`

Rejected. The waiting logic spreads over Plan, Apply, State and every other plugin that runs terraform, and a plugin
left out runs untracked.

### Start plan and apply outside terraform-exec

Rejected for its size. Starting terraform directly would let tfui forward a real second SIGINT and reach terraform's
own immediate exit on every platform, at the cost of replacing terraform-exec for the main operations.

### Always forward the first signal

Rejected. On macOS one Ctrl+C would reach terraform twice and force its immediate exit.

### A timeout on the wait

Rejected. Any fixed bound, such as terraform-exec's 60 seconds, SIGKILLs a slow but healthy stop; the second signal
is the way out.

### Pass terraform's exit code through after a signal

Rejected. An interrupted terraform exits 1, so a cancel would look like a failure.
