# Structured Error Parsing — Exploration

## Problem Statement

Terraform errors are notoriously verbose and use box-drawing characters, HCL
source references, and nested module paths. For AI agents consuming terraform
output, parsing this unstructured text wastes context and is error-prone.

This feature translates terraform stderr into structured JSON that tells a
consumer: what went wrong, which resource, what attribute, the error category,
and a suggested fix.

## Error Pattern Analysis

### Terraform Error Format (v1.x)

Terraform outputs errors in a diagnostic format:

```
Error: <headline>

  on <file>:<line>:<col>, in <block_type> "<name>":
  <line_num>: <source_line>

<body text>
```

Or for provider/API errors:

```
Error: <headline>

  with <resource_address>,
  on <file>:<line>:<col>, in resource "<type>" "<name>":
  <line_num>: <source_line>

<body text>
```

Warnings follow the same structure but use `Warning:` prefix.

### Common Error Categories

| Category | Pattern | Reliability |
|----------|---------|-------------|
| `missing_argument` | `The argument "X" is required` | HIGH — stable wording |
| `invalid_value` | `Invalid value for variable`, `Unsuitable value type` | HIGH |
| `syntax_error` | `Invalid expression`, `Invalid block definition` | HIGH |
| `provider_error` | `with <resource>`, body contains API/provider text | MEDIUM — body varies |
| `dependency_error` | `Cycle`, `Module not installed` | MEDIUM |
| `state_error` | `Error acquiring the state lock`, `state snapshot was created by Terraform` | MEDIUM |

### Regex Strategies

1. **Headline extraction**: `^│?\s*Error:\s*(.+)$` — highly reliable
2. **Resource extraction**: `with\s+(\S+),` — reliable when present
3. **Attribute extraction**: `argument\s+"([^"]+)"` or context from body — moderate
4. **File/location**: `on\s+(\S+):(\d+)` — reliable
5. **Category classification**: keyword matching on headline + body — moderate

### Known Limitations

1. **Format instability**: Terraform's error format is not a stable API. Box-drawing
   characters were added in ~1.4 and may change. We strip them for robustness.
2. **Provider errors are opaque**: The body text is provider-authored and has no
   standard structure. We can extract the resource but not reliably categorize
   the specific failure (rate limit vs permissions vs invalid param).
3. **Multi-error output**: Terraform may emit multiple `Error:` blocks in one run.
   We handle them as an array.
4. **Nested modules**: Resource addresses may be deeply nested
   (`module.vpc.module.subnets.aws_subnet.main`). We extract the full address.
5. **Terraform version variance**: Tested against 1.14. Older versions may differ.
6. **Warnings**: Parsed separately with severity "warning". Same structure.
7. **Non-diagnostic errors**: Some failures (binary not found, permission denied)
   produce free-form text without the diagnostic format. These get category
   `unknown` with the raw message.

### Viability Assessment

- **High confidence (ship it)**: Headline extraction, multi-error splitting,
  `missing_argument`, `invalid_value`, `syntax_error` categorization
- **Medium confidence (useful but fragile)**: Resource/attribute extraction from
  context lines, `provider_error` and `state_error` detection
- **Low confidence (best-effort)**: Specific provider error sub-categorization,
  suggestion generation for provider errors

### Design Decisions

1. Parse function is pure (stdin in, stdout out) — easily testable without terraform
2. Tests use static error text fixtures, not live terraform — fast and deterministic
3. jq handles JSON assembly — already a project dependency
4. Suggestions are conservative; "unknown" category gets no suggestion rather than
   a potentially misleading one
5. The function is internal (`_tfui_parse_error`) — it can be called by a future
   `--mode agent` strategy without changing the public API
