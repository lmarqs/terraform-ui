# D-02c pre-execution protocol

Written before running the experiments. These assets are disposable evidence, not production policy code.

The current Go configuration APIs are the baseline. Probe root plugin settings, Member/workspace execution
settings, empty and absent options, invalid expressions and rejected Member plugin blocks through real HCL
files. A dedicated `risk_policy` selector is **not implemented**; test rejection/ignoring as observations, never
claim selection or provenance works today. Existing session/UI regression tests supply startup and switch traces.

The separate Python candidate models only immutable local source snapshots and publication generations.
Its API is `snapshot(path)` yielding `modules` and `digest`, and `Gate.begin(context, plan_id, source_path)` /
`Gate.publish(ticket, result)` with visible `Gate.result`. It does not parse HCL, evaluate Rego, or run Terraform.

Pass criteria: exact source bytes and filename identity survive; equivalent directory copies share identity;
line ending, filename, production module and helper changes alter identity; ignored test/unrelated changes do not.
Invalid/deleted/empty/symlink source fails explicitly. A ticket contains the bytes from its own snapshot. Source
changes before publication and a late A result after B begins are rejected even when work ignores cancellation.
Beginning a new generation immediately clears old results. Repeated same Context still rejects older generations.

Falsification: silent input data loss rejects an input candidate (D-02a); mixed bytes or stale A publication rejects
this lifecycle candidate. Real HCL silently losing the selector rejects reuse of that parser route for the new
selector. Observed child parse fallback rejects unmodified lifecycle integration, not the candidate source digest.

Run all temporary files within `.ralph-tui/d02c`; use the checkout's Go toolchain and Python standard library.
Capture tool versions, exact commands and outputs. Red tests precede implementing the candidate. Existing product
behavior is observed, not repaired. This protocol supplies no security sandbox, filesystem atomicity, OPA runtime,
or completed production session integration claim.
