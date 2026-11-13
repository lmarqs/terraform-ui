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

Amended by [ADR-0021](0021-plugins-as-use-cases.md): `-macro` selects both the tape input and the recording backend, so the recording outcome pairs only with macro income.

The system has two axes -- income (how the user drives: TUI, CLI, macro tape) and outcome (what happens: live execution, command recording). These are independent. Plugins are unaware of both.

This falls directly from hexagonal architecture: plugins are the core, income is an input adapter, outcome is an output adapter. Coupling them inside a plugin would be a category error -- equivalent to saying "the HTTP adapter only works with PostgreSQL."

Most TUI tools (k9s, lazygit) are interactive-only -- they have no CLI axis and no recording axis. This project is different because tfui is a command builder with multiple interfaces, not just a visual shell.

## Consequences

- Plugins are unaware of both axes; which pairings exist is a wiring choice in `cmd/tfui/`.

## Considered Options

### Couple income to outcome in the plugins

Rejected -- a plugin that knew how it was driven or what its calls do would tie the core to its adapters, the category error above.
