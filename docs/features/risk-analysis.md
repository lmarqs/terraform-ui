---
layout: default
title: Terraform Plan Risk Analysis
parent: Features — Terraform Plan Analysis Tools
nav_order: 1
seo:
  description: Automatic risk classification for terraform plan changes. Every resource scored critical, high, medium, or low before you apply.
description: "Explains how tfui scores each planned change critical, high, medium, low or none."
audience: "Users triaging a plan, and contributors changing the risk rules"
---

## Overview

tfui scores each planned change from two inputs: the action (delete or replace, update, create) and the category of
the resource type. Deleting a data-bearing, encryption or access resource is critical; creating a resource outside
the lists below is low. The score appears as a badge on each plan row, on the headless `Risk:` line and in the Risk
view, and feeds the blast radius impact score.

## How It Works

### Risk Levels

| Level | Meaning | Example |
|-------|---------|---------|
| **Critical** | Deleting or replacing a critical or high-risk resource | Deleting an RDS instance, replacing a KMS key, deleting a VPC |
| **High** | Updating a critical or high-risk resource, or deleting a medium-risk one | Modifying IAM roles or a VPC, deleting a security group |
| **Medium** | Updating a medium-risk resource, creating a critical or high-risk one, or deleting an unlisted one | Updating security groups, modifying Route53 records, creating an IAM role |
| **Low** | Creating a medium-risk or unlisted resource, or updating an unlisted one | Creating an SNS topic, updating an `aws_instance` |
| **None** | Nothing to apply | No-op or read |

### Classification Rules

**Delete / Replace actions:**

- Critical-type resource → **Critical**
- High-risk resource → **Critical**
- Medium-risk resource → **High**
- Everything else → **Medium**

**Update actions:**

- Critical-type resource → **High**
- High-risk resource → **High**
- Medium-risk resource → **Medium**
- Everything else → **Low**

**Create actions:**

- Critical/high-risk resource → **Medium**
- Everything else → **Low**

**No-op and read:** **None**

### Resource Type Categories

These are every type in each category; the source is `internal/terraform/risk.go`.

**Critical (data-bearing / encryption):**

- AWS: `aws_db_instance`, `aws_rds_cluster`, `aws_rds_cluster_instance`, `aws_dynamodb_table`, `aws_s3_bucket`,
  `aws_efs_file_system`, `aws_fsx_lustre_file_system`, `aws_fsx_windows_file_system`, `aws_redshift_cluster`,
  `aws_elasticache_cluster`, `aws_elasticache_replication_group`, `aws_docdb_cluster`, `aws_neptune_cluster`,
  `aws_kms_key`, `aws_kms_alias`
- GCP: `google_sql_database_instance`, `google_storage_bucket`, `google_kms_key_ring`, `google_kms_crypto_key`
- Azure: `azurerm_sql_server`, `azurerm_cosmosdb_account`, `azurerm_storage_account`, `azurerm_key_vault`

**High (access / networking / compute):**

- AWS: `aws_iam_role`, `aws_iam_policy`, `aws_iam_user`, `aws_iam_group`, `aws_iam_instance_profile`, `aws_vpc`,
  `aws_subnet`, `aws_route_table`, `aws_nat_gateway`, `aws_internet_gateway`, `aws_eip`, `aws_lb`, `aws_alb`,
  `aws_elb`, `aws_ecs_cluster`, `aws_eks_cluster`, `aws_lambda_function`, `aws_cloudfront_distribution`
- GCP: `google_compute_network`, `google_compute_subnetwork`, `google_container_cluster`, `google_project_iam_member`
- Azure: `azurerm_virtual_network`, `azurerm_kubernetes_cluster`, `azurerm_role_assignment`

**Medium (security rules / DNS / messaging):**

- AWS: `aws_security_group`, `aws_security_group_rule`, `aws_network_acl`, `aws_route53_record`,
  `aws_cloudwatch_log_group`, `aws_sns_topic`, `aws_sqs_queue`, `aws_ecr_repository`
- GCP: `google_compute_firewall`, `google_dns_record_set`
- Azure: `azurerm_network_security_group`, `azurerm_dns_zone`

## How a level renders

Every view that shows a level uses the same colors:

- Critical -- bold magenta
- High -- bold red
- Medium -- amber
- Low -- green

The rules above are fixed. The [Risk Analysis plugin](../plugins/risk.md) groups a plan's changes by level, and the
[Plan plugin](../plugins/plan.md) puts a badge on each change and, in headless text output (without `-json`), the
overall level on its last line.
