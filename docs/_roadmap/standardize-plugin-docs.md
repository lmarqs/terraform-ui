---
title: Standardize Plugin Page Documentation
status: in-progress
priority: high
created: 2026-05-17
effort: medium
tags: [docs, demo, plugins, dx]
depends_on: []
---

## Summary

Standardize all 19 plugin documentation pages with a consistent template, dedicated screenshots generated exclusively by the demo package, and uniform sections across all plugins.

## Standard Template

Every plugin doc follows this exact structure:

```markdown
---
layout: default
parent: Plugins
title: [Human Name]
id: [plugin-id]
key: [keybinding or "—"]
category: [operations|navigation|action|analysis|utility]
description: [one-line description]
default_enabled: true
---

# [Human Name]

## Overview
## Screenshot
## Interactive (TUI)
## Command Line (CLI)
## Equivalence
## Configuration
## Related
```

## Demo Type Classification

**Animated GIF** (multi-step interaction):
plan, state, apply, risk, phantom, blastradius, workspace, import, taint, untaint, forceunlock, output, validate

**Static frame** (single screen capture):
version, console, init, context, chdir

## Remaining Work

### Phase 5: Rewrite All 18 Plugin Docs

Apply the standard template. Changes needed per file:

| Plugin | Screenshot | Equivalence | CLI | Other |
|--------|-----------|-------------|-----|-------|
| plan | ✓ has GIF | ✓ has | ✓ | Promote Screenshot to dedicated heading |
| state | ✓ has GIF | ✓ has | ✓ | Promote Screenshot to dedicated heading |
| risk | ✓ has GIF | ✗ add | ✗ add | Promote Screenshot to dedicated heading |
| phantom | ✓ has GIF | ✗ add | ✗ add | Promote Screenshot to dedicated heading |
| apply | ✗ replace text with GIF ref | ✓ has | ✓ | — |
| blastradius | ✗ replace text with GIF ref | ✗ add | ✗ add | — |
| workspace | ✗ replace text with GIF ref | ✗ add | ✓ | — |
| output | ✗ replace text with GIF ref | ✗ add | ✓ | — |
| validate | ✗ replace text with GIF ref | ✗ add | ✓ | — |
| import | ✗ replace text with GIF ref | ✗ add | ✓ | — |
| taint | ✗ replace text with GIF ref | ✗ add | ✓ | — |
| untaint | ✗ replace text with GIF ref | ✗ add | ✓ | — |
| forceunlock | ✗ replace text with GIF ref | ✗ add | ✓ | — |
| init | ✗ replace text with GIF ref | ✗ add | ✓ | — |
| console | ✗ replace text with GIF ref | ✗ add | ✓ | Add Exit Codes |
| version | ✗ replace text with GIF ref | ✗ add | ✓ | — |
| context | ✗ replace text with GIF ref | ✗ add | ✓ | — |
| chdir | ✗ replace text with GIF ref | ✗ add | ✓ | Add Exit Codes |

**Rewrite strategy:**
1. Add `## Screenshot` heading with GIF reference (`![Name]({{ site.baseurl }}/assets/demo/<tape-name>.gif)`)
2. Add `## Equivalence` table (Goal | CLI | TUI columns) for the 11 files missing it
3. Add `## Command Line (CLI)` for risk and phantom (analysis-only, explain no standalone CLI)
4. Normalize section order to match template
5. Normalize `key` frontmatter to `"—"` for plugins with no keybinding

**GIF name mapping** (tape name → GIF file → plugin doc):
- plan → plan-review.gif → plan.md
- state → state-browse.gif → state.md
- risk → risk-analysis.gif → risk.md
- phantom → phantom.gif → phantom.md
- apply → apply.gif → apply.md
- blastradius → blastradius.gif → blastradius.md
- workspace → workspace.gif → workspace.md
- output → output.gif → output.md
- validate → validate.gif → validate.md
- import → import.gif → import.md
- taint → taint.gif → taint.md
- untaint → untaint.gif → untaint.md
- forceunlock → forceunlock.gif → forceunlock.md
- init → init.gif → init.md
- console → console.gif → console.md
- version → version.gif → version.md
- context → context.gif → context.md
- chdir → chdir.gif → chdir.md

### Phase 6: Update References

- Update `docs/plugins/index.md` if it references screenshots
- Update README if it uses demo GIF paths
- Verify all GIF paths resolve after running `mise run demo:generate`

## Verification

```bash
mise run build
mise run demo:generate
ls docs/assets/demo/*.gif  # all 20 GIFs exist (6 existing + 14 new)
mise run docs:serve        # navigate each plugin page, verify screenshots render
```
