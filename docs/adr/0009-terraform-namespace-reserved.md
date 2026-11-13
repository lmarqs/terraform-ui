---
layout: default
title: "ADR-0009: Terraform's command namespace is reserved"
grand_parent: Development
parent: Architecture
nav_order: 0009
seo:
  description: Decision to reserve terraform's CLI command names and not shadow them
description: "Records why tfui never takes a command name terraform owns, and names its own features apart."
audience: "Contributors naming a new command or plugin"
---

# Terraform's command namespace is reserved

tfui never claims a verb that terraform owns. All terraform commands (`init`, `plan`, `apply`, `state`, `workspace`, `output`, `validate`, `console`, `import`, `taint`, `untaint`, `refresh`) pass through to terraform semantics. Novel tfui features use novel names: `scaffold`, `risk`, `phantom`, `blast-radius`.

The principle: tfui is a superset of terraform. A user's terraform muscle memory transfers directly. If `terraform plan` does X, `tfui plan` does X (with a better interface). tfui adds commands, it never redefines them.

## Consequences

- A novel feature never takes a terraform verb, even when that verb is the most natural name for it.
- A tfui plugin named after a terraform verb wraps that verb: `plugins/init` runs `terraform init`.

## Considered Options

### Let a tfui feature take a terraform verb

Rejected. The original "init" plugin created a tfui.hcl file; it was renamed to "scaffold", because `tfui init` must mean `terraform init`.
