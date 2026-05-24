---
name: cli-design
description: This skill should be used when designing CLI commands, flag conventions, output formats, I/O channel allocation, exit code semantics, pipe ergonomics, or non-interactive modes. Activates on mentions of CLI design, command interface, stdout/stderr contract, flag handling, JSON output, pipe-safe output, exit codes, spinner behavior, --ci mode, non-interactive fallback, or terraform CLI compatibility.
---

# CLI Design System

Design patterns for building CLI tools that compose well in Unix pipelines, respect terminal conventions, and provide excellent developer experience. Focused on tools that wrap or extend existing CLIs (like terraform).

**Core philosophy:** A CLI earns trust by being predictable. Data flows through stdout, progress through stderr, flags work like the tools users already know, and scripts never break when upgrading.

## CLI Design Process

```
1. What operation? → Map to existing conventions (terraform, git, kubectl)
2. What output?   → Choose channel + format per mode (human/json/ci)
3. What flags?    → Inherit from wrapped tool, add novel ones without collision
4. What errors?   → Actionable messages with hints, correct exit codes
5. Validate       → Test in pipes, CI, interactive terminal, non-TTY
```

---

## 1. The Superset Principle

When wrapping an existing CLI (terraform, kubectl, docker):

| Layer | Strategy | Example |
|-------|----------|---------|
| **Preserve** | Identical behavior for existing flags/output | `-json` produces same bytes |
| **Replace** | Better human-readable defaults (same channel) | Tree view instead of terraform's wall of text |
| **Add** | Novel commands/flags using unclaimed names | `--ci`, `risk`, `phantom` |

**Rules:**
- Same flag must never mean different things between original and wrapper
- Machine-readable formats (`-json`) are byte-for-byte compatible
- Human-readable output may improve, but same channel (stdout)
- Novel additions use names the wrapped tool hasn't claimed

---

## 2. Output Channel Contract

The fundamental rule: **stdout is for data, stderr is for humans.**

| Channel | Content | Consumer |
|---------|---------|----------|
| **stdout** | Data: JSON, tree views, resource lists, plan output | Pipes, scripts, jq, other tools |
| **stderr** | Progress: spinners, status, warnings, confirmations | Human eyes only |

### Mode matrix

| Mode | stdout | stderr |
|------|--------|--------|
| Default (TTY) | Human-readable data (tree view) | Spinner + progress (if stderr is TTY) |
| `-json` | Machine-readable JSON only | Nothing |
| `--ci` | Same as default | Nothing (no ANSI, no spinner) |
| Piped (no TTY) | Same as default | Nothing (auto-detected) |

### Critical invariants

- Piping stdout must never produce ANSI escapes or spinner artifacts
- `-json` mode produces zero bytes on stderr
- `--ci` explicitly suppresses stderr for CI runners where stderr might be a TTY
- Spinner only appears when: `!ci && !jsonOutput && isStderrTTY()`

---

## 3. Flag Design

### Compatibility flags (inherited from wrapped tool)

- Accept both single-dash (`-json`) and double-dash (`--json`) for terraform flags
- Normalize internally: `normalizeArgs()` converts to double-dash for the parser
- Unknown flags pass through unchanged (future-proof)

### Novel flags (your tool's additions)

- Always double-dash only: `--ci`, `--project`, `--macro`
- Use names the wrapped tool hasn't claimed
- Boolean flags never consume the next argument
- Value flags consume next arg only when no `=` separator

### Flag categories

| Category | Examples | Rule |
|----------|----------|------|
| **Value flags** | `--target X`, `--var-file X` | Consume next arg when no `=` |
| **Bool flags** | `--json`, `--destroy` | Never consume next arg |
| **Passthrough** | Everything after `--` | Stored as ExtraArgs, forwarded to wrapped tool |

### Passthrough ordering

```
user input → splitPassthrough("--") → normalizeArgs(before) → parse
                                    → store ExtraArgs(after)
```

`splitPassthrough` MUST run before `normalizeArgs`. The passthrough section is opaque — never normalize or validate it.

---

## 4. Exit Codes

Follow the conventions of the wrapped tool:

| Code | Meaning | When |
|------|---------|------|
| `0` | Success | Operation completed, or no changes detected |
| `1` | Error | Any failure (parse, runtime, permission) |
| `2` | Changes present | `plan` command detected drift (terraform convention) |

Rules:
- Exit 2 must ONLY appear in plan-related code paths
- Framework/parser errors → exit 1
- Signal interrupts → exit 130 (128 + SIGINT)

---

## 5. Spinner & Progress

### When to show

```
showSpinner = !ci && !jsonOutput && isStderrTTY()
```

All three conditions must hold. This ensures:
- CI gets clean logs
- JSON pipes stay pure
- Piped stderr stays artifact-free

### How to render

- Write to stderr only
- Use `\r\033[K` (carriage return + clear line) for updates
- Show elapsed time for long operations: `Planning... (12s)`
- Clear completely on finish (no trailing artifacts)
- Start after 200ms delay (avoid flash on fast operations)

---

## 6. Error Messages

- Write to stderr via the framework's error handling
- Start with lowercase (Go convention: `fmt.Errorf("plan failed: %w", err)`)
- Wrap with context: what failed + why
- Add hints for actionable recovery: `\n\nhint: check that terraform is installed`
- Never use "Press X to Y" patterns (that's TUI territory)
- Include the failing path/resource when available

---

## 7. Pipe Ergonomics

Every CLI command must work in these compositions:

```bash
# Filter pattern: stdin → transform → stdout
terraform show -json plan.out | tfui risk
cat state.json | tfui state list --state -

# Substitution pattern: drop-in replacement
tfui plan -json | jq '.type'           # identical to terraform
tfui state list | grep "aws"           # identical to terraform

# Composition pattern: novel analysis
tfui plan -json | tfui risk --json | jq '.score'
```

### Stdin conventions

- `-` means stdin (universal Unix convention)
- Detect stdin availability: `!isatty(stdin)`
- If no stdin and no file argument → error with suggestions
- Never block waiting for stdin without indicating what's expected

---

## 8. Non-Interactive Fallback

When stdin is not a TTY:
- Skip interactive modes (TUI, prompts, confirmations)
- Render output and exit immediately
- If interaction is required → error with `hint: use --auto-approve or provide input via flags`

When stdout is not a TTY:
- Suppress ANSI color/formatting in stdout
- Keep data format identical (just strip decoration)

---

## 9. JSON Output Design

### For wrapped commands (`-json`)

Byte-for-byte compatible with the original tool. No additions, no reformatting.

### For novel commands (`--json`)

Your schema. Design principles:
- Top-level object with `version` field for future evolution
- Consistent field naming (snake_case for terraform ecosystem)
- Include metadata: timestamp, source command, input file
- Errors as structured objects, not strings
- NDJSON for streaming operations (one JSON object per line)

---

## 10. Config Resolution

When your tool has its own config file:

```
CLI flags  >  env vars  >  project config  >  user config  >  defaults
```

Rules:
- Config file absence is non-fatal (standalone mode)
- Parse errors are fatal with actionable hint
- Config never contradicts CLI flags (flags always win)
- Log which config was loaded at debug level

---

## Anti-Patterns

| # | Anti-Pattern | Fix |
|---|-------------|-----|
| 1 | **Data on stderr** | Data always goes to stdout, even errors about data |
| 2 | **ANSI in piped output** | Detect TTY, strip when piped |
| 3 | **Novel flag collides with wrapped tool** | Check wrapped tool's flag namespace before naming |
| 4 | **Spinner on stdout** | Spinner goes to stderr, gated by TTY check |
| 5 | **Blocking stdin without signal** | Show "reading from stdin..." or error if no pipe |
| 6 | **Exit 0 on error** | Always exit non-zero on failure |
| 7 | **Different JSON schema in `-json` mode** | `-json` = wrapped tool's format exactly |
| 8 | **Interactive prompts in piped mode** | Detect non-TTY stdin, use flags instead |
| 9 | **Inconsistent flag naming** | Pick one style (double-dash for novel) and stick to it |
| 10 | **Swallowing wrapped tool's stderr** | Forward it unless you're explicitly replacing the output |

---

## Compatibility Checklist

Before shipping a CLI command:

- [ ] `cmd | jq '.'` works (stdout is clean JSON in -json mode)
- [ ] `cmd > /dev/null` produces no ANSI artifacts
- [ ] `cmd 2>/dev/null` suppresses all progress (only data on stdout)
- [ ] Works in `set -e` scripts (correct exit codes)
- [ ] `--ci` produces zero bytes on stderr
- [ ] Non-TTY stdin handled gracefully (error or read, no hang)
- [ ] `-json` output is byte-compatible with wrapped tool
- [ ] Novel `--json` output is stable and versioned
- [ ] Error messages include enough context to debug without re-running
- [ ] `--help` output goes to stdout (convention for piping to pager)

---

## Benchmarks & References

Tools with excellent CLI UX to study:

| Tool | Notable for |
|------|-------------|
| **ripgrep** | Perfect output channel separation, smart TTY detection |
| **jq** | Pure filter pattern, composable, predictable |
| **gh (GitHub CLI)** | Wraps API elegantly, `--json` with field selection |
| **kubectl** | Multiple output formats (`-o json/yaml/wide/name`) |
| **terraform** | Flag conventions, exit codes, `-json` streaming |
| **exa/eza** | Graceful color degradation, `--no-color` respect |
| **fd** | Smart defaults that differ from `find`, great pipe behavior |

## What This Skill is NOT

- Not a replacement for testing in real shells and CI environments.
- Not framework-specific (applies to cobra, urfave/cli, clap, click).
- Not about TUI/interactive mode — that's the tui-design skill.
- Not an excuse to reinvent existing conventions.
