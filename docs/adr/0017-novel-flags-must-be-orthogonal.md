---
layout: default
title: "ADR-0017: Novel flags must be orthogonal — no conditional validity"
grand_parent: Development
parent: Architecture
nav_order: 0017
description: Decision to require all novel flags to combine validly with every other flag
---

# Novel flags must be orthogonal — no conditional validity

Every novel flag must produce valid behavior in combination with every other flag. A flag whose semantics depend on another flag's absence is a conditional flag — it introduces a hidden dependency between axes and must not be introduced.

This extends ADR-0006 (orthogonal axes) to the flag surface specifically. The system's existing flags (`-ci`, `-json`, `-macro`, `-plan`, `-state`, `-record`) all combine freely. Any pairing is valid. Before adding a novel flag, verify that all combinations remain valid. If any combination is contradictory, the flag doesn't belong on the CLI surface.

## The incident: `-interactive`

A `-interactive` flag was proposed for form-capable plugins. It showed the TUI form for review instead of auto-running. The problem: `-interactive` + `-ci` is a contradiction — you cannot show a form in headless mode. This made it the first flag in the project conditional on another flag's absence.

## Hypothetical violations

- `-verbose` + `-json` — verbose human output is meaningless when format is JSON
- `-dry-run` + `-ci` — if dry-run only affected TUI rendering, it's conditional on having a TUI
- `-color` + `-ci` — color only matters when there's a TUI to render

## Resolution

The UX need (guided form vs auto-run) is handled by two activation paths, not a flag:

- `Activate()` — from TUI menu, shows the form for configuration
- `ActivateWithArgs(args)` — from CLI with flags, auto-submits (flags = declared intent)

Plugins are invocation-agnostic (ADR-0006). The activation path distinction is architectural, not flag-driven.
