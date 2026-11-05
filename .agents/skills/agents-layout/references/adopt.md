# Adopt the layout

One procedure sets the layout up and ends in one reviewed commit. Seed and connect never go in separate commits: a
seeded layout that nothing connects tells agents to edit a policy no harness reads.

Run every cook twice: first with `--dry-run`, read the diff it prints, then the same command with `--batch-approve`.
An agent has no terminal to answer Cook's prompt, so without `--batch-approve` Cook writes nothing and says
`approval required; nothing was written`.

## A new repository

1. **Find Cook.** Use the first that prints a version: `./cook --version`, then `mise exec -- cook --version` when
   `mise.toml` pins the-cook, then `cook --version`. If none does, stop and ask the developer how Cook is installed
   here. The commands below write it as `./cook`.
2. **Find the version.** `grep 'source: "io.github.lmarqs.cookbook:agents:' cook.lock` shows it when the repository
   already cooks the artifact. Otherwise use the latest release:
   `gh release view -R lmarqs/the-cook --json tagName -q .tagName`, without its `v`. The commands below write it as
   `<version>`.
3. **Confirm two decisions with the developer.** Which harnesses the team uses: `#connect` removes what it wrote for
   a harness left out, so each one goes in every connect as `--harness=<name>`; the commands below name Claude and
   Codex. And `bypass` in the policy: the starter's `refuse` makes Claude refuse its bypass flags, which some loop
   runners pass by default ([policy.md](policy.md#top-level)).
4. **Seed.**

   ```sh
   ./cook --dry-run io.github.lmarqs.cookbook:agents:<version>#seed
   ./cook --batch-approve io.github.lmarqs.cookbook:agents:<version>#seed
   ```

   It writes `AGENTS.md`, `.agents/README.md`, `.agents/permission/policy.yaml`, the shared directories and this
   skill, in `.agents/skills/agents-layout/`. When `AGENTS.md` already exists, Cook refuses; follow
   [An existing AGENTS.md](#an-existing-agentsmd).
5. **Fill in `AGENTS.md`** as the note at its top says, or merge the old file as
   [An existing AGENTS.md](#an-existing-agentsmd) says.
6. **Write the policy**: what agents may run here. The starter grants nothing and ends with a commented example;
   [policy.md](policy.md) has the schema.
7. **Connect.**

   ```sh
   ./cook --dry-run io.github.lmarqs.cookbook:agents:<version>#connect --harness=claude --harness=codex
   ./cook --batch-approve io.github.lmarqs.cookbook:agents:<version>#connect --harness=claude --harness=codex
   ```

8. **Review.** Stage first as step 9 does: `check.sh` reads the links and the size of `AGENTS.md` from the index, so
   before staging it fails every link. Then run the review in [SKILL.md](../SKILL.md#review) and fix what it finds.
9. **Commit once.** Stage every path `git status --short` lists that the adoption changed: the layout's paths below,
   and what the sections under [An existing setup](#an-existing-setup) moved, removed or rewired.

   ```sh
   git status --short
   git add AGENTS.md .agents .claude .codex .gitignore cook.lock
   git commit
   ```

   The links in `.claude/` and `.codex/` are committed, so a clone, a worktree, a cloud session or a CI job reads the
   layout with no cooking and no Cook installed. The `harnesses` section of `.gitignore` keeps everything else a
   harness writes there, such as `.claude/settings.local.json` or `.claude/worktrees/`, out of commits. Commit no
   real file in either directory: a file the team shares goes under `.agents/`, which the links reach, and a file
   for one person stays on their machine. [design.md](design.md#links-and-adapters-are-committed) gives the cost.

   After the commit, `git ls-files .claude .codex` lists only links, and `git ls-files -s .claude .codex` shows each
   with mode `120000`. `git check-ignore --no-index $(find .claude .codex -maxdepth 1 -type l)` prints nothing; a link
   it prints is hidden by an ignore rule, which [investigate.md](investigate.md) shows how to find.

Before and after:

```text
before                     after
  README.md                  README.md
  src/                       src/
  cook.toml                  cook.toml
                             cook.lock
                             AGENTS.md
                             .gitignore                    with the harnesses section
                             .agents/README.md
                             .agents/permission/policy.yaml
                             .agents/adapters/claude/settings.json
                             .agents/adapters/codex/config.toml
                             .agents/skills/agents-layout/
                             .agents/commands/  hooks/  memory/
                             .claude/settings.json -> ../.agents/adapters/claude/settings.json
                             .claude/skills -> ../.agents/skills
                             .claude/commands -> ../.agents/commands
                             .codex/config.toml -> ../.agents/adapters/codex/config.toml
```

`.codex/rules` and `.agents/adapters/codex/rules/cook.rules` appear once the policy has a command entry.

### A drift check in CI

A CI step fails when someone edits the policy and commits without cooking. In a repository that commits the `./cook`
launcher:

```yaml
- name: The agents layout matches its policy
  run: ./cook check io.github.lmarqs.cookbook:agents#connect
```

`check` cooks `#connect` again with the version and arguments `cook.lock` records, so the step stays as it is when
the recipe is upgraded or cooked for another harness.

## An existing setup

Each section below starts from one kind of existing setup and says which step of the procedure it changes. Read the
ones that match; most repositories match more than one. Everything still ends in the one commit of step 9.

### An existing AGENTS.md

This replaces steps 4 and 5.

1. Seed with `--skip` before the recipe, which keeps the old `AGENTS.md` and writes the rest:

   ```sh
   ./cook --dry-run --skip io.github.lmarqs.cookbook:agents:<version>#seed
   ./cook --batch-approve --skip io.github.lmarqs.cookbook:agents:<version>#seed
   ```

2. Read the template, `.agents/skills/agents-layout/templates/AGENTS.template.md`, and number the old file's lines:

   ```sh
   git show HEAD:AGENTS.md | grep -n .
   ```

3. Write the new `AGENTS.md` from the template, carrying each old line into it or elsewhere.
4. Write a line map into the commit message or the pull request, one row per numbered line:

   ```text
   old line | kept / moved / enforced / in README / dropped | where or why
   ```

   `moved` names the skill or memory file, `enforced` the policy entry or check, and `in README` the section. A
   `dropped` line needs the developer's approval before the commit.

### A root CLAUDE.md

Claude skips `AGENTS.md` while a `CLAUDE.md`, `.claude/CLAUDE.md` or `CLAUDE.local.md` exists in the directory or one
above, so the file has to go, not just stop being committed. This belongs to step 5.

1. List every such file the repository tracks:

   ```sh
   git ls-files 'CLAUDE.md' '*/CLAUDE.md' 'CLAUDE.local.md' '*/CLAUDE.local.md'
   ```

2. Move each line every agent needs into `AGENTS.md`. Drop `@AGENTS.md` imports and lines the README already says.
   Lines that are one person's preference leave the repository; hand them back to their author.
3. Remove the files from git and from the disk. `.gitignore` does not untrack a committed file:

   ```sh
   git rm CLAUDE.md
   ```

Before:

```markdown
<!-- CLAUDE.md -->
@AGENTS.md

Use pnpm, never npm.
Run `pnpm test` before every commit.
I prefer short answers.
```

After: the first two rules are in `AGENTS.md`, the last line is back with its author, and `CLAUDE.md` is gone.

### Claude path-scoped rules in .claude/rules/

Codex does not read `.claude/rules/`, and the layout keeps `.claude/` to links, so each rule moves to a place every
agent reads. This belongs to step 5.

1. For a short rule, add a section to `AGENTS.md` that names the paths it covers.
2. For a long one, or one that matters only for some tasks, make it a skill: `.agents/skills/<name>/SKILL.md`, with a
   `description` saying when to use it. Both harnesses load a skill when its description matches the task.
3. `git rm -r .claude/rules`.

Before:

```markdown
<!-- .claude/rules/migrations.md -->
---
paths: ["src/migrations/**"]
---
Never edit a migration that has been merged; add a new one.
```

After, in `AGENTS.md`:

```markdown
## Migrations (`src/migrations/`)

Never edit a migration that has been merged; add a new one.
```

### Bash rules in a committed .claude/settings.json

This belongs to step 6, and the file has to go before step 7: `#connect` refuses to put a link where a file is.

1. Translate each `permissions` rule into the policy:

   | In `.claude/settings.json` | In `policy.yaml` |
   | --- | --- |
   | `Bash(git status:*)`, the legacy prefix form | `[git, status]` under the same decision |
   | `Bash(npm run test *)` | `[npm, run, test]` |
   | `Bash(git push origin main)`, no wildcard | `[git, push, origin, main]`, which also matches later arguments, or `claude: "git push origin main"` to stay exact |
   | `Bash(git push * --force*)`, a wildcard in the middle | `claude: "git push * --force*"`; Codex does not enforce it |
   | `WebFetch(domain:docs.gradle.org)`, `mcp__github__get_*` | `claude: "<rule>"` under `tools` |

2. Move the file to the adapter, which keeps the keys the team shares, such as `hooks` or `env`; `#connect` makes
   `.claude/settings.json` a link to it and replaces `permissions` from the policy:

   ```sh
   mkdir -p .agents/adapters/claude
   git mv .claude/settings.json .agents/adapters/claude/settings.json
   ```

3. Delete `model` and `defaultMode` from it: they are each person's choice and leave the repository.
4. After step 7, compare `.agents/adapters/claude/settings.json` with `git show HEAD:.claude/settings.json`: every
   rule is there, `Bash(<words> *)` in place of `Bash(<words>:*)`.

Before:

```json
{
  "model": "opus",
  "permissions": {
    "allow": ["Bash(git status:*)", "Bash(gh pr view:*)", "Bash(gh pr list:*)", "mcp__github__get_issue"],
    "deny": ["Bash(git push --force:*)"]
  }
}
```

After:

```yaml
version: 2
commands:
  allow:
    - [git, status]
    - [gh, pr, [view, list]]
  deny:
    - [git, push, --force]
tools:
  allow:
    - claude: "mcp__github__get_issue"
```

### A hand-made policy with its own renderer

Some repositories already keep a policy file and a script that writes both harnesses' settings from it. Version 2
holds the same ideas, so the move is mechanical. This belongs to step 6.

1. Map the groups:

   | Hand-made | Version 2 |
   | --- | --- |
   | A group for every session, such as `shared: {allow, ask, deny}` | `commands: {allow, ask, deny}` |
   | A group for one kind of run, such as `worker: {deny}` | `roles: {worker: {deny}}` |
   | A quoted string read as a raw Claude pattern, such as `"mise * ::: *"` | `claude: "mise * ::: *"` |
   | A list of MCP or other tool patterns appended to Claude's allows | `claude:` entries under `tools.allow` |
   | Allows for one kind of run only | `commands` for everyone, after the team agrees; a role only denies |

2. Codex gets no role. A renderer that wrote worker denies into Codex's rules because Codex ran only as a worker now
   has two choices: add those entries to `commands.deny`, which also binds people at the keyboard, or keep Codex
   workers on `commands` alone and rely on its sandbox. [codex.md](codex.md#no-roles) explains why.
3. Find the code, tests, docs and CI steps that run the renderer or name its outputs:

   ```sh
   git grep -n -e render.py -e 'import render' -e 'from render'
   git grep -n -e workflow.rules -e .claude/settings.json
   ```

4. Move the hooks it wrote into the `hooks` key of `.agents/adapters/claude/settings.json` and into
   `.agents/adapters/codex/hooks.json`, with the scripts in `.agents/hooks/`. A session-start hook that prints context
   becomes a script in `.agents/hooks/session-start.d/`, since `#connect` owns `hooks.SessionStart`.
5. Before step 7, delete the renderer, its tests and its outputs, and rewire what step 3 found that still reads
   them: a CI drift check becomes [the `check` step](#a-drift-check-in-ci), and a script that names an output names
   the adapter or link that replaces it. `-f` replaces the starter policy the seed wrote:

   ```sh
   git rm .agents/permissions/render.py .claude/settings.json .codex/rules/workflow.rules
   git mv -f .agents/permissions/policy.yaml .agents/permission/policy.yaml
   ```

   The names are an example; use the repository's own.

Before:

```yaml
shared:
  ask:
    - "gh pr merge *--admin*"
  deny:
    - [git, push, ["--force", "-f"], origin, main]
    - "git push origin main --force*"
  read_mcp:
    - "mcp__plugin_datadog_mcp__search_*"
worker:
  deny:
    - [git, [commit, push, stash]]
    - [gh]
```

After:

```yaml
version: 2
commands:
  ask:
    - claude: "gh pr merge *--admin*"
  deny:
    - [git, push, [--force, -f], origin, main]
    - claude: "git push origin main --force*"
tools:
  allow:
    - claude: "mcp__plugin_datadog_mcp__search_*"
roles:
  worker:
    deny:
      - [git, [commit, push, stash]]
      - [gh]
```

Keep the comments that explain why an entry is there; YAML comments survive the move and every reader needs them.

### A version 1 .agents/policy.yaml

An earlier release of the agents artifact used `io.github.lmarqs:agents` coordinates, a version 1 policy at
`.agents/policy.yaml` and a `CLAUDE.md` adapter. Cook refuses that policy with `version must be 2`. This comes before
step 4.

1. Remove what the earlier release wrote. List its recipes, then discard each by the name `cook.lock` records:

   ```sh
   grep 'name:' cook.lock
   ./cook discard --dry-run io.github.lmarqs:agents#connect io.github.lmarqs:agents#instructions
   ./cook discard --batch-approve io.github.lmarqs:agents#connect io.github.lmarqs:agents#instructions
   ```

   Discard reads only `cook.lock`, so it works although those coordinates no longer resolve, and it ignores the
   version. It keeps `AGENTS.md` and the policy, which a seed wrote and nobody owns.
2. Move the policy and set its version:

   ```sh
   mkdir -p .agents/permission
   git mv -f .agents/policy.yaml .agents/permission/policy.yaml
   ```

   Change `version: 1` to `version: 2`. Move each line of `instructions` into `AGENTS.md`, then delete the field.
   `commands` keeps its shape; nested lists can now merge entries that differ in one word.
3. If a `CLAUDE.md` or `.claude/CLAUDE.md` remains, follow [A root CLAUDE.md](#a-root-claudemd).

Before:

```yaml
version: 1
instructions:
  - Run ./gradlew check before committing.
commands:
  allow:
    - [git, status]
    - [git, diff]
```

After, with the instruction line in `AGENTS.md`:

```yaml
version: 2
commands:
  allow:
    - [git, [status, diff]]
```

### Files beside the shared directories

Harnesses reach `.agents/` through the shared directories and the adapters only, so a file or directory the team
keeps directly under `.agents/` is found only through a path someone wrote down. `check.sh` warns on each. This comes
before step 8.

1. List the entries that are not the layout's:

   ```sh
   git ls-files -- .agents | sed -n 's|^\.agents/\([^/]*\).*|\1|p' | sort -u |
     grep -vxE 'README\.md|permission|adapters|skills|commands|hooks|memory'
   ```

2. Find what names each one: `git grep -n -e <entry>`.
3. Move it beside what uses it: a file a hook script loads into `hooks/`, a file one skill reads into that skill's
   directory, a durable fact into `memory/`, permission rules into `permission/policy.yaml`. A file no agent reads, such as a loop runner's configuration, leaves
   `.agents/`. Then update every path step 2 found, and the relative paths inside the moved file.

## Risks a review catches

Mechanical moves do not cover every existing setup. These fail loudly; each has a check to run before step 7.

- **A file or directory where `#connect` makes a link.** Cook refuses, saying, for example, that
  `.codex/config.toml is a file, and this recipe makes it a link to ../.agents/adapters/codex/config.toml`, and writes
  nothing. Check: `find .claude .codex -mindepth 1 -maxdepth 1 ! -type l` lists what is not a link; a
  `settings.json`, `skills`, `commands`, `config.toml`, `rules` or `hooks.json` among them stops the cook that links
  it. Move a file to the adapter of the same name, as `git mv .codex/config.toml .agents/adapters/codex/config.toml`
  does, a directory's files into `.agents/`, and cook again.
- **An ignore rule that hides a link.** Cook's `harnesses` section re-includes each harness directory, but a rule after
  it or in a global excludes file still wins. Check: the `git check-ignore --no-index` line of step 9;
  [investigate.md](investigate.md) has the row that finds the rule.
- **A key Cook owns written as a TOML table.** Cook refuses a key it owns whole when the file writes it as a `[k]`
  table, a `[k.sub]` table, a `[[k]]` array of tables or dotted keys `k.x`, saying the key `is written as` that
  shape, and writes nothing. Check: `grep -n '^[[:space:]]*\[' .agents/adapters/codex/config.toml` lists the tables.
  Rewrite the key inline, as the message says, or remove it and let the cook write it.
