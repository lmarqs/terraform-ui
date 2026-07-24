terraform {
  required_version = ">= 1.14"
  required_providers {
    local = {
      source  = "hashicorp/local"
      version = "~> 2.5"
    }
  }
}

# Evaluation error on purpose: the undeclared local makes plan, validate, and
# apply fail while `terraform init` still succeeds.
resource "local_file" "broken" {
  filename = "${path.module}/out/broken.txt"
  content  = local.nonexistent
}
