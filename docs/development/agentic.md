---
layout: default
title: Agentic development
parent: Development
nav_order: 2
description: "Coordinate agent tasks, review their deliveries, and run Beads epics through Ralph."
audience: "Contributors and coordinating agents who know Git and mise and need the repository's agent workflow"
---

# Agentic development

The coordinator reviews and delivers every task, including tasks executed through Ralph. The coordinator prepares acceptance
criteria, monitors the run, reviews every diff, verifies checks, and resolves failures before delivery.

## Roles and task tracking

The **coordinator** is the agent working with the user: it owns task selection, review, follow-up fixes, and delivery.
A **worker** is an agent assigned a task with acceptance criteria. The **runner** starts workers and records task
completion; Ralph is the runner used here. Runner completion and coordinator acceptance are separate: a closed
Bead still needs review.

Beads stores the execution tasks, grouped into an **epic**. Ralph runs those tasks in a dedicated Git worktree.
GitHub issues remain the project backlog; the epic breaks that work into tasks a worker can complete and test.

## Start a coordinated run

Start a coordinated run with `mise run ralph:run <epic> --headless`. The repository defaults to jailed Codex
workers, serial execution, and at most ten iterations. The run lives at `.worktrees/ralph/<epic>` on branch
`ralph/<epic>`. The [Beads/Ralph skill](https://github.com/lmarqs/terraform-ui/blob/main/.agents/skills/beads-ralph/SKILL.md)
owns tool setup, tracker discovery, and launcher details. Use `--iterations N` to bound the run and `--model NAME`
to select a model supported by the chosen worker.

Asking the coordinator to implement work through Ralph authorizes the run and its local task commits. The
coordinator proceeds with routine fixes and reviews. It asks for input when a scope or product decision remains
open, or when an action needs access or authorization the user has not already granted. Publishing and merging
follow the user's delivery instructions.

## Review each delivery

For each delivery, the coordinator:

1. Compares the diff and final worker response with the task's acceptance criteria.
2. Confirms the required checks passed for the delivered changes, including any task-specific regression tests.
3. Checks the Bead status, commit, worktree status, and run logs together. A completion followed by an error or retry
   needs investigation even if the Bead is already closed.
4. Resolves review findings before publishing. Workers do not push, merge, or update Beads; Ralph makes the local
   task commits and closes tasks, and the coordinator verifies those actions.

## Change the runner safely

For changes to the runner itself, execute an unchanged copy of the worker. A shell may keep reading its script
after startup, so rewriting that script during a task can cause an error after the task's checks pass. Start the
next run with the reviewed version.

