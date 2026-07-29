terraform {
  bin = "terraform"
}

member "envs/demo" {}

defaults {
  plugin "plan" {
    targets = ["terraform_data.scoped"]
  }

  plugin "state" {
    enabled = false
  }
}
