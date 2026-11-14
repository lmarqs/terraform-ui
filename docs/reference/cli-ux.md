---
layout: default
title: CLI UX Guidelines
parent: Reference
nav_order: 4
seo:
  description: UX design guidelines for the CLI interface
description: "Sets the rules every CLI command follows: output channels, flags, exit codes, errors and headless runs."
audience: "Contributors adding or changing a CLI command"
---

# CLI UX Guidelines

A command parses its flags into its plugin's `Input` and lets the plugin produce the output; the plugin runs the same
code standalone and headless. The modes, channels and exit codes a caller relies on are the
[CLI I/O contract](cli-io-contract.md); every command and flag is in the [CLI reference](cli-reference.md). This page
holds the rules a contributor follows when writing a command, each with its reason.

## 1. Output channels

- **Data goes to stdout, through the plugin's `StdoutEmitter`; status and summaries go to stderr, through its
  `StderrEmitter`; the TUI renders only on stderr.** A script can pipe stdout while the user reviews in the TUI, and
  stdout never carries ANSI sequences ([why TUI on stderr](cli-io-contract.md#why-tui-on-stderr)).
- **A plugin produces its output through the emitter interfaces (`StdoutEmitter`, `StderrEmitter`, `ExitCoder`) and
  never asks which mode it runs in.** The mode is chosen in `cmd/tfui/`; a plugin that branched on it would couple the
  core to its adapters ([ADR-0006](../adr/0006-orthogonal-income-outcome.md)). `-json` reaches the plugin as
  `Input.JSON` at `Activate` time, and the plugin decides what `Stdout()` returns.
- **A piped stdout keeps the TUI.** The user wants the review and the piped output at once
  ([why `-ci` instead of detecting a pipe](cli-io-contract.md#why--ci-instead-of-auto-detecting-stdout-pipe)).

## 2. Flags

- **`-json` changes the format and `-ci` the mode; neither depends on the other.** `tfui plan -ci -json` is headless
  with JSON output, and every pairing stays valid ([ADR-0017](../adr/0017-novel-flags-must-be-orthogonal.md)).
- **A terraform flag keeps terraform's name and works with one dash or two.** Terraform users type `-target`; cobra
  parses `--target`, so `normalizeArgs()` (`cmd/tfui/normalize.go`) double-dashes multi-letter flags before cobra
  sees them.
- **A terraform flag that takes a separate value goes in `knownValueFlags`; a boolean one in `knownBoolFlags`.** The
  normalizer must know whether the next argument belongs to the flag: `-target X` consumes `X`, `-lock false` must
  not. The sets are listed in the [CLI reference](cli-reference.md#flag-syntax).
- **A tfui-only flag takes a name terraform has not claimed and is registered with a double dash.** A name terraform
  owns would change meaning for terraform users ([ADR-0009](../adr/0009-terraform-namespace-reserved.md)).
- **A tfui-only flag combines validly with every other flag.** A flag whose meaning depends on another flag's absence
  hides a dependency between axes ([ADR-0017](../adr/0017-novel-flags-must-be-orthogonal.md)).

## 3. Exit codes

- **Exit 2 comes only from the plan plugin's `ExitCode()`.** Scripts read 2 as "plan has changes", as with terraform's
  `-detailed-exitcode`; a second source would make it ambiguous.
- **Every other failure exits 1.** A plugin's `ExitCode()` returns 1 on failure; a cobra error exits 1 through
  `SilenceUsage: true` and `os.Exit(1)` in `main.go`.

## 4. Error messages

- Errors go to stderr through cobra's error handling or `fmt.Errorf`.
- Error messages start lowercase, because a wrapped error continues another sentence (Go convention).
- Wrap errors with the step that failed, `fmt.Errorf("plan failed: %w", err)`, so the user sees where it broke.
- An actionable hint follows the error after `\n\nhint:`, so the error line stays terraform's or Go's own.
- No "Press X to Y" in CLI output: a headless run has no keys to press.

## 5. Headless runs

- **A confirmation is skipped headless only when the command's arguments declare the intent.** The action verbs
  (`taint`, `untaint`, `import`) set `Input.AutoConfirm` from `SilentStderr() || macro.Active()`, as terraform's own
  commands never prompt; nothing could answer the prompt, and the run would block until the driver timed out
  ([ADR-0022](../adr/0022-agent-environments-imply-headless.md)).
- **A command that would change infrastructure without a review fails headless unless the user opted out of the
  review**, as `terraform apply` does without `-auto-approve` ([`tfui apply`](cli-reference.md#tfui-apply)). Applying
  unconfirmed is the one failure a user cannot take back.
- **A failed run never claims success.** It reports failure the way the contract's
  [failure rows](cli-io-contract.md#headless-failures) say, because a script or an agent acts on the exit code
  alone.

## 6. Imperative commands

Workspace operations and force-unlock run at once, with no TUI, and print a status message to stderr. They are
one-shot operations: the operation is the entire interaction, so a TUI would add nothing.

```bash
tfui workspace select prod     # switches, prints status to stderr
tfui force-unlock abc123       # unlocks, prints status to stderr
```
