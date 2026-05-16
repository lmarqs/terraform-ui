---
title: Verb-First Plugin Architecture (overview)
status: planned
priority: critical
created: 2026-05-15
effort: large
tags: [architecture, ux, workflow]
depends_on: []
---

## Summary

Restructure tfui plugins to mirror terraform's verb-first CLI design. Each top-level terraform verb becomes its own plugin. Workflows chain verbs explicitly — just like the CLI — with clear post-action guidance.

---

## Guiding Principle

**Terraform verbs are plugins. Workflows chain verbs.**

| Terraform CLI | tfui Plugin | Type |
|---|---|---|
| `terraform state list/rm/mv` | state | Browser |
| `terraform plan` | plan | Browser |
| `terraform taint` | taint | Action |
| `terraform untaint` | untaint | Action |
| `terraform import` | import | Action |
| `terraform apply` | apply | Action |

**Browser plugins** own a resource list (filter, tree, inspect, pin).
**Action plugins** are transient — arrive with context, confirm, execute, return.

---

## Complete Keybinding Map (post-restructure)

### Global Plugin Switches

| Key | Plugin | Menu Visible |
|-----|--------|------|
| `s` | state | ✓ |
| `p` | plan | ✓ |
| `w` | workspace | ✓ |
| `o` | output | ✓ |
| `v` | validate | ✓ |
| `i` | init | ✓ |
| `~` | console | ✓ |
| `C` | context | ✓ |
| `R` | risk | ✓ |
| `P` | phantom | ✓ |
| `B` | blast radius | ✓ |

### Contextual Verb Keys (inside browser plugins)

| Key | In State | In Plan | Meaning |
|-----|----------|---------|---------|
| `t` | → taint plugin | → taint plugin | Taint cursor resource |
| `T` | → untaint plugin | → untaint plugin | Untaint cursor resource |
| `n` | → import plugin | — | Import at address |
| `d` | inline state rm | — | Delete from state |
| `m` | inline state mv | — | Move/rename in state |
| `e` | open $EDITOR | — | Edit .tf file |
| `a` | — | → [replan] → apply | Apply (full or targeted) |
| `u` | → forceunlock | → forceunlock | Force unlock (when locked) |

### Navigation & Interface

| Key | Action | Scope |
|-----|--------|-------|
| `Space` | Pin/unpin | Browser plugins |
| `Enter` | Inspect/confirm | Everywhere |
| `/` | Filter mode | Browser plugins |
| `!` | Batch palette | When pins > 0 |
| `Esc` | Pop/cancel/back | Everywhere |
| `q` | Home | Everywhere |
| `:` | Command mode | Everywhere |
| `ctrl+r` | Refresh/retry | Everywhere |
| `ctrl+t` | Toggle tree/flat | Browser plugins |
| `ctrl+w` | Toggle wrap | Detail views |
| `ctrl+p` | Pinned filter | Browser plugins |
| `ctrl+u` | Unpin all | Browser plugins |
| `[` / `]` | Collapse/expand all | Tree mode |
| `←` / `→` | Horizontal pan | When wrap off |
| `↑` / `↓` / `j` / `k` | Navigate | Lists |
| `g` / `G` | Jump start/end | Lists |

### Command Mode (`:`)

| Command | Target |
|---------|--------|
| `:state` | State plugin |
| `:plan` | Plan plugin |
| `:apply` | Apply plugin |
| `:taint` | Taint plugin |
| `:untaint` | Untaint plugin |
| `:import` | Import plugin |
| `:workspace` | Workspace plugin |
| `:console` | Console plugin |
| `:output` | Output plugin |
| `:validate` | Validate plugin |
| `:init` | Init plugin |
| `:risk` | Risk plugin |
| `:phantom` | Phantom plugin |
| `:blast-radius` | Blast radius plugin |
| `:q` | Quit (guarded) |
| `:q!` | Force quit |

---

## State Transitions

### Taint Plugin

```
[Idle] ──SetTargets+RequestExecute──→ [Confirming]
[Confirming] ──y/enter──→ [Loading]
[Confirming] ──n/esc──→ [DeactivateMsg → return to origin]
[Loading] ──success──→ [Done] + emit PlanInvalidatedEvent
[Loading] ──failure──→ [Error]
[Done] ──esc──→ [DeactivateMsg → return to origin]
[Done] ──p──→ [NavigateMsg{plan}]
[Error] ──esc──→ [DeactivateMsg → return to origin]
[Error] ──ctrl+r──→ [Loading] (retry)
```

### Untaint Plugin

```
(identical to taint)
```

### Import Plugin

```
[Idle] ──SetAddress+RequestExecute──→ [Form]
[Form] ──enter (valid)──→ [Confirming]
[Form] ──esc──→ [DeactivateMsg → return to origin]
[Confirming] ──y/enter──→ [Loading]
[Confirming] ──n/esc──→ [Form] (back to edit)
[Loading] ──success──→ [Done] + emit StateRefreshedEvent + PlanInvalidatedEvent
[Loading] ──failure──→ [Error]
[Done] ──esc──→ [DeactivateMsg → return to origin]
[Done] ──p──→ [NavigateMsg{plan}]
[Error] ──esc──→ [DeactivateMsg → return to origin]
[Error] ──ctrl+r──→ [Loading] (retry)
```

### Apply Plugin (revised)

```
[Idle] ──RequestApply (no targets)──→ [Confirming]
[Idle] ──RequestApply (with targets)──→ [Replanning]
[Replanning] ──success──→ [Confirming]
[Replanning] ──failure──→ [Error]
[Replanning] ──esc──→ [DeactivateMsg → return to plan]
[Confirming] ──y/enter──→ [Loading]
[Confirming] ──n/esc──→ [DeactivateMsg → return to plan]
[Loading] ──success──→ [Done] + emit PlanInvalidatedEvent
[Loading] ──failure──→ [Error]
[Done] ──esc──→ [DeactivateMsg → return to plan]
[Error] ──esc──→ [DeactivateMsg → return to plan]
[Error] ──ctrl+r──→ [Loading] (retry)
```

### State Plugin

```
[Idle] ──Activate──→ [Loading]
[Loading] ──success──→ [Done]
[Loading] ──failure──→ [Error]
[Done] ──t──→ emit TaintRequestMsg (app navigates to taint)
[Done] ──T──→ emit UntaintRequestMsg (app navigates to untaint)
[Done] ──n──→ emit ImportRequestMsg (app navigates to import)
[Done] ──d──→ [Inline confirm → Loading → Done] (state rm)
[Done] ──m──→ [Inline input → confirm → Loading → Done] (state mv)
[Done] ──ctrl+r──→ [Loading] (refresh)
[Error] ──ctrl+r──→ [Loading] (retry)
```

### Plan Plugin

```
[Idle] ──Activate──→ [Loading]
[Loading] ──success──→ [Done]
[Loading] ──failure──→ [Error]
[Done] ──a (no pins)──→ emit ApplyRequestMsg (app → apply, use saved plan)
[Done] ──a (with pins)──→ emit ApplyRequestMsg (app → apply, replan with targets)
[Done] ──t──→ emit TaintRequestMsg (app navigates to taint)
[Done] ──T──→ emit UntaintRequestMsg (app navigates to untaint)
[Done] ──ctrl+r──→ [Loading] (re-plan)
[Done] ──PlanInvalidatedEvent──→ [Loading] (auto re-plan)
[Error] ──ctrl+r──→ [Loading] (retry)
```

---

## User Workflows (Complete)

### W1: Standard Apply (no targets)

```
User: p
  → Plan runs terraform plan, saves tfplan.out
  → Shows: 3 to add, 1 to change, 1 to destroy
User: (reviews changes)
User: a
  → App → apply plugin (NavPush, returnTo=plan)
  → "Apply all changes? [y/n]"
User: y
  → terraform apply tfplan.out
  → "Applied. 5 resources affected."
  → PlanInvalidatedEvent
User: Esc
  → Return to plan (auto-replans, shows "no changes")
```

### W2: Targeted Apply (replan flow)

```
User: p
  → Full plan: 10 changes
User: Space on 2 resources (pin them)
User: a
  → App detects targets
  → Apply plugin enters Replanning state
  → terraform plan -target=A -target=B (new plan file)
  → Shows targeted plan: "2 resources, 1 add, 1 change"
User: y
  → terraform apply tfplan.out (targeted plan)
  → Success
  → PlanInvalidatedEvent
User: Esc
  → Return to plan (auto-replans full)
```

### W3: Taint from State → Plan → Apply

```
User: s
  → State loads, shows resource list
User: / web (filter)
User: t (on aws_instance.web)
  → Navigate to taint plugin (NavPush, returnTo=state)
  → "Taint aws_instance.web? [y/n]"
User: y
  → terraform taint aws_instance.web
  → "✓ Tainted. p plan  Esc back"
  → PlanInvalidatedEvent emitted
User: p
  → Navigate to plan
  → Plan runs, shows: "1 to destroy, 1 to add (replace)"
User: a → y
  → terraform apply tfplan.out
  → Resource recreated
```

### W4: Taint from Plan (sees drift, wants recreation)

```
User: p
  → Plan shows changes, including aws_instance.web with unwanted drift
User: t (on aws_instance.web)
  → Navigate to taint plugin (NavPush, returnTo=plan)
  → "Taint aws_instance.web? [y/n]"
User: y
  → terraform taint
  → Success + PlanInvalidatedEvent
User: Esc
  → Return to plan
  → Plan auto-replans (listens to PlanInvalidatedEvent)
  → Shows: resource now marked for replace
User: a → y
  → Apply
```

### W5: Batch Taint from State

```
User: s
User: Space (pin resource A)
User: Space (pin resource B)
User: Space (pin resource C)
User: !
  → Batch palette: [d] delete  [t] taint  [T] untaint
User: t
  → Navigate to taint plugin with 3 addresses
  → "Taint 3 resources? [y/n]"
  → Lists addresses
User: y
  → Taint all sequentially
  → "3/3 tainted. p plan  Esc back"
User: p
  → Plan shows 3 recreations
```

### W6: Untaint (cancel recreation)

```
User: s (or p)
User: (cursor on tainted resource)
User: T
  → Navigate to untaint plugin
  → "Untaint aws_instance.web? [y/n]"
User: y
  → terraform untaint
  → PlanInvalidatedEvent
User: Esc → return
```

### W7: Import New Resource

```
User: :import
  → Navigate to import plugin (empty form)
  → "Address: [                    ]"
  → "ID:      [                    ]"
User: types aws_instance.web, tab, types i-0abc123
User: Enter
  → "Import i-0abc123 as aws_instance.web? [y/n]"
User: y
  → terraform import aws_instance.web i-0abc123
  → "✓ Imported. p plan  Esc back"
  → StateRefreshedEvent + PlanInvalidatedEvent
User: p
  → Plan shows whether config matches imported state
```

### W8: Import from State (re-import, address pre-filled)

```
User: s
User: (cursor on aws_instance.web)
User: n
  → Navigate to import plugin (address pre-filled)
  → "Address: [aws_instance.web    ]"
  → "ID:      [                    ]"
User: types new ID
User: Enter → confirm → execute → return
```

### W9: State Surgery → Verify

```
User: s
User: d (delete aws_instance.old from state)
  → "Remove aws_instance.old from state? [y/n]"
User: y
  → terraform state rm
  → StateRefreshedEvent + PlanInvalidatedEvent
  → Hint: "p plan"
User: p
  → Plan shows: terraform wants to create aws_instance.old (it exists in config but not state)
  → User decides: remove from config, or import, or accept
```

### W10: State Move → Verify

```
User: s
User: m (on aws_instance.web)
  → "Move to: [aws_instance.web    ]"
User: changes to aws_instance.application
  → "Move aws_instance.web → aws_instance.application? [y/n]"
User: y
  → terraform state mv
  → PlanInvalidatedEvent
  → Hint: "p plan"
User: p
  → Plan shows no changes (if config also renamed) or shows drift (if config still has old name)
```

---

## Event Flow Diagram

```
┌──────────────┐     PlanInvalidatedEvent      ┌──────────────┐
│ Taint Plugin │ ─────────────────────────────→ │ Plan Plugin  │
│              │                                │ (auto-replan)│
└──────────────┘                                └──────────────┘
                                                       ↑
┌──────────────┐     PlanInvalidatedEvent              │
│Untaint Plugin│ ──────────────────────────────────────┘
└──────────────┘                                       ↑
                                                       │
┌──────────────┐  StateRefreshed+PlanInvalidated       │
│Import Plugin │ ──────────────────────────────────────┘
└──────────────┘         │                             ↑
                         │                             │
                         ↓                             │
┌──────────────┐     StateRefreshedEvent               │
│ State Plugin │ ←─────────────────────────────────────│
│ (refresh)    │     PlanInvalidatedEvent               │
│              │ ──────────────────────────────────────┘
└──────────────┘  (from inline rm/mv)

┌──────────────┐     PlanInvalidatedEvent
│ Apply Plugin │ ──────────────────────────────────────→ Plan, State
└──────────────┘
```

---

## Navigation Model

```
                    ┌─────────┐
                    │  HOME   │
                    └────┬────┘
         ┌───────┬───────┼───────┬───────┐
         ↓       ↓       ↓       ↓       ↓
      ┌─────┐ ┌─────┐ ┌─────┐ ┌─────┐ ┌─────┐
      │state│ │plan │ │work │ │outp │ │valid│  ... (NavReplace)
      └──┬──┘ └──┬──┘ └─────┘ └─────┘ └─────┘
         │       │
    t/T/n│  t/T/a│
         │       │
         ↓       ↓
      ┌─────────────────┐
      │  Action Plugins  │  (NavPush — returnTo=origin)
      │                  │
      │  taint           │
      │  untaint         │
      │  import          │
      │  apply           │
      │  forceunlock     │
      └─────────────────┘
```

**NavReplace**: lateral switch, no history (state, plan, workspace, output, validate, etc.)
**NavPush**: preserves return context (taint, untaint, import, apply, forceunlock)

---

## Implementation Order & Dependencies

```
Phase 1 (no dependencies, parallel-safe):
  ├── console-keybinding       (small, frees t key)
  ├── taint-plugin             (medium, new plugin)
  └── untaint-plugin           (medium, new plugin)

Phase 2 (depends on Phase 1):
  ├── import-plugin            (medium, new plugin)
  ├── plan-contextual-verbs    (small, depends on taint/untaint existing)
  └── state-plugin-slim        (small, depends on taint/untaint/import existing)

Phase 3 (depends on Phase 2):
  └── apply-replan-targets     (medium, depends on plan plugin changes)
```

### Priority Matrix

| Item | Priority | Effort | Risk if Skipped |
|------|----------|--------|-----------------|
| apply-replan-targets | Critical | Medium | User applies unreviewed changes |
| taint-plugin | High | Medium | Workflow misalignment with terraform |
| untaint-plugin | High | Medium | Workflow misalignment with terraform |
| import-plugin | High | Medium | Workflow misalignment with terraform |
| state-plugin-slim | High | Small | Incoherent plugin responsibilities |
| plan-contextual-verbs | High | Small | Missing natural workflow path |
| console-keybinding | Medium | Small | Keybinding conflict blocks others |

---

## CLI Surface (matching verbs)

Each verb plugin has a corresponding CLI command:

```bash
# Browser plugins (output to stdout)
tfui state                    # List resources (tree view)
tfui state -json              # List resources (JSON)
tfui plan                     # Run plan, show tree
tfui plan -json               # Run plan, terraform-compatible JSON
tfui plan --target X          # Targeted plan

# Action plugins (execute and exit)
tfui apply                    # Plan + confirm + apply
tfui apply --auto-approve     # Plan + apply (no confirm)
tfui taint <address>          # Taint resource
tfui untaint <address>        # Untaint resource
tfui import <address> <id>    # Import resource

# State sub-commands (genuine state operations)
tfui state rm <address>       # Remove from state
tfui state mv <src> <dst>     # Move in state
```

---

## What Stays Unchanged

- Workspace plugin (already standalone)
- Output plugin (already standalone)
- Validate plugin (already standalone)
- Init plugin (already standalone)
- Risk/Phantom/BlastRadius plugins (analysis plugins, not verbs)
- Context/Chdir plugins (meta-navigation)
- ForceUnlock plugin (already action-style)
- Pin service (unchanged semantics)
- Event bus (unchanged, gains more subscribers)
- Service interface (unchanged)
