# The project root declares a backend it never initializes: any terraform
# command that runs here instead of in the selected member fails with
# "Backend initialization required", which is how the tests below detect a
# subcommand that ignores --chdir.
terraform {
  backend "s3" {
    bucket = "tfui-nonexistent-bucket"
    key    = "terraform.tfstate"
    region = "us-east-1"
  }
}
