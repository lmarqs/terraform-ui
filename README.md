# terraform-ui

Animated terminal feedback for `terraform plan` and `terraform apply` — spinner, progress bar, elapsed timer, and tree-view diff — in pure bash.

**Type:** CLI + embeddable bash library
**Invocation:** `tfui plan`, `tfui apply`, or `source lib/tfui.sh`
**Input:** Terraform root module directory
**Output:** Tree-view diff (stdout), animations (fd3)
**Dependencies:** bash 3.2+, jq, terraform

## What It Looks Like

During a plan or apply, the terminal shows a live spinner with elapsed time and a progress bar:

```
⠋ Planning (12s)
  Progress: 8/20 [████████████░░░░░░░░] 40%
```

When the plan completes, a tree-view diff is printed to stdout:

```
+ aws_instance.web
~ aws_security_group.allow_tls
- aws_iam_role.old_role
-/+ aws_db_instance.main

Plan: 1 to add, 1 to change, 1 to destroy.
```

Symbols: `+` create, `~` update, `-` delete, `-/+` replace.

## Why

Running `terraform plan` on large modules produces verbose output with no progress indication during the wait. `terraform-ui` wraps these commands with a live animated spinner and progress bar, then outputs a concise tree diff showing only what changes. Use it to wrap terraform in scripts, in CI pipelines (with `--mode plain`), or interactively in the terminal.

## Quick Start

```bash
# Install
curl -fsSL https://raw.githubusercontent.com/lmarqs/terraform-ui/main/scripts/install.sh | bash

# Preview changes
tfui plan --dir ./modules/vpc

# Plan, confirm, and apply
tfui apply --dir ./modules/vpc
```

## Install

### mise (recommended)

Add to your project's `mise.toml`:

```toml
[tools]
"github:lmarqs/terraform-ui" = { version = "latest", exe = "tfui", extract_all = "true", bin_path = "bin" }
```

Then run:

```bash
mise install
```

To pin a specific version, replace `"latest"` with a version number (e.g. `"0.36.4"`). The latest version is in the [`VERSION`](VERSION) file and on the [releases page](https://github.com/lmarqs/terraform-ui/releases).

### curl

```bash
curl -fsSL https://raw.githubusercontent.com/lmarqs/terraform-ui/main/scripts/install.sh | bash
```

### Basher

```bash
basher install lmarqs/terraform-ui
```

### Manual

Download the tarball from [GitHub Releases](https://github.com/lmarqs/terraform-ui/releases), extract it, and add `bin/` to your PATH.

## Usage

Use the **CLI** as a drop-in wrapper for `terraform plan`/`apply`. Use the **library** when embedding terraform-ui into your own bash scripts or task runners.

### CLI

```bash
# Preview changes (like terraform plan)
tfui plan --dir ./modules/vpc

# Plan, confirm, and apply (like terraform apply)
tfui apply --dir ./modules/vpc

# Auto-approve (CI-friendly)
tfui apply --dir ./modules/vpc --auto-approve --mode plain

# Pass arguments to terraform
tfui plan --dir ./modules/vpc -- -target=aws_instance.web -var="env=prod"
```

### Library

```bash
source "/path/to/terraform-ui/lib/tfui.sh"

plan_file=$(mktemp)
tfui_init "$MODULE_DIR" "auto"
tfui_plan "Planning module: $MODULE" --out "$plan_file"
if tfui_confirm "$plan_file"; then
  tfui_apply "$plan_file" "Applying module: $MODULE"
fi
```

## Reference

### CLI Commands

#### `tfui plan [options] [-- terraform_args]`

Run terraform plan and display a tree view of changes.

| Option | Default | Description |
|--------|---------|-------------|
| `--dir <path>` | `.` | Terraform root module directory |
| `--mode <mode>` | `auto` | UI mode: auto, rich, simple, plain |
| `--message <msg>` | `Planning` | Spinner message |
| `--` | | Pass remaining args to terraform |

#### `tfui apply [options] [-- terraform_args]`

Plan, confirm, and apply changes (full lifecycle).

| Option | Default | Description |
|--------|---------|-------------|
| `--dir <path>` | `.` | Terraform root module directory |
| `--mode <mode>` | `auto` | UI mode: auto, rich, simple, plain |
| `--message <msg>` | `Planning` | Spinner message |
| `--auto-approve` | *(prompt)* | Skip confirmation |
| `--` | | Pass remaining args to terraform |

Additional commands: `tfui version`, `tfui help`.

### Library API

| Function | Description |
|----------|-------------|
| `tfui_init <dir> [mode]` | Initialize working directory and choose UI strategy |
| `tfui_plan <msg> [args] --out <file>` | Run terraform plan, render tree view |
| `tfui_confirm <file> [--auto-approve]` | Check for changes, optionally prompt user |
| `tfui_apply <file> <msg> [args]` | Apply the saved plan |

### UI Modes

| Mode | Strategy | Description |
|------|----------|-------------|
| `auto` | *(detected)* | Rich if terminal available, plain otherwise |
| `rich` | progress | Two-line UI: spinner + progress bar |
| `simple` | spinner | One-line spinner with elapsed time |
| `plain` | silent | No UI output, captures silently |

### File Descriptors

| FD | Purpose |
|----|---------|
| stdout (1) | Tree-view output (data) |
| stderr (2) | Error messages |
| fd3 | Terminal UI (animations, progress bar) |

### Environment Variables

| Variable | Description |
|----------|-------------|
| `TF_CLI_ARGS_plan` | Extra args passed to `terraform plan` (native terraform) |
| `TF_CLI_ARGS_apply` | Extra args passed to `terraform apply` (native terraform) |

## Requirements

- bash 3.2+ (macOS default works)
- jq
- terraform

## Development

```bash
mise install              # Install tools (jq, bats, terraform)
mise run setup            # Install BATS helper libraries
mise run test:run         # Syntax check + run tests
mise run build [version]  # Package dist/ (reads VERSION by default)
mise run coverage:run     # Coverage via Docker + kcov
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for commit conventions and test guidelines.

## License

MIT
