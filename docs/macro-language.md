---
layout: default
title: Macro Language
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

Macro mode requires `--plan` or `--state` (always read-only, no TTY needed):

```bash
# Run tape against a plan file
tfui --plan ./plan.json --macro ./scripts/verify-plan.tape

# Run tape against state
tfui --state ./state.json --macro ./scripts/check-state.tape

# Both plan and state
tfui --plan ./plan.json --state ./state.json --macro ./scripts/full-check.tape

# Tape from stdin
echo "wait ready; key p; assert view aws_instance" | tfui --plan ./plan.json --macro -

# CI pipeline
terraform show -json tfplan.out | tfui --plan - --macro ./tests/verify-plan.tape
```

No TTY is required — macro mode drives the BubbleTea model directly without opening a terminal.

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
tfui --macro "key p; wait ready; assert view create"
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

### Verify plan view renders correctly

```tape
# Navigate to plan plugin
key p
wait ready

# Check expected content
assert view create
assert view aws_instance

# Capture for golden file comparison
screenshot ./golden/plan-basic.txt
```

### Navigate and inspect a resource

```tape
key s
wait ready
key enter
wait view values
assert view instance_type
screenshot ./golden/state-detail.txt
key esc
```

### CI visual regression

```bash
#!/bin/bash
terraform show -json tfplan.out > plan.json
tfui --plan ./plan.json --macro ./tests/macros/verify-plan.tape
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
