---
name: beads-ralph
description: "Beads task tracking, Beads Viewer triage and ralph-tui runs in this repository: finding ready work, claiming, creating and closing beads, ranking an epic with bv, and launching or cleaning a ralph-tui run in its worktree. Use before running bd, bv or .ralph-tui/bin/*, or when splitting work into beads."
---

# Beads and ralph-tui

Beads holds the task graph; Beads Viewer (`bv`) ranks it; ralph-tui works through an epic's beads with unattended
workers, one worktree per run.

`bd`, `bv` and `ralph-tui` are installed globally by each person; mise.toml pins none of them.
Start a run over an epic with `mise run ralph:run <epic>`, adding `--headless` to log instead of drawing the TUI and
`--agent claude` for Claude workers. Codex workers are the default and run without a sandbox, with the person's
credentials and network; the prompt keeps them from committing, pushing or running `bd`, and ralph-tui commits and
closes each bead.

The sections below are written by the Cook recipes that set each tool up.
<!-- cook:begin io.github.lmarqs.cookbook:beads#tracker/tracker -->
<!-- io.github.lmarqs.cookbook:beads#tracker --share=team --instructions=.agents/skills/beads-ralph/SKILL.md -->
- Tracker policy: cook, recook and change policy in the main checkout; one policy applies to all linked worktrees
  of this physical clone. In a normal Git checkout, return there before setup or a policy change:

  ```sh
  main_checkout="$(dirname -- "$(git rev-parse --path-format=absolute --git-common-dir)")"
  cd "$main_checkout"
  ```

  If you cooked in a linked worktree, return here and recook with the recorded arguments from that cook's root
  `.gitignore` tracker section; preserve `--share` and every `--instructions` target. Separate team and private
  policies require separate physical clones or explicitly selected native stores. Tracker configures only its
  kitchen's .beads; it does not configure external stores. A branch-local cook does not establish independent policy.
- Beads tasks: after installing `bd` 1.3.1, run `bd bootstrap --yes`, then `bd dolt status` in the main checkout.
  Code checkout alone has no task store. Bootstrap can read published team history and persist `sync.remote`.
  Use `bd ready`, `bd create`, `bd update <id> --claim` and `bd close <id>` for ordinary task work.
  For first publication, review the intended `origin` with `git remote get-url origin`, publish the source branch,
  and review a clean index/worktree with `git status` before `bd dolt push --yes`. This consents to publishing the
  complete task history, persists `sync.remote` and creates a source commit under your Git identity. Inspect
  `git diff --check` and `git log -1 --oneline`, then `git push origin <branch>` to share that configuration.
  Later sync uses `bd dolt pull` and `bd dolt push`. Linked worktrees share the main checkout's store without
  `BEADS_DIR`. Follow `cook help io.github.lmarqs.cookbook:beads#tracker` for setup warnings and mode changes.
<!-- cook:end io.github.lmarqs.cookbook:beads#tracker/tracker -->
<!-- cook:begin io.github.lmarqs.cookbook:beads#viewer/viewer -->
<!-- io.github.lmarqs.cookbook:beads#viewer --instructions=.agents/skills/beads-ralph/SKILL.md -->
Use Beads Viewer (`bv` 0.25.2) with `bd` 1.3.1 for graph inspection and recommendations. An initialized native
Beads store is required. Install and bootstrap by the recipe's After cooking instructions before using Viewer.
Exports are compatibility snapshots; the native tracker remains authoritative.

Before triage, select one store and refresh successfully. For an external store, supply an absolute native
`BEADS_DIR` in this shell. Otherwise resolve the main checkout's store through Git's common directory, including
from a linked worktree:

```sh
beads_dir="${BEADS_DIR:-$(dirname -- "$(git rev-parse --path-format=absolute --git-common-dir)")/.beads}"
bd --db "$beads_dir" export -o "$beads_dir/issues.jsonl" &&
  bv --db "$beads_dir" --robot-triage
```

Explicit Viewer `--db` overrides `BEADS_DB`, which otherwise overrides `BEADS_DIR`. A team worktree's local
`.beads/config.yaml` can shadow automatic main-store discovery; pass the same selected directory to both tools.
External exports stay in that store. This repository's ignore rules do not control the external repository's
staging or sync policy. Review its policy separately.

Keep export failures and Viewer diagnostics visible. A failed automatic refresh with an existing export can exit 0
while reporting `source_authority.state=partial`, `claim_safe=false`, `readiness=provisional`, source `stale=true`
and export warnings. Missing export plus failed refresh exits 1. Require successful export before presenting current
recommendations, inspect source authority and warnings, and retain stale output only as diagnostic information.

Fresh local bootstrap can omit native `metadata.json`: Viewer can rank the snapshot but returns
`diagnostic_top_pick` and a message instead of an actionable top-level ID. Ralph 0.12.0 `beads-bv` then falls back
to `bd ready`. Native remote bootstrap can create metadata; do not fabricate it or promise graph selection when
the response cannot support it.

Agents use robot commands such as `--robot-triage`; people can use `bv --db "$beads_dir"` for the interactive TUI
after a successful export. Selection never claims or reserves a task. Check current tracker state and use
`bd --db "$beads_dir" update <id> --claim` to claim; use `bd` for ordinary task creation, updates and closure.
Worker restrictions in ralph prompts apply to that runner's workers.
<!-- cook:end io.github.lmarqs.cookbook:beads#viewer/viewer -->
<!-- cook:begin io.github.lmarqs.cookbook:ralph#ralph-tui/ralph-tui -->
<!-- io.github.lmarqs.cookbook:ralph#ralph-tui --harness=claude --harness=codex --worker-role=claude:worker --worker-settings=codex:.agents/adapters/codex/worker.toml --worktree-setup=.worktrees/setup --worktree-teardown=.worktrees/teardown --worker-check=scripts/worker-check --tracker=beads-bv --tracker=json --instructions=.agents/skills/beads-ralph/SKILL.md -->
- Loop runs: `.ralph-tui/bin/run <name> --agent <entry> --epic <id>`, or `--prd <file>`, runs ralph-tui in
  `.worktrees/ralph/<name>` on branch `ralph/<name>`; `.ralph-tui/bin/clean <name>` removes both once the
  branch is delivered.
<!-- cook:end io.github.lmarqs.cookbook:ralph#ralph-tui/ralph-tui -->
