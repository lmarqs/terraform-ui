var_file "base.tfvars" {}

workspace "production" {
  var_file "prod.tfvars" {}
  var "environment" { value = "prod" }
}

workspace "staging" {
  var_file "staging.tfvars" {}
  var "environment" { value = "staging" }
}

workspace "dev-*" {
  var "environment" { value = "dev" }
}
