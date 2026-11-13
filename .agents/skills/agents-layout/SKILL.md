---
name: agents-layout
description: Shared .agents/ layout that gives every harness in a repository the same instructions, permissions, skills and subagents. Use it to adopt the layout; change agent instructions, permissions, roles, skills or subagents; investigate an agent that ignores the setup or runs what the team did not expect; or learn why the layout is shaped this way. Not for the repository's own code, or a person's harness settings outside this repository.
argument-hint: "[adopt | review [--fix]]"
---

# Agents layout

## Invoked with arguments

The words after the skill's name choose a procedure.

| Arguments | Do |
| --- | --- |
| `adopt` | Follow [references/adopt.md](references/adopt.md) from its first step |
| `review` | Run [the review](references/review.md#review) and report it; change no file |
| `review --fix` | Run the review, then [fix what it finds](references/review.md#review---fix) |
| None | Pick the task from [the table below](#pick-the-task) |

## The layout

One repository, many ways of working: one person runs a harness at the keyboard, another leaves a different one
working alone, a third has one harness coordinate another's workers through a loop runner. The layout gives all of
them the same knowledge and the same agreed permissions, and leaves each run free to pick its harness, model, mode
and role.

## Owners

Every file here has one owner. Most surprises come from editing a file someone else owns.

| Owner | Files | Changed by |
| --- | --- | --- |
| The team | `AGENTS.md`, `.agents/permission/policy.yaml`, `.agents/skills/`, `hooks/` with its own session-start scripts in `hooks/session-start.d/`, `memory/`, `agents/`, `.agents/README.md`, and every key in the adapter files that Cook does not own | Editing and committing, through review |
| Cook | `.agents/skills/agents-layout/` | Cooking `io.github.lmarqs.cookbook:agents#seed` with a new release |
| A tool's recipe | A script in `.agents/hooks/session-start.d/`, at the path the person names by a parameter of that recipe | Cooking that recipe with that parameter |
| Cook | The renders: from the policy, `permissions` in `.agents/adapters/claude/settings.json`, `sandbox_mode`, `approval_policy` and `sandbox_workspace_write.network_access` in `.agents/adapters/codex/config.toml`, `.agents/adapters/codex/rules/cook.rules` and `.agents/adapters/claude/roles/`; from `.agents/agents/`, `.agents/adapters/<harness>/agents/` | Cooking `io.github.lmarqs.cookbook:agents#connect` after a team file changes |
| Cook | The wiring: `.agents/hooks/session-start` and `hooks.SessionStart` in `.agents/adapters/claude/settings.json` and `.agents/adapters/codex/hooks.json`; the links in `.claude/` and `.codex/`; the `harnesses` section of `.gitignore` | Cooking `io.github.lmarqs.cookbook:agents#connect` |
| Each person | `.claude/settings.local.json` | Nobody else; Cook never writes it and git ignores it |
| A loop runner | Its own configuration and the worktrees it creates | Whoever sets it up; it reads the layout through [the contract](references/design.md#worktrees-and-loop-runners) |

```text
AGENTS.md                         instructions for every agent; each harness reads it directly
.agents/
  README.md                       what lives where, and the command line for each kind of run
  permission/policy.yaml          what agents may run; the one file to edit for permissions
  adapters/<harness>/             each harness's settings: the policy as Cook renders it, other keys the team's
  skills/ hooks/ memory/ agents/
.claude/  .codex/                 committed links into .agents/, plus each person's local files
```

Cook's `io.github.lmarqs.cookbook:agents` recipes write it: `#seed` creates it, this skill included, and `#connect`
renders the policy and makes the links. Commands here write the recipe version as `<version>`; use the one in `cook.lock`.
`cook --help` lists Cook's options and commands, and
`cook help io.github.lmarqs.cookbook:agents:<version>#<recipe>` prints a recipe's help with each parameter, its
values and default, without running it. Read it before cooking with a parameter this skill does not show, such as
`#connect`'s `--worktree-*`.

The layout cannot decide how people work; anyone can override it on their own machine. It guides: shared changes
reach every agent when they go where every agent reads them, and personal ones stay out of the team's files.

## Shared directories

Everything the team shares with agents, apart from `AGENTS.md` and the policy, goes in one of four directories under
`.agents/`. Each harness reaches them in its own way; nothing needs copying per harness.

| Directory | Holds | Claude reaches it | Codex reaches it |
| --- | --- | --- | --- |
| `skills/` | One directory per skill: `SKILL.md`, whose frontmatter `name` and `description` say when to use it, and the files it refers to | Through the `.claude/skills` link | Natively |
| `hooks/` | Scripts that hook settings run, by their path from the repository root, and the files they load | The `hooks` key of `.claude/settings.json` | `.codex/hooks.json` |
| `memory/` | Durable facts about the repository that the code and `README.md` do not show; never secrets or personal data | When `AGENTS.md` points to a file | When `AGENTS.md` points to a file |
| `agents/` | One Markdown file per subagent, as [subagents.md](references/subagents.md) describes | Through `.claude/agents`, rendered by `#connect` | Through `.codex/agents`, rendered by `#connect` |

- **A skill** is for knowledge an agent needs only for some tasks: it loads when its description matches, so it costs
  nothing otherwise. A rule every task needs belongs in `AGENTS.md`.
- **A subagent** is a helper with its own prompt that an agent delegates a task to. It is written once and rendered
  for each harness, so both run the same prompt.
- **A hook** runs on a harness event, such as before a command or after an edit, with no agent deciding. The script
  is shared; when it runs is set per harness, because the events and their input differ. Session start is the
  exception: a script in `hooks/session-start.d/` that prints text reaches every connected harness as session
  context, and gets the harness name as `$1`.
- **Memory** holds what a past session learned and every later one needs, such as why a dependency is pinned. List
  each file in `AGENTS.md` with the question it answers, so agents know when to read it.

## Pick the task

| Task | Read |
| --- | --- |
| Set the layout up, or move an existing `CLAUDE.md`, `.claude/` or `.codex/` setup into it | [references/adopt.md](references/adopt.md) |
| Fill in or review `AGENTS.md` | [references/agents-md.md](references/agents-md.md) |
| Write or review `policy.yaml` | [references/policy.md](references/policy.md) |
| Write or review a shared subagent | [references/subagents.md](references/subagents.md) |
| An agent ignores the instructions, asks when it should not, or runs what it should not | [references/investigate.md](references/investigate.md) |
| Set up a loop runner, or decide where worktrees go | [references/design.md](references/design.md#worktrees-and-loop-runners) |
| Know what Claude Code or Codex reads, trusts and enforces here, and how it was tested | [references/claude.md](references/claude.md), [references/codex.md](references/codex.md) |
| Understand why the layout is shaped this way, or argue for changing it | [references/design.md](references/design.md) |
| Know what a term of the layout means, such as role, subagent or adapter | [references/glossary.md](references/glossary.md) |

## Where a change goes

| To | Edit | Then |
| --- | --- | --- |
| Tell every agent something | `AGENTS.md` | Commit |
| Allow, ask before or deny a command in every session | `commands` in `.agents/permission/policy.yaml` | Cook `#connect`, review, commit |
| Deny a command in one kind of run, such as unattended workers | `roles.<role>.deny` in the same file | Cook `#connect`, review, commit |
| Share a skill, hook script or durable fact | `.agents/skills/`, `hooks/` or `memory/` | Commit; the links expose it |
| Share a subagent | `.agents/agents/<name>.md` | Cook `#connect`, review, commit |
| Give every agent context as its session starts | A script in `.agents/hooks/session-start.d/` that prints it, named with a number first to set its order, such as `10-status.sh`; the harness name is `$1` | Commit; Codex runs it once the hooks are trusted in the main checkout, its worktrees included, [codex.md](references/codex.md#trust) |
| Run a shared hook script on another harness event | The `hooks` key of `.agents/adapters/claude/settings.json`, and `.agents/adapters/codex/hooks.json`; `SessionStart` is Cook's in both | Commit |
| Change another shared harness setting, such as `env` or an MCP server | That key in `.agents/adapters/claude/settings.json` or `.agents/adapters/codex/config.toml` | Commit |
| Change something for yourself | `.claude/settings.local.json` | Nothing to commit |
| Choose the harness, model, mode or role of one run | The launch command, from Launching in `.agents/README.md` | Nothing to commit |

For example, to let every agent run the test suite:

```yaml
# .agents/permission/policy.yaml
commands:
  allow:
    - [./gradlew, test]
```

```sh
cook --dry-run io.github.lmarqs.cookbook:agents:<version>#connect --harness=claude --harness=codex
cook --batch-approve io.github.lmarqs.cookbook:agents:<version>#connect --harness=claude --harness=codex
git status --short
git add AGENTS.md .agents .claude .codex .gitignore cook.lock
git commit
```

Read the diff the dry run prints before approving: an agent has no terminal to answer Cook's prompt, so without
`--batch-approve` Cook writes nothing.

The diff adds `Bash(./gradlew test *)` to `.agents/adapters/claude/settings.json` and a `prefix_rule` to
`.agents/adapters/codex/rules/cook.rules`, and `cook.lock` records them. The first command entry also creates that
rules file and its `.codex/rules` link.

## Guidance

- **Edit the policy, not what is rendered from it.** The keys Cook owns come from `policy.yaml`: a hand edit to one
  reaches one harness only, and the next cook proposes undoing it. Every other key in the harness files is the
  team's: edit it in place, and cooking keeps it.
- **Keep `CLAUDE.md`, `.claude/CLAUDE.md` and `CLAUDE.local.md` out of the repository.** While any of them exists in
  the directory of an `AGENTS.md` or one above, Claude stops reading that `AGENTS.md`. `.gitignore` keeps them out of
  commits; a local one still affects whoever has it.
- **Widen in `commands`, narrow in roles.** A role only adds denies, so a worker never gets more than a person at the
  keyboard. When a role needs something `commands` lacks, that is a change for everyone; agree it with the team.
- **Use `ask` sparingly.** Nobody answers in an unattended run, so an `ask` refuses there, and every unattended run that
  reaches it stops.
- **Leave models out of shared settings.** A run's model is a person's or a loop runner's choice; a model in the
  shared settings overrides everyone's. A subagent may name its own, which applies only while it runs.
- **Cook every harness the team uses.** `#connect` removes what it wrote for harnesses left out, and those files are
  committed, so cooking only your own harness removes a teammate's.
- **Keep secrets out of `.agents/`.** Everything there is committed and read by every agent.
- **Treat the policy as an agreement, not a lock.** While `bypass` is `refuse`, Claude refuses its bypass flags, but
  `claude --setting-sources user` or `codex exec --ignore-rules` still skips the policy on one machine. A rule every
  machine must obey belongs in the organization's managed settings.
