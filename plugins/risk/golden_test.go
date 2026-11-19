package risk

import (
	"testing"

	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/sdktest"
)

func newGoldenPlugin() *Plugin {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	return p
}

func TestView_Given_Idle_ShouldRender_NoPlanMessage(t *testing.T) {
	p := newGoldenPlugin()
	p.status = sdk.StatusIdle

	sdktest.AssertGolden(t, p.View(80, 18))
}

func TestView_Given_NoChanges_ShouldRender_NoRiskMessage(t *testing.T) {
	p := newGoldenPlugin()
	p.status = sdk.StatusDone
	p.groups = nil

	sdktest.AssertGolden(t, p.View(80, 18))
}

// goldenRow classifies address at level, with reason when it is not empty.
func goldenRow(address string, level sdk.RiskLevel, reason string) sdk.PolicyResourceAssessment {
	row := sdk.PolicyResourceAssessment{Address: address, Status: sdk.ResourceClassified, Level: &level}
	if reason != "" {
		row.Reasons = []sdk.PolicyReason{{Code: "destructive", Message: reason}}
	}
	return row
}

func TestView_Given_MixedRisk_ShouldRender_GroupedChanges(t *testing.T) {
	high := sdk.RiskHigh
	p, _, _ := newAssessedPlugin(t, &sdk.PlanSummary{Changes: []sdk.PlanChange{
		{Resource: sdk.Resource{Address: "aws_db_instance.main"}, Action: sdk.ActionDelete},
		{Resource: sdk.Resource{Address: "aws_security_group.web"}, Action: sdk.ActionUpdate},
		{Resource: sdk.Resource{Address: "aws_instance.web"}, Action: sdk.ActionCreate},
		{Resource: sdk.Resource{Address: "aws_s3_bucket.logs"}, Action: sdk.ActionCreate},
	}}, sdk.PolicyAssessment{Status: sdk.AssessmentComplete, OverallLevel: &high, Resources: []sdk.PolicyResourceAssessment{
		goldenRow("aws_db_instance.main", sdk.RiskHigh, "destructive operation"),
		goldenRow("aws_security_group.web", sdk.RiskMedium, ""),
		goldenRow("aws_instance.web", sdk.RiskLow, ""),
		goldenRow("aws_s3_bucket.logs", sdk.RiskLow, ""),
	}})
	runLifecycleCmd(p, p.Activate(Input{}))

	sdktest.AssertGolden(t, p.View(80, 18))
}

func TestView_Given_CriticalRisk_ShouldRender_CriticalBanner(t *testing.T) {
	critical := sdk.RiskCritical
	p, _, _ := newAssessedPlugin(t, &sdk.PlanSummary{Changes: []sdk.PlanChange{
		{Resource: sdk.Resource{Address: "aws_rds_cluster.production"}, Action: sdk.ActionDelete},
	}}, sdk.PolicyAssessment{Status: sdk.AssessmentComplete, OverallLevel: &critical, Resources: []sdk.PolicyResourceAssessment{
		goldenRow("aws_rds_cluster.production", sdk.RiskCritical, "destructive operation"),
	}})
	runLifecycleCmd(p, p.Activate(Input{}))

	sdktest.AssertGolden(t, p.View(80, 18))
}
