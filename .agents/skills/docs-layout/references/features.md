# Features

Holds what the system does, one page per unit of behavior. Diátaxis type: reference. A page answers "what does it
do, and why is it so?"

A fact belongs here when it states what a unit owes its users and the code cannot be read for it: a threshold, a
rule, who may do what. A limit an interface's callers rely on is the interface's; a reason that meets the decision
bar is a record, and one below it stays here; operating the unit is a runbook.

What makes a good feature page:

- It opens with what the unit is and why it is so, as it happened: a reason too small for a record has no other
  owner.
- It follows the unit's own structure, its inputs, states and outputs, so a reader finds a fact where the product
  puts it. A rule that is arithmetic over state carries a worked example.
- It names what the unit does not handle, each with its workaround, and links the decisions behind it.

## Example

This shows one good page, not the page. `docs/features/sessions.md` in the made-up `acme`:

```markdown
---
description: Explains how long a session lasts, what renews it and what ends it.
audience: Whoever changes login
keywords: [session timeout, idle logout]
---
# Sessions

A session keeps a user logged in until logout or 30 minutes without a request. The limit was 10 minutes until 2023, when
shoppers paying on slow connections lost their carts mid-checkout.

Each request renews the session for 30 minutes; logout deletes it at once. Every replica reads the same
[session store](../adr/0007-keep-sessions-in-a-shared-store.md), so a user stays logged in whichever one answers.
```
