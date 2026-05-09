terraform {
  required_version = ">= 1.14"
  required_providers {
    local = {
      source  = "hashicorp/local"
      version = "~> 2.5"
    }
  }
}

resource "local_file" "alpha" {
  filename = "${path.module}/out/alpha.txt"
  content  = "alpha"
}

resource "local_file" "beta" {
  filename = "${path.module}/out/beta.txt"
  content  = "beta"
}
