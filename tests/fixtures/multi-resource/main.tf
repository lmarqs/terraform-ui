terraform {
  required_version = ">= 1.14"
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

resource "local_file" "three" {
  filename = "${path.module}/out/three.txt"
  content  = "three"
}

resource "local_file" "four" {
  filename = "${path.module}/out/four.txt"
  content  = "four"
}

resource "local_file" "five" {
  filename = "${path.module}/out/five.txt"
  content  = "five"
}
