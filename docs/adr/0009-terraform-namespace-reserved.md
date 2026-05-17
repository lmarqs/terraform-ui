---
layout: default
title: "ADR-0009: Terraform's command namespace is reserved"
grand_parent: Development
parent: Architecture
nav_order: 0009
description: Decision to reserve terraform's CLI command names and not shadow them
---

# Terraform's command namespace is reserved

tfui never claims a verb that terraform owns. All terraform commands (`init`, `plan`, `apply`, `state`, `workspace`, `output`, `validate`, `console`, `import`, `taint`, `untaint`, `refresh`) pass through to terraform semantics. Novel tfui features use novel names: `scaffold`, `risk`, `phantom`, `blast-radius`.

This is why the original "init" plugin was renamed to "scaffold" -- `tfui init` must mean `terraform init`, not "create a tfui.hcl file."

The principle: tfui is a superset of terraform. A user's terraform muscle memory transfers directly. If `terraform plan` does X, `tfui plan` does X (with a better interface). tfui adds commands, it never redefines them.
