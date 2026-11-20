---
layout: default
title: "ADR-0006: Orthogonal axes: income and outcome are independent"
grand_parent: Development
parent: Architecture
nav_order: 0006
seo:
  description: Decision to keep input mode and output format as independent orthogonal axes
description: "Records why how tfui is driven and what its terraform calls do are independent axes."
audience: "Contributors adding a mode, an output format or a CLI flag"
---

# Orthogonal axes: income and outcome are independent

The system has two axes -- income (how the user drives: TUI, CLI, macro tape) and outcome (what happens: live execution, command recording). Plugins are unaware of both, so a plugin works under any pairing; which pairings the CLI offers is wiring in `cmd/tfui/`, and the [CLI reference](../reference/cli-reference.md) lists the flags that select them.

This falls directly from hexagonal architecture: plugins are the core, income is an input adapter, outcome is an output adapter. Coupling them inside a plugin would be a category error -- equivalent to saying "the HTTP adapter only works with PostgreSQL."

Most TUI tools (k9s, lazygit) are interactive-only -- they have no CLI axis and no recording axis. This project is different because tfui is a command builder with multiple interfaces, not just a visual shell.

## Consequences

- Adding a pairing, or an input or outcome, changes only the wiring in `cmd/tfui/`; no plugin changes.

## Considered Options

### Couple income to outcome in the plugins

Rejected -- a plugin that knew how it was driven or what its calls do would tie the core to its adapters, the category error above.
