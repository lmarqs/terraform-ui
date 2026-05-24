terraform {
  bin = "terraform"
}

member "modules/vpc" {}
member "modules/ecs" {}

cache {
  staleness_threshold = "5m"
}

defaults {
  parallelism = 10
  lock        = true

  var_file "common/tags.tfvars" {}

  plugin "risk" {
    enabled = true
    level   = "high"
  }
}
