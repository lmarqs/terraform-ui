# Shared subagents

A subagent is a named helper with its own instructions, which an agent delegates a task to. Each one is written once,
in `.agents/agents/<name>.md`. Cooking `io.github.lmarqs.cookbook:agents#connect` renders it into the format of each
harness it connects, under `.agents/adapters/`, and links each harness's agent directory to the renders. Written for
Claude Code 2.1.304 and Codex 0.160.1.

## The file

```markdown
---
name: bug-hunter
description: Reviews a staged diff for correctness bugs.
claude:
  tools: [Read, Grep, Glob]
  model: opus
codex:
  model: gpt-5.5
  model_reasoning_effort: high
---
You review diffs. Report only bugs you can show with a failing input.
```

| Key | Required | Holds |
| --- | --- | --- |
| `name` | Yes | The name each harness calls the subagent by; the file is named `<name>.md` |
| `description` | Yes | When to use the subagent; each harness reads it to decide when to delegate |
| `claude` | No | Keys Claude Code takes as given, from its subagent front matter, such as `tools`, `model` or `effort` |
| `codex` | No | Keys Codex takes as given, from its custom agent file, such as `model`, `model_reasoning_effort` or `skills` |

The text after the front matter is the prompt, and both harnesses get the same one. A prompt that has to differ by
harness belongs in two subagents, each with the block of one harness.

Each file sits directly under `.agents/agents/`, with a Markdown extension. Other files there, such as `.gitkeep`,
are not subagents.

## What each harness gets

| Harness | File | Holds | Link |
| --- | --- | --- | --- |
| Claude | `.agents/adapters/claude/agents/<name>.md` | Front matter with `name`, `description` and the `claude` keys, then the prompt | `.claude/agents` |
| Codex | `.agents/adapters/codex/agents/<name>.toml` | `name`, `description`, the prompt as `developer_instructions`, and the `codex` keys | `.codex/agents` |

A `model` in a harness block applies only while that subagent runs; the run's own model stays the person's or the
loop runner's choice. A harness block's values reach that harness unchanged, and the other harness never sees them: model names, tool names
and every other value are the harness's own. A key the harness does not know is the harness's to report, as Claude
Code does in its `--debug` log.

`#connect` renders only for the harnesses it is cooked with. Deleting a subagent's file and cooking again removes its
renders, and the links once no subagent is left.

## Refusals

Cooking stops, names the file, and writes nothing, when a subagent file:

- has a top-level key other than `name`, `description`, `claude` and `codex`;
- lacks `name` or `description`;
- has a `claude` or `codex` value that is not a map;
- is not named after its `name`;
- has no front matter between two `---` lines.

A directory under `.agents/agents/` is refused too.

## Where each harness looks

Claude Code reads subagents from `.claude/agents/` and ignores a front matter key it does not know, as its
[subagent documentation](https://code.claude.com/docs/en/sub-agents) says. Codex reads custom agents from `*.toml`
files under `.codex/agents/`, each needing `name`, `description` and `developer_instructions`, and loads a project's
only once the project is trusted ([codex.md](codex.md#trust)). Both were tested by asking each harness to list its
subagents.
