---
layout: default
title: Terraform Plan Risk Analysis
parent: Features — Terraform Plan Analysis Tools
nav_order: 1
seo:
  description: Risk classifications for eligible Terraform plan mutations, with built-in action and resource-type rules or a complete project policy.
description: "Explains the built-in mutation classifications and how they differ from complete custom policies."
audience: "Users triaging a plan, and contributors changing the risk rules"
---

## Overview

Risk classifies eligible resource mutations independently with one selected complete policy. Plan's badges,
Risk's grouped review and command report, and Blast Radius's risk input share the assessment. The
[Risk plugin guide](../plugins/risk.md) explains how to open the view, supply a plan and select a policy.

The rules below describe the **built-in policy**, selected when no custom source is configured. A custom source
owns every classification and reason, including its explicit defaults; it never gains hidden built-in fallback.
The [policy contract](../reference/risk-policy-contract.md) owns eligibility, input/result validation and failure
semantics. Reads, no-ops, pure moves/imports and output-only changes are not assessed, rather than classified
as `none`. Mutation accompanying a move/import is assessed normally.

## How It Works

### Risk Levels

| Level | Meaning | Example |
|-------|---------|---------|
| **Critical** | Deleting or replacing a critical or high-risk resource | Deleting an RDS instance, replacing a KMS key, deleting a VPC |
| **High** | Updating a critical or high-risk resource, or deleting a medium-risk one | Modifying IAM roles or a VPC, deleting a security group |
| **Medium** | Updating a medium-risk resource, creating a critical or high-risk one, or deleting an unlisted one | Updating security groups, modifying Route53 records, creating an IAM role |
| **Low** | Creating a medium-risk or unlisted resource, or updating an unlisted one | Creating an SNS topic, updating an `aws_instance` |
| **None** | Explicit policy classification | A custom policy may explicitly return `none`; the built-in mutation rules below do not |

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

**No-op and read:** Not assessed.

### Resource Type Categories

These are every type in each category; the production policy source is `internal/riskpolicy/builtin.rego`.

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

The [Risk Analysis plugin](../plugins/risk.md) groups mutations by their selected-policy levels. The
[Plan plugin](../plugins/plan.md) shows the same classifications and explanations. A complete nonempty assessment
has an overall level equal to its highest classification; unavailable or partial assessment has no overall level.
Blast Radius combines valid selected-policy risk with its existing counts and destructive-action arithmetic;
its impact category is distinct from a risk classification.

## Built-in Lambda explanations

All updates of `aws_lambda_function` retain the built-in **high** classification. The explanation distinguishes
observed evidence without lowering configuration-only updates:

- Known changes to `filename`, `s3_bucket`, `s3_key`, `s3_object_version`, `image_uri` or `source_code_hash`
  identify changed code-source attributes, not deployed bytes or deployment ownership.
- Unknown or sensitive source evidence states that code equality cannot be established.
- Other observed attribute changes describe configuration evidence and do not establish a code upload.
- With no more specific evidence, the policy returns its explicit action/type explanation.

Known code-source changes take explanation precedence over uncertain source evidence, then configuration changes.
An unchanged hash, including a hash that stays null or absent, does not invent a source change; an unknown hash
remains uncertain.
The [authoring guide](../guides/risk-policy-authoring.md) provides a complete custom Lambda example and tests.

## Limits

These classifications are heuristics, not a guarantee that a change is safe. A phantom label is advisory:
eligible phantom mutations are still assessed normally. Missing or invalid evidence, policy compilation failure
and invalid or missing classifications produce unavailable risk rather than `none`. Partial classifications
remain inspectable, but cannot produce an aggregate or a successful command report.

For exact distinctions among excluded changes, empty eligibility, explicit `none` and unavailable assessment,
see the [policy contract](../reference/risk-policy-contract.md#assessment-aggregation-and-user-facing-states).
