# Claude Code

What Claude Code reads in this layout, how the policy reaches it, and what it trusts. Tested with Claude Code 2.1.303
or 2.1.304 (2.1.304 for skill arguments, worktrees, bypass flags and `--setting-sources`), 2.1.306 for the session
start and 2.1.326 for auto mode, except where a line says it was not tested. [design.md](design.md) says why the
layout is shaped around these facts; [codex.md](codex.md) holds the same facts for Codex.

## What it reads

| Path | What Claude takes from it |
| --- | --- |
| `AGENTS.md` | The instructions, from 2.1.277 on, while no `CLAUDE.md`, `.claude/CLAUDE.md` or `CLAUDE.local.md` sits in its directory or one above; any of the three makes Claude read it instead (tested, `CLAUDE.local.md` included). A person can turn `AGENTS.md` support off under "Project instructions" in `/config`, as Claude Code's changelog says; not tested |
| `.claude/settings.json`, a link to `.agents/adapters/claude/settings.json` | The project settings: the policy as `permissions`, `hooks` and every other key the team writes. Claude reads project settings only there |
| `.claude/skills`, a link to `.agents/skills/` | Skills. Claude finds them only there, and does not read `.agents/skills/` (tested) |
| `.claude/agents`, a link to `.agents/adapters/claude/agents/` | The shared subagents, rendered from `.agents/agents/` ([subagents.md](subagents.md)); Claude reads subagents from `.claude/agents/` and ignores a front matter key it does not know, per its documentation; `/agents` lists a rendered one (tested with 2.1.318) |
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

## Auto mode

From 2.1.283 on, an interactive session starts in auto mode unless the person chose another mode: Claude's classifier,
instead of the person, decides each command no rule covers, and blocks what it judges destructive or aimed outside the
repository, such as a force push or `git reset --hard`
([permission modes](https://code.claude.com/docs/en/permission-modes)). The policy's rules are read first:

| The policy says | In auto mode | How it is known |
| --- | --- | --- |
| `deny` | Refused before the classifier sees it | Claude's docs |
| `ask` | A person is asked; an unattended run refuses it | Unattended tested; attended from Claude's docs |
| `allow` | Runs without the classifier, unless the rule is one auto mode drops | Claude's docs |
| Nothing | The classifier decides | Tested: `claude -p --permission-mode auto` ran a `touch` no rule listed |

So in a session at the keyboard, `ask` is what brings a person in; an unlisted command no longer does.

On entering auto mode Claude drops the allow rules broad enough to run arbitrary code, and restores them on leaving
it (Claude's docs; not tested). For the rules `#connect` renders, that means:

| Entry | Renders | Why it is dropped |
| --- | --- | --- |
| `[python3]` | `Bash(python3 *)` | An interpreter followed by anything |
| `[timeout]` | `Bash(timeout *)` | A wrapper followed by anything |
| `[npm, run]` | `Bash(npm run *)` | A package manager's run command |
| `{claude: "*"}`, `{claude: "* --version"}` | `Bash(*)`, `Bash(* --version)` | A rule that starts with a wildcard |
| `{claude: "Agent(...)"}`, `{claude: "Monitor"}` under `tools` | As given | Agent and Monitor rules |

A narrow entry, such as `[npm, run, test]` or `[./gradlew, check]`, stays in effect.

What a team can and cannot share about auto mode:

| Setting | Shared through the layout | How it is known |
| --- | --- | --- |
| `permissions.disableAutoMode: "disable"` | Yes: `#connect` writes it while the policy's `auto` is `refuse`. `claude -p --permission-mode auto` then starts in `default` and refuses an unlisted command | Tested |
| `permissions.defaultMode: "auto"` | No: Claude gives it no effect in project settings, and the layout sets no `defaultMode` ([design.md](design.md#why-mode-is-not-a-policy-field)) | Claude's docs |
| `autoMode`, the classifier's `environment`, `allow`, `soft_deny` and `hard_deny` | No: Claude reads it from `~/.claude/settings.json`, managed settings and `--settings` only, so that a repository cannot loosen its own classifier ([auto mode configuration](https://code.claude.com/docs/en/auto-mode-config)) | Claude's docs |
| An instruction, such as "never force push" | Claude's docs say the classifier reads the instruction files Claude loads; that this covers `AGENTS.md` was not tested | Claude's docs |

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
adds its role's `--settings`. `claude -p` with no mode starts in `default`, not in auto mode (tested with 2.1.326).
With `--permission-mode auto` the classifier decides each unlisted command instead of refusing it, an `ask` entry is
still refused (tested), and a command the classifier blocks does not run while the run goes on (Claude's docs).

While the policy's `bypass` is `refuse`, or absent, `#connect` writes
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
| `claude --permission-mode auto`, or an interactive session, which starts in it | Stays inside the policy's `deny` and `ask`; a command no entry lists runs when the classifier lets it, without a person. Refused while `auto` is `refuse` | Tested unattended |
| A rule saved to the project settings from a session | Writes through the `.claude/settings.json` link into the shared adapter | Not tested |
