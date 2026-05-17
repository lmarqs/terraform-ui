---
title: Standardize Plugin Page Documentation
status: in-progress
priority: high
created: 2026-05-17
effort: large
tags: [docs, demo, plugins, dx]
depends_on: []
---

## Summary

Standardize all 19 plugin documentation pages with a consistent template, dedicated screenshots generated exclusively by the demo package, and uniform sections across all plugins.

## Problem

Plugin docs are inconsistent: only 4/19 have animated GIFs, sections appear in different orders, several lack CLI/Equivalence sections, and screenshots are a mix of text mockups and real demos. There's no single authoritative template that all plugins follow.

## Proposal

### Standard Template

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

### Demo Type Classification

**Animated GIF** (multi-step interaction):
plan, state, apply, risk, phantom, blastradius, workspace, import, taint, untaint, forceunlock, output, validate

**Static frame** (single screen capture):
version, console, init, context, chdir

### Naming Convention

- Tape files: `demo/tapes/<plugin-id>.tape` (keep existing names: plan-review, state-browse, risk-analysis, etc.)
- GIF outputs: `docs/assets/demo/<same-name-as-tape>.gif` (keep existing names)
- Fixture files: `demo/fixtures/<plugin-id>.json` per plugin where predictable side-effect data is needed

## Implementation Phases

### Phase 1: Extend MacroService for Richer Demos (DONE: resize fix)

The MacroService currently returns empty/stub data for Output, Validate, WorkspaceList, and Version. Cache-backed returns are needed so demos show real content.

**Files to modify:**
- `internal/terraform/service_cache.go` — Add fields: outputs, diagnostics, workspaces + Seed/Get methods
- `internal/terraform/macro_service.go` — Read from cache instead of returning stubs
- `cmd/tfui/main.go` — Add flags: `--outputs`, `--validate-result`, `--workspaces`
- `cmd/tfui/session.go` — Wire new flags through `seedCache()`

**Completed work:**
- Fixed GIF truncation bug: `Recorder.Resize()` method added, `Runner` calls it on `CmdResize` so manifest dimensions match actual tape resize
- Removed duplicate test functions in `plugins/plan/output_test.go`
- Fixed goimports formatting in 7 test files

### Phase 2: Fixture Files

Create per-plugin fixture files with predictable side-effect data:

| Fixture | Purpose |
|---------|---------|
| `demo/fixtures/outputs.json` | 5 output values (string, list, sensitive, url, object) |
| `demo/fixtures/validate.json` | 2 warnings (deprecated attr, unused var) |
| `demo/fixtures/workspaces.json` | 4 workspaces (default, production, staging, development) |
| `demo/fixtures/plan-large.json` | (existing) 16 resources for plan/risk/blast/phantom demos |
| `demo/fixtures/state-large.json` | (existing) 8 resources for state browsing |

### Phase 3: Tape Files

**Keep existing tapes (same names):**
- `hero.tape`, `plan-review.tape`, `state-browse.tape`, `pin-target.tape`, `risk-analysis.tape`, `phantom.tape`

**Create new tapes:**

| Tape | Type | Key interaction |
|------|------|-----------------|
| `apply.tape` | animated | p → space (pin) → a → y (confirm) |
| `blastradius.tape` | animated | b → scroll → enter (expand module) |
| `workspace.tape` | animated | w → scroll → enter (switch) |
| `import.tape` | animated | s → navigate → n → type ID → enter |
| `taint.tape` | animated | s → navigate → t → y |
| `untaint.tape` | animated | s → navigate → T → y |
| `forceunlock.tape` | animated | emit lock deploy-bot → u → y |
| `output.tape` | animated | o → scroll → enter (inspect) |
| `validate.tape` | animated | v → scroll → enter (expand) |
| `version.tape` | static | V or :version enter |
| `console.tape` | static | ~ |
| `init.tape` | static | :init enter |
| `context.tape` | static | C |
| `chdir.tape` | static | :chdir enter |

### Phase 4: Update `demo/generate.sh`

Pass new fixture flags to all recordings:
```bash
"$TFUI" \
  --plan "$FIXTURES/plan-large.json" \
  --state "$FIXTURES/state-large.json" \
  --outputs "$FIXTURES/outputs.json" \
  --validate-result "$FIXTURES/validate.json" \
  --workspaces "$FIXTURES/workspaces.json" \
  --macro "$tape" \
  --record "$OUTPUT/$name/"
```

Add optional single-tape mode for development:
```bash
./demo/generate.sh [path-to-binary] [tape-name]
```

### Phase 5: Rewrite All 19 Plugin Docs

Apply the standard template to every file:

| Plugin | Current gaps |
|--------|-------------|
| plan | Move Screenshot up, add Equivalence |
| state | Move Screenshot up, add Equivalence |
| risk | Add CLI section, Equivalence, move Screenshot up |
| phantom | Add CLI section, Equivalence, move Screenshot up |
| apply | Replace text mockups with GIF ref |
| blastradius | Add CLI, Equivalence, replace text screenshots |
| workspace | Add Equivalence, replace text screenshots |
| output | Add Equivalence, replace text screenshots |
| validate | Add Equivalence, replace text screenshots |
| import | Add Equivalence, replace text screenshots |
| taint | Add Equivalence, replace text screenshots |
| untaint | Add Equivalence, replace text screenshots |
| forceunlock | Add Equivalence, replace text screenshots |
| init | Add Equivalence, replace text screenshots |
| console | Add Equivalence, replace text screenshots |
| version | Add Equivalence, replace text screenshots |
| context | Add Equivalence, replace text screenshots |
| chdir | Add Equivalence, replace text screenshots |

Frontmatter: normalize `key` field to `"—"` when no keybinding exists.

### Phase 6: Update References

- Update any docs referencing old GIF names if tapes are renamed
- Update `docs/plugins/index.md` if it references screenshots
- Update README if it uses demo GIF paths

## Verification

```bash
mise run build
mise run demo:generate
ls docs/assets/demo/*.gif  # all 19+ GIFs exist
mise run test:unit
mise run docs:serve        # navigate each plugin page
mise run check:lint && mise run check:vet
```

## Progress

- [x] Fix GIF truncation (Recorder.Resize + Runner integration)
- [x] Fix duplicate test functions in plan plugin
- [x] Fix goimports in 7 test files
- [x] ServiceCache: add outputs/diagnostics/workspaces fields and Seed/Get methods
- [x] MacroService: read from cache for Output/Validate/WorkspaceList
- [x] CLI flags: --outputs, --validate-result, --workspaces
- [x] Create fixture files (outputs.json, validate.json, workspaces.json)
- [ ] Create 14 new tape files
- [ ] Update generate.sh
- [ ] Rewrite all 19 plugin docs to standard template
- [ ] Update references
