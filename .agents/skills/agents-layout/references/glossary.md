# Glossary

Each term of the layout has one meaning, which every file of this skill uses. Where a harness uses another word for
the same thing, the term lists it under Avoid.

| Term | Meaning | Avoid |
| --- | --- | --- |
| Harness | A program that runs an agent, such as Claude Code or Codex | AI Agent, tool |
| Run | One start of a harness, which fixes its mode, role and model | Session, except in a harness's own names such as `SessionStart` |
| Agent | Whatever acts within a run; a word for prose, never a setting | |
| Mode | Whether someone answers the run's questions: attended or unattended | |
| Role | A named narrowing of the policy for one kind of run: `roles.<role>.deny` in `policy.yaml`, which only adds denies | Profile |
| Worker | A run started by a loop runner, usually unattended | |
| Subagent | A helper with its own prompt that an agent delegates a task to, written once in `.agents/agents/<name>.md` | Custom agent, agent role, agent definition |
| Policy | `.agents/permission/policy.yaml`: what agents may run without asking, what asks, and what never runs | Permissions file |
| Shared directory | A directory under `.agents/` that any harness can use: `skills/`, `hooks/`, `memory/` or `agents/` | |
| Adapter | `.agents/adapters/<harness>/`: one harness's own files, committed | |
| Harness directory | `.claude/` or `.codex/`, which a harness reads at a fixed path | |
| Render | A file or key Cook writes into an adapter from a source in `.agents/`, which the next cook rewrites | Generated file |
| Link | A committed symbolic link in a harness directory that points into a shared directory or an adapter | |
| Personal file | Any file in a harness directory that is not a link; git ignores it | Local file |

Cook's own terms, such as owner, recipe and cooking, are in its glossary,
`docs/features/glossary.md` in the-cook's repository.
