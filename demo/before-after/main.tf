# A plan for the landing page's before/after section. Nothing here talks to AWS: the credentials are fake, the
# provider skips every account check, and generate.sh plans with -refresh=false against terraform.tfstate.
terraform {
  required_version = ">= 1.14"
  required_providers {
    aws = { source = "hashicorp/aws", version = "~> 5.80" }
  }
}

provider "aws" {
  region                      = "us-east-1"
  access_key                  = "demo"
  secret_key                  = "demo"
  skip_credentials_validation = true
  skip_requesting_account_id  = true
  skip_metadata_api_check     = true
  skip_region_validation      = true
}

module "networking" {
  source = "./modules/networking"
}

module "compute" {
  source            = "./modules/compute"
  subnet_ids        = module.networking.subnet_ids
  security_group_id = module.networking.security_group_id
}

module "storage" {
  source = "./modules/storage"
}
