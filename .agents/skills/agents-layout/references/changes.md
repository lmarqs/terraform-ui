# Where a change goes

Which file a change to the layout is made in, and what follows it.

| To | Edit | Then |
| --- | --- | --- |
| Tell every agent something | `AGENTS.md` | Commit |
| Allow, ask before or deny a command in every session | `commands` in `.agents/permission/policy.yaml` | Cook `#connect`, review, commit |
| Deny a command in one kind of run, such as unattended workers | `roles.<role>.deny` in the same file | Cook `#connect`, review, commit |
| Share a skill, hook script or memory file | `.agents/skills/`, `hooks/` or `memory/` | Commit; the links expose it |
| Share a subagent | `.agents/agents/<name>.md` | Cook `#connect`, review, commit |
| Give every agent context as its session starts | A script in `.agents/hooks/session-start.d/` that prints it, named with a number first to set its order, such as `10-status.sh`; the harness name is `$1` | Commit; Codex runs it once the hooks are trusted in the main checkout, its worktrees included, [codex.md](codex.md#trust) |
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
