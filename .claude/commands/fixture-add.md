---
allowed-tools: Bash(terraform:*), Bash(mkdir:*), Bash(ls:*), Read, Edit, Write
description: Add a new terraform test fixture
---

## Context

- Existing fixtures: !`ls tests/fixtures/`
- Fixture helper: !`cat tests/helpers/fixtures.bash`
- Terraform version: !`terraform version | head -1`

## Instructions

Create a new terraform fixture for testing. Follow this process:

1. Create directory `tests/fixtures/<name>/`
2. Write `main.tf` with the TARGET state (what terraform will plan against)
3. If the scenario needs pre-existing state (update, delete, replace):
   a. Temporarily write a "seed" config as main.tf
   b. Run `terraform init -backend=false -input=false`
   c. Run `terraform apply -auto-approve -input=false`
   d. Replace main.tf with the target config
   e. Clean up: remove .terraform/, .terraform.lock.hcl, terraform.tfstate.backup
   f. Keep: main.tf + terraform.tfstate (+ out/ files if needed)
4. If no pre-existing state needed (create scenario): just main.tf, no state file
5. Verify with `terraform plan` that it produces the expected actions
6. Use `required_version = ">= 1.14"` and hashicorp/local provider `~> 2.5`
7. Prefer `local_file` for create/delete/replace, `terraform_data` for update
