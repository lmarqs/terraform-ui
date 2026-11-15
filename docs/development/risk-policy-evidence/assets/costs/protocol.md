# D-03c execution amendment, declared before measurement

Use D-03a's protocol and limits, D-01a's delegated first-version candidates and D-03b's
frozen OPA v1.19.1 capabilities. Baseline remains immutable revision
60167777494b622ca55a203f05deb8e127357d23; current HEAD has identical go.mod/go.sum
and release YAML. Export this revision to baseline and candidate /tmp trees. No live Go
module edits. Candidate adds an early explicit `__d03` dispatch in tfui main and a Go
probe with reachable OPA preparation/evaluation; retain exact patch and resolved locks.
Use Go1.25.5, identical stripped flags, CGO=0, concurrency two, four release targets.

Run sequentially: dependency resolution/prefetch, supervisor preflight, five paired cold
builds per target alternating baseline/candidate first, one priming build per variant/target,
then five cached paired builds alternating order. Each cold trial has an empty private
GOCACHE; warm trials share only their variant/target priming cache. Module cache is
prefetched and shared within this scratch experiment. Save all command/time/RSS/exit
records, artifact bytes/hash. No failed sample is removed or assigned successful cost.
A failed 120-second build remains evidence; use delegated authority to record whether a
workable first version exists, rather than silently lifting guards.

Runtime variants: combined candidate, standalone identical probe, and decode/dispatch-only
probe (same executable with overhead mode, no prepare/Eval). Generate N=1,100,1000 copies
of D-03a minimal resource with unique addresses, <=1MiB total; each Eval sees one resource.
Use a complete explicit constant-low default plus an update rule with the same complete
value, exercising resource-local action access. This is a representative small policy,
not a worst-case production workload. Reuse D-03b's frozen positive capabilities. Decode
with UseNumber outside Eval. Cold: parse, prepare once, first Eval then remaining N-1;
warm: parse, prepare once, one untimed priming Eval, then 100 passes over N resources.
Check every result's exact level/reason/cardinality, count checks and prepare calls.
Five fresh processes per N/phase/variant; one process, one evaluator, no concurrency.
Overhead variant decodes and dispatches N resources using the same loops without OPA.
Capture separate parse/prepare/first-Eval/total-Eval nanoseconds plus whole process wall/RSS.
Run the frozen 256-equality policy separately with cancelled and 100ms contexts; this
checks cooperative cancellation outcomes, not a hard latency promise. External guards
remain 1GiB/30s. Runtime on other architectures is unmeasured (no native runners).

Before timing, freeze probe/protocol/runner/capability/fixture bytes and hashes, module
locks/graph and host/toolchain metadata. Record source origin from pinned module metadata.
Record dependency fetch separately. GNU time RSS includes wrapper/process-tree high water,
not allocation totals. Retain median/range with no outlier removal and no invented budget.
The coordinator-selected cost judgment is provisional first-version acceptance under owner
delegation, not a claim that the owner selected exact numbers or production interfaces.
Decision-ticket publication is coordinator-owned: provide a ready-to-publish #116 record,
linked to #107, without changing remote tracking state in this worker run.
