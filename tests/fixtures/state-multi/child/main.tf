terraform {
  required_version = ">= 1.14"
}

resource "terraform_data" "one" {
  input = "one"
}

resource "terraform_data" "two" {
  input = "two"
}
