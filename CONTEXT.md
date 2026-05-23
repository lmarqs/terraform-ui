# terraform-ui

A k9s-style interactive TUI for Terraform operations. Single Go binary, plugin architecture, BubbleTea framework.

## Language

### Navigation & Structure

**Project**:
The root directory where `tfui.hcl` lives. Defines the boundary of a managed terraform workspace collection.
_Avoid_: repo, root, monorepo

**Chdir**:
A selected member directory within a project. Each chdir is a terraform root module.
_Avoid_: scope, module, target, directory

**Member**:
A directory declared in `tfui.hcl` via `member "path" {}` blocks. The set of valid chdirs.
_Avoid_: child, subdirectory, component

**Workspace**:
A Terraform workspace within a chdir. Standard terraform concept, no redefinition.
_Avoid_: environment, stage

**Context**:
The complete operating environment: Project + Chdir + Workspace + resolved execution parameters (var-files, vars, parallelism, lock settings, scoped service). Represents "where am I operating and with what parameters." Replaced atomically on context switch — never patched field-by-field.
_Avoid_: session, scope, state (when meaning location), execution context, resolved options

### Plugin System

**Plugin**:
A self-contained feature module implementing `sdk.Plugin`. Owns its view logic, types, messages, and state.
_Avoid_: component, widget, module (when meaning feature)

**Frame**:
A sub-view within a plugin, managed via a stack. Controls rendering and input within that plugin's screen.
_Avoid_: page, screen, panel (when meaning sub-view within a plugin)

**SDK**:
The public contract (`pkg/sdk/`) that plugins depend on. The only allowed import for plugin code.
_Avoid_: API, library, framework (when referring to the plugin contract)

### UI Zones

**Actions Bar**:
A row of button chips (cyan background, black text) inside the bordered plugin frame, pinned to the bottom. Shows terraform mutation keys only. Owned and rendered by the plugin.
_Avoid_: toolbar, action palette, command bar

**Hint Bar**:
The single-line footer outside the bordered frame. Shows UI/navigation keys only (ctrl+key, punctuation, Enter, Esc, q).
_Avoid_: status bar (when meaning the hint line), footer (ambiguous)

**Scroll Gutter**:
A vertical column at the right edge of the content area showing viewport position. `▲` top cap, `┃` thumb, `│` track, `▼` bottom cap. Only visible when content overflows.
_Avoid_: scrollbar (implies interactivity)

### Operations

**Plan**:
Preparation of terraform changes. Owns all planning, including targeted replans. Produces a plan file as its output artifact. Never hands off to Apply until the plan file matches the user's current intent (including pins as targets).
_Avoid_: preview, dry-run

**Apply**:
Confirmation and execution of a prepared plan. Receives a plan file and executes it — never re-derives, re-plans, or reads targets independently. In TUI flow: confirms and runs `terraform apply planfile.out`. In CLI flow (`tfui apply --target=X`): maps to terraform's own plan+apply-in-one-shot mode.
_Avoid_: deploy, execute (as standalone term)

**Replan**:
Re-running `terraform plan` within the Plan plugin to incorporate changed inputs (new pins, refresh). Equivalent to `ctrl+r` but may include `-target` flags from pins. Always owned by Plan — never by Apply.
_Avoid_: re-plan inside apply, targeted apply

**Pin**:
A user-selected resource marked for targeted operations. Pins scope `plan` to specific resources. Plugin-derived state — scoped to the current Context, dies on context switch.
_Avoid_: selection, mark, target (as noun for the set)

**Macro**:
A recorded sequence of TUI interactions for deterministic replay. Used for automated testing.
_Avoid_: script, recording, tape (when meaning the concept — "tape" is the file format)

### Service Layer

**ExecService**:
The live terraform execution adapter. Shells out to the terraform binary.
_Avoid_: runner, executor, live service

**MacroService**:
The recording adapter. Captures commands without executing them.
_Avoid_: mock service, dry-run service, recorder

**ServiceCache**:
Typed, source-aware cache shared by both service strategies. Pre-seeded from CLI flags at startup.
_Avoid_: store, buffer, data layer

## Relationships

- A **Project** contains one or more **Members**
- A **Chdir** is exactly one **Member** selected at runtime
- A **Workspace** exists within a **Chdir**
- A **Context** is one **Project** + one **Chdir** + one **Workspace** + their resolved execution parameters; replaced atomically on switch
- A **Plugin** contains one or more **Frames** (via a stack)
- A **Pin** targets a resource address; pins are shared across **Plugins** within the current **Context** and die on context switch
- Both **ExecService** and **MacroService** read from the same **ServiceCache**
- The **Actions Bar** lives inside the bordered frame; the **Hint Bar** lives outside it
- The **Scroll Gutter** spans content rows only (not the actions bar)

### Data Flow

Data flows downstream; invalidation flows upstream. Each node receives input from its parent, derives its own state, and resets when the parent signals change.

```
Context ──→ Plan ──→ Apply
```

- **Context** is owned by the app. Plugins are downstream consumers — they read from it, never write to it.
- **Plugins derive state** from Context (pins, plan file, filtered views). When Context changes, all derived state is invalidated — full reset.
- **Plan produces a plan file** as its output artifact. If pins exist, Plan runs a targeted plan (`terraform plan -target=X`). The plan file is only handed to Apply when it matches the user's current intent.
- **Apply consumes the plan file**. It confirms and executes — never re-plans, never reads targets, never accesses shared options. Apply's sole input is the artifact Plan produced.
- **Invalidation flows upstream**: a state-mutating operation (state rm, taint) signals "plan is stale" — it never modifies the parent's state directly.

Architectural enforcement: see ADR-0019 (unidirectional data flow).

## Example dialogue

> **Dev:** "When the user switches **Chdir**, does the **Workspace** reset?"
> **Domain expert:** "Yes — each **Member** has its own workspace state. Switching **Chdir** reloads workspaces for that member. The **Context** changes entirely."

> **Dev:** "If I **Pin** resources in the state view, then navigate to plan, are pins still there?"
> **Domain expert:** "Yes — **Pins** are shared across **Plugins** within the current **Context**. That's how the targeted plan flow works: pin in state or plan, then plan uses those as targets."

> **Dev:** "What happens to pins when I switch **Chdir**?"
> **Domain expert:** "Gone. Pins are scoped to the **Context**. New context = fresh start. A pin on `aws_instance.web` in module A means nothing in module B."

> **Dev:** "User pins 3 resources and presses `a` to apply. Who runs the targeted plan?"
> **Domain expert:** "**Plan** does. Plan checks: do my pins match the current plan file? If not, it **Replans** with targets, then hands the plan file to **Apply**. Apply just confirms and runs."

> **Dev:** "Can Apply ever run `terraform plan` internally?"
> **Domain expert:** "Only in CLI mode (`tfui apply --target=X`) — that maps directly to terraform's own plan+apply-in-one-shot. In TUI flow, Apply receives a plan file and that's it. All planning belongs to Plan."

## Flagged ambiguities

- "context" — Resolved: **Context** is the complete operating environment (Project+Chdir+Workspace+resolved parameters). In code, the init-time DI container currently named `sdk.Context` will be renamed to avoid collision with this domain concept.
- "scope" — Deprecated. Was previously used for what is now **Chdir**. Do not use in new code or documentation.
- "module" — Ambiguous: could mean a Go module, a Terraform module, or a feature. Use **Plugin** for features, **Member** for terraform root modules.
