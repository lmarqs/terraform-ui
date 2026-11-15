# D-03 runtime protocol, prepared before execution

Audience: the D-03 executor and owner reviewing feasibility. This protocol pins a disposable experiment and declares observations. No evaluator, capability control, build or cost measurement has run. Public behavior and cost acceptance remain open.

## Execution gate

Before controls or measurements, attach explicit D-01a **experimental** candidates identifying (1) trusted policy authors, exact external/nondeterministic builtin promise, sensitive input exposure and permitted diagnostic/output disclosure; (2) input envelope, entrypoint, result cardinality/shape, allowed levels/reasons, completeness and malformed/undefined/empty/conflicting-result treatment. Candidates may be provisional and falsifiable; they are not public acceptance. Neither this proposed allowlist nor example query supplies that authorization.

Without those candidates one can record source revisions, module declarations, host tool versions, release matrix, proposed fixtures/method and artifact integrity. After candidates are supplied, raw compile/evaluation distinctions and costs can be recorded without choosing public semantics; control success is judged against the explicit candidate, and shipping still requires owner acceptance. No fallback classification, aggregation formula, thresholds or sensitivity promise is implied here.

## Version, fixtures and capabilities

Use manifest.json's exact OPA tag and revision; source rationale is in maintenance.md. Resolve into a new disposable module only with Go1.25.5 and GOTOOLCHAIN=local; save go.mod/go.sum, module graph, tool env and downloaded source hashes. Verify tag revision before running. Never edit repository go.mod/go.sum, upgrade Go, override transitive requirements or quietly substitute another release. A dependency rejection is feasibility evidence. An owner-approved version change gets a new manifest/protocol revision before execution. v1.15.1 is only a historical comparison lead.

fixtures.json preserves a JSON integer beyond float64 precision, list order, null versus absent, masks and replacement orders. Decode using json.Decoder.UseNumber before conversion; independently compare original JSON values and candidate evaluator echo results. Fixtures are synthetic and resource-local, not an approved input schema. Hash fixture bytes before every run. Generate only the bounded variants declared there. Source, serialized input and channels obey manifest bounds.

Construct capabilities from the pinned version, retaining only the exact manifest builtin names and its v1 language features; abort on missing builtin names. Save the complete resultant capability JSON and hash, including feature/future-keyword lists, before execution. Empty AllowNet is additional defense; do not infer a sandbox from it. Use raw local modules, no supplied compiler, no remote loading, no custom resolvers/builtins, strict builtin errors, print disabled, no print hook or tracer. Review pinned API option implementations before harness implementation; research described v1.15.0, not a verified v1.19.1 control implementation. Do not silently broaden the allowlist to make fixtures compile.

## Result and cancellation matrix

Each row is a separately hashed local Rego v1 module with package d03. Record prepare error, eval error and code, ResultSet length, expression count and typed value, validation outcome, context cause, stdout/stderr, print/trace hooks and duration separately. Do not use err == nil as the success criterion. Generated modules and harness source must be frozen before execution.

| Fixture/module body sketch | Observation to distinguish |
| --- | --- |
| result := {"level":"low","reason":"synthetic"} | Defined example object; public validity depends on D-01a. |
| unrelated := true | Undefined result, even if Go error is nil. |
| result := [] ; separately result := {} ; separately result := null | Defined empty array/object/null, distinct from undefined and each other. |
| result := 42 ; separately object with missing reason or unknown level | Defined but candidate-invalid shape/domain; validation failure distinct from evaluator error. |
| result := input.x if input.a; result := input.y if input.b; input a/b true, x=1,y=2 | Competing complete values; keep runtime conflict distinct from static compile failure. |
| result := [ | Parse/preparation failure. |
| result := nonexistent_builtin(1) | Compile failure for missing builtin. |
| result := div(1, input.zero), input.zero=0 | Executed strict builtin failure. Also run strict=false in a separately labelled control. |
| result := "reachable"; unused := div(1, input.zero), input.zero=0 | Unqueried rule's builtin error stays unexecuted; strict mode cannot validate unreachable expressions. Compare strict=false. |
| Same normal and 256-condition modules with already-cancelled context, then 100ms context deadline | Separate cancellation from success, interpreter error and external termination. No hard latency guarantee. |
| result := input | Synthetic fidelity echo; compare raw numeric/absence/order/masks before any result validator. |

Builtin fixtures must call actual Rego operators/syntax accepted by the pinned API; the executor records resulting source and hashes before running. These sketches are case definitions, not claims of compilation success.

## Capability and disclosure matrix

For each denied builtin test a direct executed call and a call in an unqueried rule. Expected candidate boundary is compile rejection even if unreachable; any successful compilation reopens the capability choice. Use no permissive network positive control and no real secrets.

| Attempt | Data / observation |
| --- | --- |
| http.send | GET http://127.0.0.1:<harness-port>/probe; loopback listener request counter must stay zero. |
| net.lookup_ip_addr | localhost; observe resolver instrumentation and compilation result; no remote hostname. Missing instrumentation means no DNS non-use claim beyond compile rejection. |
| time.now_ns | Capture compile/error/result. |
| rand.intn, uuid.rfc4122 | Separate modules; capture compile/error/result. |
| opa.runtime | Capture compile/error/result, especially environment leakage; child has minimal synthetic env. |
| print(input.change.after.nested.token) | With print disabled capture both streams and hooks; print may be erased rather than rejected. Explicitly distinguish suppression from compile denial. |
| trace(input.change.after.nested.token) | Denied builtin plus absent tracer; capture compile result and every diagnostic channel. |
| sprintf("%s", [input.change.after.nested.token]) returned in result; direct echo; synthetic token in source/error | Search raw results, validation errors, stdout/stderr and print/trace channels for D03_SYNTHETIC_SECRET_ONLY. Capabilities cannot prevent policies returning their input or source literals. Judge only the chosen trust/disclosure promise. |

Child environment contains only required PATH/GOROOT, GOTOOLCHAIN=local, CGO_ENABLED=0, locale, TZ=UTC and synthetic marker; no inherited cloud credentials, real HOME files, telemetry, user config, Terraform process or real network requests. The loopback listener is harness infrastructure, not policy permission. Bounds and process limits below constrain damage; they do not constitute a general security sandbox.

## External guards and observability

Run each subprocess under `prlimit --as=<bytes> --core=0` with `timeout`, in its own process group. Build guard: 4 GiB address space/120s; probe guard: 1 GiB address space/30s. These are experimental safety ceilings, not accepted performance budgets. The address-space limit is an external memory bound applied by the kernel; Go context and GOMEMLIMIT are cooperative and do not replace it. Process count is bounded only by the wall deadline and the process-group kill: RLIMIT_NPROC counts every process of the user, so it is not a per-child bound. Before OPA execution, preflight with harmless synthetic children that an over-limit allocation fails, a timeout kills the whole group, and no child survives. If prlimit or timeout is missing, stop and report the missing guard.

Outer supervisor starts each child in a new process group, enforces monotonic wall deadline, sends TERM then KILL after 2s to the entire group, drains bounded channels concurrently and kills on channel overflow. The Linux command template is `/usr/bin/time -v -o <rss.txt> prlimit --as=1073741824 --core=0 -- /usr/bin/timeout --signal=TERM --kill-after=2s 30s <probe> <case>`; build uses 4294967296 and 120s. Keep the external supervisor outside the child group; confirm no group member survives afterward. Capture exit/signal, timeout status, allocation failures, and peak RSS. GNU time RSS is KiB on this host; retain raw output. Probe deadline must include preparation, not just Eval. Mark limited/allocation-failure/overflow trials incomplete rather than assigning cost or successful assessment.

## Reproducible build and measurement protocol

Use immutable clean baseline revision and hashes from manifest. If branch changes, create a new baseline manifest rather than compare different trees. Export that revision to two scratch copies outside the live module (baseline and candidate). Resolve each graph separately; no production files change. Baseline is ordinary tfui. Candidate adds a reachable, explicitly invoked evaluation path in the scratch tfui main package, prepares a local policy and evaluates fixture input using pinned rego. Record the exact patch, harness source, graph and hashes before running. A blank import, dead code, or evaluator-only binary cannot establish combined tfui feasibility or size. Keep standalone evaluator results separately labelled.

Use identical flags for all targets: `GOTOOLCHAIN=local CGO_ENABLED=0 GOMAXPROCS=2 GOOS=<os> GOARCH=<arch> go build -p=2 -trimpath -buildvcs=false -ldflags='-s -w -X main.version=d03-experiment' -o <artifact> ./cmd/tfui`. Execute through repository-aligned mise or explicitly aligned Go1.25.5 GOROOT/PATH. Match release stripping/version flags; trimpath and disabled VCS are controlled experiment flags, not a claim of byte-identical goreleaser output. Record go env, go version, compiler path, OS/kernel/CPU/RAM, git hashes, GOAMD64/GOARM64 defaults, build tags and environment. Isolate GOCACHE/GOMODCACHE under scratch storage.

| Target | Baseline/candidate build & bytes | Execution |
| --- | --- | --- |
| linux/amd64 | Required independently | Native host cold/warm/RSS; combined path invocation required. |
| linux/arm64 | Required independently | Native runner needed; otherwise unmeasured. |
| darwin/amd64 | Required independently | Native runner needed; otherwise unmeasured. |
| darwin/arm64 | Required independently | Native runner needed; otherwise unmeasured. |

Cross-compilation proves only buildability. Do not execute via emulation and call that native runtime evidence. Set CGO_ENABLED=0 consistently as an experiment choice; the release YAML does not explicitly set it.

Prefetch dependencies under external guard before timed builds; report fetch duration separately. For each target, take five paired baseline/candidate builds with alternating order. Use fresh empty GOCACHE per cold compilation; use one untimed priming build then five cached builds for warm compilation. No global cache clearing. Capture command wall/user/sys/RSS, byte count, sha256, stderr and exit status. Compare byte and absolute/relative duration deltas only for successful pairs. Candidate dependency minimum-version conflicts remain failures; standalone success cannot repair them.

For N=1,100,1000 independent mutations, run five fresh processes per variant. Cold runtime records process elapsed separately from in-process parse/compile/prepare time and first Eval time. Warm runtime prepares once per process, performs one untimed evaluation, then 100 sequential Eval calls over each N case, recording total and per-evaluation timings and full correctness checks. Record prepare count; do not hide a recompile in warm evaluation. Use input parsing outside Eval timer and report its duration separately. Keep concurrency one, GOMAXPROCS=2. Run the bounded 256-condition expression and deadline cases separately; no escalation of workload after a timeout.

Collect process peak RSS for each sample, including a fixture decode/dispatch-only harness to contextualize measurement overhead. Baseline tfui is used for build/size/compile comparison; its lack of evaluator means there is no honest baseline Eval latency. Report standalone evaluator and combined candidate timings separately. Preserve all five samples, median/range, units, failed samples, caps, hashes and structured errors. No outlier removal. The owner judges acceptable costs; no performance pass threshold is defined. Unsupported target, failed control or infeasible combined graph reopens the corresponding architecture choice before production.

## Evidence handoff

Execution records need candidate IDs for D-01a, frozen modules/harness/fixtures/capability JSON, exact commands, source and tool hashes, per-case raw outputs, results matrix, four-target build matrix, complete cost samples and limitations. Required columns: case, candidate, target, phase, expected candidate behavior, observed error/result cardinality/type, streams, duration, RSS, memory peak, external-limit outcome, verdict and evidence paths. Keep operational observations separate from public semantics and owner cost acceptance. This preparation completes no D-03 runtime/controls/cost acceptance criteria.
