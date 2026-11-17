package terraform

import (
	"testing"

	"github.com/lmarqs/terraform-ui/internal/riskpolicy"
	"github.com/open-policy-agent/opa/v1/ast"
)

func TestClassifyRisk(t *testing.T) {
	tests := []struct {
		name     string
		change   PlanChange
		expected RiskLevel
	}{
		// Delete actions on critical resources -> Critical
		{
			name: "delete of RDS instance is critical",
			change: PlanChange{
				Resource: Resource{Type: "aws_db_instance"},
				Action:   ActionDelete,
			},
			expected: RiskCritical,
		},
		{
			name: "delete of S3 bucket is critical",
			change: PlanChange{
				Resource: Resource{Type: "aws_s3_bucket"},
				Action:   ActionDelete,
			},
			expected: RiskCritical,
		},
		{
			name: "delete-then-create of RDS cluster is critical",
			change: PlanChange{
				Resource: Resource{Type: "aws_rds_cluster"},
				Action:   ActionDeleteThenCreate,
			},
			expected: RiskCritical,
		},
		{
			name: "create-then-delete of DynamoDB table is critical",
			change: PlanChange{
				Resource: Resource{Type: "aws_dynamodb_table"},
				Action:   ActionCreateThenDelete,
			},
			expected: RiskCritical,
		},
		// Delete actions on high-risk resources -> Critical
		{
			name: "delete of IAM role is critical",
			change: PlanChange{
				Resource: Resource{Type: "aws_iam_role"},
				Action:   ActionDelete,
			},
			expected: RiskCritical,
		},
		{
			name: "delete of VPC is critical",
			change: PlanChange{
				Resource: Resource{Type: "aws_vpc"},
				Action:   ActionDelete,
			},
			expected: RiskCritical,
		},
		// Delete actions on medium-risk resources -> High
		{
			name: "delete of security group is high",
			change: PlanChange{
				Resource: Resource{Type: "aws_security_group"},
				Action:   ActionDelete,
			},
			expected: RiskHigh,
		},
		{
			name: "delete of route53 record is high",
			change: PlanChange{
				Resource: Resource{Type: "aws_route53_record"},
				Action:   ActionDelete,
			},
			expected: RiskHigh,
		},
		// Delete actions on unknown resources -> Medium
		{
			name: "delete of unknown resource is medium",
			change: PlanChange{
				Resource: Resource{Type: "local_file"},
				Action:   ActionDelete,
			},
			expected: RiskMedium,
		},
		// Update actions on critical resources -> High
		{
			name: "update of RDS instance is high",
			change: PlanChange{
				Resource: Resource{Type: "aws_db_instance"},
				Action:   ActionUpdate,
			},
			expected: RiskHigh,
		},
		{
			name: "update of S3 bucket is high",
			change: PlanChange{
				Resource: Resource{Type: "aws_s3_bucket"},
				Action:   ActionUpdate,
			},
			expected: RiskHigh,
		},
		// Update actions on high-risk resources -> High
		{
			name: "update of IAM role is high",
			change: PlanChange{
				Resource: Resource{Type: "aws_iam_role"},
				Action:   ActionUpdate,
			},
			expected: RiskHigh,
		},
		// Update actions on medium-risk resources -> Medium
		{
			name: "update of security group is medium",
			change: PlanChange{
				Resource: Resource{Type: "aws_security_group"},
				Action:   ActionUpdate,
			},
			expected: RiskMedium,
		},
		// Update actions on unknown resources -> Low
		{
			name: "update of unknown resource is low",
			change: PlanChange{
				Resource: Resource{Type: "local_file"},
				Action:   ActionUpdate,
			},
			expected: RiskLow,
		},
		// Create actions on critical resources -> Medium
		{
			name: "create of RDS instance is medium",
			change: PlanChange{
				Resource: Resource{Type: "aws_db_instance"},
				Action:   ActionCreate,
			},
			expected: RiskMedium,
		},
		// Create actions on high-risk resources -> Medium
		{
			name: "create of IAM role is medium",
			change: PlanChange{
				Resource: Resource{Type: "aws_iam_role"},
				Action:   ActionCreate,
			},
			expected: RiskMedium,
		},
		// Create actions on medium-risk resources -> Low
		{
			name: "create of security group is low",
			change: PlanChange{
				Resource: Resource{Type: "aws_security_group"},
				Action:   ActionCreate,
			},
			expected: RiskLow,
		},
		// Create actions on unknown resources -> Low
		{
			name: "create of simple resource is low",
			change: PlanChange{
				Resource: Resource{Type: "local_file"},
				Action:   ActionCreate,
			},
			expected: RiskLow,
		},
		// NoOp and Read actions -> None
		{
			name: "no-op action is none",
			change: PlanChange{
				Resource: Resource{Type: "aws_db_instance"},
				Action:   ActionNoOp,
			},
			expected: RiskNone,
		},
		{
			name: "read action is none",
			change: PlanChange{
				Resource: Resource{Type: "aws_db_instance"},
				Action:   ActionRead,
			},
			expected: RiskNone,
		},
		// Google and Azure resources
		{
			name: "delete of google SQL instance is critical",
			change: PlanChange{
				Resource: Resource{Type: "google_sql_database_instance"},
				Action:   ActionDelete,
			},
			expected: RiskCritical,
		},
		{
			name: "delete of azure key vault is critical",
			change: PlanChange{
				Resource: Resource{Type: "azurerm_key_vault"},
				Action:   ActionDelete,
			},
			expected: RiskCritical,
		},
		{
			name: "update of google container cluster is high",
			change: PlanChange{
				Resource: Resource{Type: "google_container_cluster"},
				Action:   ActionUpdate,
			},
			expected: RiskHigh,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClassifyRisk(&tt.change)
			if result != tt.expected {
				t.Errorf("ClassifyRisk() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestOverallRisk(t *testing.T) {
	tests := []struct {
		name     string
		changes  []PlanChange
		expected RiskLevel
	}{
		{
			name:     "empty changes returns none",
			changes:  []PlanChange{},
			expected: RiskNone,
		},
		{
			name: "single low risk change",
			changes: []PlanChange{
				{Risk: RiskLow},
			},
			expected: RiskLow,
		},
		{
			name: "highest risk is selected from mixed changes",
			changes: []PlanChange{
				{Risk: RiskLow},
				{Risk: RiskMedium},
				{Risk: RiskCritical},
			},
			expected: RiskCritical,
		},
		{
			name: "all same risk level",
			changes: []PlanChange{
				{Risk: RiskHigh},
				{Risk: RiskHigh},
			},
			expected: RiskHigh,
		},
		{
			name: "critical is highest even when not last",
			changes: []PlanChange{
				{Risk: RiskCritical},
				{Risk: RiskLow},
				{Risk: RiskMedium},
			},
			expected: RiskCritical,
		},
		{
			name: "none among real risks is irrelevant",
			changes: []PlanChange{
				{Risk: RiskNone},
				{Risk: RiskMedium},
				{Risk: RiskNone},
			},
			expected: RiskMedium,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := OverallRisk(tt.changes)
			if result != tt.expected {
				t.Errorf("OverallRisk() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestRiskTypeLists_GivenBuiltinPolicy_ShouldMatchClassifyRiskTypeMaps(t *testing.T) {
	policy := builtinPolicyTypeSets(t)

	tests := []struct {
		rule string
		want map[string]bool
	}{
		{"critical_types", criticalTypes},
		{"high_types", highRiskTypes},
		{"medium_types", mediumRiskTypes},
	}
	for _, tt := range tests {
		t.Run(tt.rule, func(t *testing.T) {
			got, ok := policy[tt.rule]
			if !ok {
				t.Fatalf("builtin.rego defines no %s set", tt.rule)
			}
			for resourceType := range tt.want {
				if !got[resourceType] {
					t.Errorf("%s is in the Go type map but missing from builtin.rego %s", resourceType, tt.rule)
				}
			}
			for resourceType := range got {
				if !tt.want[resourceType] {
					t.Errorf("%s is in builtin.rego %s but missing from the Go type map", resourceType, tt.rule)
				}
			}
		})
	}
}

func builtinPolicyTypeSets(t *testing.T) map[string]map[string]bool {
	t.Helper()
	modules := riskpolicy.BuiltinModules()
	if len(modules) != 1 {
		t.Fatalf("builtin policy has %d modules, want 1", len(modules))
	}
	module, err := ast.ParseModule(modules[0].Name, modules[0].Source)
	if err != nil {
		t.Fatalf("parse builtin policy: %v", err)
	}
	sets := make(map[string]map[string]bool)
	for _, rule := range module.Rules {
		if rule.Head.Value == nil {
			continue
		}
		set, ok := rule.Head.Value.Value.(ast.Set)
		if !ok {
			continue
		}
		members := make(map[string]bool)
		set.Foreach(func(term *ast.Term) {
			if text, ok := term.Value.(ast.String); ok {
				members[string(text)] = true
			}
		})
		sets[string(rule.Head.Name)] = members
	}
	return sets
}
