# terraform-ui

Animated terminal UI for `terraform plan` and `terraform apply` operations.

Provides spinner, elapsed timer, progress bar, and tree-view diff output — all in pure bash.

## Features

- Spinner with elapsed time counter
- Progress bar tracking resource count during plan/apply
- Tree-view output showing planned changes (`+`, `~`, `-`, `-/+`)
- Three display modes: rich (progress bar), simple (spinner only), plain (silent)
- Works on macOS (bash 3.2+) and any Linux distro
- Single dependency: `jq`

## Install

### curl

```bash
curl -fsSL https://raw.githubusercontent.com/lmarqs/terraform-ui/main/scripts/install.sh | bash
```

### Basher

```bash
basher install lmarqs/terraform-ui
```

### Homebrew

```bash
brew tap lmarqs/terraform-ui
brew install terraform-ui
```

### Manual

Download the tarball from [GitHub Releases](https://github.com/lmarqs/terraform-ui/releases), extract it, and add `bin/` to your PATH.

## Usage

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

#### `tfui version`

Print version string (reads from VERSION file).

#### `tfui help`

Print usage information.

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

## Architecture

```
bin/tfui                 — CLI entry point
lib/tfui.sh              — library (source to embed)
VERSION                  — version (single source of truth)
tests/                   — BATS test suite
tests/fixtures/          — real terraform projects for integration tests
scripts/install.sh       — curl installer
```

### Layers

```
┌─────────────────────────────────────────────┐
│  CLI  (bin/tfui)                            │
│  Argument parsing, fd3 setup, dispatch      │
├─────────────────────────────────────────────┤
│  Public API  (tfui_*)                       │
│  init, plan, confirm, apply                 │
├─────────────────────────────────────────────┤
│  Orchestration  (_tfui_run*)                │
│  Strategy resolution, phase sequencing      │
├─────────────────────────────────────────────┤
│  Strategies  (_tfui_strategy_*)             │
│  silent, spinner, progress                  │
├──────────────────────┬──────────────────────┤
│  UI Engine           │  Execution           │
│  (_tfui_ui_*)        │  (_tfui_exec)        │
│  Animation, layout,  │  Working dir, output │
│  render, format      │  capture, exit codes │
├──────────────────────┴──────────────────────┤
│  Renderer  (_tfui_render_*)                 │
│  Plan tree view (jq)                        │
├─────────────────────────────────────────────┤
│  State & Lifecycle                          │
│  (_tfui_state_*, _tfui_lifecycle_*)         │
└─────────────────────────────────────────────┘
```

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

## License

MIT
