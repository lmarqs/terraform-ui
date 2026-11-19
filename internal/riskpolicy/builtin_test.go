package riskpolicy

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/lmarqs/terraform-ui/internal/config"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

func builtinSnapshot(t *testing.T) Snapshot {
	t.Helper()
	snapshot, err := LoadSource(config.RiskPolicySelection{}, BuiltinModules())
	if err != nil {
		t.Fatalf("built-in complete policy is unavailable: %v", err)
	}
	return snapshot
}

func TestAssess_GivenBuiltinPolicy_ShouldPreserveFixedTypeAndActionSeverities(t *testing.T) {
	snapshot := builtinSnapshot(t)
	for _, tc := range []struct {
		name, types             string
		create, update, destroy sdk.RiskLevel
	}{
		{"critical", `aws_db_instance aws_rds_cluster aws_rds_cluster_instance aws_dynamodb_table aws_s3_bucket aws_efs_file_system aws_fsx_lustre_file_system aws_fsx_windows_file_system aws_redshift_cluster aws_elasticache_cluster aws_elasticache_replication_group aws_docdb_cluster aws_neptune_cluster aws_kms_key aws_kms_alias google_sql_database_instance google_storage_bucket google_kms_key_ring google_kms_crypto_key azurerm_sql_server azurerm_cosmosdb_account azurerm_storage_account azurerm_key_vault`, sdk.RiskMedium, sdk.RiskHigh, sdk.RiskCritical},
		{"high", `aws_iam_role aws_iam_policy aws_iam_user aws_iam_group aws_iam_instance_profile aws_vpc aws_subnet aws_route_table aws_nat_gateway aws_internet_gateway aws_eip aws_lb aws_alb aws_elb aws_ecs_cluster aws_eks_cluster aws_lambda_function aws_cloudfront_distribution google_compute_network google_compute_subnetwork google_container_cluster google_project_iam_member azurerm_virtual_network azurerm_kubernetes_cluster azurerm_role_assignment`, sdk.RiskMedium, sdk.RiskHigh, sdk.RiskCritical},
		{"medium", `aws_security_group aws_security_group_rule aws_network_acl aws_route53_record aws_cloudwatch_log_group aws_sns_topic aws_sqs_queue aws_ecr_repository google_compute_firewall google_dns_record_set azurerm_network_security_group azurerm_dns_zone`, sdk.RiskLow, sdk.RiskMedium, sdk.RiskHigh},
		{"unlisted", `aws_instance aws_lambda_layer_version local_file`, sdk.RiskLow, sdk.RiskLow, sdk.RiskMedium},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, typ := range strings.Fields(tc.types) {
				for _, action := range []struct {
					actions []string
					level   sdk.RiskLevel
				}{
					{[]string{"create"}, tc.create}, {[]string{"update"}, tc.update}, {[]string{"delete"}, tc.destroy}, {[]string{"delete", "create"}, tc.destroy}, {[]string{"create", "delete"}, tc.destroy},
				} {
					t.Run(typ+"/"+strings.Join(action.actions, "-"), func(t *testing.T) {
						resource := evaluationResource(typ+".worker", typ)
						resource.Actions = action.actions
						resource.Input.Change.Actions = action.actions
						result := Assess(context.Background(), snapshot, sdk.PolicyEvidence{Resources: []sdk.PolicyResourceEvidence{resource}})
						if result.Status != "complete" || result.Resources[0].Level == nil || *result.Resources[0].Level != action.level || len(result.Resources[0].Reasons) != 1 {
							t.Fatalf("%s %v lost fixed severity %s: %+v", typ, action.actions, action.level, result)
						}
						if !reflect.DeepEqual(result.Resources[0].Actions, action.actions) {
							t.Fatalf("action order changed: %+v", result.Resources[0])
						}
						if result.Policy.Kind != "builtin" || result.Policy.ID != "builtin/v1" {
							t.Fatalf("built-in ownership lost: %+v", result.Policy)
						}
					})
				}
			}
		})
	}
}

func TestAssess_GivenBuiltinLambdaEvidence_ShouldExplainInContractPriorityOrder(t *testing.T) {
	snapshot := builtinSnapshot(t)
	for _, tc := range []struct {
		name, actions, before, after, masks, code string
		paths                                     [][]any
		level                                     sdk.RiskLevel
	}{
		{"create uncertain", "[\"create\"]", "null", "{}", `,"after_unknown":true`, "mutation-action", nil, sdk.RiskMedium},
		{"delete", "[\"delete\"]", `{"filename":"old"}`, "null", "", "mutation-action", nil, sdk.RiskCritical},
		{"delete create", "[\"delete\",\"create\"]", `{"filename":"old"}`, `{"filename":"new"}`, "", "mutation-action", nil, sdk.RiskCritical},
		{"create delete", "[\"create\",\"delete\"]", `{"filename":"old"}`, `{"filename":"new"}`, "", "mutation-action", nil, sdk.RiskCritical},
		{"filename unchanged hash", `["update"]`, `{"filename":"old","source_code_hash":"same"}`, `{"filename":"new","source_code_hash":"same"}`, "", "lambda-code-source-change", [][]any{{"before", "filename"}, {"after", "filename"}}, sdk.RiskHigh},
		{"filename null hash", `["update"]`, `{"filename":"old","source_code_hash":null}`, `{"filename":"new","source_code_hash":null}`, "", "lambda-code-source-change", [][]any{{"before", "filename"}, {"after", "filename"}}, sdk.RiskHigh},
		{"filename absent hash", `["update"]`, `{"filename":"old"}`, `{"filename":"new"}`, "", "lambda-code-source-change", [][]any{{"before", "filename"}, {"after", "filename"}}, sdk.RiskHigh},
		{"filename unknown hash", `["update"]`, `{"filename":"old","source_code_hash":"old"}`, `{"filename":"new"}`, `,"after_unknown":{"source_code_hash":true}`, "lambda-code-source-change", [][]any{{"before", "filename"}, {"after", "filename"}}, sdk.RiskHigh},
		{"all changed source fields", `["update"]`, `{"filename":"old","s3_bucket":"old","s3_key":"old","s3_object_version":"old","image_uri":"old","source_code_hash":"old"}`, `{"filename":"new","s3_bucket":"new","s3_key":"new","s3_object_version":"new","image_uri":"new","source_code_hash":"new"}`, "", "lambda-code-source-change", [][]any{{"before", "filename"}, {"after", "filename"}, {"before", "s3_bucket"}, {"after", "s3_bucket"}, {"before", "s3_key"}, {"after", "s3_key"}, {"before", "s3_object_version"}, {"after", "s3_object_version"}, {"before", "image_uri"}, {"after", "image_uri"}, {"before", "source_code_hash"}, {"after", "source_code_hash"}}, sdk.RiskHigh},
		{"unknown hash", `["update"]`, `{"source_code_hash":"old"}`, `{}`, `,"after_unknown":{"source_code_hash":true}`, "lambda-source-uncertain", nil, sdk.RiskHigh},
		{"sensitive source", `["update"]`, `{"filename":"old"}`, `{"filename":"private"}`, `,"after_sensitive":{"filename":true}`, "lambda-source-uncertain", nil, sdk.RiskHigh},
		{"memory with uncertain source", `["update"]`, `{"memory_size":128,"filename":"old"}`, `{"memory_size":256}`, `,"after_unknown":{"filename":true}`, "lambda-source-uncertain", nil, sdk.RiskHigh},
		{"memory known equal source", `["update"]`, `{"filename":"same","s3_bucket":"same","s3_key":"same","s3_object_version":"same","image_uri":"same","source_code_hash":"same","memory_size":128}`, `{"filename":"same","s3_bucket":"same","s3_key":"same","s3_object_version":"same","image_uri":"same","source_code_hash":"same","memory_size":256}`, "", "lambda-configuration-update", [][]any{{"before", "memory_size"}, {"after", "memory_size"}}, sdk.RiskHigh},
		{"layers update", `["update"]`, `{"layers":["old"]}`, `{"layers":["new"]}`, "", "lambda-configuration-update", [][]any{{"before", "layers"}, {"after", "layers"}}, sdk.RiskHigh},
		{"no known change", `["update"]`, `{}`, `{}`, "", "type-action-default", nil, sdk.RiskHigh},
		{"whole known null", `["update"]`, `null`, `null`, "", "type-action-default", nil, sdk.RiskHigh},
		{"known null hash", `["update"]`, `{"source_code_hash":null}`, `{"source_code_hash":null}`, "", "type-action-default", nil, sdk.RiskHigh},
		{"whole unknown", `["update"]`, `null`, `null`, `,"after_unknown":true`, "lambda-source-uncertain", nil, sdk.RiskHigh},
	} {
		t.Run(tc.name, func(t *testing.T) {
			summary := acquiredEvidence(t, fmt.Sprintf(`{"actions":%s,"before":%s,"after":%s%s}`, tc.actions, tc.before, tc.after, tc.masks))
			result := Assess(context.Background(), snapshot, *summary.PolicyEvidence)
			if result.Status != "complete" || result.Resources[0].Level == nil || *result.Resources[0].Level != tc.level || len(result.Resources[0].Reasons) != 1 {
				t.Fatalf("Lambda evidence was not classified at %s: %+v", tc.level, result)
			}
			reason := result.Resources[0].Reasons[0]
			if reason.Code != tc.code {
				t.Errorf("reason code=%s, want %s: %+v", reason.Code, tc.code, reason)
			}
			if tc.paths != nil {
				actual, err := json.Marshal(reason.Paths)
				if err != nil {
					t.Fatal(err)
				}
				expected, err := json.Marshal(tc.paths)
				if err != nil {
					t.Fatal(err)
				}
				if string(actual) != string(expected) {
					t.Errorf("paths=%s, want %s", actual, expected)
				}
			}
			if strings.Contains(strings.ToLower(reason.Message), "placeholder") || strings.Contains(strings.ToLower(reason.Message), "externally managed") || strings.Contains(strings.ToLower(reason.Message), "memory-only") {
				t.Errorf("reason invents unsupported ownership or equality: %s", reason.Message)
			}
		})
	}
}
