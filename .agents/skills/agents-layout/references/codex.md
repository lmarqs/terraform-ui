# Codex

What Codex reads in this layout, how the policy reaches it, and what it trusts. Tested with codex-cli 0.160.1, except
where a line names another version or says it was not tested. [design.md](design.md) says why the layout is shaped
around these facts; [claude.md](claude.md) holds the same facts for Claude Code.

## What it reads

| Path | What Codex takes from it |
| --- | --- |
| `AGENTS.md` | The instructions, natively; it ignores `.codex/AGENTS.md` (tested) |
| `.agents/skills/` | Skills, natively; a skill invoked as `$agents-layout review --fix` gets the prompt as typed (tested) |
| `.codex/config.toml`, a link to `.agents/adapters/codex/config.toml` | The project configuration: the sandbox and approval keys the policy sets, and every other key the team writes |
| `.codex/rules`, a link to `.agents/adapters/codex/rules/` | The command rules, `cook.rules` among them, loaded whole |
| `.codex/hooks.json`, a link to `.agents/adapters/codex/hooks.json` | The hooks, read from the main checkout even in a worktree ([Worktrees](#worktrees)) |

Codex reads project configuration only under `.codex/`, and reads all of it only in a trusted project
([Trust](#trust)). It does not read `.agents/commands/` or `.claude/rules/`.

## Permissions

`#connect` sets `sandbox_mode = "workspace-write"`, `approval_policy = "on-request"` and
`sandbox_workspace_write.network_access = false`, and renders each `commands` entry as one `prefix_rule` in
`.agents/adapters/codex/rules/cook.rules`, keeping the alternatives as nested lists: `allow` stays `allow`, `ask`
becomes `prompt` and `deny` becomes `forbidden` ([policy.md](policy.md#prefix)). A `{claude: "<pattern>"}` entry becomes
the comment `# Not enforced by Codex, Claude-only: <pattern>`, because Codex cannot match it, and `tools` renders
nothing. `codex execpolicy check --rules .agents/adapters/codex/rules/cook.rules <command>` prints the decision the
rules give a command.

### No roles

A role needs a way for one run to add rules that another run does not get. Codex has none:

- `CODEX_PERMISSION_PROFILE` is a variable Codex sets for the commands it runs, not one that selects a profile.
- `prefix_rules` in configuration exist only in `requirements.toml`, which an administrator controls.
- `--profile <name>` layers a profile from the person's own configuration, not from the repository.
- No option names a rules file; Codex loads the project's `.codex/rules` whole, or skips them with `--ignore-rules`.

So Cook renders no Codex role, and a Codex run gets `commands` only. A team that runs Codex only as a worker can move
the worker's denies into `commands.deny`, at the cost of binding people at the keyboard too, or rely on Codex's
sandbox for workers. `-c <key>=<value>` sets configuration keys for one run, such as the sandbox's, but no rules.

## Trust

Codex keeps two kinds of trust in each person's `~/.codex/config.toml`, both keyed by the main checkout's absolute
path and shared by its worktrees:

- **The project.** Codex loads `.codex/` only in a trusted project: in a clone nobody trusted it skips the rules and
  the configuration and applies its read-only default sandbox. It asks when started interactively. codex-cli 0.160.1
  ignores `-c 'projects."<path>".trust_level="trusted"'` and prints `unrecognized configuration setting`.
- **Each hook.** Codex records one entry per hook, as
  `hooks.state."<path>/.codex/hooks.json:session_start:0:0".trusted_hash = "sha256:<hash>"`, and runs a hook only
  while its content matches that hash. A repository at a new path runs none of its hooks, a clone trusted as a project
  included, and an entry that changed since it was trusted runs nowhere, until someone trusts it again. `codex exec`
  skips both without a warning. So a cook that changes `hooks.SessionStart`, such as the first one in a repository
  whose team had its own entry, turns the session start off in every person's Codex until each of them trusts it
  again. codex-cli 0.160.1 has no command that trusts hooks: start `codex` interactively in the main checkout and
  accept the hook review. `--dangerously-bypass-hook-trust` runs the hooks for one run.

To check a checkout, run `codex exec --sandbox read-only "<any prompt>" </dev/null 2>err.log` there, without the
bypass flag: a trusted `SessionStart` prints `hook: SessionStart` and `hook: SessionStart Completed` to `err.log`, an
untrusted one neither.

## Hooks

`#connect` owns `hooks.SessionStart` in `hooks.json`. Codex runs a relative hook command from the session's
directory, so a session started in a subdirectory fails to find `.agents/hooks/session-start` (tested); the command
starts from `git rev-parse --show-toplevel` instead, runs the dispatcher with the argument `codex`, and Codex adds the
`additionalContext` it prints to the session (tested). codex-cli 0.160.1 has no hook event for creating or removing a
worktree. Every other event under `hooks` is the team's.

## Worktrees

A worktree, inside the checkout or beside it, takes the main checkout's trust, project and hooks alike, and needs none
of its own: no `hooks.state` entry appears for it. It reads `.codex/rules` from its own checkout, but runs the hook
list of the main checkout's `.codex/hooks.json`, not its own, so a hook change reaches a worktree only once the main
checkout has it ([tested](#how-it-is-tested)).

## Unattended runs

`codex exec -c approval_policy=never` has nobody to ask, so a `prompt` rule refuses there; not tested. Codex has no
setting that refuses its bypass flags, and needs one less: its rules hold under
`--dangerously-bypass-approvals-and-sandbox`, which refuses a `prompt` rule instead of running it (tested). While the
policy's `bypass` is `refuse`, `cook.rules` opens with a comment naming the runs that step outside the rules, since no
setting refuses them.

## What steps outside the policy

| Route | Effect | How it is known |
| --- | --- | --- |
| `codex exec --ignore-rules` | Skips `cook.rules` | Tested |
| An untrusted project, including a run with `--ignore-user-config` | Skips `.codex/`, so the rules and the sandbox settings | Tested |
| `codex exec --approve-for-me` | An automatic reviewer approves `prompt` rules; `forbidden` still holds | Tested |
| `codex exec --dangerously-bypass-approvals-and-sandbox`, `--sandbox danger-full-access` | The rules hold; the sandbox is gone | Rules tested; sandbox from `codex exec --help` |
| `codex exec -c <key>=<value>` | Sets any configuration key for one run, the sandbox and approval keys included | `codex exec --help`; not tested |

## How it is tested

The trust claims above come from one probe, which anyone can rerun when Codex changes. It needs a Codex login and
costs one short model call per row.

1. Point `CODEX_HOME` at a new directory holding a copy of `~/.codex/auth.json` and an empty `config.toml`, so no
   trust from the person's own configuration leaks in. A trusted parent directory, such as the home directory, would
   otherwise trust every repository under it.
2. Create a repository outside `/tmp` and commit three files: `.codex/rules/probe.rules` with
   `prefix_rule(pattern = ["touch", "rule-probe"], decision = "forbidden")`; `.codex/hooks.json` with a `SessionStart`
   hook running `sh -c 'touch "$(git rev-parse --show-toplevel)/hook-probe"'`; and `.codex/config.toml` with
   `sandbox_mode = "workspace-write"` and `approval_policy = "never"`.
3. Add a worktree inside it and one beside it with `git worktree add`, and `git clone` it to a third path.
4. Start `codex` interactively in the main checkout, in a terminal, and accept both the project and the hook trust
   questions. `config.toml` then holds `projects."<path>"` and
   `hooks.state."<path>/.codex/hooks.json:session_start:0:0"`.
5. In each checkout, run `codex exec --json -C <dir> "Run exactly this shell command and nothing else: touch
   rule-probe ." </dev/null`. A missing `rule-probe` with a `rejected: policy forbids` error means the rules applied;
   a `hook-probe` file means the hook ran. Delete both files between runs.

Results with codex-cli 0.160.1:

| Checkout | `.codex/rules` applied | Hook ran |
| --- | --- | --- |
| Main checkout, project and hooks trusted | Yes | Yes |
| Worktree inside it | Yes | Yes |
| Worktree beside it | Yes | Yes |
| Worktree, with `--dangerously-bypass-hook-trust` | Yes | Yes |
| Clone nobody trusted | No, and the read-only default sandbox applied, so its `config.toml` was skipped too | No |
| Clone trusted as a project in `config.toml`, hooks never accepted, same `hooks.json` | Yes | No |
| Main checkout, after its trusted `SessionStart` entry changed | Yes | No, until trusted again |
| Worktree whose own `hooks.json` differs from the main checkout's (codex-cli 0.159.0) | Yes | The main checkout's hook ran, not the worktree's |
