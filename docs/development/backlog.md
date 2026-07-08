---
layout: default
title: Backlog Maintenance
parent: Development
nav_order: 3
description: How the terraform-ui backlog is tracked — project board, status workflow, and label taxonomy
---

# Backlog Maintenance

All planned work — features, bugs, refactors, and roadmap items — is tracked as
**GitHub issues** organized on the **[terraform-ui project board](https://github.com/users/lmarqs/projects/2)**.
There is no in-repo backlog file: `docs/_roadmap/` was retired and its contents
migrated to issues (see [Roadmap](#roadmap) below).

## Where things live

| Concern | Home |
|---------|------|
| Backlog & prioritization | [Project #2](https://github.com/users/lmarqs/projects/2) |
| Individual work items | [Issues](https://github.com/lmarqs/terraform-ui/issues) |
| Design decisions (why) | [ADRs](../adr/) |
| Domain vocabulary | [`CONTEXT.md`](https://github.com/lmarqs/terraform-ui/blob/main/CONTEXT.md) |

## Status workflow

Every issue on the board carries a **Status** (single-select field). The flow is
strictly left-to-right; issues move one column at a time.

| Status | Meaning |
|--------|---------|
| **Backlog** | Captured, not yet scheduled. The default for ideas and roadmap items. |
| **Todo** | Ready to work — scoped well enough to start. |
| **In Progress** | Actively being worked (branch/commits in flight). |
| **In Review** | Committed and pushed; awaiting verification / review. |
| **Done** | Merged to `main` and verified. |

Guidelines:

- Pull from **Todo**, not **Backlog** — if something isn't Todo-ready, scope it first.
- Keep **In Progress** small (ideally one item at a time; this repo ships small commits directly to `main`).
- An issue reaches **Done** only after `mise run check:lint && mise run test:unit && mise run check:build` pass (plus `mise run test:macro` for UI changes).

## Label taxonomy

GitHub's native *Issue Types* are org-only, so **types are expressed as labels**.
Every issue gets exactly one **type** label; **area/status** are carried by the
board, not by labels.

| Label | Use for |
|-------|---------|
| `bug` | Incorrect behavior in shipped code. |
| `enhancement` | New user-facing feature or capability. |
| `refactor` | Internal restructuring, no behavior change (see the architecture deep-dives). |
| `ux` | TUI/CLI experience polish (hints, keybindings, output). |
| `tech-debt` | Known correctness risk or debt to pay down. |
| `documentation` | Docs-only changes. |
| `roadmap` | A planned feature migrated from the old `docs/_roadmap/`. |

Labels compose: a roadmap feature that is also a refactor carries `roadmap,refactor`;
a correctness-risk bug carries `bug,tech-debt`.

## Issue conventions

- **Title**: imperative and specific — `State Browser: pinning a resource wipes the list`, not `pin bug`. Prefix architecture refactors with `[arch]`.
- **Body**: Summary → Root cause / Evidence (with `file:line`) → Fix → Verification. Cross-link related issues and ADRs.
- **No speculative issues**: file work that serves a real, current need (mirrors the *No Speculative Code* rule in [`CLAUDE.md`](https://github.com/lmarqs/terraform-ui/blob/main/.claude/CLAUDE.md)).
- **Close via commits**: end the commit body with `Fixes #NN` so merging to `main` closes the issue automatically. Use [conventional commit](https://www.conventionalcommits.org/) prefixes (`fix:`, `feat:`, `refactor:`, …) so semantic-release versions correctly.

## Roadmap

The roadmap is the set of open issues labelled
[`roadmap`](https://github.com/lmarqs/terraform-ui/issues?q=is%3Aissue+is%3Aopen+label%3Aroadmap).
Completed roadmap items are **closed**, never marked "done" in place — the same
delete-on-completion discipline the old `docs/_roadmap/` collection used.

## Common operations (`gh` CLI)

```bash
# Create a typed issue
gh issue create --repo lmarqs/terraform-ui \
  --title "Plan: filter changes by action type" \
  --label "roadmap,ux,enhancement" --body-file ./notes.md

# Add an issue to the board and capture its item id
item=$(gh project item-add 2 --owner lmarqs --url <issue-url> --format json | jq -r .id)

# Set that item's Status (field/option ids are stable per project)
gh project field-list 2 --owner lmarqs          # discover the Status field id
gh project item-edit --id "$item" --project-id <project-id> \
  --field-id <status-field-id> --single-select-option-id <option-id>
```

Field and option ids are discoverable via `gh project field-list 2 --owner lmarqs`
and a GraphQL query on the `Status` field's `options`.
