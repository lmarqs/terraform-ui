terraform {
  required_providers {
    local = {
      source  = "hashicorp/local"
      version = "~> 2.5"
    }
  }
}

resource "local_file" "result" {
  filename = "${path.module}/out/result.txt"
  content  = "applied successfully"
}
