# Acquisition experiment protocol

Declared before execution, 2026-11-15. Disposable overlay tests; no production integration.

Candidate: a seeded reviewed plan remains available through real `Session.RunPlugin` with an initial selected Chdir and a nonexistent Terraform binary. Reject this lifecycle candidate if the session attempts Terraform instead of returning the seeded resource rows. Root seeded review is the control.

Additional observations to implement after the first red run: count fake Terraform plan/show/version calls for direct seeded Plan, live Plan, repeated live Plan, Plan followed by PlanJSON, and Risk Analyze of the reviewed summary. Retain raw show JSON from the same saved artifact and compare exact fixture bytes and SHA-256. Reject PlanJSON as an evidence-retrieval candidate if it generates another plan. Reject summary-only evidence if original JSON cannot be recovered losslessly. No claim about real Terraform cost or OPA is supported by this fixture.

Expected existing behavior is distinct from candidate acceptance. `ACQUISITION_CANDIDATE=1` keeps the unsupported lifecycle expectation executable; baseline observations must explicitly assert its current failure without changing production code.
