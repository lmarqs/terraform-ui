# Claude Code

What Claude Code reads in this layout, how the policy reaches it, and what it trusts. Tested with Claude Code 2.1.303
or 2.1.304 (2.1.304 for skill arguments, worktrees, bypass flags and `--setting-sources`), and 2.1.306 for the session
start, except where a line says it was not tested. [design.md](design.md) says why the layout is shaped around these
facts; [codex.md](codex.md) holds the same facts for Codex.

## What it reads

| Path | What Claude takes from it |
| --- | --- |
| `AGENTS.md` | The instructions, from 2.1.277 on, while no `CLAUDE.md`, `.claude/CLAUDE.md` or `CLAUDE.local.md` sits in its directory or one above; any of the three makes Claude read it instead (tested, `CLAUDE.local.md` included). A person can turn `AGENTS.md` support off under "Project instructions" in `/config`, as Claude Code's changelog says; not tested |
| `.claude/settings.json`, a link to `.agents/adapters/claude/settings.json` | The project settings: the policy as `permissions`, `hooks` and every other key the team writes. Claude reads project settings only there |
| `.claude/skills`, `.claude/commands`, links to `.agents/skills/` and `.agents/commands/` | Skills and slash commands. Claude finds them only there, and does not read `.agents/skills/` (tested) |
| `.claude/settings.local.json` | Each person's own settings, merged over the project's; git ignores it |
| `.claude/rules/` | Path-scoped rules, which Codex does not read; the layout keeps none, [adopt.md](adopt.md#claude-path-scoped-rules-in-clauderules) moves them |

A skill invoked as `/agents-layout review --fix` gets the words after its name below its file as
`ARGUMENTS: review --fix` (tested).

## Permissions

`#connect` renders the policy into `permissions` of `.agents/adapters/claude/settings.json`: each `commands` entry
becomes one `Bash(<words> *)` rule per combination of its alternatives, and each `{claude: "<pattern>"}` entry, under
`commands` or `tools`, is written as given ([policy.md](policy.md#prefix)). Claude merges every settings file, and
deny wins over ask, and ask over allow, whichever file each comes from.

A role renders to `.agents/adapters/claude/roles/<role>.json`, holding only `permissions.deny`. A run adds it with
`--settings .agents/adapters/claude/roles/<role>.json`; Claude merges it over the project settings, and a deny there
wins over a project allow (tested).

## Trust

Claude ignores the project's `permissions.allow` in a workspace nobody has trusted, and `claude -p` says so:
`Ignoring 1 permissions.allow entry ... workspace has not been trusted`. Start `claude` in the main checkout once,
interactively, and accept the trust question. It runs the project's hooks even in a workspace nobody trusted (tested
with `claude --print`).

## Hooks

`#connect` owns `hooks.SessionStart` in the settings. Its command starts from `$CLAUDE_PROJECT_DIR`, runs the dispatcher
`.agents/hooks/session-start` with the argument `claude`, and Claude adds the `additionalContext` it prints to the
session (tested with 2.1.306). With a `--worktree-*` param, `#connect` also owns `hooks.WorktreeCreate` and
`hooks.WorktreeRemove`, which cover the worktrees Claude Code itself creates; its help lists their limits. Every other
event under `hooks` is the team's.

## Worktrees

Claude keys trust on the clone's main checkout. In a worktree of a clone nobody trusted, its warning names
`projects["<main checkout>"]`; in a worktree of a trusted clone, it needs no trust of its own and reads the main
checkout's `.claude/settings.local.json` (tested, with a deny, in a worktree that had no local file of its own).

| Where the worktree is | What follows (tested) |
| --- | --- |
| Inside the main checkout | The main checkout's `git status` lists the directory until the project ignores it. Claude reads the worktree's own `AGENTS.md`, not the main checkout's `AGENTS.md` or `CLAUDE.md`. A `CLAUDE.local.md` in the main checkout, or in any directory between it and the worktree, reaches the worktree and turns its `AGENTS.md` off |
| Beside the repository or elsewhere | Nothing in the main checkout's directory reaches it; trust and the local settings still follow the main checkout |

## Unattended runs

`claude -p --permission-prompts none` refuses whatever would ask, so an `ask` entry stops an unattended run; a worker
adds its role's `--settings`. While the policy's `bypass` is `refuse`, or absent, `#connect` writes
`permissions.disableBypassPermissionsMode: "disable"`. Tested in an untrusted workspace, and the first row in a trusted
one too:

| Run | Without the key | With it |
| --- | --- | --- |
| `claude -p --dangerously-skip-permissions`, a command the policy does not allow | Runs | Refused: it needs approval, and nobody can give it |
| `--permission-mode bypassPermissions`, or `=bypassPermissions` | Runs | Refused |
| A command the policy denies, under `--dangerously-skip-permissions` | Refused | Refused |
| `--setting-sources user --dangerously-skip-permissions` | Runs | Runs: the project's settings, the key included, are skipped |

The key works from a `--settings` layer too (tested), and Claude's docs say it works from any scope
([permissions](https://code.claude.com/docs/en/permissions)).

## What steps outside the policy

| Route | Effect | How it is known |
| --- | --- | --- |
| `claude --dangerously-skip-permissions`, `--permission-mode bypassPermissions` | Refused while `bypass` is `refuse`; with `allow`, runs every command not denied | Tested |
| `claude --setting-sources user` | Skips the project's settings: the policy and the bypass refusal | Tested |
| A rule saved to the project settings from a session | Writes through the `.claude/settings.json` link into the shared adapter | Not tested |
