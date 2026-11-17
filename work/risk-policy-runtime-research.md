# Embedded policy evaluation: capabilities and failure semantics

Research for [Establish embedded OPA capabilities and failure semantics](https://github.com/lmarqs/terraform-ui/issues/111), part of [Design the risk policy contract architecture](https://github.com/lmarqs/terraform-ui/issues/107). Inspected 2026-11-14. This is source research, not a runtime spike or an accepted contract.

## Version and build boundary

The candidate for a disposable experiment is **OPA v1.15.0**, commit `f9e7302d69595f413840344bd642653d2a99766e`. Its [module declaration](https://github.com/open-policy-agent/opa/blob/f9e7302d69595f413840344bd642653d2a99766e/go.mod) requires Go 1.25.0, matching tfui's declared baseline. This establishes declared toolchain compatibility only; it does not prove the combined dependency graph builds. It is a research pin, not a recommendation to ship this older release without reviewing current maintenance and advisories.

The tfui baseline inspected is `394a60ae46ee9d6fbb6efeffe5947fce3ac0980a`: [go.mod](https://github.com/lmarqs/terraform-ui/blob/394a60ae46ee9d6fbb6efeffe5947fce3ac0980a/go.mod) declares Go 1.25.0; [.goreleaser.yaml](https://github.com/lmarqs/terraform-ui/blob/394a60ae46ee9d6fbb6efeffe5947fce3ac0980a/.goreleaser.yaml) builds Linux and Darwin on amd64 and arm64. No supported-target build, binary-size, latency, or memory measurement was performed. A module's dependency list alone does not establish what the final executable links.

## Evaluation matrix

| Case | Source-backed observation | Consequence for the design discussion |
| --- | --- | --- |
| Preparation | `PrepareForEval` parses and compiles modules/query and returns an error on failure. A prepared query accepts per-evaluation input. [Rego API](https://github.com/open-policy-agent/opa/blob/f9e7302d69595f413840344bd642653d2a99766e/v1/rego/rego.go#L1726) | Distinguish preparation failure from an assessment. Reuse must respect source identity. |
| Undefined query | An empty `ResultSet` means undefined. Normal evaluation returns `nil, nil` when there are no results. [Result representation](https://github.com/open-policy-agent/opa/blob/f9e7302d69595f413840344bd642653d2a99766e/v1/rego/resultset.go), [evaluation path](https://github.com/open-policy-agent/opa/blob/f9e7302d69595f413840344bd642653d2a99766e/v1/rego/rego.go#L2315) | Checking only the Go error is insufficient. Undefined must remain distinct from a defined empty collection. |
| Defined result | Expressions contain arbitrary values; the API also exposes bindings. `Allowed()` recognizes a specific boolean result shape. [Result representation](https://github.com/open-policy-agent/opa/blob/f9e7302d69595f413840344bd642653d2a99766e/v1/rego/resultset.go) | A risk result needs tfui validation of cardinality, shape, and domain semantics. Boolean authorization convenience is not that validator. |
| Builtin failure | The evaluator records ordinary builtin errors and can continue; halt errors propagate. [Interpreter](https://github.com/open-policy-agent/opa/blob/f9e7302d69595f413840344bd642653d2a99766e/v1/topdown/eval.go#L2085). `StrictBuiltinErrors(true)` makes builtin errors fatal. [Rego API](https://github.com/open-policy-agent/opa/blob/f9e7302d69595f413840344bd642653d2a99766e/v1/rego/rego.go#L1250) | Use strict behavior in the candidate harness; distinguish evaluated failures from rules that simply do not match. Strict mode does not execute unreachable expressions. |
| Runtime conflict/type/error | Topdown error codes distinguish cancellation, conflict, type, builtin and internal errors. Conflict includes competing values for one complete document. [Error definitions](https://github.com/open-policy-agent/opa/blob/f9e7302d69595f413840344bd642653d2a99766e/v1/topdown/errors.go) | Preserve failure identity internally. A syntactically valid but contract-invalid result remains tfui's error to detect. |
| Cancellation | The Rego adapter connects context completion to interpreter cancellation; the interpreter checks the cancellation state. [Adapter](https://github.com/open-policy-agent/opa/blob/f9e7302d69595f413840344bd642653d2a99766e/v1/rego/rego.go#L2300), [check](https://github.com/open-policy-agent/opa/blob/f9e7302d69595f413840344bd642653d2a99766e/v1/topdown/eval.go#L411) | Cooperative cancellation is supported. This evidence provides neither a hard termination deadline nor a memory limit for evaluation, compilation, or builtin work. |

## Capability and diagnostic boundary

Compiler capabilities list available builtins and language features. `Capabilities` and `UnsafeBuiltins` need special care with caller-supplied compilers: their module-compilation options are not automatically applied to that compiler. With raw-string modules, print calls are erased unless enabled; hooks and query tracers are separate controls. [Rego API](https://github.com/open-policy-agent/opa/blob/f9e7302d69595f413840344bd642653d2a99766e/v1/rego/rego.go#L1193). The candidate should avoid a supplied compiler until its configuration is explicitly tested.

The builtin registry includes HTTP, DNS, random values, current time, and runtime information. It marks nondeterministic builtins, but that marker is not an application trust policy. [Registry](https://github.com/open-policy-agent/opa/blob/f9e7302d69595f413840344bd642653d2a99766e/v1/ast/builtins.go). The `NondeterministicBuiltins` option controls partial evaluation, not a general deny switch for normal evaluation. [Option implementation](https://github.com/open-policy-agent/opa/blob/f9e7302d69595f413840344bd642653d2a99766e/v1/rego/rego.go#L964).

`AllowNet` distinguishes nil (unrestricted) from empty (no allowed hosts). Its struct comment describes schema-reference fetching, while the HTTP implementation also checks it; this source/comment mismatch argues against treating the field as a complete sandbox. [Capabilities](https://github.com/open-policy-agent/opa/blob/f9e7302d69595f413840344bd642653d2a99766e/v1/ast/capabilities.go#L82), [HTTP checks](https://github.com/open-policy-agent/opa/blob/f9e7302d69595f413840344bd642653d2a99766e/v1/topdown/http.go#L374). A positive builtin allowlist, no custom external resolvers, and no remote schema loading are candidates to test, not a proven isolation boundary.

Inference: suppressing print and trace channels does not prevent a policy from returning sensitive input in its result. Error messages and source locations also need review. The trust/sensitive-data decision must determine what values policies receive and what tfui may render; the evaluator cannot infer Terraform sensitivity from arbitrary JSON.

## Smallest falsification experiments

These experiments belong to the later prototype tickets; none ran here.

1. **Result distinction:** evaluate undefined entrypoint, defined empty collection, scalar, malformed object, and conflicting complete rules. Pass when the harness distinguishes each outcome and no undefined/error case passes contract validation. A failure changes the adapter or result boundary.
2. **Builtin errors:** use invalid builtin inputs in an executed branch and an unreachable branch, with strict mode on and off. Pass when executed errors fail under the selected mode; document the unreachable case. A mismatch revisits failure promises.
3. **Capabilities and disclosure:** attempt HTTP, DNS, current time, randomness, print, trace, and sensitive-value interpolation under a candidate allowlist. Capture all output and deny network at the harness boundary to detect attempts. Pass only against the owner's chosen trust model; any escape changes capabilities or the input/output boundary.
4. **Cancellation/cost:** run bounded large-plan and expensive-policy fixtures, with an external harness deadline and resource cap. Measure cancellation response, compilation/evaluation cost, and memory; never infer a hard limit from cooperative cancellation. Unacceptable costs or control gaps reopen runtime integration.
5. **Build compatibility:** use a disposable module on all four tfui targets, record exact toolchain/dependency versions and binary-size delta. Build failure invalidates this candidate; successful compilation does not settle acceptable runtime cost.

## Research resolution

The required API distinctions and controls exist in the inspected source, with the limitations above. A minimal candidate uses `github.com/open-policy-agent/opa/v1/rego`, local module snapshots, a direct document query, strict builtin errors, an explicit capability set, disabled print/tracing, a context, and separate tfui result validation. This is a testable candidate, not the contract decision.

Still open: owner trust model and sensitive input access; completeness/aggregation semantics; validated report shape; reload identity; acceptable resource costs; measured cancellation behavior and supported builds. Those are already assigned to downstream map tickets. No production dependency or application file changed. Markdown whitespace/link-source checks ran; build, lint, unit, coverage, and macro checks did not run because this branch contains only a research note and no evaluator.
