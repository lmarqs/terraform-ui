---
layout: default
title: CLI I/O Contract
description: How tfui CLI commands handle stdin, stdout, stderr, and flags relative to terraform
---

# CLI I/O Contract

## Design Principle

**tfui is a superset of terraform. All terraform flags work identically. Our additions use names terraform hasn't claimed.**

A user who types `tfui` instead of `terraform` must never be surprised by broken behavior. Scripts, CI pipelines, and pipe workflows must work unmodified when substituting `terraform` for `tfui`. The only visible change is better human-readable output.

### Three Layers

```
Layer 1: Terraform-compatible (PRESERVE)
  -json flag → identical bytes to terraform
  state list/pull/push → identical behavior
  Exit codes → identical
  stdin (state push) → identical

Layer 2: Better defaults (REPLACE)
  plan/apply/validate default stdout → cleaner, same channel
  Spinner on stderr → same role as terraform's progress output

Layer 3: Novel (ADD)
  Commands terraform doesn't have (risk, phantom, blast-radius)
  Flags terraform doesn't have (--ci, --project, --macro)
  TUI mode (--plan, --state, --macro)
```

### Rules

1. **Preserve what exists.** If terraform has a flag and it produces output, tfui produces the same output in the same format on the same channel.
2. **Replace only the useless.** Terraform's default human-readable stdout (plan text, apply text) is bloated and unparseable by machines. Nobody consumes it programmatically — machine consumers use `-json`. We replace it with better human-readable output (tree view, summaries). Same channel (stdout), better content.
3. **Add freely what doesn't exist.** Novel commands and flags use names terraform hasn't claimed. No collision risk.
4. **Never contradict.** Same flag must never mean different things. If terraform's `-json` produces NDJSON events, ours does too — even if our enriched format would be "more useful."

## Terraform I/O Baseline

| Command | stdin | stdout | stderr | Exit |
|---------|-------|--------|--------|------|
| `terraform plan` | — | Human plan text | Warnings/progress | 0/1/2 |
| `terraform plan -json` | — | NDJSON events | — | 0/1/2 |
| `terraform plan -out=file` | — | Human plan text | Warnings | 0/1/2 |
| `terraform apply file` | — | Apply results | Progress | 0/1 |
| `terraform apply -json file` | — | NDJSON events | — | 0/1 |
| `terraform show -json file` | — | Structured JSON | — | 0/1 |
| `terraform show file` | — | Human-readable | — | 0/1 |
| `terraform state list` | — | Addresses (one/line) | — | 0/1 |
| `terraform state show addr` | — | HCL attributes | — | 0/1 |
| `terraform state rm addr` | — | Confirmation message | — | 0/1 |
| `terraform state mv src dst` | — | Confirmation message | — | 0/1 |
| `terraform state pull` | — | State JSON | — | 0/1 |
| `terraform state push` | State JSON | — | — | 0/1 |
| `terraform validate` | — | Human diagnostics | — | 0/1 |
| `terraform validate -json` | — | JSON diagnostics | — | 0/1 |
| `terraform output` | — | Human outputs | — | 0/1 |
| `terraform output -json` | — | JSON outputs | — | 0/1 |
| `terraform import addr id` | — | Success message | Progress | 0/1 |
| `terraform refresh` | — | Refresh results | Progress | 0/1 |

## tfui I/O Table

### Commands mirroring terraform

| Command | Flags | stdin | stdout | stderr | Exit | Delta |
|---------|-------|-------|--------|--------|------|-------|
| `tfui plan` | | — | Tree view | Spinner | 0/1/2 | Replaced: bloated text → tree |
| `tfui plan` | `-json` | — | NDJSON events | — | 0/1/2 | Identical |
| `tfui plan` | `-out=file` | — | Tree view | Spinner | 0/1/2 | Replaced: bloated text → tree |
| `tfui plan` | `-json -out=file` | — | NDJSON events | — | 0/1/2 | Identical |
| `tfui plan` | `--ci` | — | Tree view | — | 0/1/2 | Additive flag |
| `tfui plan` | `-json --ci` | — | NDJSON events | — | 0/1/2 | Additive flag |
| `tfui apply` | `file` | — | Apply summary | Spinner | 0/1 | Replaced: bloated text → summary |
| `tfui apply` | `-json file` | — | NDJSON events | — | 0/1 | Identical |
| `tfui apply` | `--ci` | — | Apply summary | — | 0/1 | Additive flag |
| `tfui show` | `-json file` | — | Structured JSON | — | 0/1 | Identical |
| `tfui show` | `file` | — | Human-readable | — | 0/1 | Replaced: better formatting |
| `tfui state list` | | — | Addresses (one/line) | — | 0/1 | Identical |
| `tfui state show` | `addr` | — | HCL attributes | — | 0/1 | Identical |
| `tfui state rm` | `addr` | — | Confirmation message | — | 0/1 | Identical |
| `tfui state mv` | `src dst` | — | Confirmation message | — | 0/1 | Identical |
| `tfui state pull` | | — | State JSON | — | 0/1 | Identical |
| `tfui state push` | | State JSON | — | — | 0/1 | Identical |
| `tfui import` | `addr id` | — | Success message | Spinner | 0/1 | Identical |
| `tfui validate` | | — | Enriched diagnostics | — | 0/1 | Replaced: same data, better format |
| `tfui validate` | `-json` | — | JSON diagnostics | — | 0/1 | Identical |
| `tfui output` | | — | Human outputs | — | 0/1 | Identical |
| `tfui output` | `-json` | — | JSON outputs | — | 0/1 | Identical |
| `tfui output` | `name` | — | Single value | — | 0/1 | Identical |
| `tfui refresh` | | — | Refresh results | Spinner | 0/1 | Identical |
| `tfui scaffold` | `--yes` | — | HCL content | — | 0/1 | Novel |
| `tfui workspace show` | | — | Workspace name | — | 0/1 | Identical |
| `tfui workspace list` | | — | Workspace names | — | 0/1 | Identical |
| `tfui workspace select` | `name` | — | — | — | 0/1 | Identical |
| `tfui workspace new` | `name` | `-lock`, `-lock-timeout` | — | — | 0/1 | Identical |
| `tfui workspace delete` | `name` | `-force`, `-lock`, `-lock-timeout` | — | — | 0/1 | Identical |

### Novel commands (no terraform equivalent)

| Command | Flags | stdin | stdout | stderr | Exit |
|---------|-------|-------|--------|--------|------|
| `tfui risk` | | Plan JSON | Risk report (human) | — | 0/1 |
| `tfui risk` | `--json` | Plan JSON | Risk JSON (our schema) | — | 0/1 |
| `tfui phantom` | | Plan JSON | Phantom report (human) | — | 0/1 |
| `tfui phantom` | `--json` | Plan JSON | Phantom JSON (our schema) | — | 0/1 |
| `tfui blast-radius` | | Plan JSON | Blast graph (human) | — | 0/1 |
| `tfui blast-radius` | `--json` | Plan JSON | Blast JSON (our schema) | — | 0/1 |

### TUI mode (interactive, own world)

| Command | Flags | stdin | stdout | stderr | Exit |
|---------|-------|-------|--------|--------|------|
| `tfui` | | — | (alt screen) | — | 0/1 |
| `tfui` | `--plan file` | — | (alt screen) | — | 0/1 |
| `tfui` | `--state file` | — | (alt screen) | — | 0/1 |
| `tfui` | `--plan -` | Plan JSON | (alt screen) | — | 0/1 |
| `tfui` | `--state -` | State JSON | (alt screen) | — | 0/1 |
| `tfui` | `--macro tape --plan file` | — | Commands | — | 0/1 |
| `tfui` | `--macro tape` | — | Commands | — | 0/1 |

### Additive flags (no collision)

| Flag | Effect | Scope |
|------|--------|-------|
| `--ci` | Suppress stderr spinner/progress | All execution commands |
| `--project dir` | Set project root | All commands |
| `--terraform-bin path` | Override binary | All commands |
| `--chdir member` | Select chdir member | All commands |
| `--config key=val` | Override config | All commands |

## Benchmarks

### Pipe scenario: user replaces `terraform` with `tfui` in scripts

```bash
# Before:                                    After:
terraform plan -out=tfplan.out | tee log     tfui plan -out=tfplan.out | tee log
# ✓ log gets tree (better). Binary file identical.

terraform plan -json | jq '.type'            tfui plan -json | jq '.type'
# ✓ identical NDJSON. jq works same.

terraform show -json tfplan.out | infracost  tfui show -json tfplan.out | infracost
# ✓ identical JSON. infracost works.

terraform state list | grep "aws"            tfui state list | grep "aws"
# ✓ identical output. grep works.

terraform state pull > backup.json           tfui state pull > backup.json
# ✓ identical state JSON.

terraform validate -json | jq '.diagnostics' tfui validate -json | jq '.diagnostics'
# ✓ identical JSON.

terraform output -json | jq '.ep.value'      tfui output -json | jq '.ep.value'
# ✓ identical JSON.
```

### Pipe scenario: tfui novel commands chained

```bash
# Plan → risk analysis
tfui show -json tfplan.out | tfui risk
# stdout: risk report. Consumes terraform-compatible JSON.

# Plan → risk → filter
tfui show -json tfplan.out | tfui risk --json | jq '.high_risk[]'
# Each stage: stdin=JSON, stdout=JSON

# Macro → inspect → execute
tfui --macro deploy.tape --plan ./tfplan.out        # prints commands
tfui --macro deploy.tape --plan ./tfplan.out | sh   # executes them
```

### Combined flags verification

```bash
# terraform:
terraform plan -json -out=plan.out
# stdout: NDJSON events | side effect: plan.out written | exit: 0/1/2

# tfui:
tfui plan -json -out=plan.out
# stdout: NDJSON events | side effect: plan.out written | exit: 0/1/2
# IDENTICAL in every way.
```

## Decisions and Tradeoffs

### Why replace terraform's default stdout?

Terraform's default human-readable output is:
- 50+ lines of attribute diffs for a single resource
- Impossible to scan quickly for "what's changing?"
- Not consumed by any machine tool (they all use `-json`)
- Mixed warnings and plan content

tfui replaces this with a tree view (action + address, summary line, risk). Same information density for humans, orders of magnitude less noise. Because no tool parses the human text, this replacement breaks nothing.

### Why not put the tree view on stderr?

Considered and rejected. Reasoning:

- `tfui plan > file.txt` would produce an empty file (surprising)
- `tfui plan | grep "aws"` would match nothing (surprising)
- `tfui plan | less` would show nothing (surprising)
- Every terraform-adjacent tool (tflint, infracost, checkov) puts human output on stdout
- The curl model (data=stdout, commentary=stderr) doesn't fit because our tree IS the output, not commentary about it

The tree view replaces terraform's bloated text on the same channel. It's the "better default" — not metadata about the operation.

### Why preserve `-json` exactly?

Considered: make `-json` output our enriched format (risk, phantom annotations). Rejected because:

- User types `tfui plan -json | jq` expecting the same fields as `terraform plan -json`
- CI tools parse terraform's NDJSON schema — adding fields is OK, changing structure is not
- If `-json` means different things in terraform and tfui, the user maintains two mental models
- Our enrichments get their own novel commands (`tfui risk --json`), no collision

### Why `--ci` and not reusing terraform's flags?

Terraform has no "suppress progress" flag. It shows progress when stderr is a TTY and suppresses when not. tfui follows the same isatty heuristic but adds `--ci` as an explicit override for cases where stderr IS a TTY but the user still wants clean output (certain CI runners, terminal multiplexers).

### Why `--json` on novel commands uses our schema?

`tfui risk --json` produces our risk schema because terraform has no `risk` command — there's no existing format to collide with. We own the namespace entirely. Same for `phantom`, `blast-radius`, and any future enrichment command.

### Why novel commands read terraform JSON from stdin?

`tfui risk` consumes the output of `tfui show -json` (which is terraform-compatible plan JSON). This means:

```bash
terraform show -json tfplan.out | tfui risk    # works
tfui show -json tfplan.out | tfui risk         # same thing
```

The input format is terraform's. The output format is ours. We extend the pipeline without breaking it.

### Why TUI mode is separate from CLI?

The TUI (`tfui --plan`, `--state`, `--macro`) is our interactive product — not a terraform operation. It uses alt-screen, handles keyboard input, manages state. This is a fundamentally different interface that terraform has no equivalent for, so we have full design freedom:

- `--plan file` accepts binary plan files (for review AND apply)
- `--plan -` accepts JSON from stdin (view-only, can't apply stdin)
- `--state -` accepts state JSON from stdin (view-only in TUI)
- `--macro` outputs commands to stdout (the user pipes to sh if desired)

These flags never conflict with terraform because terraform has no TUI mode.

## What We Don't Support

Terraform flags that affect output format are not supported because terraform-exec discards terraform's stdout. tfui reconstructs output from structured data:

- `-no-color` (we control our own colors)
- `-compact-warnings` (we control our own warning display)
- `-detailed-exitcode` on apply (we use standard 0/1)

These are documented as unsupported. Users who need them should use terraform directly.
