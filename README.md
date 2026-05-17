# terraform-ui

A k9s-style interactive terminal UI for Terraform. Plan, analyze risk, inspect blast radius, and apply — all from a keyboard-driven TUI.

**Type:** Go CLI + interactive TUI (BubbleTea)
**Invocation:** `tfui` (TUI), `tfui plan`, `tfui apply` (non-interactive)
**Input:** Terraform root module or monorepo with `tfui.hcl`
**Dependencies:** Go 1.25+, terraform (or tofu/terragrunt)

## What It Looks Like

```
┌─────────────────────────────────────────────────────────┐
│  terraform-ui                        workspace: default │
├─────────────────────────────────────────────────────────┤
│                                                         │
│   [p] Plan          [R] Risk Analysis                   │
│   [a] Apply         [P] Phantom Changes                 │
│   [s] State         [B] Blast Radius                    │
│   [w] Workspaces    [o] Outputs                         │
│   [v] Validate      [t] Console                         │
│   [C] Context       [I] Scaffold                        │
│                                                         │
├─────────────────────────────────────────────────────────┤
│  q quit  / filter  : command                            │
└─────────────────────────────────────────────────────────┘
```

## Why

Running `terraform plan` on large modules produces verbose output with no structure. Reviewing state, managing workspaces, and assessing risk requires switching between multiple commands. terraform-ui wraps all of this into a single interactive terminal — navigate changes as a tree, see risk badges inline, pin resources for targeted operations, and apply without leaving the TUI.

## Quick Start

```bash
# Install
brew install lmarqs/tap/tfui

# Or via go install
go install github.com/lmarqs/terraform-ui/cmd/tfui@latest

# Launch interactive TUI
cd my-terraform-project
tfui

# Or use non-interactive mode
tfui plan --project ./infra
tfui apply --project ./infra
```

## Install

### Homebrew (recommended)

```bash
brew install lmarqs/tap/tfui
```

### Go install

```bash
go install github.com/lmarqs/terraform-ui/cmd/tfui@latest
```

### Binary download

Download the latest release from [GitHub Releases](https://github.com/lmarqs/terraform-ui/releases). Extract the binary and place it on your `PATH`.

```bash
# Example: Linux amd64
curl -sL https://github.com/lmarqs/terraform-ui/releases/latest/download/tfui_linux_amd64.tar.gz | tar xz
sudo mv tfui /usr/local/bin/
```

### mise

Add to your project's `mise.toml`:

```toml
[tools]
"github:lmarqs/terraform-ui" = { version = "latest", exe = "tfui", extract_all = "true", bin_path = "bin" }
```

Then run:

```bash
mise install
```

To pin a specific version, replace `"latest"` with a version number (e.g. `"1.4.0"`). The latest version is in the [`VERSION`](VERSION) file and on the [releases page](https://github.com/lmarqs/terraform-ui/releases).

## Features

- **Interactive Plan Review** — Navigate changes as a tree, expand attribute diffs, see risk badges
- **Risk Analysis** — Automatic classification of changes as critical/high/medium/low
- **Blast Radius** — Visualize affected modules and resource dependencies
- **Phantom Change Detection** — Identify no-op changes that terraform incorrectly reports
- **Live Apply** — Per-resource progress tracking with real-time status
- **State Browser** — Navigate, inspect, taint, untaint, move, and remove state resources
- **Workspace Management** — List, switch, create, and delete workspaces
- **Monorepo Support** — Discover and select chdir members via `tfui.hcl`
- **Pin & Target** — Pin resources with `Space`, then plan/apply only those targets
- **Non-Interactive Mode** — `tfui plan` and `tfui apply` for CI pipelines and scripts

## Usage

### Interactive TUI (default)

```bash
tfui                           # TUI in current directory
tfui --project ./infra         # TUI scoped to project directory
tfui --plan ./tfplan.out       # TUI with pre-computed plan
tfui --state ./terraform.tfstate  # TUI with pre-loaded state
```

### Non-Interactive CLI

```bash
# Plan with tree view on stdout
tfui plan
tfui plan -target=aws_instance.web
tfui plan -json                     # NDJSON (terraform-compatible)
tfui plan --ci                      # suppress spinner for CI

# Apply
tfui apply tfplan.out
tfui apply --ci

# State operations
tfui state rm aws_instance.old
tfui state mv aws_instance.old aws_instance.new
tfui state import aws_instance.web i-1234567890

# Workspaces
tfui workspace list
tfui workspace select production
tfui workspace new staging
```

Exit codes: `0` success, `1` error, `2` changes present (plan).

### Navigation

| Key | Action |
|-----|--------|
| `p` | Plan |
| `a` | Apply |
| `s` | State browser |
| `w` | Workspaces |
| `o` | Outputs |
| `v` | Validate |
| `R` | Risk analysis |
| `P` | Phantom changes |
| `B` | Blast radius |
| `C` | Context (chdir/workspace) |
| `/` | Filter |
| `:` | Command mode |
| `q` | Quit / back |
| `Space` | Pin (target for plan/apply) |
| `Enter` | Inspect / expand |

## Configuration

Create an optional `tfui.hcl` in your project root for monorepo support:

```hcl
terraform {
  bin = "terraform"
}

member "modules/vpc" {}
member "modules/ecs" {}
member "environments/prod" {}

defaults {
  parallelism = 10
  var_file "common/tags.tfvars" {}
}
```

Two modes:
- **Standalone** (no config): TUI skin over terraform, CWD = working dir
- **Project** (`tfui.hcl` present): full config resolution, member directories, workspace overrides

See [docs/guides/configuration.md](docs/guides/configuration.md) for all options.

## Requirements

- Go 1.25+ (for building from source)
- terraform, tofu, or terragrunt on `PATH`

## Development

```bash
mise install              # Install tools (go, golangci-lint, terraform, node)
mise run setup            # Install CI dependencies (npm + gotestsum)
mise run dev              # Run TUI in development mode
mise run check:lint       # Lint (golangci-lint)
mise run test:unit        # Unit tests
mise run test:coverage    # Coverage report
mise run build            # Cross-platform binaries (goreleaser snapshot)
```

See [docs/guides/getting-started.md](docs/guides/getting-started.md) for contributor setup and [docs/development/testing.md](docs/development/testing.md) for test strategy.

## Documentation

- [Getting Started](docs/guides/getting-started.md) — Installation and first run
- [Configuration](docs/guides/configuration.md) — `tfui.hcl` reference
- [CLI Reference](docs/reference/cli-reference.md) — All commands and flags
- [Architecture](docs/development/architecture.md) — Internal design
- [Plugins](docs/plugins/) — Plugin catalog
- [TUI UX Spec](docs/reference/tui-ux.md) — Navigation and interaction patterns
- [CLI I/O Contract](docs/reference/cli-io-contract.md) — stdout/stderr specification
- [Macro Language](docs/reference/macro-language.md) — Tape DSL for automated testing
- [Roadmap](docs/roadmap.md) — Planned features

## License

MIT
