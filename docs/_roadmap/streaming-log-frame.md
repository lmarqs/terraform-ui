---
title: Streaming Log Frame for Terraform Commands
status: idea
priority: high
created: 2026-05-21
effort: medium
tags: [ux, plan, apply, init, streaming]
depends_on: []
---

## Summary

Show real-time terraform output during `plan`, `apply`, and `init` execution instead of a static spinner.

## Need

Users running terraform commands have no visibility into what is happening during execution:
- `init` can be slow (downloading providers) with zero feedback
- `apply` shows only "Applying changes... 5s" while resources are being created/modified/destroyed
- `plan` shows only "Running terraform plan... Xs" while state refresh happens
- On failure, the error context disappears when the view transitions

Current workaround: run terraform in a separate terminal to see output.

## Expected UX

A reusable `StreamFrame` (in `pkg/sdk/frames/`) is pushed onto the plugin's frame stack during command execution:

- Output lines appear in real-time as terraform produces them
- Auto-scrolls to the bottom; pauses if the user scrolls up manually
- `G` jumps back to the bottom and resumes auto-scroll

**After success:** auto-navigates to the natural result (plan→tree, apply→back to plan, init→deactivates) — identical to today's behavior. The log remains accessible via `L` from the result view.

**After error:** remains on the log so the user can read the full failure output before pressing `Esc`.

**Cancellation:**
- `^c` (1×) → sends SIGINT (terraform graceful shutdown)
- `^c` (2×) → shows confirmation: "Force cancel? Infrastructure may be left in partial state. (y/n)"

## Design Decisions

| Decision | Choice |
|----------|--------|
| Surface | TUI only (CI mode unchanged) |
| Placement | Full-screen frame within the plugin (frame stack) |
| Component | Reusable `StreamFrame` in `pkg/sdk/frames/` |
| Streaming mechanism | `io.Writer` added to `PlanOptions`, `ApplyOptions`, `InitOptions`; ExecService wires to terraform-exec `WithStdout`/`WithStderr`; MacroService ignores (no-op) |

## Implementation Sketch

1. Add `Writer io.Writer` field to `PlanOptions`, `ApplyOptions`, `InitOptions` in `pkg/sdk/types.go`
2. Create `pkg/sdk/frames/stream.go` — `StreamFrame` implementing the `Frame` interface with line buffer, auto-scroll, scrollbar gutter, and internal `ConfirmFrame` overlay for force-cancel
3. Wire `io.Writer` in `internal/terraform/exec/service.go` via `WithStdout`/`WithStderr`
4. Integrate in `plugins/plan/plan.go`, `plugins/apply/apply.go`, `plugins/init/result_frame.go`
