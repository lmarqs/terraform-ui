---
layout: default
title: Macro Language
parent: Reference
nav_order: 3
description: Tape DSL reference for automated TUI interaction
---

# Macro Language

tfui includes a macro system for automated TUI interaction. Macros are used for testing, CI visual regression, and repeatable workflows.

## Purpose

The macro system serves three roles:

1. **Self-verification** — after modifying a plugin's View(), run a tape to confirm the output without manually opening the TUI
2. **CI visual regression** — tape files assert expected content in rendered views; failures produce non-zero exit codes
3. **Reproducible demos** — capture screenshots at specific navigation points for documentation

## Invocation

No TTY is required — macro mode drives the BubbleTea model directly without opening a terminal.

```bash
# Run tape against a plan file
tfui -plan ./plan.json -macro ./scripts/verify-plan.tape

# Run tape against state
tfui -state ./state.json -macro ./scripts/check-state.tape

# Both plan and state
tfui -plan ./plan.json -state ./state.json -macro ./scripts/full-check.tape

# Without pre-loaded data (for project-level flows like scaffold, chdir)
tfui -project ./my-infra -macro ./scripts/test-scaffold.tape

# Tape from stdin
echo "wait ready; key p; assert view aws_instance" | tfui -plan ./plan.json -macro -

# CI pipeline
terraform show -json tfplan.out | tfui -plan - -macro ./tests/verify-plan.tape
```

When `-plan`/`-state` are provided, the app renders pre-loaded data. Without them, the app starts in live mode (for testing project-level flows like scaffold wizard, chdir selection). In both cases, mutations are never executed.

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
├── frame_0001.txt      # ANSI frame after each command
├── frame_0002.txt
└── ...
```

The generated `recording.tape` is directly replayable: `tfui -macro ./recording/recording.tape`.

Frames can be stitched into GIFs using `demo/stitch.sh` (converts to asciicast, renders via `agg` or `vhs`).

## Command Output (stdout)

Every terraform operation triggered during macro playback is recorded and printed to stdout after the macro completes. This enables piping macros into a shell:

```bash
# Generate and execute terraform commands via macro
tfui -plan ./plan.json -state ./state.json -macro ./apply.tape | sh

# Use tofu instead of terraform
tfui -plan ./plan.json -state ./state.json -macro ./apply.tape -terraform-bin tofu | sh

# Preview what would run
tfui -plan ./plan.json -state ./state.json -macro ./taint.tape
# stdout:
# terraform workspace show
# terraform state list
# terraform taint aws_instance.web
```

## Safety

Macro mode **never executes real terraform commands**. All mutating operations (apply, state rm, taint, import, etc.) are recorded and printed to stdout — they are never delegated to the terraform binary. The user chooses whether to execute them by piping to `sh`.

Read operations (plan, state list, workspace show) delegate to the inner service to provide data for the UI to render. When `-plan`/`-state` are provided, reads use pre-loaded file data. Without them, reads may call the terraform binary (e.g., `terraform workspace show`) but never perform mutations.

## Tape Format

One command per line. Empty lines and lines starting with `#` are ignored.

```tape
# This is a comment

key p
wait ready
assert view create
screenshot /tmp/plan-view.txt
```

Inline mode uses semicolons as separators (single line):

```bash
tfui -macro "key p; wait ready; assert view create"
```

## Commands

### `key <key>`

Send a key event to the TUI.

```tape
key p          # single character
key enter      # special key
key esc        # escape
key space      # space bar
key ctrl+c     # ctrl combination
key /          # punctuation
key :          # command mode
```

**Supported special keys:** `enter`, `esc`, `tab`, `backspace`, `up`, `down`, `left`, `right`, `space`, `ctrl+c`, `ctrl+w`, `ctrl+t`, `ctrl+s`

**Single characters:** any single printable character (`a`–`z`, `0`–`9`, punctuation)

### `wait ready`

Block until the active plugin reports `Ready() == true`. Times out after a configurable duration.

```tape
key p
wait ready     # wait for plan plugin to finish loading
```

### `wait view <substring>`

Block until the rendered view contains the specified substring.

```tape
key p
wait view to add    # wait until "to add" appears in the view
```

The substring is everything after `view ` (spaces preserved):

```tape
wait view 3 resources to create    # matches "3 resources to create"
```

### `assert view <substring>`

Immediately check that the rendered view contains the substring. Fails the macro if not found.

```tape
assert view create          # fail if "create" not in view
assert view aws_instance    # fail if "aws_instance" not in view
```

### `screenshot <path>`

Write the current rendered view (ANSI-stripped) to a file.

```tape
screenshot /tmp/plan-output.txt
screenshot ./golden/state-view.txt
```

### `resize <width> <height>`

Change the terminal dimensions for rendering.

```tape
resize 120 40    # wide terminal
resize 80 24     # default size
```

### `sleep <duration>`

Pause execution. Use sparingly — primarily for demos and recordings.

```tape
sleep 500ms
sleep 2s
```

Accepts Go duration format: `100ms`, `1s`, `1m30s`.

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | All assertions passed |
| 1 | Assertion failure |
| 2 | Syntax error in tape |
| 3 | Timeout waiting for condition |

## Examples

### Apply all planned changes

```tape
wait ready
key p
wait ready
key a
wait view Apply plan
key y
wait view Are you sure
key y
```

```bash
# Outputs: terraform apply
tfui -plan ./plan.json -state ./state.json -macro ./apply.tape | sh
```

### Apply targeted (pinned resource)

```tape
wait ready
key p
wait ready
key space
key a
wait view Apply plan
key y
wait view Are you sure
key y
```

```bash
# Outputs: terraform apply -target=aws_instance.web
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
    app := buildTestApp(mockService)
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
- No interactive input simulation (text fields, prompts)
- `wait ready` checks that the view is non-empty and not in a loading state
- Screenshots include raw ANSI escape codes (lipgloss styling); use `cat` to view with colors, or pipe through `sed` to strip
- Default terminal size is 80x24; use `resize` to change
