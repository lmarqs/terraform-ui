---
layout: default
title: "ADR-0017: Novel flags must be orthogonal — no conditional validity"
grand_parent: Development
parent: Architecture
nav_order: 0017
seo:
  description: Decision to require all novel flags to combine validly with every other flag
description: "Records why every flag tfui adds must combine validly with every other flag."
audience: "Contributors adding a CLI flag"
---

# Novel flags must be orthogonal — no conditional validity

Amended by [ADR-0021](0021-plugins-as-use-cases.md): the two activation paths became one method, `Activate(input Input)`, and the plugin's typed `Input` carries the difference.

Every novel flag must produce valid behavior in combination with every other flag. A flag whose semantics depend on another flag's absence is a conditional flag — it introduces a hidden dependency between axes and must not be introduced.

This extends ADR-0006 (orthogonal axes) to the flag surface specifically. The novel flags (`-ci`, `-json`, `-macro`, `-plan`, `-state`, `-record`) all combine freely. Any pairing is valid.

The incident that forced the rule: a `-interactive` flag was proposed for form-capable plugins. It showed the TUI form for review instead of auto-running. The problem: `-interactive` + `-ci` is a contradiction — you cannot show a form in headless mode. This made it the first flag in the project conditional on another flag's absence.

## Consequences

- Before adding a novel flag, verify that all combinations remain valid. If any combination is contradictory, the flag doesn't belong on the CLI surface. Hypothetical violations:
  - `-verbose` + `-json` — verbose human output is meaningless when format is JSON
  - `-dry-run` + `-ci` — if dry-run only affected TUI rendering, it's conditional on having a TUI
  - `-color` + `-ci` — color only matters when there's a TUI to render
- The UX need behind `-interactive` (guided form vs auto-run) is handled by the activation path, not a flag. The TUI menu activates the plugin with a zero `Input`, which shows the form for configuration; a CLI command fills `Input` from its flags, and a plugin given flag values auto-submits (flags = declared intent). `plugins/init` works this way.
- Plugins are invocation-agnostic (ADR-0006). The activation path distinction is architectural, not flag-driven.

## Considered Options

### A `-interactive` flag

Rejected — conditional on the absence of `-ci`, as the incident above shows.

### Two activation methods

The resolution as first recorded: `Activate()` from the TUI menu shows the form, and `ActivateWithArgs(args)` from the CLI auto-submits. Replaced by the typed `Input` of ADR-0021.
