terraform {
  required_version = ">= 1.14"
  required_providers {
    local = {
      source  = "hashicorp/local"
      version = "~> 2.5"
    }
  }
}

resource "local_file" "moved" {
  filename = "${path.module}/out/new-location.txt"
  content  = "same content"
}
