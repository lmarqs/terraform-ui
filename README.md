# terraform-ui

Animated terminal feedback for `terraform plan` and `terraform apply` — spinner, progress bar, elapsed timer, and tree-view diff — in pure bash.

**Type:** CLI + embeddable bash library
**Invocation:** `tfui plan`, `tfui apply`, or `source lib/tfui.sh`
**Input:** Terraform root module directory
**Output:** Tree-view diff (stdout) or structured JSON (`--mode agent`), animations (fd3)
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
| `plain` | silent | No UI output, tree-view on stdout |
| `agent` | agent | No UI output, structured JSON on stdout |

### Agent Mode

Use `--mode agent` when terraform-ui is consumed by AI agents, MCP tools, or automation that needs structured data instead of human-readable text.

```bash
tfui plan --dir ./modules/vpc --mode agent
```

Output:

```json
{
  "has_changes": true,
  "summary": { "add": 1, "change": 1, "destroy": 1, "replace": 0 },
  "changes": [
    { "action": "create", "address": "aws_instance.web", "risk": "low" },
    { "action": "update", "address": "aws_security_group.allow_tls", "risk": "medium" },
    { "action": "delete", "address": "aws_iam_role.old_role", "risk": "high" }
  ],
  "risk_level": "high",
  "destructive": true
}
```

Each change includes a `risk` classification based on resource type and action:

| Risk | Trigger |
|------|---------|
| `critical` | Delete/replace databases, storage, KMS keys |
| `high` | Delete/replace IAM, networking, compute clusters; update critical resources |
| `medium` | Update high-risk resources; modify security groups, DNS |
| `low` | Create operations on non-critical resources |

The plan-level `risk_level` is the maximum across all changes. `destructive` is `true` if any change is a delete or replace.

### Noise Reduction

terraform-ui includes two noise reduction features for cleaner plan output:

**Phantom change detection** — identifies "updates" where nothing actually changed (tag ordering, computed defaults, provider normalization):

```bash
source lib/tfui.sh
_tfui_filter_phantom_changes "$plan_json_file"
```

```json
{
  "phantom_changes": 13,
  "real_changes": 2,
  "phantom_resources": ["aws_security_group.default", "aws_route_table.main"]
}
```

**Module-level grouping** — groups flat resource lists by module path for better signal:

```bash
_tfui_group_by_module "$plan_json_file"
```

```json
{
  "by_module": {
    "module.vpc": { "summary": { "add": 0, "change": 3, "destroy": 0 }, "changes": [...] },
    "root": { "summary": { "add": 1, "change": 0, "destroy": 1 }, "changes": [...] }
  }
}
```

Human-readable grouped output:

```bash
_tfui_render_grouped_plan_tree "$plan_json_file"
```

```
module.vpc (3 to change)
  ~ aws_route_table.private
  ~ aws_subnet.private[0]
  ~ aws_subnet.private[1]
root (1 to add, 1 to destroy)
  + aws_instance.web
  - aws_iam_role.old_role
```

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
mise install              # Install tools (jq, bats, terraform, node)
mise run setup            # Install npm + BATS dependencies
mise run test:run         # Syntax check + run tests
mise run build [version]  # Package dist/ (reads VERSION by default)
mise run coverage:run     # Coverage via Docker + kcov
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for commit conventions and test guidelines.

## License

MIT
