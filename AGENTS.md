> **This file is an unfilled template.** Fill it in only when the developer asks. If they ask for other work in an
> interactive session, say once that it is unfilled, then do that work. To fill it in:
>
> - Replace every `[[fill: …]]`, table cells included; `grep -n '\[\[fill:' AGENTS.md` finds them all. Write `none`
>   in a row that does not apply. Delete a section only when nothing else in this file points to it.
> - Link the README section instead of restating it, and fix the README in the same change when it is wrong.
> - Run every command you write. Mark one you could not run `(unverified)`; the next agent to run it removes the mark.
> - Never write a rule its source does not state: the code, a config file, the README, CI or the developer.
> - When replacing an existing `AGENTS.md`, follow `.agents/skills/agents-layout/references/adopt.md`.
> - `.agents/skills/agents-layout/references/agents-md.md` says why the fixed text is here; keep it unless it is
>   wrong for this repository.
> - Run `sh .agents/skills/agents-layout/scripts/check.sh`; when this note is all it reports, delete the note.

# [[fill: project name]]

[[fill: what this repository is and who uses it, in one or two sentences, with a link to the README]]

Stack: [[fill: each language, framework and service with its major version and the file that pins it]]

## Quick reference

- Install or add a dependency: [[fill: both commands]]
- Run: [[fill: command]]
- Build: [[fill: command]]
- Test one file: [[fill: command]]
- Lint or format one file: [[fill: command]]
- Generate code: [[fill: command, or none]]
- Local services: [[fill: start and stop commands, or none]]
- Worktrees: [[fill: where they go, how to create one ready to run and remove one with its data, or none]]
- Quick check, seconds on what changed: [[fill: command]]
- Done check, all that must pass before a change is done: [[fill: commands in order, time taken, services needed]]

## Boundaries

**Always**

- Run the Quick check before moving on. A change is done when the Done check passes.
- Report each check's result, or that it did not run.
- Update this file in the change that alters a command, path or rule it names.
- Keep this file under 800 words; past 1,200, move a section into a skill or memory file before adding. All
  `AGENTS.md` files together stay under 16 KiB.
- [[fill: another rule for every change here, or none]]

**Never**

- Commit secrets or credentials, or put personal data in logs, fixtures or commits.
- Edit generated files by hand: [[fill: paths, or none]]
- Make a check pass by weakening it: skipping a test, suppressing a warning, loosening an assertion.

**Stop and ask** before [[fill: such as new dependencies, schema changes, rewriting pushed history]], and when a
check fails again after a fix, a command is refused, or a decision is left open.

Enforced by [[fill: the file that makes agent tools refuse these, or none]]. A refusal is the rule working.

## Where things go

- New code: [[fill: the directory for each kind of code, related repositories, and the design document]]
- Tests: [[fill: directory and naming, with an example file]]
- Settings: [[fill: the file a new setting goes in, where secrets live (names only), and which source wins]]
- Decisions and domain rules: [[fill: directory or document]]
- A procedure an agent repeats: a skill in `.agents/skills/`.
- A trap in one area: `.agents/memory/`, listed under Deep dive.

## Rules

- A behavior change ships with a test; a bug fix with one that failed before the fix.
- Comments record only reasons the code cannot show.
- Change only what the task needs, matching the surrounding code.
- [[fill: a convention review enforces, with a file that follows it, or none]]
- Commits: [[fill: one change per commit; branch, message, pull request and tracker rules]]
- Deploy: [[fill: what triggers a deploy, where to see it worked, and the rollback runbook, or none]]

## Gotchas

At most five traps any task can hit.

- [[fill: what goes wrong when you do what, and what to do instead]]

## Deep dive (read on demand)

- [[fill: topic, and the document, skill or memory file that covers it]]
