terraform {
  required_version = ">= 1.14"
}

resource "terraform_data" "doc" {
  input = "updated"
}
