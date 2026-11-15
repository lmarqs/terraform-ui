---
layout: default
title: Risk policy Context and source identity evidence
parent: Development
description: "Records configuration traces, disposable source identity experiments and the D-02 first-version conclusion."
audience: "Contributors publishing D-01c and implementing source selection and assessment lifecycle"
---

# Context and source identity

D-02c examines the [D-01b source/lifecycle candidate](source-template-decisions.md) against checkout
`bedb40b8d26e02e22f8cd65a5bc98645ac9a294b`. This is evidence from disposable prototypes and current code,
not a shipped `risk_policy` setting, Rego evaluator or production assessment lifecycle. The
[contract](../../reference/risk-policy-contract.md) owns eligibility and failure semantics.

## Criteria and method

The test-writer recorded the [protocol](assets/context-source/protocol.md) before execution and wrote
behavioral assertions before the candidate implementation. Silent input loss rejects an input candidate;
observed mixed source bytes or stale A publication rejects a lifecycle candidate. Selected invalid or deleted
source must never become built-in success. Tests assert loaded bytes, provenance and publication results,
not only an options map. A missing candidate first produced the retained
[red capture](assets/context-source/candidate-red.txt).

Three evidence layers remain distinct:

- Go overlays execute actual configuration and app code with disposable HCL files. These observe current
  behavior, including rejected assumptions; passing baseline assertions do not endorse those behaviors.
- A Python model exercises the proposed path precedence, immutable module bytes, framed digest and generation
  guard with an explicitly ordered evaluation/publication schedule. It does not implement HCL parsing,
  Rego compilation, cancellation, or a concurrent app integration.
- D-02a/b assets are rerun rather than treating their earlier captures as sufficient evidence.

Toolchain: Go 1.25.5 linux/amd64 and Python 3.13.14. No OPA, provider or network service is used. All fixtures
are synthetic, local and disposable. Commands run from this worktree root; temporary files and Go build caches
stay under `.ralph-tui/`.

## Current code trace and rejected integration assumptions

`cmd/tfui/root.go` loads root HCL before building the session; malformed root HCL returns a command error,
while absent root HCL selects zero-config. Root plugin settings reach the registry from `Resolve(root,nil,...)`.
`internal/config/hcl_root.go` discards unrecognized defaults through `Remain` and skips generic plugin expressions
that cannot be evaluated. It retains no declaring-directory provenance in resolved option strings.
Therefore a dedicated selector cannot reuse that extraction route without strict parsing changes.

`internal/config/hcl_child.go` explicitly rejects Member top-level and workspace `plugin` blocks. This is
current behavior, replacing the historical silently ignored blocks. The proposed dedicated `risk_policy`
Member attribute is unsupported. Do not restore Member plugin inheritance or pass manually assembled option
maps off as supported HCL. `mergePlugins` still merges per key when called with constructed settings; this is
not complete-policy source replacement.

`internal/ui/app.go` loads child configuration at the startup directory, including initial `-chdir`; it ignores
that load error. Context switches load the new child and log errors, then replace the child pointer even on
failure. Root-as-Member startup attempts to parse root HCL as child; locked blocks can fail and be discarded.
`withResolvedConfig` resolves the actual workspace against the resulting root/child settings. Root configuration
is retained from startup rather than reread at every boundary. These paths are insufficient for strict policy
selection or source reload: malformed Member selection must not silently resurrect the inherited policy.

Context replacement itself is atomic and emits previous/next snapshots. It is not a policy publication guard.
The current Risk plugin only exposes explicit `Analyze(summary)` and has no source snapshot, evaluator,
Context-change handler or generation identity. There is no real policy A/B race to execute yet. The app's
`requireIdle` also rejects Context switching during operations that hold its busy lock. Future asynchronous
assessment must fit that lifecycle; the ordered prototype does not prove app-level cancellation or headless
termination.

## Reproduce and results

```sh
python3 -B docs/development/risk-policy-evidence/assets/context-source/test_candidate.py
bash docs/development/risk-policy-evidence/assets/context-source/run-current.sh
python3 -B -m unittest discover -s docs/development/risk-policy-evidence/assets/fidelity -p test_fidelity.py -v
bash docs/development/risk-policy-evidence/assets/acquisition/run.sh
mkdir -p .ralph-tui/d02c-fidelity/tmp .ralph-tui/d02c-fidelity/cache
cp docs/development/risk-policy-evidence/assets/fidelity/baseline.go.txt .ralph-tui/d02c-fidelity/main.go
TMPDIR="$PWD/.ralph-tui/d02c-fidelity/tmp" GOCACHE="$PWD/.ralph-tui/d02c-fidelity/cache" go run ./.ralph-tui/d02c-fidelity
```

The [Go overlay](assets/context-source/current_test.go.txt) passes two experiment functions;
[current output](assets/context-source/current.txt) retains the actual diagnostics. The
[Python suite](assets/context-source/test_candidate.py) passes eleven tests against the
[candidate](assets/context-source/candidate.py); [successful output](assets/context-source/candidate-green.txt)
is retained (capture trailing whitespace removed). The coordinator independently reran both. Initial red used an unavailable test adapter, not
production failures. The additional [selection-race red](assets/context-source/selection-race-red.txt) exposed
an actual prototype defect: digest-only rechecking accepted a changed selected path with identical bytes.
The corrected gate also rechecks the resolver's selected path. Code review found that a subdirectory named
`ignored.rego` was incorrectly rejected. The test-writer reproduced that failure in
[subdirectory red](assets/context-source/subdirectory-red.txt); the corrected loader ignores directories
without recursion while still rejecting eligible symlinks.

| Experiment | Observed result and boundary |
| --- | --- |
| Root/Member/workspace and relative origin | Model loads distinct winning bytes from root, Member or exact workspace paths; inherited selection retains root origin; omitted inherits and empty resets. Exact match beats first glob. HCL-to-model plumbing is unimplemented. |
| Zero-config, absent and empty | Real absent generic option remains absent; empty generic string is retained. Zero-config app retains initial Chdir. Built-in reset is a selection-model result, not an existing evaluator. |
| Invalid configuration | Real malformed root fails; Member dedicated attributes and plugin blocks fail parsing; invalid generic root expression disappears. Invalid Member startup/switch falls back to root execution settings. Reject these fallback routes for policy selection. |
| Module identity | Exact source bytes and framed SHA-256 match hand-derived expected data; copied directories match; rename, line ending, helper edits/addition/deletion change digest; test and unrelated files do not. |
| Invalid/deleted source | Missing path, directory with no modules, selected test module, selected/ancestor/module symlink raise ValueError. Deleting a source before publication rejects it. Rego syntax and actual permission-denied behavior are not evaluated. |
| Mixed read | Injected differing first/verification reads reject acquisition. The retained mapping is immutable. This detects observed changes, not every possible filesystem swap. |
| A begins, B begins, A completes | A is rejected before and after B publication, even though A work never cooperates with cancellation. Same-Context new generations also reject old tickets. This is a deterministic schedule, not a running app evaluator race. |
| Source bytes/selection edited while evaluating | Ticket retains its original bytes; source edit/deletion or a changed selected path rejects publication. The resolver is injected; production must reread strict HCL including provenance. |

The model's workspace matching uses Python `fnmatch` for these simple ASCII fixture patterns; production must
keep Go `filepath.Match` semantics. The model has no built-in loader, compilation cache or assessment reuse path.
A resolver callback demonstrates path rechecking only, not strict configuration parsing, origin-only changes,
or full Context/plan identity construction. Each `begin` takes caller-supplied Context/plan identity and starts
a generation; production must actually invoke it on all invalidation boundaries. These limitations prevent
claiming that the model alone proves the full proposed lifecycle.

## First-version conclusion from D-02

Retain the D-01a tagged lossless representation and D-01b declaration-relative complete-source replacement,
framed source digest, immutable compilation snapshot and generation-checked publication as first-version
implementation choices. These are contributor conclusions under delegated authority, not owner-selected
exact interfaces. The bounded experiments support feasibility; they reject reuse of the unmodified display,
configuration and startup seed paths.

Fresh D-02a reproduction: all 11 projection/independence experiments pass. The real Go probe again reports
array-order and null-field changes as phantom, display integer `9007199254740992` and exponent `1000`, versus
number-preserving tokens `9007199254740993` and `1e3`. Thus display strings remain rejected as policy evidence,
and phantom normalization remains advisory. The [fidelity page](input-fidelity.md) documents the discovered
opaque-ancestor mask-validation limitation; the candidate is not a complete validator.

Fresh D-02b reproduction: all five acquisition experiment functions pass their observational assertions.
Seeded direct reads use zero plan/show calls; live Plan then explicit Risk.Analyze uses one plan/one show;
PlanJSON replans. Real seeded Session startup at root uses zero plans, while initial Member Chdir makes one
unintended plan. The nonexistent-binary Member case still fails rather than reviewing its seed. These are
rejected product assumptions, not fixed behavior. See [counts and exact boundaries](acquisition-counts.md).

US-001 must retain original evidence at acquisition, preserve the seed short-circuit and fix initial-Chdir seed
propagation with real Session tests. US-002 must add a strict, provenance-preserving dedicated HCL selector and
separate assessment failure from usable Plan review. US-004 must integrate generation, Context/plan/selection
identity, boundary reload and error/cancellation terminal statuses. Do not retrieve reviewed evidence by
replanning through PlanJSON. Apply and Terraform pass-through behavior remain outside this experiment.

The model does not establish filesystem atomicity against adversarial swaps, OPA validity of module bytes,
real permission-denied behavior across platforms, runtime memory/termination bounds, or production A/B race
safety. D-03 owns evaluator controls and costs; production stories must exercise those controls through the
real session. A bounded model pass cannot close those integration requirements.

## Verification for this delivery

- Source candidate: **PASS**, eleven Python behavior experiments.
- Current config/UI overlay: **PASS**, two Go experiment functions.
- D-02a reproduction: **PASS**, eleven Python experiments and the real Go probe; observed display loss retained.
- D-02b reproduction: **PASS** observational baseline; initial-Chdir seed preservation remains **REJECTED**.
- `mise run docs:build`: **PASS**.
- `mise run docs:check`: **PASS**, 77 HTML files including internal link hashes.
- `git diff --check` and new evidence whitespace check: **PASS** after removing capture trailing spaces.
- Quick `mise run check:build` and affected-package `mise run test:unit`: **NOT APPLICABLE / not run**, no
  production package changes; the disposable Go overlay and probe were compiled and executed explicitly.
- Done `mise run check:lint`, `mise run test:unit`, `mise run check:build`: **NOT APPLICABLE / not run**,
  evidence assets and documentation only.
- `mise run test:coverage`: **NOT APPLICABLE / not run**, no production branches changed.
- `mise run test:macro`: **NOT APPLICABLE / not run**, no UI behavior changed.
