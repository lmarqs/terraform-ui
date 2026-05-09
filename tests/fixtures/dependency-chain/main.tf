terraform {
  required_version = ">= 1.14"
  required_providers {
    local = {
      source  = "hashicorp/local"
      version = "~> 2.5"
    }
  }
}

# Root resource - will be destroyed to test blast radius
resource "local_file" "database" {
  filename = "${path.module}/out/database.txt"
  content  = "database-connection-string"
}

# Depends on database (depth 1)
resource "local_file" "app_config" {
  filename = "${path.module}/out/app_config.txt"
  content  = local_file.database.content

  depends_on = [local_file.database]
}

# Depends on database (depth 1)
resource "local_file" "backup" {
  filename = "${path.module}/out/backup.txt"
  content  = "backup-of-${local_file.database.content}"

  depends_on = [local_file.database]
}

# Depends on app_config (depth 2 from database)
resource "local_file" "web_server" {
  filename = "${path.module}/out/web_server.txt"
  content  = "server-using-${local_file.app_config.content}"

  depends_on = [local_file.app_config]
}

# Independent resource - not in blast radius
resource "local_file" "independent" {
  filename = "${path.module}/out/independent.txt"
  content  = "standalone"
}
