---
layout: default
title: CLI UX Guidelines
parent: Reference
nav_order: 4
seo:
  description: UX design guidelines for the CLI interface
description: "Sets the rules every CLI command follows: execution modes, output channels, flags and errors."
audience: "Contributors adding or changing a CLI command"
---

# CLI UX Guidelines

## 1. Design Principle

Every `tfui <command>` launches the actual plugin in a standalone TUI. The TUI renders on stderr, output goes to stdout on exit. `-ci` or `CI=1` disables the TUI for headless use.

This follows the fzf model: interactive UI on one fd, structured output on another.

## 2. Two Execution Modes

| Mode | TUI | stdout | Trigger |
|------|-----|--------|---------|
| **Standalone** | Alt-screen on stderr | Plugin output on exit | Default (stderr is TTY) |
| **CI** | None | Plugin output immediately | `-ci`, `CI=1`, agent environment, stderr not TTY |

Mode resolution:
```go
if -ci OR CI=1:                → CI mode
if CLAUDECODE or AI_AGENT set:  → CI mode (agent environment, ADR-0022)
if stderr not TTY:              → CI mode
otherwise:                      → Standalone TUI
```

## 3. Output Channel Rules

| Channel | Content | Never |
|---------|---------|-------|
| **stdout** | Data output: tree view, JSON, resource lists, summaries (written on TUI exit) | TUI rendering, ANSI sequences |
| **stderr** | TUI rendering (alt-screen in standalone mode) | Data output |

### Critical invariants

- `-json` flag: changes output FORMAT (JSON vs human-readable), not mode
- `-ci` flag: changes execution MODE (headless vs TUI), not format
- Both flags are orthogonal: `tfui plan -ci -json` = headless + JSON output
- Piped stdout: TUI still renders on stderr (fzf model)

## 4. Flag Conventions

### Terraform compatibility
- All terraform flags work with a single dash (`-json`, `-target`) or a double dash (`--json`, `--target`)
- `normalizeArgs()` converts single-dash flags to double-dash for cobra
- Unknown multi-letter flags are double-dashed too, so a future terraform flag reaches cobra as `--name`; single-letter flags pass through unchanged

### Novel flags (tfui-only)
- Registered in cobra with a double dash (`--ci`); `normalizeArgs()` also accepts the single-dash spelling used throughout these docs (`-ci`, `-project`, `-macro`, `-record`, `-plan`, `-state`, `-terraform-bin`, `-config`, `-chdir`)
- Use names terraform hasn't claimed — no collision risk
- `-plan` and `-state` are available on ALL commands (pre-seed data; [CLI Reference](cli-reference.md) says which reads use it)
- `-macro` is available on ALL commands (on root: drives full TUI headlessly; on subcommands: drives standalone plugin headlessly, outputs recorded commands)
- Every novel flag must combine validly with every other flag (see ADR-0017)

### Passthrough (`--`)
- Everything after `--` is stored as ExtraArgs
- `splitPassthrough()` runs BEFORE `normalizeArgs()`
- ExtraArgs are appended to the `terraform plan` and `terraform apply` commands via sdk options
- ExecService does NOT forward ExtraArgs directly (uses typed API)

### Flag normalization sets

**Value flags** (consume next arg when no `=`):
`target`, `var`, `var-file`, `replace`, `out`, `parallelism`, `lock-timeout`, `chdir`, `workspace`, `input`, `backend-config`, `plugin-dir`, `get`

**Bool flags** (never consume next arg):
`json`, `lock`, `destroy`, `refresh-only`, `compact-warnings`, `upgrade`, `reconfigure`, `force-copy`, `backend`

## 5. Exit Codes

| Code | Meaning | Commands |
|------|---------|----------|
| `0` | Success (no changes, or operation completed) | All |
| `1` | Error / validation failure | All |
| `2` | Changes present | `plan` only |

Rules:
- Exit 2 must ONLY come from plan plugin's `ExitCode()` method
- cobra errors → exit 1 (handled by `SilenceUsage: true` + `os.Exit(1)`)
- Macro errors → exit code from `macro.RunError.Code`

## 6. Binary Resolution Priority

```
-config terraform.bin=X  >  -terraform-bin flag  >  tfui.hcl terraform { bin }  >  "terraform"
```

## 7. Pipe Ergonomics

### Standalone TUI + pipe (fzf model)

```bash
# User reviews plan in TUI, quits, then output flows to grep:
tfui plan | grep "aws_"

# User reviews plan in TUI, quits, then JSON flows to jq:
tfui plan -json | jq '.resource_changes[].address'

# User browses state in TUI, quits, then addresses flow to wc:
tfui state | wc -l
```

### CI mode in scripts

```bash
# No TUI, immediate output:
CI=1 tfui plan -json | jq '.resource_changes[].address'
tfui validate -ci -json | jq '.valid'

# Auto-detected (stderr not TTY):
tfui plan -json > plan-output.json
```

## 8. Tree View Format

Default human-readable output for `tfui plan`:

```
+ aws_s3_bucket.logs
~ aws_iam_role.api
- aws_security_group.old
-/+ aws_instance.web

Plan: 1 to add, 2 to change, 1 to destroy.
Risk: high
```

Symbols: `+` create, `~` update, `-` delete, `-/+` replace, `<=` read. A replace counts as a change. `Risk:` appears only when a change carries risk.

## 9. Error Message Formatting

- Errors go to stderr via cobra's error handling or `fmt.Errorf`
- Error messages start with lowercase (Go convention)
- No "Press X to Y" patterns in CLI output
- Wrap errors with context: `fmt.Errorf("plan failed: %w", err)`
- Hints use `\n\nhint:` suffix for actionable guidance

## 10. `-ci` Mode

Purpose: disable TUI entirely for headless/scripted use.

Detection:
1. `-ci` flag on the command
2. `CI=1` environment variable (`CI=true` does not match)
3. `CLAUDECODE` or `AI_AGENT` set to a non-empty value (agent environment, ADR-0022)
4. stderr is not a TTY (auto-detected)

Effects:
- No TUI rendering (no alt-screen, no keyboard input)
- Plugin runs headlessly via macro driver
- Output written directly to stdout when plugin reaches Ready state
- Exit codes unchanged

## 11. Analysis plugins

Risk, phantom and blast radius are TUI plugins (`R`, `P`, `B`); they have no CLI command.

## 12. Imperative Commands (no TUI)

Workspace operations and force-unlock are direct CLI commands — they execute immediately without TUI because they are one-shot imperative actions:

```bash
tfui workspace select prod     # switches, prints status to stderr
tfui workspace new staging     # creates, prints status to stderr
tfui force-unlock abc123       # unlocks, prints status to stderr
```
