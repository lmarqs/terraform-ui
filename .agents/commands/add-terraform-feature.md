---
allowed-tools: Bash(find:*), Read, Write, Edit, Bash(mise run:*)
description: Add a new terraform feature (service method, parser, etc.)
---

## Add a new terraform feature

Extend the terraform service layer in `internal/terraform/`.

Steps:
1. Read `internal/terraform/service.go` for the Service interface
2. Add the new method to the `Service` interface
3. Implement it on `TerraformService` using `tfexec`
4. If new types are needed, add them to `internal/terraform/parser.go`
5. Add unit tests in the corresponding `*_test.go` file
6. Run `mise run test:unit` to verify

Key patterns:
- Use `tfexec.Terraform` from hashicorp/terraform-exec
- Parse JSON output with `github.com/hashicorp/terraform-json`
- Domain types go in parser.go (Resource, PlanChange, etc.)
- Business logic (risk, phantom, grouping) gets its own file
- Always add table-driven tests
