---
description: "CLI design decisions, pre-seeded cache, config loading, and terraform flag handling"
globs: ["cmd/**"]
---

Full spec: `docs/cli-ux.md`

# CLI Design

## Pre-Seeded Cache (`--plan`, `--state`)

```bash
tfui --plan ./plan.json
tfui --state ../terraform.tfstate
terraform show -json tfplan.out | tfui --plan -
tfui --plan ./plan.json --state ./state.json
```

When `--plan` or `--state` provided:
- `ServiceCache` is pre-seeded with parsed data; `ExecService` serves reads from cache
- Header shows `[pre-seeded]` badge
- Mutating hints hidden from status bar

## Design Decisions

Full I/O contract: **`docs/cli-io-contract.md`**

Core principle: tfui is a superset of terraform. All terraform flags work identically. Our additions use names terraform hasn't claimed.

Three interfaces:
- TUI: `tfui` (interactive BubbleTea)
- CLI: `tfui plan`, `tfui apply` (stdout tree/JSON)
- MCP: `tfui mcp` (future, structured protocol)

Behavior matrix:
- `tfui plan` → stdout: tree view, stderr: spinner (if TTY)
- `tfui plan --ci` → stdout: tree view, stderr: nothing
- `tfui plan -json` → stdout: NDJSON (terraform-compatible), stderr: nothing

Rules:
- `-json` → identical output to terraform's
- Default stdout → our enriched tree view
- `--ci` → suppress stderr
- Novel commands (`risk`, `phantom`, `blast-radius`) → our schema
- `show_spinner = !ci && isStderrTTY()`

Binary resolution:
- `--terraform-bin` > `--config terraform.bin=X` > `tfui.hcl terraform { bin = "..." }` > `"terraform"`

`--` passthrough:
- `splitPassthrough()` separates args at `--`
- ExtraArgs stored for `MacroService` (recorded in command flags)
- `ExecService` does NOT forward ExtraArgs (terraform-exec typed API)

Exit codes: `0` = success, `1` = error, `2` = changes present

## Config (`tfui.hcl`)

HCL format. Everything optional. No config file = standalone mode.

```hcl
terraform { bin = "terraform" }
member "modules/vpc" {}
member "modules/ecs" {}
cache { staleness_threshold = "5m" }
ai { enabled = true; provider = "bedrock"; region = "us-east-1" }
defaults {
  parallelism = 10
  lock = true
  var_file "common/tags.tfvars" {}
  plugin "risk" { level = "high" }
}
```

Two modes:
- Standalone (no tfui.hcl): CWD = terraform dir, `--chdir` = raw passthrough
- Project (tfui.hcl found): full resolution, chdir validated against members

Resolution chain: Root defaults → Child top-level → Workspace block → CLI flags → `--` passthrough

Key functions: `config.LoadRoot(dir)`, `config.LoadChild(dir)`, `config.Resolve(root, child, workspace)`
