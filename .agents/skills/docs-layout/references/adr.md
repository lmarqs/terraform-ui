# Architecture decisions

Holds why the system is built as it is: each foundation decision, the options weighed and why the choice won.
Diátaxis type: explanation. A record answers "why is it built this way?", so a reader months later can tell whether
the reasons still hold before changing it.

A fact belongs here when reversing it changes more than one feature or module, it is hard to reverse, it surprises a
reader new to it, and it came from a real trade-off. How one feature works, a choice one change can reverse and a
convention a linter enforces are not: every record reads as binding, and a record of a detail freezes it.

What makes a good record:

- The title is the decision taken, as a sentence, and the file is `NNNN-<slug>.md`, numbered once and never reused,
  since old references name it.
- It tells what happened first: the situation, what it cost, what was tried. Then the forces the options are judged
  by. A reader tests the decision against that story.
- Each option appears at its best, with the force it fails. Options that had to be invented mean no decision.
- It names the product chosen and why that one, in a sentence: that reason has no other owner.
- A changed decision is edited in place and keeps its number; one reversed outright leaves, and its facts that still
  hold move to the page they fit.

## Example

This shows one good page, not the page. `docs/adr/0007-keep-sessions-in-a-shared-store.md` in the made-up `acme`:

```markdown
---
description: Records why sessions live in one store every replica reads.
audience: Whoever changes login or deploys
---
# Keep sessions in a shared store

In 2023 the web tier went to four replicas, and each deploy logged out everyone on the restarted one: 300
"logged out mid-checkout" tickets the first month. Sticky sessions, the stopgap, did not survive a deploy. A session
must survive deploys, end at once on logout, and hold a 20 KB cart.

Signed cookies keep no server state, but cannot be ended early and a cart exceeds the 4 KB a cookie holds. Sticky
sessions need nothing new, and lose every user of a restarted replica. A store every replica reads meets all three.
It is Redis, already run for the job queue.

Any replica serves any request; the store joins what a deploy depends on.
```
