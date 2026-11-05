resource "aws_s3_bucket" "logs" {
  bucket = "acme-logs"
  tags   = { Env = "prod", Owner = "infra" }
}

# primary is gone from this file: it moved to primary_v2, so terraform plans to destroy the old instance.
resource "aws_db_instance" "primary_v2" {
  identifier          = "acme-primary-v2"
  engine              = "postgres"
  engine_version      = "16.4"
  instance_class      = "db.r6g.large"
  allocated_storage   = 100
  username            = "acme"
  manage_master_user_password = true
  skip_final_snapshot = false
  tags                = { Env = "prod" }
}
