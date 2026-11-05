# Runbooks

Holds what to do when a trigger fires: an alert, a request, a symptom. Diátaxis type: how-to.

A fact belongs here when someone acts on it under pressure. Why goes to the pages that explain.

What makes a good runbook:

- It covers one trigger, is named after it, and its body opens with the outcome and impact.
- Each step is one action with the output that shows it worked, so the responder verifies as they go.
- For a symptom, it names the likely causes and a remedy for each, and when the cause isn't obvious, it starts with
  checks that tell them apart.
- It gains what the responder learned each time the trigger fires.

## Example

This shows one good page, not the page. `docs/runbooks/redis-down.md` in the made-up `acme`:

```markdown
---
description: Restores Redis when its alert fires, each step with the output to expect.
audience: Whoever answers the Redis alert
---
# Redis down

While it is down every user is logged out. Needs `kubectl` access to `acme-prod`.

1. Check the pod: `kubectl -n acme get pod -l app=redis`. Expected: one pod, `Running`.
2. If it is in `CrashLoopBackOff`, restart it: `kubectl -n acme rollout restart statefulset/redis`. Expected:
   `Running` within two minutes.

If it stays down, restore the last snapshot with `make redis-restore`.
```
