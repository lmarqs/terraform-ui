---
name: add-terraform-feature
allowed-tools: Bash(find:*), Read, Write, Edit, Bash(mise run:*)
description: Add a new terraform operation to the Service (interface, implementations, domain types)
---

## Add a new terraform operation

Plugins reach terraform only through `sdk.Service`. A new operation touches the interface and every implementation.

Steps:
1. The `test-writer` agent writes the failing test first.
2. Add the method to `Service` in `pkg/sdk/service.go`. Options go in an `<Op>Options` struct beside it.
3. Implement it on `ExecService` in `internal/terraform/exec/` using `tfexec.Terraform`; group it with its kind
   (`state_ops.go`, `workspace_ops.go`) or in `service.go`.
4. Implement it on `MacroService` in `internal/terraform/macro_service.go`: a mutation records an `sdk.Command` and
   touches nothing, a read serves the cache.
5. Implement it on `MockService` in `pkg/sdk/sdktest/mock_service.go` so plugin tests can stub it.
6. Run `mise run test:unit ./pkg/... ./internal/terraform/...`.

Key patterns:
- Domain types plugins see live in `pkg/sdk/` (`resource.go`, `change.go`); parse terraform's JSON with
  `github.com/hashicorp/terraform-json`.
- Analysis over plan or state (risk, phantom, grouping) gets its own file in `internal/terraform/`.
- Tests are table-driven; `internal/terraform/exec` is excluded from the coverage gate, the rest is held at 100%.
- Terraform validates its own inputs; pass its error through unchanged (`tfui-conventions`).
