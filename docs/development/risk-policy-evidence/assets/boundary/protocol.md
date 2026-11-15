# D-03b execution amendment (frozen before probes)

Baseline fdfdcb39b0f2fea08fcc9174e36bcde7952b949d; OPA v1.19.1 and Go1.25.5
from D-03a. This is a standalone control experiment, not the D-03c combined build/cost experiment.
Use the D-01a query data.tfui.risk.v1.classification and exact level/reasons shape.
The d03.result/reason sketches in D-03a are superseded for these controls.
Use the D-01a positive builtin list plus plus/minus/mul/div solely to exercise strict
arithmetic failure. Save the full resulting capabilities; no production allowlist is established.
Reject print and trace using parsed AST calls before OPA preparation in the candidate path;
separate raw controls bypass that rejection to measure OPA erasure/compile behavior.
No print hook or query tracer is installed; channel fields report that absence, not an
instrumented invocation count. DNS has no resolver instrumentation; only compile rejection
can be concluded. HTTP uses a counted loopback server, never a permissive control.

Fixtures: valid, undefined, empty array/object/null, malformed scalar/missing reason/unknown
level; dynamic complete-rule conflict; parse and compile failures; strict/lenient executed
and unqueried division by zero. Denied builtins run in queried and unqueried rules.
Synthetic sensitive input is replaced by a tagged opaque node before the marked probe;
unmarked input, source literal and raw error probes intentionally demonstrate disclosure.
No real secrets. Source <=64KiB, input and channels <=1MiB. Freeze sources/hashes before probes.
Use D-03a's prlimit/timeout/process-group supervisor, preflight memory and descendant killing.
Each probe is independent under 1GiB/30s; build under 4GiB/120s. Dependency fetch is separately
guarded. Capture raw channels, errors, cardinality, values, validation, duration and GNU time RSS.
Validation covers the fixture result shapes and empty paths only; it is no production validator.
No release builds, cost benchmark, fidelity matrix or cancellation acceptance is claimed here.

Pre-probe environment correction: the first guarded dependency fetch failed because Go's
checksum database needs GOPATH or HOME. Retain fetch-initial channels; add explicit scratch
GOPATH=/tmp/tfui-d03b/gopath. No HOME or user credential/config inheritance is added.
