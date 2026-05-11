terraform {
  required_providers {
    local = {
      source  = "hashicorp/local"
      version = "~> 2.5"
    }
  }
}

resource "local_file" "one" {
  filename = "${path.module}/out/one.txt"
  content  = "one"
}

resource "local_file" "two" {
  filename = "${path.module}/out/two.txt"
  content  = "two"
}
