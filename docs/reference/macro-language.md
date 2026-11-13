---
layout: default
title: Macro Language
parent: Reference
nav_order: 3
seo:
  description: Tape DSL reference for automated TUI interaction
description: "Specifies the tape language macros are written in, and how to record and replay them."
audience: "Contributors writing macro tapes or tests"
---

# Macro Language

A tape is a text file of one command per line (`key`, `wait`, `assert`, `screenshot`, `resize`, `sleep`, `emit`). `tfui -macro ./file.tape` replays it headlessly against pre-loaded or empty data and prints the terraform commands the session would run. Tapes verify a view after a change, assert rendered content in CI, and capture screenshots for demos.

## Invocation

No TTY is required: macro mode drives the BubbleTea model directly.

```bash
# Run tape against a plan file
tfui -plan ./plan.json -macro ./scripts/verify-plan.tape

# Run tape against state
tfui -state ./state.json -macro ./scripts/check-state.tape

# Both plan and state
tfui -plan ./plan.json -state ./state.json -macro ./scripts/full-check.tape

# Without pre-loaded data (for project-level flows like chdir selection)
tfui -project ./my-infra -macro ./scripts/test-chdir.tape

# Tape from stdin (one line, no trailing newline, semicolons between commands)
printf 'wait ready; key p; assert view aws_instance' | tfui -plan ./plan.json -macro -

# CI pipeline
terraform show -json tfplan.out | tfui -plan - -macro ./tests/verify-plan.tape
```

When `-plan`/`-state` are provided, the app renders pre-loaded data. Without them, reads return empty data and the app starts at the chdir picker (for testing project-level flows like chdir selection). In both cases, terraform never runs.

## Recording (`-record`)

The `-record <dir>` flag captures rendered frames during macro playback (or interactive use). It is orthogonal to `-macro`:

```bash
# Record macro playback as frames (for GIF generation)
tfui -plan ./plan.json -macro ./demo.tape -record ./recording/

# Record an interactive session (generates tape + frames)
tfui -plan ./plan.json -record ./my-session/
```

Output directory structure:
```
recording/
├── manifest.json       # Frame timing + terminal dimensions
├── recording.tape      # Reconstructed tape (interactive mode only)
├── session.log         # Debug log (JSON lines)
├── frame_0001.txt      # Rendered frame at start, then after each command
├── frame_0002.txt
└── ...
```

The generated `recording.tape` is directly replayable: `tfui -macro ./recording/recording.tape`.

Frames can be stitched into GIFs using `demo/stitch.sh` (converts to asciicast, renders via `agg` or `vhs`).

## Command Output (stdout)

Every terraform operation triggered during macro playback, plan runs included, is recorded and printed to stdout after the macro completes. This enables piping macros into a shell:

```bash
# Generate and execute terraform commands via macro
tfui -plan ./plan.json -state ./state.json -macro ./apply.tape | sh

# Use tofu instead of terraform
tfui -plan ./plan.json -state ./state.json -macro ./apply.tape -terraform-bin tofu | sh

# Preview what would run
tfui -plan ./plan.json -state ./state.json -macro ./taint.tape
# stdout:
# terraform taint aws_instance.web
```

## Safety

Macro mode never runs the terraform binary. Plan, apply, state rm, taint, import and the other operations are recorded and printed to stdout; the user chooses whether to execute them by piping to `sh`.

Reads (plan data, state list, show) come from the `-plan`/`-state` data; without it they return empty results, and the workspace reads as `default`.

## Tape Format

One command per line. Empty lines and lines starting with `#` are ignored.

```tape
# This is a comment

key p
wait ready
assert view aws_instance
screenshot /tmp/plan-view.txt
```

A comment takes a whole line: text after a command is read as its arguments.

A tape given on stdin as a single line, with no trailing newline, may separate commands with semicolons:

```bash
printf 'key p; wait ready; assert view aws_instance' | tfui -plan ./plan.json -macro -
```

## Commands

### `key <key>`

Send a key event to the TUI.

```tape
key p
key enter
key space
key ctrl+t
key /
```

**Supported special keys:** `enter`, `esc`, `tab`, `backspace`, `up`, `down`, `left`, `right`, `space`, `ctrl+c`, `ctrl+r`, `ctrl+w`, `ctrl+t`, `ctrl+s`

**Single characters:** any single printable character (`a`–`z`, `0`–`9`, punctuation)

### `wait ready`

Block until the rendered view is non-empty and contains no `Loading`. Times out after 5 seconds, like every `wait`.

```tape
key p
wait ready
```

### `wait view <substring>`

Block until the rendered view contains the specified substring.

```tape
key p
wait view to add
```

The substring is everything after `view `, with runs of spaces collapsed to one: `wait view 1 to add` matches "Plan: 1 to add".

### `assert view <substring>`

Immediately check that the rendered view contains the substring. Fails the macro if not found.

```tape
key p
wait ready
assert view aws_instance
```

### `screenshot <path>`

Write the current rendered view to a file, styling included as the terminal's color profile renders it: raw ANSI escape codes on a color terminal, plain text otherwise.

```tape
screenshot /tmp/plan-output.txt
screenshot ./golden/state-view.txt
```

### `resize <width> <height>`

Change the terminal dimensions for rendering.

```tape
resize 120 40
```

The default size is 80x24.

### `sleep <duration>`

Pause execution. Use sparingly — primarily for demos and recordings.

```tape
sleep 500ms
sleep 2s
```

Accepts Go duration format: `100ms`, `1s`, `1m30s`.

### `emit <event>`

Inject an app event, as if it came from the event bus: `stale` (plan invalidated), `refreshed` (state refreshed), `lock <who>` (state lock detected) or `lock-clear` (lock cleared).

```tape
emit lock deploy-bot
emit lock-clear
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | All assertions passed |
| 1 | Assertion failure, or a screenshot that cannot be written |
| 2 | Syntax error in tape |
| 3 | Timeout waiting for condition |

## Examples

### Apply all planned changes

```tape
wait ready
key p
wait view aws_instance.web
key a
wait view Are you sure
key y
```

```bash
# Outputs: terraform plan -out=<tmp>.tfplan, then terraform apply <tmp>.tfplan
tfui -plan ./plan.json -state ./state.json -macro ./apply.tape | sh
```

### Apply targeted (pinned resource)

```tape
wait ready
key p
wait view aws_instance.web
key space
key a
wait view Are you sure
key y
```

```bash
# Outputs: the plan run on entry, terraform plan -out=<tmp>.tfplan -target=aws_instance.web,
# then terraform apply <tmp>.tfplan (targets go on plan, never on apply: ADR-0019)
tfui -plan ./plan.json -state ./state.json -macro ./apply-targeted.tape | sh
```

### Taint a resource

```tape
wait ready
key s
wait view aws_instance.web
key t
wait view Taint
key y
```

```bash
# Outputs: terraform taint aws_instance.web
```

### Delete from state

```tape
wait ready
key s
wait view aws_instance.web
key d
wait view Remove
key y
```

```bash
# Outputs: terraform state rm aws_instance.web
```

### CI visual regression

```bash
#!/bin/bash
terraform show -json tfplan.out > plan.json
tfui -plan ./plan.json -macro ./tests/macros/verify-plan.tape
```

### CI apply via macro

```bash
#!/bin/bash
terraform show -json tfplan.out > plan.json
terraform state pull > state.json
tfui -plan ./plan.json -state ./state.json -macro ./deploy.tape | sh
```

## Programmatic Driver (Go API)

For Go test files, use the driver directly instead of tape:

```go
import "github.com/lmarqs/terraform-ui/internal/macro"

func TestPlanView(t *testing.T) {
    app := newApp(t) // any tea.Model, typically a ui.App over a test service
    d := macro.NewDriver(app, 80, 24)
    d.Init()

    d.SendKey("p")
    err := d.WaitUntil(func(v string) bool {
        return strings.Contains(v, "create")
    }, 5*time.Second)
    if err != nil {
        t.Fatal(err)
    }

    if !d.ViewContains("aws_instance") {
        t.Error("expected aws_instance in view")
    }
}
```

## Limitations

- No loops, conditionals, or variables (use Go tests for complex logic)
- Every `wait` times out after 5 seconds
