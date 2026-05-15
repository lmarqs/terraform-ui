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
The umbrella concept combining Project + Chdir + Workspace. Represents "where am I operating right now."
_Avoid_: session, scope, state (when meaning location)

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

### Operations

**Pin**:
A user-selected resource marked for targeted operations. Pins scope `plan` and `apply` to specific resources.
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
- A **Context** is the combination of one **Project** + one **Chdir** + one **Workspace**
- A **Plugin** contains one or more **Frames** (via a stack)
- A **Pin** targets a resource address; pins are shared across **Plugins** via PinService
- Both **ExecService** and **MacroService** read from the same **ServiceCache**

## Example dialogue

> **Dev:** "When the user switches **Chdir**, does the **Workspace** reset?"
> **Domain expert:** "Yes — each **Member** has its own workspace state. Switching **Chdir** reloads workspaces for that member. The **Context** changes entirely."

> **Dev:** "If I **Pin** resources in the state view, then navigate to plan, are pins still there?"
> **Domain expert:** "Yes — **Pins** are global to the session, shared across all **Plugins**. That's how the plan→apply flow works: pin in state or plan, then apply only pinned."

## Flagged ambiguities

- "context" — Resolved: **Context** is ONLY the umbrella concept (Project+Chdir+Workspace). Never used for `sdk.Context` (the struct passed to plugins at Init) in domain conversation. In code, `sdk.Context` is a dependency injection container, not the domain concept.
- "scope" — Deprecated. Was previously used for what is now **Chdir**. Do not use in new code or documentation.
- "module" — Ambiguous: could mean a Go module, a Terraform module, or a feature. Use **Plugin** for features, **Member** for terraform root modules.
