---
layout: default
title: "ADR-0020: Rich domain types over primitive obsession"
grand_parent: Development
parent: Architecture
nav_order: 0020
description: Decision to use named types for domain concepts rather than bare primitives
---

# Rich domain types over primitive obsession

Core domain values (workspace names, resource addresses, lock modes, refresh strategies) were bare primitives (`string`, `*bool`) with implicit invariants scattered across call sites. We replaced them with named types that encode domain identity at the type level, making it impossible to mix a workspace name with a file path or a resource address at compile time.

We chose named types (not interfaces or builder patterns) because the values are simple — they carry identity, not behavior. The type system provides safety; struct literal construction stays idiomatic Go.

## Key decisions

- **Named string types** (`Workspace`, `Address`, `LockTimeout`, `DiagnosticSeverity`) for values that are passed through to terraform. No validation at our layer — terraform owns format validation.
- **Named int enums** (`RefreshMode`, `LockMode`) to replace confusing two-field encodings (`*bool` + `bool`) with a single field where the zero value means "terraform default."
- **Service interface unchanged** — service methods keep `string` params at the I/O boundary. Conversion happens at the app layer, not the plugin layer. This prevents a cascade through MockService and every plugin test.
- **`LockModeFromPtr` bridge** — a single exported converter for the config layer (which returns `*bool` from HCL parsing). The bridge lives on the SDK so both `internal/ui` and `cmd/tfui` can use it without duplication.

## Considered Options

### Builder pattern for events/messages

Factories like `sdk.NewContextSwitch().Chdir("x").Workspace("y").Build()` would catch missing fields at runtime. Rejected — the named types already prevent passing wrong values, and the events are constructed in only 2-3 well-tested places each. The extra indirection is not justified.

### Interface for PlanFile (planned, not yet implemented)

The roadmap proposes `PlanFile` as an opaque struct with a `Cleanup()` method whose behavior depends on how the file was created (temp vs user-provided). This is a future phase — the current ADR covers only the foundation types.

### Validation in constructors

We could reject empty strings or invalid patterns in `NewWorkspace()`. Rejected per the "no speculative code" rule — terraform validates names, and our layer just passes them through.
