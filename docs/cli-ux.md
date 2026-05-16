---
layout: default
title: CLI UX Guidelines
nav_order: 14
description: UX design guidelines for the CLI interface
nav_exclude: true
---

# CLI UX Guidelines — terraform-ui

## 1. Design Principle

tfui is a superset of terraform. All terraform flags work identically. Our additions use names terraform hasn't claimed. A user who types `tfui` instead of `terraform` must never be surprised by broken behavior.

## 2. Output Channel Rules

| Channel | Content | Never |
|---------|---------|-------|
| **stdout** | Data output: tree view, JSON, resource lists, plan summaries | Spinner, progress, status messages |
| **stderr** | Spinner, progress messages, warnings, operational confirmations | Data output, JSON, tree views |

### Critical invariants

- `-json` mode: stdout = JSON only, stderr = nothing
- `--ci` mode: stdout = tree/summary, stderr = nothing
- Default mode: stdout = tree/summary, stderr = spinner (if TTY)
- Pipe-safe: stdout must always be machine-parseable or human-readable data

## 3. Spinner Behavior

```go
showSpinner := !ci && !jsonOutput && isStderrTTY()
```

All three conditions must hold for spinner to appear:
- NOT in `--ci` mode
- NOT in `-json` mode
- stderr IS a TTY

Spinner output rules:
- Uses ANSI control sequences: `\r\033[K` (carriage return + clear line)
- Shows elapsed time for long operations (`(Xs)` suffix)
- Cleared on completion (line fully erased)
- Never leaves artifacts in piped output

## 4. Flag Conventions

### Terraform compatibility
- All terraform flags work with single dash (`-json`, `-target`) or double dash (`--json`, `--target`)
- `normalizeArgs()` converts single-dash terraform flags to double-dash for cobra
- Unknown flags are left unchanged (future terraform flags don't break)

### Novel flags (tfui-only)
- Always double-dash: `--ci`, `--project`, `--macro`, `--plan`, `--state`, `--terraform-bin`, `--config`, `--chdir`
- Use names terraform hasn't claimed — no collision risk

### Passthrough (`--`)
- Everything after `--` is stored as ExtraArgs
- `splitPassthrough()` runs BEFORE `normalizeArgs()`
- ExtraArgs passed to Plan/Apply via sdk options
- ExecService does NOT forward ExtraArgs directly (uses typed API)

### Flag normalization sets

**Value flags** (consume next arg when no `=`):
`target`, `var`, `var-file`, `replace`, `out`, `parallelism`, `lock`, `lock-timeout`, `chdir`, `workspace`, `input`, `backend`, `backend-config`, `plugin-dir`, `get`

**Bool flags** (never consume next arg):
`json`, `destroy`, `refresh-only`, `compact-warnings`, `upgrade`, `reconfigure`, `force-copy`

## 5. Exit Codes

| Code | Meaning | Commands |
|------|---------|----------|
| `0` | Success (no changes, or operation completed) | All |
| `1` | Error | All |
| `2` | Changes present | `plan` only |

Rules:
- `os.Exit(2)` must ONLY appear in plan-related code paths
- cobra errors → exit 1 (handled by `SilenceUsage: true` + `os.Exit(1)`)
- Macro errors → exit code from `macro.RunError.Code`

## 6. Binary Resolution Priority

```
--terraform-bin flag  >  --config terraform.bin=X  >  tfui.hcl terraform { bin }  >  "terraform"
```

Checked in `PersistentPreRunE`:
1. CLI flag already set → skip HCL
2. HCL has `terraform.bin` AND flag is empty → use HCL value
3. Neither → default to `"terraform"`

## 7. Config Loading Rules

- `LoadRoot()` called in `PersistentPreRunE` (runs for ALL commands)
- `ConfigNotFoundError` is non-fatal (standalone mode)
- HCL parse errors return with `hint: check HCL syntax in tfui.hcl`
- Child config cannot declare: `terraform`, `member`, `cache`, `ai`, `defaults` (locked blocks)
- Resolution chain: Root defaults → Child top-level → Workspace block → CLI flags → `--` passthrough

## 8. Pipe Ergonomics

These must always work:

```bash
# Substitution (terraform → tfui)
tfui plan -json | jq '.type'              # identical NDJSON
tfui show -json file | infracost          # identical JSON
tfui state list | grep "aws"              # identical output
tfui validate -json | jq '.diagnostics'   # identical JSON
tfui output -json | jq '.ep.value'        # identical JSON

# Pipeline (novel commands)
tfui show -json file | tfui risk          # plan JSON → risk report
tfui show -json file | tfui risk --json | jq  # full JSON pipeline

# Non-interactive fallback
terraform show -json plan.out | tfui --plan -   # stdin to TUI (view-only)
tfui --plan ./file.json                         # auto-renders without TTY
```

## 9. Tree View Format

Default human-readable output for `tfui plan`:

```
+ aws_s3_bucket.logs
~ aws_iam_role.api
- aws_security_group.old
-/+ aws_instance.web

Plan: 1 to add, 1 to change, 1 to destroy.
Risk: high
```

Symbols: `+` create, `~` update, `-` delete, `-/+` replace, `<=` read

## 10. Error Message Formatting

- Errors go to stderr via cobra's error handling or `fmt.Errorf`
- Error messages start with lowercase (Go convention)
- No "Press X to Y" patterns in CLI output
- Wrap errors with context: `fmt.Errorf("plan failed: %w", err)`
- Hints use `\n\nhint:` suffix for actionable guidance

## 11. `--ci` Mode

Purpose: explicit override for CI runners where stderr IS a TTY but user wants clean output.

Effects:
- Suppresses spinner (no ANSI on stderr)
- Suppresses any progress/status messages
- stdout remains identical (tree view or JSON)
- Does NOT affect exit codes

## 12. Non-Interactive Fallback

When stdin is not a TTY and `--plan`/`--state` is provided:
- Skip TUI (no alt-screen)
- Render tree view or resource list to stdout
- Exit immediately after rendering

When stdin is not a TTY and no `--plan`/`--state`:
- Print error with suggestions for non-interactive alternatives
- Exit 1

## 13. Novel Commands

Commands with no terraform equivalent (`risk`, `phantom`, `blast-radius`):
- Read plan JSON from stdin (terraform-compatible input)
- Default output: human-readable report on stdout
- `--json` output: our schema (not terraform's) on stdout
- No spinner (read from stdin, write to stdout — pure filter)
- Exit 0 on success, 1 on error
