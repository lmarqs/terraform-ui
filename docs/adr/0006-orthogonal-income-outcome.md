---
layout: default
title: "ADR-0006: Orthogonal axes: income and outcome are independent"
grand_parent: Development
parent: Architecture
nav_order: 0006
description: Decision to keep input mode and output format as independent orthogonal axes
---

# Orthogonal axes: income and outcome are independent

The system has two axes -- income (how the user drives: TUI, CLI, macro tape) and outcome (what happens: live execution, command recording). These are independent. Any income can pair with any outcome. Plugins are unaware of both.

This falls directly from hexagonal architecture: plugins are the core, income is an input adapter, outcome is an output adapter. Coupling them would be a category error -- equivalent to saying "the HTTP adapter only works with PostgreSQL."

Most TUI tools (k9s, lazygit) are interactive-only -- they have no CLI axis and no recording axis. This project is different because tfui is a command builder with multiple interfaces, not just a visual shell.
