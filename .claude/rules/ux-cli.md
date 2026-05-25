---
description: "CLI UX: standalone/CI modes, I/O contract, flag conventions, exit codes"
globs: ["cmd/**"]
---

Full spec: `docs/reference/cli-ux.md`

# CLI UX Rules

## Execution Model

Every `tfui <command>` launches the plugin in a standalone TUI (on stderr). Output goes to stdout on exit. `-ci` or `CI=1` disables the TUI.

Two modes:
- **Standalone TUI**: alt-screen on stderr, plugin output to stdout on exit (fzf model)
- **CI**: no TUI, headless execution via macro driver, output to stdout immediately

Mode resolution:
```go
if -ci OR CI=1:     → CI mode
if stderr not TTY:   → CI mode
otherwise:           → Standalone TUI
```

Behavior matrix:
- `tfui plan` → TUI on stderr, tree view to stdout on quit
- `tfui plan -ci` → no TUI, tree view to stdout immediately
- `tfui plan -json` → TUI on stderr, JSON to stdout on quit
- `tfui` (no args) → full TUI on stdout (unchanged, no output)

## Pre-Seeded Cache (`-plan`, `-state`)

```bash
tfui -plan ./plan.json
tfui -state ../terraform.tfstate
terraform show -json tfplan.out | tfui -plan -
tfui -plan ./plan.json -state ./state.json
```

When `-plan` or `-state` provided:
- `ServiceCache` is pre-seeded with parsed data; `ExecService` serves reads from cache
- Header shows `[pre-seeded]` badge
- Mutating hints hidden from status bar

## Key Interfaces

Plugins receive input and produce output via hexagonal ports (ADR-0021):

**Input port** (per-plugin typed Input + Activate):
- Each plugin exports `Input` struct + `Activate(input Input) tea.Cmd`
- `--json` flows as `Input.JSON bool` — plugin decides what to do with it
- cobra wiring lives in `cmd/tfui/<plugin>_command.go`

**Output port** (optional channel-specific emitter interfaces):
- `StdoutEmitter`: `Stdout() ([]byte, error)` — bytes for stdout
- `StderrEmitter`: `Stderr() []byte` — post-quit stderr (warnings, summaries)
- `ExitCoder`: `ExitCode() int` — process exit code

## Flag Conventions

- `-json` → changes output FORMAT (JSON vs human-readable)
- `-ci` → changes execution MODE (headless vs TUI)
- Both are orthogonal: `tfui plan -ci -json` = headless + JSON

Flag scoping:
- `-plan`, `-state`, `-macro`, `-record`: available on ALL commands (persistent flags)
- On root: `-macro` drives the full multi-plugin TUI headlessly
- On subcommands: `-macro` drives the standalone plugin headlessly, outputs recorded commands
- `-record <dir>`: captures ANSI frames + manifest.json (orthogonal to `-macro`)
- All other flags: persistent (available everywhere)
- Every novel flag must combine validly with every other flag — no conditional validity (see ADR-0017)

Binary resolution:
- `-terraform-bin` > `-config terraform.bin=X` > `tfui.hcl terraform { bin = "..." }` > `"terraform"`

`--` passthrough:
- `splitPassthrough()` separates args at `--`
- ExtraArgs stored for `MacroService` (recorded in command flags)
- `ExecService` does NOT forward ExtraArgs (terraform-exec typed API)

Exit codes: `0` = success, `1` = error, `2` = changes present (plan only)

## Config (`tfui.hcl`)

HCL format. Everything optional. No config file = standalone mode.

```hcl
terraform { bin = "terraform" }
member "modules/vpc" {}
member "modules/ecs" {}
cache { staleness_threshold = "5m" }
ai { enabled = true; provider = "bedrock"; region = "us-east-1" }
defaults {
  parallelism = 10
  lock = true
  var_file "common/tags.tfvars" {}
  plugin "risk" { level = "high" }
}
```

Two modes:
- Standalone (no tfui.hcl): CWD = terraform dir, `-chdir` = raw passthrough
- Project (tfui.hcl found): full resolution, chdir validated against members

Resolution chain: Root defaults → Child top-level → Workspace block → CLI flags → `--` passthrough

Key functions: `config.LoadRoot(dir)`, `config.LoadChild(dir)`, `config.Resolve(root, child, workspace)`
