---
name: docs-layout
description: Teaches where a fact goes in docs/, the source of truth people and agents share, and why. Writes the map, docs/README.md, with the team, then writes and reviews pages by it. Use it to record a decision, describe a feature, interface, standard, telemetry, runbook, architecture or tenets, add a glossary term, place a fact, review docs/, or adopt an existing docs tree.
argument-hint: "[map | adopt | review [--fix] | <kind> [title] | term <term>]"
---

# docs-layout

Each fact reads best on one page, its owner; other pages link it. The map, `docs/README.md`, says which kinds of page
this repository keeps and what a good page of each achieves. [references/map.md](references/map.md) teaches where a
fact goes and why.

## Three rules

**The map is this repository's, and where it and this skill differ, the map wins.** A kind's existing pages show its
shape here. A reference's example is one good page in a made-up project, not the page to copy.

**A page never holds a placeholder.** Ask the person for what the sources do not give. What stays unknown is left out
and reported as an open question; a blank in a committed page stays there.

**An existing page is edited for its facts.** Fix what is wrong, missing or stated twice. Change its structure only
when its shape fails a reader, and say how.

## Invoked with arguments

| Arguments | Do |
| --- | --- |
| `map` | [Write the map](#map), or add or remove a kind on it |
| `<kind> [title]` | [Write a page](#write-a-page) of that kind |
| `term <term>` | Add the entry to `docs/glossary.md` in its existing form, or as [references/glossary.md](references/glossary.md) shows, link a new glossary from the map, and replace its rejected synonyms where they name this concept |
| `review` | [Review](#review) and report; files stay as they are |
| `review --fix` | Review, ask the person about each difference that needs their decision, and fix what they agree to |
| `adopt` | Follow [references/adopt.md](references/adopt.md) |
| None | Pick what the request fits; with no map, start with `map` |

## Map

When `docs/README.md` is absent or names no kind, ask the person what the repository holds that its code and
history show nowhere else, and what went wrong for want of it: a decision argued again, behavior nobody could state,
a procedure found by trial. Pick the kinds those facts need: [adr](references/adr.md),
[features](references/features.md), [api](references/api.md), [standards](references/standards.md),
[observability](references/observability.md), [runbooks](references/runbooks.md),
[architecture](references/architecture.md), [philosophy](references/philosophy.md). Fewer kinds read better; one
with no facts yet waits. Write the map by [references/map.md](references/map.md), in the team's language, with a
worked example from this repository. On a map with kinds, add or remove one in its own words.

## Write a page

1. Search `docs/` for the subject, by its term and synonyms. When a page owns it, edit that page.
2. Place a new page by the map, by the question its facts answer and never by their subject.
   When **no kind fits**, say so and offer: outside `docs/`, a kind the team declares, or nowhere. A forced kind
   misleads every reader who trusts the map.
3. Gather facts from the code, the contract, the history, the issue and the person. Ask first what went wrong, what
   it cost and what was tried: only the person may know it.
4. Write the body, then the front matter. Take the shape from the kind's existing pages, else from the map and the
   reference.
5. When the kind keeps a directory, list the page in its guide, `docs/<kind>/README.md`, created with the first page.
   Link a new guide or one-page kind from the map, and replace other pages' copies of its facts with links.
6. Report the files written and the open questions.

## Review

Compare each page with the map, and report where it differs from these signs:

- It sits in a declared kind, or is the glossary, and is reached from the map; its links resolve.
- It achieves what the map describes for its kind, and its kind was not forced: no part had to be invented to fit.
- Each fact is stated on its owner, in the glossary's terms.
- Its front matter tells a reader from a listing whether to open it, and the body bears it out.
- Each reason tells what happened first, and the claims hold against the code.
- It holds no placeholder.
- Its shape did not change in a commit that changed no fact.

Give each difference with the description it differs from and the step that would close it. A page that serves its
readers in a shape of its own is not a difference. The team decides.
