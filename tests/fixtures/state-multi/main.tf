terraform {
  required_version = ">= 1.14"
}

resource "terraform_data" "alpha" {
  input = "alpha"
}

resource "terraform_data" "beta" {
  input = "beta"
}

resource "terraform_data" "gamma" {
  input = "gamma"
}

module "child" {
  source = "./child"
}
