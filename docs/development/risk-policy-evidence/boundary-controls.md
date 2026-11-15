---
layout: default
title: Risk policy boundary control prerequisite audit
parent: Development
description: "Records the missing experiment prerequisites that prevent reproducing D-03b boundary controls in this checkout."
audience: "Contributors preparing D-03a and resuming D-03b"
---

# Boundary control prerequisite audit

D-03b cannot yet reproduce result and capability controls at inspected commit
`7db0658724060509a1b0d5c8a1555f8ac3c431a8`. This is a repository prerequisite audit,
not evaluator evidence. No OPA compilation or evaluation ran and no control is reported as passed.

The [D-01a candidates](first-version-candidates.md) provide the input, result and trust choices.
The required D-03a protocol (bead `terraform-ui-4xa.6`) is absent from this checkout. The
[runtime research](../risk-policy-runtime-research.md#version-and-build-boundary) identifies
OPA v1.15.0 only as a research pin and explicitly requires maintenance review before selecting
an experiment version. It supplies neither executed controls nor a bounded harness protocol.

## Reproduce the audit

From the repository root, inspect the committed baseline:

```sh
git rev-parse HEAD
git log -5 --oneline
git ls-files '*risk*' '*protocol*' '*experiment*' '*spike*'
rg -n 'D-03a|4xa\.6|target matrix|measurement method|external harness|protocol' docs tasks
```

The baseline contains only `first-version-candidates.md` in the evidence directory.
The PRD requires a declared fixture/toolchain/baseline/target/measurement protocol before
execution, while D-01a's handoff requests that future D-03 experiments retain modules,
inputs, harness invocation and raw channels. Neither is the missing protocol itself.
The test-writer independently inspected the checkout and confirmed this gap; no behavioral
test was added for this prose-only audit.

## Resume boundary

D-03a must first supply its reviewed candidate version, declared fixtures and protocol,
including the external process time/memory limits and output/error capture method.
D-03b can then reproduce the undefined/defined-empty/malformed/conflict/compile/builtin
failure cases and executed/unreachable capability probes against that recorded baseline.
The experiment must retain the exact version-specific capabilities artifact it uses.
Choosing a version or protocol here would bypass the bead's explicit dependency ordering.

This finding does not reject the D-01a candidate or reopen its accepted scope. It establishes
no capability protection, confidentiality, runtime cost or sandbox guarantee. No production
code, dependencies, Apply behavior or tracking state changed.
