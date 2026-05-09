# Policy-as-Annotation: Design Exploration

## Goal

Surface informational annotations about terraform plans — not enforcement (pass/fail),
but structured context so agents and operators can make informed decisions.

Example: "this plan opens port 22 to 0.0.0.0/0" or "a database is being destroyed"
as structured JSON warnings that downstream tools can consume.

## What jq Can Match

Terraform plan JSON (`terraform show -json`) exposes:

```json
{
  "resource_changes": [
    {
      "address": "aws_security_group.web",
      "type": "aws_security_group",
      "change": {
        "actions": ["create"],
        "before": null,
        "after": { "ingress": [...], "name": "web", ... },
        "after_unknown": { ... }
      }
    }
  ]
}
```

### Matchable with jq (what we implement)

| Criterion | jq approach | Complexity |
|-----------|-------------|------------|
| Resource type (exact) | `.type == "aws_db_instance"` | Trivial |
| Resource type (pattern) | `.type \| test("aws_security_group.*")` | Simple |
| Action (membership) | `.change.actions \| inside(["create","delete"])` or array intersection | Simple |
| Resource address (pattern) | `.address \| test("module\\.prod\\..*")` | Simple |
| Provider | `.provider_name \| test("aws")` | Simple |

### Possible but fragile (out of scope for v1)

| Criterion | Why fragile |
|-----------|-------------|
| Deep attribute matching (e.g., `after.ingress[].cidr_blocks`) | Schema varies by resource; nested arrays are hard to traverse generically |
| Cross-resource references | Would need multiple passes, graph traversal |
| Before/after diff analysis | Requires type-specific schema knowledge |
| Conditional matching (if X then check Y) | jq can do it, but expression authoring becomes a mini-language |

### What OPA/Sentinel Do Better

- Rego/Sentinel have type-aware traversal, iteration over nested structures
- They handle schema versioning, provider-specific attributes
- They provide formal proof of policy compliance

### Our Niche

We provide lightweight, zero-dependency (just jq) annotation that:
- Ships with sensible defaults
- Is configured via a single JSON file
- Produces structured output for agent consumption
- Runs locally without infrastructure

## Policy File Format

```json
{
  "rules": [
    {
      "id": "unique-identifier",
      "description": "Human-readable explanation shown in warnings",
      "severity": "critical|high|medium|low",
      "match": {
        "resource_type": "regex_pattern",
        "action": ["create", "delete", "update", "replace"]
      }
    }
  ]
}
```

### Design Decisions

1. **JSON over YAML** — jq parses it natively, no additional dependencies
2. **Regex for resource_type** — jq's `test()` handles PCRE-lite patterns
3. **Action as array** — match if any action in the rule matches the plan action
4. **No attribute inspection in v1** — scope limited to type + action matching
5. **Severity levels** — critical, high, medium, low (for prioritization, not enforcement)

## File Discovery

Search order (first found wins):
1. `.tfui-policy.json` in the terraform working directory
2. Walk parent directories up to filesystem root
3. Fall back to built-in default rules

This mirrors how tools like `.editorconfig` or `.eslintrc` work.

## Output Format

The policy evaluation result is a JSON object on stdout:

```json
{
  "warnings": [
    {
      "rule_id": "database-delete",
      "severity": "critical",
      "resource": "aws_db_instance.main",
      "message": "Database resource being destroyed"
    }
  ],
  "policy_summary": {
    "critical": 1,
    "high": 0,
    "medium": 0,
    "low": 0,
    "passed": 5
  }
}
```

## Built-in Default Rules

Ship reasonable defaults covering common high-risk operations:

- **database-delete**: Database resources being destroyed (critical)
- **database-replace**: Database resources being replaced (critical)
- **storage-delete**: Storage/bucket resources being destroyed (high)
- **network-delete**: VPC/subnet destruction (high)
- **iam-change**: IAM role/policy modifications (medium)
- **encryption-change**: KMS key or encryption config changes (medium)

## Limitations (Honest Assessment)

1. **Type-only matching** — cannot inspect resource attributes (e.g., specific ports)
2. **No cross-resource analysis** — each resource evaluated independently
3. **Pattern-based** — may over-match (e.g., all security groups, not just public ones)
4. **Single-file scope** — no rule inheritance or composition across files
5. **No suppression mechanism in v1** — cannot silence specific warnings per resource

These limitations are acceptable for an annotation layer. For enforcement, use OPA/Sentinel.

## Implementation Plan

1. `_tfui_policy_discover` — walk up directories for `.tfui-policy.json`
2. `_tfui_policy_defaults` — emit built-in rules as JSON to stdout
3. `_tfui_policy_load` — discover or fall back to defaults
4. `_tfui_policy_evaluate` — run rules against plan JSON, emit warnings
5. `tfui_policy` — public API function composing the above
6. CLI integration: `tfui plan --policy` flag or always-on
