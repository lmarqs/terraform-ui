# Real Terraform plan fixtures for Risk

`risk-delete/plan.json`, `risk-forget/plan.json` and `risk-import/plan.json` are `terraform show -json` output from
Terraform 1.14.9 (the `terraform` pinned in `mise.toml`) on built-in `terraform_data` resources with local state, so no
provider or credential is involved. `risk-deposed/plan.json` is written by hand: Terraform emits a deposed object only
after a failed create-before-destroy replace, which a local fixture cannot cause.

Each directory also holds the `tfui.hcl` and `policy.rego` that `tfui risk -plan <plan.json> -project <dir>` reads.
`tests/integration/risk_real_plans_test.go` runs all four.

## What each plan holds

| Fixture | Resource change | Shape Risk relies on |
| --- | --- | --- |
| `risk-delete` | `terraform_data.doomed`, actions `["delete"]` | `"after": null` with `"after_unknown": {}` |
| `risk-forget` | `terraform_data.kept`, actions `["forget"]` | `"after": null` with `"after_unknown": {}` |
| `risk-import` | `terraform_data.adopted`, actions `["update"]` | `"importing": {"id": "known-import-id"}` |

## Regenerating

Run each block in an empty scratch directory with the pinned `terraform` first on `PATH`. Then copy `plan.json` over
the fixture's, after removing the top-level `timestamp` key (the only non-deterministic value) and indenting with two
spaces.

`risk-delete`: apply a resource, empty the configuration, plan.

```sh
cat > main.tf <<'TF'
resource "terraform_data" "doomed" {
  input = "doomed"
}
TF
terraform init -input=false && terraform apply -auto-approve -input=false
: > main.tf
terraform plan -out=plan.tfplan -input=false
terraform show -json plan.tfplan > plan.json
```

`risk-forget`: apply a resource, replace it with a `removed` block that keeps the object, plan.

```sh
cat > main.tf <<'TF'
resource "terraform_data" "kept" {
  input = "kept"
}
TF
terraform init -input=false && terraform apply -auto-approve -input=false
cat > main.tf <<'TF'
removed {
  from = terraform_data.kept

  lifecycle {
    destroy = false
  }
}
TF
terraform plan -out=plan.tfplan -input=false
terraform show -json plan.tfplan > plan.json
```

`risk-import`: plan an import block with a literal ID against empty state.

```sh
cat > main.tf <<'TF'
import {
  to = terraform_data.adopted
  id = "known-import-id"
}

resource "terraform_data" "adopted" {
  input = "adopted"
}
TF
terraform init -input=false
terraform plan -out=plan.tfplan -input=false
terraform show -json plan.tfplan > plan.json
```
