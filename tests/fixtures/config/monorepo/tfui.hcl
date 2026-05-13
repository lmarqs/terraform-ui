terraform {
  bin = "terraform"
}

chdir {
  members = [
    "modules/vpc",
    "modules/ecs",
  ]
}

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
