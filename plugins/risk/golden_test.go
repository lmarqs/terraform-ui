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

func TestView_Given_MixedRisk_ShouldRender_GroupedChanges(t *testing.T) {
	p := newGoldenPlugin()
	p.status = sdk.StatusDone
	p.overall = sdk.RiskHigh
	p.total = 4
	p.groups = []RiskGroup{
		{
			Level: sdk.RiskHigh,
			Changes: []sdk.PlanChange{
				{Resource: sdk.Resource{Address: "aws_db_instance.main"}, Action: sdk.ActionDelete, Risk: sdk.RiskHigh},
			},
		},
		{
			Level: sdk.RiskMedium,
			Changes: []sdk.PlanChange{
				{Resource: sdk.Resource{Address: "aws_security_group.web"}, Action: sdk.ActionUpdate, Risk: sdk.RiskMedium},
			},
		},
		{
			Level: sdk.RiskLow,
			Changes: []sdk.PlanChange{
				{Resource: sdk.Resource{Address: "aws_instance.web"}, Action: sdk.ActionCreate, Risk: sdk.RiskLow},
				{Resource: sdk.Resource{Address: "aws_s3_bucket.logs"}, Action: sdk.ActionCreate, Risk: sdk.RiskLow},
			},
		},
	}

	sdktest.AssertGolden(t, p.View(80, 18))
}

func TestView_Given_CriticalRisk_ShouldRender_CriticalBanner(t *testing.T) {
	p := newGoldenPlugin()
	p.status = sdk.StatusDone
	p.overall = sdk.RiskCritical
	p.total = 1
	p.groups = []RiskGroup{
		{
			Level: sdk.RiskCritical,
			Changes: []sdk.PlanChange{
				{Resource: sdk.Resource{Address: "aws_rds_cluster.production"}, Action: sdk.ActionDelete, Risk: sdk.RiskCritical},
			},
		},
	}

	sdktest.AssertGolden(t, p.View(80, 18))
}
