---
layout: default
title: Risk policy result and capability controls
parent: Development
description: "Records reproducible OPA result, strict-error, capability and disclosure controls for D-03b."
audience: "Contributors reviewing D-03b and preparing D-03c or the first-version contract publication"
---

# Result and capability controls

Forty bounded standalone OPA v1.19.1 cases reproduce the D-01a result distinctions and reduced-capability
candidate on Linux/amd64 with Go1.25.5. Undefined output is not success, strict builtin errors require execution,
and disabled print is erased rather than rejected. Explicit source checks reject print/trace. Sensitive
redaction must occur before evaluation; capabilities do not prevent disclosure of unmarked input, authored
literals or raw compiler errors. This supports trusted local authors and fixed public diagnostics, with no
hostile-policy sandbox or general confidentiality claim.

These are disposable experiments, not implemented Risk behavior. The earlier prerequisite audit at
`7db0658724060509a1b0d5c8a1555f8ac3c431a8` is superseded: the protocol now exists. This run inspected baseline
`fdfdcb39b0f2fea08fcc9174e36bcde7952b949d`, the [D-01a candidates](first-version-candidates.md),
[D-03a manifest](d03-preparation/manifest.json) and [protocol](d03-preparation/protocol.md).
The [execution amendment](assets/boundary/protocol.md) was written before probes to reconcile their query,
result and capability differences. No production dependency, Apply behavior or tracking state changed.

## Reproduce the experiment

Run from this worktree on Linux with Go1.25.5 installed at the path in the runner, Python 3, GNU time,
prlimit and timeout. The runner needs module-download access and writes only its tracked output directory
and scratch builds/caches under `/tmp`. A rerun replaces observations and uses a new scratch path and loopback port.

```sh
python3 docs/development/risk-policy-evidence/assets/boundary/run.py
python3 docs/development/risk-policy-evidence/assets/boundary/test_controls.py
```

The [runner](assets/boundary/run.py), [Go probe](assets/boundary/main.go.txt) and
[acceptance checks](assets/boundary/test_controls.py) are retained. The test-writer ran seven failing evidence
checks before harness implementation; all seven pass against the captured observations. These check measured
fixture outcomes; they are not a production validator test suite.

[results.json](assets/boundary/output/results.json) contains every case's preparation/evaluation error and error
code, result/expression cardinality, typed value, fixture validation, context cause, process exit, raw stdout/stderr,
wall duration and RSS-file reference. Each case also has a process record and GNU time RSS record in that directory.
Print and trace fields are empty because no hook or tracer is installed; they are not instrumented call counters.
Errors and values are returned through a separate bounded JSON observation file, leaving raw process streams literal.
All fixture sources/inputs, capability JSON, module lock and harness bytes were hashed before the probes;
[frozen-sha256.json](assets/boundary/output/frozen-sha256.json) records them. Only synthetic markers appear in evidence.

## Result and strict-error observations

The query is `data.tfui.risk.v1.classification`. The valid fixture has exactly `level: low` and one reason
with `code: synthetic`, a nonblank message and empty paths. The disposable validator checks the fixture subset;
it does not establish all reason/path bounds in D-01a or validate paths against tagged input trees.

| Cases | Preparation | Evaluation and cardinality | Fixture outcome |
| --- | --- | --- | --- |
| `valid` | Success | One result, one expression, object | Valid |
| `undefined` | Success | No error, zero results | Undefined; resource failure in candidate |
| `empty_array`, `empty_object`, `null`, `scalar` | Success | One result, one expression; respectively array/object/null/number | Invalid, distinct from undefined |
| `missing_reasons`, `unknown_level` | Success | One object | Invalid |
| `conflict` | Success | `eval_conflict_error`, no result | Runtime complete-rule conflict; no fallback |
| `parse_error` | Parse failure | Not evaluated | Unavailable |
| `unknown_builtin` | Compile/type failure | Not evaluated | Unavailable |
| `builtin_executed_strict` | Success | `eval_builtin_error: div: divide by zero` | Evaluation failure |
| `builtin_executed_nonstrict` | Success | No error, zero results | Undefined |
| `builtin_unreachable_strict`, `builtin_unreachable_nonstrict` | Success | One valid object | Unqueried division never executed |

The conflict uses two rules whose values come from input, avoiding a static constant conflict. Strict errors
therefore improve detection of executed builtin failures, but cannot certify unqueried expressions. Query
cardinality and result validation remain separate from Go error checking. No multi-result query was generated;
the probe rejects non-unit cardinality without selecting an arbitrary value.

## Capability and disclosure observations

The experiment uses D-01a's exact positive builtin names plus `plus`, `minus`, `mul`, `div` for arithmetic controls.
The full pinned compiler feature/future-keyword set is retained in
[capabilities.json](assets/boundary/output/capabilities.json); no unrestricted registry is passed to preparation.
`AllowNet` is an empty Go slice (omitted by OPA's JSON `omitempty` tag). Local parsed modules, no supplied compiler,
custom builtins/resolvers, remote schemas, print hook or query tracer are used. Pinned option implementations
were inspected in `v1/rego/rego.go` and capabilities/compiler source; their hashes and the verified module origin
are retained with the experiment. The standalone module resolved without overriding transitive requirements;
this does not prove compatibility with tfui's dependency graph.

| Attempt, in queried and unqueried rules | Observed boundary | Conclusion |
| --- | --- | --- |
| `http.send` to counted loopback listener | Compile rejection; zero listener requests | HTTP call denied for these modules |
| `net.lookup_ip_addr("localhost")` | Compile rejection | DNS builtin denied; no resolver instrumentation, so no broader DNS claim |
| `time.now_ns` | Compile rejection | Clock builtin denied |
| `rand.intn`, `uuid.rfc4122` | Separate compile rejections | These randomness/UUID builtins denied |
| `opa.runtime` | Compile rejection | Runtime/environment builtin denied |
| `trace` in raw OPA path | Compile rejection | Trace absent from capabilities |
| `print(input.token)` with print disabled, raw path | Valid result; empty stdout/stderr and no hooks | Print erased; capabilities alone do not reject source use |
| `print` and `trace`, explicit parsed-AST candidate path | Fixed source rejection, before preparation | Keep explicit source check for first version |

Both raw and candidate print paths cover queried and unqueried rules. The AST prototype inspects expression
operators and nested call terms; production must apply equivalent checks to every selected module/helper rather
than searching source text. These controls establish the listed direct calls only, not an exhaustive adversarial
bypass proof.

| Synthetic disclosure case | Observation | Implication |
| --- | --- | --- |
| Marked value replaced with `{"kind":"sensitive"}`, then echo/interpolation | Marker absent from serialized input, results, errors and streams | Pre-evaluation redaction removes this value; echo is shape-invalid, interpolation is valid |
| Unmarked token interpolated by allowed `sprintf` | Marker in a valid reason | Trusted author/input boundary remains necessary |
| Token literal authored in policy | Marker in a valid reason | Capabilities cannot certify authored text confidentiality |
| Token used as unknown function name | Marker in raw compiler error | Never expose raw OPA errors as public diagnostics |

The marked fixture uses a preconstructed opaque node, not the production projector. End-to-end masking belongs
to US-001/US-003 and the [fidelity evidence](input-fidelity.md). The fixture exposes no neighboring resource or
external data. No real secret, real DNS target or permissive network positive control is used.

## External guards and retained failures

Before OPA, a 512MiB Python allocation under a 128MiB address-space ceiling raised `MemoryError` with exit 42.
A one-second timeout returned 124 and terminated a spawned descendant; `/proc` inspection found no running child
(a zombie pending reaping is explicitly allowed). See [preflight.json](assets/boundary/output/preflight.json)
and its raw process/RSS captures.

Every fetch/build uses 4GiB/120s, and every probe uses 1GiB/30s with core dumps disabled. A supervisor owns a separate
process group, drains bounded streams, kills on overflow/deadline and sends TERM then KILL to the entire child group.
It drains pipes after termination, records truncation/drain status and checks `/proc` for running members of every
trial group before accepting completion; zombies pending reaping are excluded.
Source is bounded at 64KiB and serialized input/output channels at 1MiB. Evaluation also has a cooperative 100ms
context; it is not the hard memory/termination bound. All forty probes completed with process exit 0 and no external
limit event; evaluator errors are intentional observations, not process successes interpreted as assessments.
GNU time retains RSS per trial; no cost budget, release-target feasibility or benchmark claim follows from these runs.

The first dependency fetch failed because the minimal environment lacked GOPATH/HOME; its
[raw failure](assets/boundary/output/fetch-initial.process.json) is retained. Adding only a scratch GOPATH allowed
checksum verification and resolution; no inherited HOME/configuration or credentials were added. The module origin
matches D-03a's `54896f9f28515d14b31a93e6ef5737bb85866e30` and its checksum is recorded in the lock/output.

## First-version conclusion and next bead

Keep the trusted-local-author candidate, mandatory pre-evaluation sensitive redaction, positive deterministic
capabilities, strict executed builtin errors, explicit AST rejection of print/trace, complete-result validation
and fixed public diagnostics. Raw errors belong only in this synthetic experiment. Disabled print alone is
insufficient to enforce the selected source contract; undefined output must explicitly fail classification.
The arithmetic additions are experimental controls and do not authorize broadening the production builtin list.

D-03c can use the pinned version, lock, capability artifact and bounded guard approach for combined tfui
build/size/cost measurements. It still must test the real combined dependency graph and all release targets;
standalone success cannot substitute for those measurements. D-01c should publish the tested contract under the
owner's delegated first-version authority, while keeping these limitations explicit.
