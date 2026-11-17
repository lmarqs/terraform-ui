---
layout: default
title: Risk policy release-target costs
parent: Development
description: "Records reproducible combined tfui/OPA build, size, evaluation and memory measurements for D-03c."
audience: "Contributors reviewing embedding costs before publishing the first-version risk policy contract"
---

# Release-target and evaluation costs

Accept the measured embedding cost for the first version under the owner’s delegated authority: all four
release targets build, and the small policy evaluates 1000 independent mutations without a guard failure.
This is a contributor-selected judgment, not an owner-selected numerical budget.

This evidence compares ordinary tfui with a disposable tfui binary that embeds and explicitly invokes OPA
v1.19.1. It measures buildability and costs; it does not implement policy acquisition, lifecycle, reporting or
Apply changes. The [D-03b controls](boundary-controls.md) separately establish the tested result and capability
boundary. Production adoption and maintenance/advisory review remain separate from this cost experiment.

## Reproduce the measurements

Run on Linux/amd64 with the exact Go1.25.5 path in the runner, Python 3, GNU time, prlimit, timeout, git and diff.
Module-download access is required. From the repository root:

```sh
python3 docs/development/risk-policy-evidence/assets/costs/run.py
python3 docs/development/risk-policy-evidence/assets/costs/test_costs.py
python3 docs/development/risk-policy-evidence/assets/costs/validate.py
python3 docs/development/risk-policy-evidence/assets/costs/report.py
```

The [execution amendment](assets/costs/protocol.md) was declared before measurements, refining the
[D-03a protocol](d03-preparation/protocol.md) and [manifest](d03-preparation/manifest.json) with the
[D-01a candidates](first-version-candidates.md). The runner exports immutable baseline
`60167777494b622ca55a203f05deb8e127357d23` twice under `/tmp`. Current run HEAD
`8c43cf994391c917287cb4628296ac48b0cefb2c` has identical production dependencies and release configuration.
No live Go module or production source is edited. Scratch caches and executables can be deleted after review;
the tracked harness, fixtures, patch, locks and hashes reproduce them.

The candidate [main patch](assets/costs/output/candidate-main.patch) adds an early `__d03` dispatch and the
[probe](assets/costs/probe.go.txt) in the scratch tfui main package. This reachable path prepares a local policy
and evaluates independent inputs. A blank import or standalone evaluator would not establish combined feasibility.
The standalone runtime comparison uses the identical probe and resolved candidate dependency graph.

The release matrix matches `.goreleaser.yaml`: Linux and Darwin, amd64 and arm64. Every variant uses:

```text
GOTOOLCHAIN=local CGO_ENABLED=0 GOMAXPROCS=2 GOOS=<os> GOARCH=<arch>
go build -p=2 -trimpath -buildvcs=false \
  -ldflags='-s -w -X main.version=d03-experiment' -o <artifact> ./cmd/tfui
```

Stripping/version flags match release intent; trimpath, disabled VCS and CGO=0 are controlled experiment
choices, not a claim of byte-identical Goreleaser artifacts. Dependencies are fetched before timed builds;
fetch and prefetch have separate process/RSS records. Each target has five cold pairs with alternating order,
each using a new empty private GOCACHE, then one untimed priming build per variant and five cached pairs.
Module downloads share a prefetched scratch cache; no global cache is cleared. Resolved baseline go.mod is
unchanged; module prefetch adds go.sum checksums, retained alongside the original manifest hashes.

[results.json](assets/costs/output/results.json) retains raw channels, exit/guard outcomes, supervisor wall
seconds, per-sample bytes/hashes, RSS, graphs, toolchain and host metadata. Raw GNU time files retain command
elapsed/user/system seconds, faults and I/O. [summary.md](assets/costs/output/summary.md) gives all five-sample
medians and full ranges, paired duration deltas and size deltas without removing outliers.
Supervisor wall time includes launch, pipe draining and process-group checks. GNU command elapsed time is
separate and quantized to hundredths of a second; its shortest samples can legitimately read zero.
Neither measures a dedicated-host performance guarantee.

## Runtime workload and limits

The frozen [fixtures](assets/costs/fixtures.json) contain D-03a's minimal mutation, an explicit complete low
classification and an update rule returning that same object. This is a small representative policy; it does
not model a complete built-in rule set or worst-case custom source. Each process constructs N=1,100,1000
resource-local inputs with distinct addresses. The evaluator receives one resource at a time, never the list.
Correctness checks require one result/expression and the exact level/reason object for every call.

Cold runtime prepares once and evaluates N resources. Warm runtime prepares once, performs one untimed prime,
then 100 passes over N resources; preparation is not hidden inside the warm loop. Five fresh processes run
for each N/phase in combined, standalone and overhead variants. Input construction/parse includes fixture
read/decode with UseNumber, cloning, address generation and bounded serialization, outside the Eval timer.
Source AST parsing, preparation and first measured Eval have separate nanosecond fields. Eval totals sum
only `query.Eval`, excluding context creation and result validation; whole-process times include those costs.
The overhead variant uses the same standalone executable to decode and dispatch without preparation/Eval;
it retains library initialization and does not represent a separate small tfui baseline or full policy overhead.

Only Linux/amd64 runtime is measured. Linux/arm64 and both Darwin targets require native runners for runtime
costs; their cross-builds establish buildability only. No emulation is presented as native evidence.

Preflight runs before dependencies/builds: a 512MiB allocation under 128MiB address space must fail, and a
one-second timeout must kill a spawned descendant. Every build/fetch uses 4GiB/120s and every runtime probe
1GiB/30s, with core dumps disabled. These are experimental safety ceilings, not accepted performance budgets.
The separate supervisor drains bounded channels, kills the whole process group, and checks running survivors
and complete pipes. Zombies pending reaping are excluded. Source is limited to 64KiB and generated input and
channels to 1MiB. Context cancellation is cooperative and does not establish hostile-policy isolation.
GNU time reports maximum resident set for the command and descendants; this is not summed simultaneous
memory across compiler processes, total allocated bytes or a hard product memory bound.

The 256-equality fixture is separate from costs, exercised with an already cancelled context and with a 100ms
context. A bounded expression may complete before its deadline; only the recorded outcome establishes whether
cancellation actually occurred. No larger workload is substituted to force a timeout.

## Measured outcomes and first-version judgment

Measured on 2026-11-15, Linux/amd64, Intel i7-12700H, about 19.4GiB reported host RAM, GOMAXPROCS=2.
All 80 timed release builds, eight priming builds, 90 runtime samples and two bounded cancellation probes
completed with exit 0, no external-limit event, complete untruncated streams and no running process-group
survivors. The setup smoke invocation also returned the expected classification. All repeated artifacts for
a given target/variant have identical hashes. Cross-build success does not establish signing, native launch
or native performance on macOS/arm64.

The combined graph resolves OPA v1.19.1 without manual dependency overrides. Go minimum-version selection
upgrades shared requirements, including Cobra 1.8.1 → 1.10.2, go-runewidth 0.0.16 → 0.0.19, pflag
1.0.5 → 1.0.10 and the existing golang.org/x modules. The measured candidate delta includes these graph
changes; it is not an isolated attribution of every byte to OPA alone. Full application regression tests
against those upgrades belong to production integration, not this evidence-only run. Module origin matches `54896f9f28515d14b31a93e6ef5737bb85866e30`;
locks and graph are retained. Prior D-03b control evidence remains applicable to the same OPA source and
frozen capability profile, but its entire matrix was not rerun in the combined binary. This experiment
adds combined classification/cancellation evidence, not an exhaustive revalidation of every transitive change.

Build medians below use supervisor wall seconds; complete ranges, command CPU times, peak RSS and paired
absolute/relative deltas are in the generated summary. Sizes are uncompressed bytes, invariant across samples.

| Target | Baseline bytes | Candidate bytes | Extra bytes | Increase | Cold seconds baseline → candidate | Cached seconds baseline → candidate |
| --- | --- | --- | --- | --- | --- | --- |
| linux/amd64 | 9,429,176 | 26,165,432 | 16,736,256 | 177.49% | 10.603 → 23.183 | 0.260 → 0.549 |
| linux/arm64 | 8,847,544 | 24,576,184 | 15,728,640 | 177.77% | 10.862 → 23.681 | 0.264 → 0.533 |
| darwin/amd64 | 9,617,808 | 26,632,224 | 17,014,416 | 176.91% | 10.611 → 23.294 | 0.279 → 0.603 |
| darwin/arm64 | 9,065,842 | 25,205,330 | 16,139,488 | 178.03% | 10.845 → 23.851 | 0.292 → 0.623 |

The extra binary cost is 15.0–16.2MiB (about 177–178% over baseline). Paired median cold build increases
are 12.6–13.0 seconds; cached increases are 0.27–0.33 seconds. The largest observed candidate build RSS is
366.1MiB, versus 249.7MiB for baseline. These maxima describe the observed commands, not simultaneous
aggregate memory of all compiler children.

Combined runtime medians follow. Cold Eval totals cover one pass over N resources; warm totals cover 100
passes. Per-Eval warm cost divides each process’s sum by its actual call count before taking the median.
Preparation stays separate; there is no meaningful baseline tfui Eval latency because baseline has no evaluator.

| N | Cold process supervisor ms | Cold Eval total ms | Warm Eval µs per mutation | Warm 100-pass process supervisor ms | Cold / warm RSS MiB |
| --- | --- | --- | --- | --- | --- |
| 1 | 11.957 | 0.113 | 35.745 | 16.637 | 16.250 / 18.906 |
| 100 | 17.507 | 3.511 | 23.378 | 283.539 | 19.062 / 22.754 |
| 1000 | 57.614 | 25.009 | 23.049 | 2669.778 | 25.938 / 27.965 |

For the combined N=1 cold case, median source parsing is 0.119ms, preparation 0.346ms and first Eval
112.645µs. At N=1000, input construction/parse is 14.421ms cold, distinct from 25.009ms total Eval.
At N=1000 warm, 100,000 checked calls take 2.305 seconds summed Eval, 2.670 seconds supervisor wall,
and at most 28.375MiB reported RSS across the five processes. The standalone counterpart measures
23.559µs per warm Eval; the dispatch-only counterpart measures 0.045µs, with median RSS 24.863MiB
and 17.969MiB respectively. These contextualize initialization/dispatch costs; they are not a subtraction-based
estimate of production policy overhead.

The already-cancelled 256-equality query returns an evaluation-cancellation error with `context canceled`,
zero results and process exit 0 because the harness successfully records the expected error. The 100ms
case completes successfully before deadline. It establishes no forced deadline interruption or worst-case
termination guarantee. Both remain below external guards; see the bounded process records.

**Decision:** proceed with the embedded Go/Rego approach for the first version and retain this version/profile
as the measured candidate for D-01c. The binary growth and doubled cold compilation are material, accepted
tradeoffs for a single distributable executable with project-owned complete policies. Cached builds remain
subsecond in this experiment, and preparation-once with independent resource evaluation has workable observed
costs at the declared plan sizes. The measured build matrix gives no reason to reopen ADR-0024 on portability.
No numerical budget or performance promise is inferred from the safety ceilings or these sample values.

This judgment does not select a production dependency automatically: D-01c must carry forward the pinned
maintenance/advisory limitations, trusted-author boundary, redaction and source controls. It must not advertise
runtime measurements on the three unmeasured targets. Full built-in/custom modules, larger value trees, actual
plan acquisition and TUI/session behavior require downstream implementation tests. Policy snapshots should
reuse preparation as designed; do not reinterpret these warm timings as recompilation-per-resource timings.
Apply, Terraform pass-through output and project dependencies remain unchanged.

## Review, retained attempts and decision record

The test-writer recorded four missing-observation failures before implementation; all four pass against the
completed samples. Supplemental integrity checks verify frozen bytes/origin, priming builds, stable artifacts,
channels, process groups and cancellation outcomes. No failed sample or outlier was dropped from a measurement
series. The earlier setup attempts are retained separately:

- `output/run-initial.log`: the extracted supervisor lacked its json import and stopped during preflight capture.
  The import was fixed before the measured run.
- `output/setup-interrupted/`: setup was deliberately stopped before timed builds to ensure every executable used
  the finalized Eval-only timing probe. The fresh run re-exported both trees and re-resolved dependencies.
- `output/integrity-red.txt`: delivery integrity caught post-freeze presentation changes to summarize.py. Its exact
  frozen bytes were restored and verified; report.py separately contains the presentation enhancements. The
  measurement harness, fixture, capability, lock, process records and frozen manifest were not rewritten.

The frozen summarize.py is retained for integrity/reconstruction. Use report.py for the final tables: it labels
supervisor time accurately, derives GNU command elapsed/user/sys time and shows relative duration deltas.
The external supervisor and experiment are disposable evidence tools, not production application abstractions.

The [verified decision-ticket record](https://github.com/lmarqs/terraform-ui/issues/116#issuecomment-6889936801)
for #116, linked to architecture
map [#107](https://github.com/lmarqs/terraform-ui/issues/107), contains the same first-version judgment and its
limitations. The protocol’s local-publication handoff is superseded only for this evidence comment by the bead’s
explicit requirement to record outcomes in the decision ticket; issue/board/Bead status is unchanged.

Production Quick build/affected-package tests, Done lint/full unit/build, coverage and UI macros are not
applicable to this delivery: only documentation and disposable standalone evidence assets change. The release
experiment builds are distinct from those unrun repository gates. Documentation validation uses docs:build
followed by docs:check; the progress log records their actual outcomes and the coordinator's delivery review.
