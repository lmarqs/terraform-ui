# `-plan` flag: support binary `.tfplan` format

## Problem

The `-plan` flag currently only accepts JSON plan output. Users who have a binary `.tfplan` file (the default output of `terraform plan -out=`) must manually run `terraform show -json <file>` before passing it to tfui. This is friction.

## Design

Detect format by source:

| Usage | Detected format | Behavior |
|-------|----------------|----------|
| `-plan -` (stdin) | JSON | Seed cache directly (current behavior) |
| `-plan ./plan.json` (`.json` extension) | JSON | Seed cache directly (current behavior) |
| `-plan ./plan.tfplan` (anything else) | Binary | tfui runs `terraform show -json <file>` for display; stores path for apply |

## Changes

- **`cmd/tfui/main.go` (`seedCache`)** — when planURI is not stdin and not `.json`, treat as binary: call `terraform show -json <path>` to get JSON for cache seeding, and store the original path for apply consumption
- **`internal/terraform/exec/service.go`** — expose `ShowPlanJSON(ctx, path) ([]byte, error)` wrapping `terraform show -json`
- **Apply flow** — when a binary plan file was provided via CLI, apply uses that path directly (no re-plan needed)

## Complexity

Medium. Touches the I/O boundary (`ExecService`), cache seeding, and the service interface. Needs integration test with a real `.tfplan` fixture.
