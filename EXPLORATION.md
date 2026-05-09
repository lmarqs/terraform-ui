# EXPLORATION: `tfui state` subcommand

## Goal

Provide a compact structured summary of terraform state for AI agents and humans.
Instead of N calls to `terraform state list` + `terraform state show`, one command
returns resource counts by type, module grouping, and a full resource list.

## Design Decision: `state pull` vs `state list` + `state show`

### Option A: `terraform state pull`
- Single command, returns full state JSON
- Parseable with jq in one pass
- Contains all resource metadata (type, module, provider, addresses)
- Works with local and remote backends
- Output is the raw state file (version 4 format)

### Option B: `terraform state list` + N `terraform state show`
- Requires N+1 commands
- Each `state show` returns a single resource
- Much slower for large states
- Output format varies by resource type

### Decision: **Option A — `terraform state pull`**

Rationale:
- Single command = faster execution, simpler error handling
- Full JSON = parse once with jq, extract all needed fields
- Agent use case benefits most from speed (agents call this before every plan)
- State JSON v4 has a stable schema: `.resources[]` with `type`, `name`, `module`, `provider`

## State JSON v4 Schema (relevant fields)

```json
{
  "resources": [
    {
      "module": "module.vpc",        // absent for root module
      "mode": "managed",             // "managed" or "data"
      "type": "aws_instance",
      "name": "web",
      "provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
      "instances": [...]
    }
  ]
}
```

## Output Modes

### Agent mode (`--mode agent`)
JSON to stdout — machine-parseable, no UI chrome:
```json
{
  "total_resources": 42,
  "by_type": { "aws_instance": 3, "aws_security_group": 5 },
  "by_module": { "module.vpc": 12, "root": 22 },
  "resources": [
    { "address": "aws_instance.web", "type": "aws_instance", "module": "root", "provider": "aws" }
  ]
}
```

### Plain/simple/rich modes
Human-readable summary to stdout (UI feedback on fd3 if applicable):
```
State: 42 resources
  aws_instance         3
  aws_security_group   5
  ...
Modules: vpc (12) | root (22)
```

## Implementation Plan

1. `_tfui_render_state_json` — jq filter on state pull output, produces agent JSON
2. `_tfui_render_state_text` — jq filter producing human-readable text
3. `tfui_state` — public API: pull state, render based on mode
4. `_tfui_cli_state` — CLI subcommand with `--dir`, `--mode` options

## Viability

- `terraform state pull` works with all backends (local, S3, GCS, etc.)
- jq is already a required dependency
- No new dependencies needed
- State JSON v4 format is stable since Terraform 0.12
- For states with no resources, output a clear "empty state" message
- Data sources (mode=data) are excluded from counts (they're read-only lookups)
