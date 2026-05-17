---
layout: default
title: Risk Analysis
parent: Features
nav_order: 1
description: Automatic risk classification for terraform plan changes
---

# Risk Analysis

terraform-ui automatically classifies every planned change by risk level based on the resource type and action.

## Risk Levels

| Level | Meaning | Example |
|-------|---------|---------|
| **Critical** | Data loss or encryption key destruction likely | Deleting an RDS instance, replacing a KMS key |
| **High** | Infrastructure disruption or access change | Deleting a VPC, modifying IAM roles, destroying load balancers |
| **Medium** | Security rule or DNS change | Updating security groups, modifying Route53 records |
| **Low** | Safe change with minimal blast radius | Creating a new resource, updating tags |

## Classification Rules

### Delete / Replace actions

- Critical-type resource → **Critical**
- High-risk resource → **Critical**
- Medium-risk resource → **High**
- Everything else → **High**

### Update actions

- Critical-type resource → **High**
- High-risk resource → **High**
- Medium-risk resource → **Medium**
- Everything else → **Medium**

### Create actions

- Critical/high-risk resource → **Medium**
- Everything else → **Low**

## Resource Type Categories

### Critical (data-bearing / encryption)

AWS: `aws_db_instance`, `aws_rds_cluster`, `aws_dynamodb_table`, `aws_s3_bucket`, `aws_efs_file_system`, `aws_redshift_cluster`, `aws_elasticache_cluster`, `aws_kms_key`

GCP: `google_sql_database_instance`, `google_storage_bucket`, `google_kms_key_ring`, `google_kms_crypto_key`

Azure: `azurerm_sql_server`, `azurerm_cosmosdb_account`, `azurerm_storage_account`, `azurerm_key_vault`

### High (access / networking / compute)

AWS: `aws_iam_role`, `aws_iam_policy`, `aws_vpc`, `aws_subnet`, `aws_eks_cluster`, `aws_lambda_function`, `aws_cloudfront_distribution`

GCP: `google_compute_network`, `google_container_cluster`, `google_project_iam_member`

Azure: `azurerm_virtual_network`, `azurerm_kubernetes_cluster`, `azurerm_role_assignment`

### Medium (security rules / DNS / messaging)

AWS: `aws_security_group`, `aws_route53_record`, `aws_sns_topic`, `aws_sqs_queue`

GCP: `google_compute_firewall`, `google_dns_record_set`

Azure: `azurerm_network_security_group`, `azurerm_dns_zone`

## Using in TUI

Press `r` from the home screen after running a plan to see the risk analysis view. Changes are grouped by risk level with color coding:

- 🔴 Critical — red
- 🟠 High — orange
- 🟡 Medium — yellow
- 🟢 Low — green
