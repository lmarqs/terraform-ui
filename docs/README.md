# docs/

The documentation of tfui that its code, its issues and its history do not hold. Every page except this map is
published to [the site](https://lmarqs.github.io/terraform-ui) by `.github/workflows/pages.yaml`; a page's directory is
part of its URL, so a page stays where it is and this map says what each directory is for. The sidebar is
`_data/nav.yml`; how the site is built is in [Development](development/index.md).

Plans and their state live in GitHub issues and the project board ([backlog](development/backlog.md)); the glossary
is [`CONTEXT.md`](../CONTEXT.md) at the repository root.

## Kinds of page

### Decisions: `adr/`

A record of one choice that shaped tfui and is costly to reverse: what forced it, the options weighed and what it
costs. A fact belongs here when someone would otherwise argue the choice again. A good record lets a reader who
wants to change the choice see what broke before it and what the change would bring back. Records keep their
numbers. [The list](adr/index.md).

### Features: `features/` and `plugins/`

What tfui does for its user, and why it does it so. `plugins/` holds one page per plugin, its keys, views, flags and
settings, with `id`, `key` and `category` in front matter for [the catalog](plugins/index.md); `features/` holds the
analyses that span plugins or need a page of reasoning, such as how risk is scored. A fact belongs here when a user
relies on it and the code is the only other place it shows. A good page states the rules, thresholds and limits a
user meets, in the order the user meets them, with a worked example where a rule is arithmetic. A rule one plugin
owns is stated on its plugin page and linked from the feature page.

### Interfaces: `reference/cli-reference.md`, `reference/cli-io-contract.md`, `reference/macro-language.md`

What a caller of tfui depends on: every command and flag, what goes to stdout and stderr in each mode with the exit
codes, and the tape language macros are written in. A fact belongs here when a script, an agent or a tape breaks if
it changes. A good page is complete for its interface and says, for each item, what it accepts and what it returns.

### Standards: `reference/cli-ux.md`, `reference/tui-ux.md`, `development/testing.md`

The rules every new command, view or test follows, where no check enforces them. A fact belongs here when it
constrains something a contributor writes, wherever it is written: how a key is bound, where a confirmation lives,
which level a test runs at. A good standard gives each rule its reason, so a case its words leave open can be decided
by its intent. The `tfui-cli-ux`, `tfui-tui-ux` and `tfui-conventions` skills summarize these pages for agents and
link them; the page owns the rule.

### Architecture: `development/architecture.md`

How the parts fit together: the SDK, the app internals, the plugins, the event bus and the runtimes, with the
directory each lives in. A fact belongs here when it is about the whole and no single file shows it. A good page
lets a reader go from a box on the diagram to the code, and links the decision behind each shape.

### Guides: `guides/`

Declared by this team. Step by step from nothing to a working setup: installing, configuring `tfui.hcl`, scaffolding
a project. A fact belongs here when a new user needs it in that order to get started; what a setting does is stated
on the plugin page and the guide links it. A good guide ends with tfui running on the reader's project.

### Contributing: `development/backlog.md`, `development/index.md`

Declared by this team. How a contributor works here: filing and moving issues, the board's statuses and labels, and
how the site is laid out. A fact belongs here when a contributor follows it as a procedure and it is not a rule of
the code. A good page lets a contributor do the task without asking.

## Where a fact goes: targets with a saved plan

Terraform refuses `-target` together with a saved plan file, and tfui's pipeline passes a plan file to apply. That one
subject lands on four pages:

- Why the pipeline puts targets on plan and never on apply, and the target-leak bug that settled it:
  [ADR-0019](adr/0019-unidirectional-data-flow.md), a decision.
- What apply does when resources are pinned (it replans with `-target`, shows the plan, then applies it):
  [the Apply plugin](plugins/apply.md), a feature.
- That `tfui apply -target=X` with no plan file plans and applies in one shot: [the CLI reference](reference/cli-reference.md),
  an interface.
- That a destructive operation such as apply opens a confirmation summarizing what will change:
  [TUI UX](reference/tui-ux.md), a standard.

## Front matter

`description` says what the page is, in one sentence opening with a verb that names its kind, such as "Records why"
or "Lists every flag", so a reader of a listing knows whether to open it. `audience` says who opens it and when.
`seo_description`, when present, is what search engines and link previews show in place of `description`
(`_plugins/seo_description.rb`). Keep `layout`, `title`, `parent`, `nav_order`, `permalink`, and on plugin pages `id`,
`key`, `category` and `default_enabled`: the site reads them.

## Misfilings we meet

- A plugin's rule restated on a feature page, or the reverse. Giveaway: the same threshold or key on both pages; keep
  it on the plugin page.
- A CLI or TUI convention written into one plugin's page. Giveaway: it names a key layer, a bar or an output channel
  that every plugin shares; it belongs to a standard.
- A decision's reasoning written into a plugin page. Giveaway: "we chose" or a rejected alternative; it belongs to a
  record, linked from the page.
- A plan or status kept in a page. Giveaway: a list updated by hand as work lands; it belongs to an issue.
