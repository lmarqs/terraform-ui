terraform {
  required_providers {
    local = {
      source  = "hashicorp/local"
      version = "~> 2.5"
    }
  }
}

resource "local_file" "good" {
  filename = "${path.module}/out/good.txt"
  content  = "good"
}

# /proc rejects new files, so this resource always fails while the one above
# succeeds — an apply that stops partway.
resource "local_file" "doomed" {
  filename = "/proc/tfui-does-not-exist/doomed.txt"
  content  = "doomed"
}
