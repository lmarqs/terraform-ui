# Why `AGENTS.md` holds what it holds

Read this when filling in `AGENTS.md` from the template `#seed` writes,
[templates/AGENTS.template.md](../templates/AGENTS.template.md), or when deciding what to add to it or cut.
Every agent session reads `AGENTS.md`, so each line costs every session: the template holds what the evidence says
agents use from such a file, and nothing the repository already shows. The reasons below name their sources; the full
list is at the end. Tested with Claude Code 2.1.304.

## What the evidence says

- Agents follow the instructions in a context file: a tool the file names is used far more often than one it does
  not. Instructions are the content that works ([1], §4.3).
- A repository overview does not help an agent reach the right files sooner, and LLM-written files mostly repeat the
  README. A context file should hold only what the README lacks ([1], §1, §4.3).
- A context file raises an agent's cost by about a fifth, because the agent runs the checks and tools it names, not
  because the file is long. Length shows no correlation with success or cost ([1], §4.2, appendix B).
- Real files have a median of 335 to 535 words and six or seven top-level sections, grow by small additions and are
  almost never pruned. Testing is their most common topic, present in three files out of four; security appears in
  one out of seven ([2], tables 3 and 5).
- Files that work are specific: exact commands with their flags, examples rather than descriptions, a stack with
  versions, and boundaries in three tiers: always, ask before acting, and never. The constraint teams most often
  wish they had written is that the agent never deletes a failing test ([3]).
- A long file makes the agent ignore its rules. Each line stays only while removing it would cause mistakes; what the
  agent can read from the code, standard conventions and "write clean code" do not belong ([4]).
- The nearest `AGENTS.md` to the edited file wins, so rules for one package go in a file in that package; Codex stops
  reading the chain at 32 KiB ([5], [6]).

## What follows for the template

- **Budgets as numbers.** Real files only grow ([2]), and a long file makes the agent ignore its rules ([4]), so
  Boundaries states the limits. 800 words is one and a half times the top of the 335 to 535-word median ([2]), room
  for a repository's own commands and rules. Past 1,200 words, a section moves into a skill, which loads only when its
  description matches the task, or into a memory file that Deep dive names. All `AGENTS.md` files together stay under
  16 KiB, half of the 32 KiB at which Codex stops reading the chain ([5], [6]); at about seven bytes a word, a root
  file at 1,200 words leaves half of that for files nested in packages. The numbers are this layout's choices within
  those sources, not findings of their own.
- **The template's own text is small.** Its fixed text, outside the blanks and the note, is 300 words or fewer, so a
  file within 800 words gives most of them to the repository.
- **Commands in one place.** Quick reference holds every command, including installing and adding a dependency, and
  defines the Quick check and the Done check. Agents use the tools a file names ([1], §4.3), and files that work give
  exact commands with their flags ([3]), so each command is written once, where an agent looks for it; the other
  sections state rules and places, and none repeats a command.
- **Rules are written out; overviews are not.** Boundaries and Rules state what holds in any repository, such as never
  weakening a check to make it pass ([3]) and comments only for what the code cannot show. Boundaries has the three
  tiers: Always, Stop and ask, and Never ([3]). Stop and ask covers decisions an agent leaves to the developer; a
  command that needs approval every time is an `ask` entry in the policy, which the harness enforces. The stack is
  one line with major versions ([3]). Tests get a place and a rule, because testing is the most common topic in real
  files ([2]).
- **One marker in Quick reference.** A command its writer could not run is marked `(unverified)`, and the next agent
  to run it removes the mark, so a command is either verified here or flagged as not yet.
- **The README is linked, not restated.** An overview does not help an agent reach the right files, and LLM-written
  files mostly repeat the README ([1], §1, §4.3). The note asks for a link to the README section, and a fix to the
  README in the same change when it is wrong, so the fact lives in one place.
- **Gotchas is capped at five.** Every session reads `AGENTS.md`, so it keeps only the traps any task can hit. A trap
  in one area goes to `.agents/memory/`, named under Deep dive, and costs nothing until a task touches that area.
- **Worktrees stay explicit**, with a line for creating one ready to run and removing one with its data, because
  removal is the step agents skip. The line is the project's, whatever creates the worktrees: a person, a script or
  any loop runner. A worktree inside the repository reads its own `AGENTS.md`, not the main checkout's, but a
  `CLAUDE.local.md` in a directory above it still turns Claude's reading of `AGENTS.md` off (tested).
- **The note speaks only when asked.** The file is loaded every session, so a note addressed to every session
  interrupts every task until someone fills the file in, and an instruction to fill it in would make the first agent
  rewrite it during an unrelated task. The note says the file is unfilled only when the developer asks for other work
  in an interactive session, and is deleted once `scripts/check.sh` reports nothing else.
- **One kind of blank.** Every blank is `[[fill: …]]`, on its own line, so
  `grep -n '\[\[fill:' AGENTS.md` finds them all; a single bracket collides with real content such as `[-D]`, an empty
  cell is a blank no search finds, and GitHub renders `<angle brackets>` as HTML tags and hides them. No sample values
  appear, because agents copy them into repositories where they are wrong.

Not established: whether a repository's filled template improves what agents produce there. One benchmark on one
repository ran four fixed tasks with its old `AGENTS.md`, with the filled template and with no file, once per task,
file and harness: 24 runs, one per cell, in Claude Code 2.1.304 and Codex CLI 0.160.1. The old and the filled file
gave the same results: each run ran the check its file names and kept every rule that applied. Without a file, both
harnesses skipped the repository's Python check, one run wrote its session note into a versioned memory file, and the
runs used about 40% more tokens. One run per cell does not separate the old file from the filled one, and the gap to
no file is observed, not established. Establishing it needs the setup in [1], a fixed task set across several agents
and models with repeated runs. The check within reach is narrower: a fresh session sets up, makes a small change and
passes the Done check from `AGENTS.md` alone.

## Sources

1. Gloaguen et al., *Evaluating AGENTS.md: Are Repository-Level Context Files Helpful for Coding Agents?*
   (2026), <https://arxiv.org/abs/2602.11988>. Controlled study on SWE-bench and CTXbench.
2. *Agent READMEs: An Empirical Study of Context Files for Agentic Coding* (2025),
   <https://arxiv.org/abs/2511.12884>. 2,303 context files from 1,925 repositories.
3. GitHub, *How to write a great agents.md: lessons from over 2,500 repositories*,
   <https://github.blog/ai-and-ml/github-copilot/how-to-write-a-great-agents-md-lessons-from-over-2500-repositories/>.
4. Anthropic, *Claude Code best practices*, section "Write an effective CLAUDE.md",
   <https://code.claude.com/docs/en/best-practices>.
5. OpenAI, *Custom instructions with AGENTS.md*, <https://learn.chatgpt.com/docs/agent-configuration/agents-md>.
6. agents.md, <https://agents.md/>.
