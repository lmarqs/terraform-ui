# Shared agent layout

Every coding agent in this repository reads the same knowledge and the same policy. What differs between two people
or two runs is only what each one selects at launch: the harness, the model, whether someone is watching, and the
role. The `agents-layout` skill in `skills/` explains the layout in full: where a change goes, how to adopt it, how to
investigate an agent that misbehaves, and why it is shaped this way.

## What lives where

| Path | Holds | Decided by |
| --- | --- | --- |
| `/AGENTS.md` | Instructions for every agent, whatever its harness or role. Each harness reads it natively | The team |
| `skills/` | One directory per skill with its `SKILL.md`. Codex reads it natively; Claude through `/.claude/skills` | The team |
| `commands/` | One Markdown file per slash command. Claude reads it through `/.claude/commands`; Codex does not | The team |
| `hooks/` | Scripts that each harness's hook settings run, by their path from the repository root, and the files they load. Each `session-start.d/*.sh` prints context that every harness adds to a session as it starts, with the harness name as `$1` | The team; Cook owns the `session-start` dispatcher |
| `memory/` | Durable facts the code and `README.md` do not show, read when `AGENTS.md` points to them; no secrets | The team |
| `permission/policy.yaml` | What agents may run: `commands` and `tools` for every session, `roles` that narrow it | The team |
| `adapters/<harness>/` | Each harness's settings. Cook renders the policy into `permissions` in `claude/settings.json`, `sandbox_mode`, `approval_policy` and `sandbox_workspace_write.network_access` in `codex/config.toml`, and `codex/rules/cook.rules`, and owns `hooks.SessionStart` in `claude/settings.json` and `codex/hooks.json`, which runs `hooks/session-start` | Cook |
| `adapters/<harness>/`, every other key and file | Settings such as the other events under `hooks`, `env` and `[mcp_servers]`, written in the harness's own format | The team |
| `adapters/claude/roles/` | One file of denies per role, added at launch | Cook |
| `/.claude/`, `/.codex/` links | Committed, so a fresh clone, a cloud session or CI reads the setup with no cooking | Cook |
| `/.claude/settings.local.json` | Personal preferences and standing approvals | Each person; Cook never writes it |

After editing `permission/policy.yaml`, cook the adapters again with every harness the team uses, using the version in
`cook.lock`:

```sh
./cook io.github.lmarqs.cookbook:agents:<version>#connect --harness=claude --harness=codex
```

`./cook help io.github.lmarqs.cookbook:agents:<version>#connect` lists its other parameters, and `./cook --help`
Cook's own options.

## Ways of working

| Axis | Examples | Chosen | Lives in |
| --- | --- | --- | --- |
| Role | developer, worker, reviewer | Per run, at launch | `permission/policy.yaml` `roles`, rendered to `adapters/claude/roles/` |
| Mode | attended, unattended | Per run, at launch | Launch flags: unattended runs refuse what would ask |
| Harness | Claude, Codex | Per person or loop runner | `adapters/<harness>/` |
| Model | Opus, Sonnet, GPT | Per person, or per role in a loop runner | Personal settings or the runner's configuration, never the policy |

## Launching

The developer role is the base, with no role file. Add `--model <model>` to any line. The roles below are examples:
a role exists when `permission/policy.yaml` defines it.

| Harness | Mode | Role | Command |
| --- | --- | --- | --- |
| Claude | attended | developer | `claude` |
| Claude | attended | reviewer | `claude --settings .agents/adapters/claude/roles/reviewer.json` |
| Claude | unattended | developer | `claude -p --permission-prompts none "<task>"` |
| Claude | unattended | worker | `claude -p --permission-prompts none --settings .agents/adapters/claude/roles/worker.json "<task>"` |
| Codex | attended | developer | `codex` |
| Codex | unattended | developer | `codex exec -c approval_policy=never "<task>"` |

Codex has no roles: it has no option that adds a rules file to one run, so every Codex run gets the policy's
`commands` alone. A role's denies reach only Claude runs.

Claude applies the project's allow rules only in a workspace someone has trusted, so start `claude` here once,
interactively, before relying on unattended runs. Codex loads `.codex/rules` only in a trusted project. A worktree of
this clone, inside it or beside it, needs no setup of its own: the layout is committed, both harnesses take its trust
from this checkout, Codex's hook trust included, and Claude reads this checkout's `.claude/settings.local.json` there.
Codex trusts each hook as it was when accepted: after a cook or a pull changes `.codex/hooks.json`, start `codex` here
interactively and accept the hook review again, or Codex skips the changed hooks without a message.

## Loop runners

A loop runner, which runs agents over a series of tasks, often in worktrees, can rely on the paths above: `AGENTS.md`,
the links in `.claude/` and `.codex/`, and the role files in `adapters/claude/roles/`. Cooking keeps them where they
are.

- Set the runner's command to the unattended line of the harness it starts, with the role file for a Claude worker,
  and leave the coordinator, if it has one, in the developer role.
- While `permission/policy.yaml` says `bypass: refuse`, Claude refuses `--dangerously-skip-permissions` and
  `--permission-mode bypassPermissions`, which many runners pass by default. Codex keeps its rules under its bypass
  flag, but `--ignore-rules`, `--ignore-user-config` and a clone nobody trusted skip them, and `--approve-for-me` lets
  a reviewer approve the asks; configure Codex runners without them.
- How a worktree becomes runnable and how one is removed are in the Worktrees line of `/AGENTS.md`'s Quick
  reference.

## Personal settings

Each person customizes in their harness's own settings, never in this directory. `.claude/settings.local.json` holds
one person's settings for this project; Claude writes "don't ask again" approvals there. External user settings may
override this behavior.

Claude reads an `AGENTS.md` only while no `CLAUDE.md`, `.claude/CLAUDE.md` or `CLAUDE.local.md` exists in its
directory or one above; any of them makes it skip that `AGENTS.md`. `.gitignore` keeps them out of commits, so none
reaches the team, but a local one still takes effect for whoever has it. Reading `AGENTS.md` needs Claude Code 2.1.277 or later.

This directory sets what the team shares, not what a person may do. A personal `allow` does not outrank a team `ask`
or `deny`, but anyone can still override the shared setup on their machine: `claude --setting-sources user` skips the
project's settings, the bypass refusal included, and `codex exec --ignore-rules` skips the rules. Enforcing a rule on
everyone takes the organization's managed settings, outside the repository.
