---
name: tfui-cli-ux
description: "tfui CLI UX rules: Standalone vs Headless, headless failure contract, pre-seeded -plan/-state cache, input and output ports, flag conventions and scoping, binary resolution, -- passthrough, exit codes, tfui.hcl config resolution. Use before editing cmd/ or a plugin's Input, Stdout, Stderr or ExitCode."
---

Full spec: `docs/reference/cli-ux.md`. I/O contract: `docs/reference/cli-io-contract.md`. Every command and flag:
`docs/reference/cli-reference.md`.

# CLI UX Rules

## Execution Model

Every `tfui <command>` launches its plugin as a Standalone TUI on stderr; output goes to stdout on exit. `-ci`, `CI=1`, an agent environment or a non-TTY stderr runs it Headless instead.

Two modes:
- **Standalone**: alt-screen on stderr, plugin output to stdout on exit (fzf model)
- **Headless**: no TUI, the macro driver runs the plugin, output to stdout immediately

Mode resolution (`resolveSilentStderrFrom`, see ADR-0022):
```go
if -ci OR CI=1:                → Headless
if CLAUDECODE or AI_AGENT set:  → Headless (agent environment)
if stderr not TTY:              → Headless
otherwise:                      → Standalone
```

Headless operational failures keep stdout empty and exit 1. Terraform errors pass through on stderr; Risk policy
failures use fixed diagnostics. Risk threshold matches emit a complete report and exit 3; see the I/O contract
for channels and terminal outcomes. Action verbs (taint/untaint/import) skip their confirm prelude headlessly via `Input.AutoConfirm` set at the cmd boundary.

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
- On subcommands: `-macro` drives the command's plugin headlessly, outputs recorded commands
- `-record <dir>`: captures ANSI frames + manifest.json (orthogonal to `-macro`)
- `-dry-run`: root only; the interactive full TUI on stderr against `MacroService`, recorded commands to stdout on exit
- Root flags (`-project`, `-terraform-bin`, `-config`, `-chdir`, `-ci`, `-json`): persistent (available everywhere)
- Command-specific flags (`-target`, `-risk-threshold`, `-lock`, `-force`, `-backend`, …): local to their command
- Every novel flag must combine validly with every other flag — no conditional validity (see ADR-0017)

Binary resolution:
- `-terraform-bin` > `-config terraform.bin=X` > `tfui.hcl terraform { bin = "..." }` > `"terraform"`

`--` passthrough:
- `splitPassthrough()` separates args at `--`
- ExtraArgs stored for `MacroService` (recorded in command flags)
- `ExecService` does NOT forward ExtraArgs (terraform-exec typed API)

Exit codes: `0` = success, `1` = error, `2` = changes present (plan only), `130` = interrupted by SIGINT,
`143` = terminated by SIGTERM (all commands). Risk-only exit `3` and its complete-report semantics are defined
in `docs/reference/cli-io-contract.md#risk-command-v1`.

Interrupts: tfui owns SIGINT/SIGTERM (`cmd/tfui/interrupt.go`). The first signal interrupts terraform once, waits for
it with no timeout, runs plugin `Shutdown`, writes no stdout and exits 130/143; a second signal exits at once with
`terraform was stopped without finishing` on stderr, still running `Shutdown` (ADR-0023).

## Config (`tfui.hcl`)

HCL format. Everything optional. No config file = Zero-config.

```hcl
terraform { bin = "terraform" }
member "modules/vpc" {}
member "modules/ecs" {}
defaults {
  parallelism = 10
  lock = true
  var_file "common/tags.tfvars" {}
  plugin "risk" { threshold = "high" }
}
```

Two modes:
- Zero-config (no tfui.hcl): CWD = terraform dir, `-chdir` = raw passthrough
- Project (tfui.hcl found): full resolution, chdir validated against members

Resolution chain: Root defaults → Child top-level → Workspace block → CLI flags → `--` passthrough

Key functions: `config.LoadRoot(dir)`, `config.LoadChild(dir)`, `config.Resolve(root, child, workspace)`

Risk settings: `docs/reference/risk-policy-contract.md#command-threshold-and-report-v1` owns threshold values,
precedence, and the rejection of the `level` key with an error that names `threshold`. The complete-policy selector is
separate from plugin settings; consult `docs/reference/risk-policy-contract.md#source-selection` before changing
resolution.
Member plugin blocks remain rejected.
