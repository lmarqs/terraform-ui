# Memory

`.agents/memory/` holds traps. A trap is something in the repository that looks right and is not: a command, a
file, a check or a setting that gives a wrong result to a run that does the plain thing. Memory keeps the traps of
one area, a directory, a module or a tool a task touches, in one file per area, as `.agents/memory/migrations.md`.
A run reads the file only when its task touches the area: `AGENTS.md` names each file under Deep dive with the
question it answers, and a file no `AGENTS.md` names is never read.

## Where a learned fact goes

Most of what a run learns has an owner already, and a copy in memory drifts from it. Take the first row that
fits:

| The fact is | It goes to |
| --- | --- |
| A secret, or about a person | Nowhere in the repository |
| What this run did, what is left to do, or a plan | The tracker, or the commit message |
| One person's preference | That person's harness settings, outside the repository |
| Shown where a run on the task reads anyway: the code it changes, a config file, the README, the history | Nowhere; name the file where a reader needs it |
| Enforced by a check, a hook or the policy that exists now or that this change adds | The check, the hook or `policy.yaml` |
| Needed by every task | `AGENTS.md`: a rule, or one of its five gotchas |
| A procedure an agent repeats | A skill in `.agents/skills/` |
| About the product, so people need it too: a decision, how a feature behaves, a standard, a trap of the product | The repository's documentation; the page owns it and memory links it. With no documentation, memory holds it until the team writes the page |
| A trap of working in one area: its commands, its files, its checks, its settings, including one whose check is more work than this change | `.agents/memory/<area>.md` |
| Anything else | Nowhere yet: the report asks the team where it goes |

## How to recognize a trap

A run knows what it saw before it knows it met a trap. Each of these is one:

| The run saw | The trap |
| --- | --- |
| A command ran clean and did something its name does not say | A command that looks right and is not |
| A change passed where it was made and broke elsewhere | Files that must change together, with nothing pointing from one to the other |
| An edit was overwritten or reverted | A file something else writes: generated, rendered or owned by a tool |
| Exit 0, "done" or silence, and the work was not done | A success signal that is false |
| A setting is in the file and had no effect | A setting something overrides, shadows or never loads |
| A rule or a check let through what it reads as forbidding | A guard narrower than it reads |

The list is open: whatever looked right and was not is a trap. A run that was slow and not wrong met none. A file
that was hard to find gets a line under Where things go in `AGENTS.md`, and a costly routine becomes a skill.

## What earns an entry

A trap the table sends to memory earns its entry when each answer is yes:

- Would a fresh run fall into it again without the entry?
- Did it happen: a person's correction, a failed check, a command's output? A guess about why something works
  waits for one of those.
- Does it name the area it applies to, so a reader outside that area can skip it?

## How an entry reads

The fact first, as an instruction a reader can follow. Then why: what happened, and what it cost. Then when it
applies, where that is not plain. The reason lets a reader judge a case the words leave open, and tell when the
entry has stopped being true. `.agents/memory/migrations.md` in a made-up project:

```markdown
# Migrations

What to know before adding or changing a file in `db/migrations/`.

- Number a new migration from `main`, not from the branch: run `git fetch` and take the next number after
  `origin/main`'s last. Two branches each took `0042`, both passed their checks, and the deploy ran only one.
- `make db-migrate` exits 0 when the database is down, and applies nothing. Run `make db-status` after it and
  read the last number applied; a run reported a migration as tested that had never run.
```

A file opens with the question it answers, the one `AGENTS.md` repeats under Deep dive. It states what is true now,
with no dates, authors or log of changes: the history holds those.

## When an entry leaves

Memory that only grows is read less each time it is opened. An entry leaves in the change that makes it untrue or
gives it another owner:

- A check, a hook or a policy entry comes to enforce it.
- The code changes so the trap is gone.
- Every task comes to need it, and it moves to `AGENTS.md`.
- People come to need it, and it moves to a documentation page.

A file that answers two questions is split, and each half named under Deep dive. An empty file is deleted with its
line in `AGENTS.md`.

## Who writes it

An agent writes the entry in the change where a person corrects it, or where a failure reveals a trap the next
run would meet, and names the entry in its report. The entry is committed and reviewed like every file the
team owns: whoever changes it steers every agent that reads it.

A harness's own memory is another thing. Claude Code's auto memory is written by Claude under
`~/.claude/projects/<project>/memory/` and stays on one person's machine (Claude's docs; not tested), so nothing
there reaches a teammate, another harness or a loop runner's worker. What the team needs is written here.
