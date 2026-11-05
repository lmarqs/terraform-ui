# Interfaces

Holds what an interface's contract file leaves out: who calls it and why, authentication, versioning, deprecation and
limits. An interface is any contract another system depends on: an HTTP API, an event stream, a queue, a
webhook. Diátaxis type: reference.

A fact belongs here when a caller needs it and the definition file, an OpenAPI document, a proto or a schema, does
not hold it. The definition is the contract; a copy of it in prose drifts.

What makes a good interface page:

- It covers one interface, and its front matter says what it is for and who calls it.
- It links the definition file and how the reference is built from it.

## Example

This shows one good page, not the page. `docs/api/orders.md` in the made-up `acme`:

```markdown
---
description: Introduces the orders API to a partner, and points to its contract.
audience: Whoever integrates a partner or changes the API
---
# Orders API

Partners asked for it in 2024, when they were keying 200 emailed orders a day into acme by hand.
[openapi/orders.yaml](../../openapi/orders.yaml) is the contract; `make api-docs` builds the reference.

Partners send an API key in `X-Acme-Key`. A key allows 600 requests a minute; a 429 carries `Retry-After`.
```
