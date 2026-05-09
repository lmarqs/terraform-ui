package terraform

import "github.com/lmarqs/terraform-ui/pkg/sdk"

var criticalTypes = map[string]bool{
	"aws_db_instance": true, "aws_rds_cluster": true, "aws_rds_cluster_instance": true,
	"aws_dynamodb_table": true, "aws_s3_bucket": true, "aws_efs_file_system": true,
	"aws_fsx_lustre_file_system": true, "aws_fsx_windows_file_system": true,
	"aws_redshift_cluster": true, "aws_elasticache_cluster": true,
	"aws_elasticache_replication_group": true, "aws_docdb_cluster": true,
	"aws_neptune_cluster": true, "aws_kms_key": true, "aws_kms_alias": true,
	"google_sql_database_instance": true, "google_storage_bucket": true,
	"google_kms_key_ring": true, "google_kms_crypto_key": true,
	"azurerm_sql_server": true, "azurerm_cosmosdb_account": true,
	"azurerm_storage_account": true, "azurerm_key_vault": true,
}

var highRiskTypes = map[string]bool{
	"aws_iam_role": true, "aws_iam_policy": true, "aws_iam_user": true,
	"aws_iam_group": true, "aws_iam_instance_profile": true,
	"aws_vpc": true, "aws_subnet": true, "aws_route_table": true,
	"aws_nat_gateway": true, "aws_internet_gateway": true,
	"aws_eip": true, "aws_lb": true, "aws_alb": true, "aws_elb": true,
	"aws_ecs_cluster": true, "aws_eks_cluster": true,
	"aws_lambda_function": true, "aws_cloudfront_distribution": true,
	"google_compute_network": true, "google_compute_subnetwork": true,
	"google_container_cluster": true, "google_project_iam_member": true,
	"azurerm_virtual_network": true, "azurerm_kubernetes_cluster": true,
	"azurerm_role_assignment": true,
}

var mediumRiskTypes = map[string]bool{
	"aws_security_group": true, "aws_security_group_rule": true,
	"aws_network_acl": true, "aws_route53_record": true,
	"aws_cloudwatch_log_group": true, "aws_sns_topic": true,
	"aws_sqs_queue": true, "aws_ecr_repository": true,
	"google_compute_firewall": true, "google_dns_record_set": true,
	"azurerm_network_security_group": true, "azurerm_dns_zone": true,
}

// ClassifyRisk assigns a risk level to a plan change based on the combination
// of its action type (create, update, delete, replace) and the criticality of
// the resource type (e.g., databases and KMS keys are critical).
func ClassifyRisk(change *PlanChange) RiskLevel {
	resourceType := change.Resource.Type

	switch change.Action {
	case ActionDelete, ActionDeleteThenCreate, ActionCreateThenDelete:
		if criticalTypes[resourceType] || highRiskTypes[resourceType] {
			return RiskCritical
		}
		if mediumRiskTypes[resourceType] {
			return RiskHigh
		}
		return RiskHigh

	case ActionUpdate:
		if criticalTypes[resourceType] || highRiskTypes[resourceType] {
			return RiskHigh
		}
		if mediumRiskTypes[resourceType] {
			return RiskMedium
		}
		return RiskMedium

	case ActionCreate:
		if criticalTypes[resourceType] || highRiskTypes[resourceType] {
			return RiskMedium
		}
		return RiskLow

	default:
		return RiskNone
	}
}

// OverallRisk delegates to the SDK implementation.
var OverallRisk = sdk.OverallRisk
