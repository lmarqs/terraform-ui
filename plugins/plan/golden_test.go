package plan

import (
	"io"
	"log/slog"
	"testing"

	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/sdktest"
)

func newGoldenPlugin() *Plugin {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.log = slog.New(slog.NewTextHandler(io.Discard, nil))
	return p
}

func TestView_Given_Idle_ShouldRender_PromptToRun(t *testing.T) {
	p := newGoldenPlugin()
	p.status = sdk.StatusIdle

	sdktest.AssertGolden(t, p.View(80, 18))
}

func TestView_Given_Loading_ShouldRender_RunningMessage(t *testing.T) {
	p := newGoldenPlugin()
	p.status = sdk.StatusLoading

	sdktest.AssertGolden(t, p.View(80, 18))
}

func TestView_Given_Error_ShouldRender_ErrorMessage(t *testing.T) {
	p := newGoldenPlugin()
	p.status = sdk.StatusError
	p.errMsg = "Error running plan: insufficient permissions"

	sdktest.AssertGolden(t, p.View(80, 18))
}

func TestView_Given_NoChanges_ShouldRender_UpToDateMessage(t *testing.T) {
	p := newGoldenPlugin()
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{Changes: nil}

	sdktest.AssertGolden(t, p.View(80, 18))
}

func TestView_Given_Changes_ShouldRender_ChangeList(t *testing.T) {
	p := newGoldenPlugin()
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "aws_instance.web"}, Action: sdk.ActionCreate, Risk: sdk.RiskLow},
			{Resource: sdk.Resource{Address: "aws_security_group.allow_ssh"}, Action: sdk.ActionUpdate, Risk: sdk.RiskMedium},
			{Resource: sdk.Resource{Address: "aws_db_instance.main"}, Action: sdk.ActionDelete, Risk: sdk.RiskHigh},
		},
		ToCreate: 1,
		ToUpdate: 1,
		ToDelete: 1,
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()

	sdktest.AssertGolden(t, p.View(80, 18))
}

func TestView_Given_Changes_WithSelection_ShouldRender_HighlightedRow(t *testing.T) {
	p := newGoldenPlugin()
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "aws_instance.web"}, Action: sdk.ActionCreate, Risk: sdk.RiskLow},
			{Resource: sdk.Resource{Address: "aws_security_group.allow_ssh"}, Action: sdk.ActionUpdate, Risk: sdk.RiskMedium},
			{Resource: sdk.Resource{Address: "aws_db_instance.main"}, Action: sdk.ActionDelete, Risk: sdk.RiskHigh},
		},
		ToCreate: 1,
		ToUpdate: 1,
		ToDelete: 1,
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()
	p.MoveDown()

	sdktest.AssertGolden(t, p.View(80, 18))
}

func TestView_Given_InspectView_ShouldRender_AttributeDiffs(t *testing.T) {
	p := newGoldenPlugin()
	p.pins = sdk.NewPinService()
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{
				Resource: sdk.Resource{Address: "aws_instance.web", Type: "aws_instance", ProviderName: "registry.terraform.io/hashicorp/aws"},
				Action:   sdk.ActionUpdate,
				Risk:     sdk.RiskMedium,
				AttributeDiffs: []sdk.AttributeDiff{
					{Key: "instance_type", OldValue: "t3.micro", NewValue: "t3.small"},
					{Key: "tags.Name", OldValue: "web-old", NewValue: "web-new"},
					{Key: "password", Sensitive: true},
				},
			},
		},
		ToUpdate: 1,
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()
	p.inspectSelected()

	sdktest.AssertGolden(t, p.renderDetail(80, 18))
}

func TestView_Given_PhantomChange_ShouldRender_PhantomMarker(t *testing.T) {
	p := newGoldenPlugin()
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "aws_instance.web"}, Action: sdk.ActionUpdate, IsPhantom: true},
			{Resource: sdk.Resource{Address: "aws_s3_bucket.logs"}, Action: sdk.ActionCreate, Risk: sdk.RiskLow},
		},
		ToCreate: 1,
		ToUpdate: 1,
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()

	sdktest.AssertGolden(t, p.View(80, 18))
}

func TestView_Given_CriticalRisk_ShouldRender_RiskWarning(t *testing.T) {
	p := newGoldenPlugin()
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "aws_rds_cluster.production"}, Action: sdk.ActionDelete, Risk: sdk.RiskCritical},
		},
		ToDelete: 1,
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()

	sdktest.AssertGolden(t, p.View(80, 18))
}

func TestView_Given_PinnedChange_ShouldRender_PinMarker(t *testing.T) {
	p := newGoldenPlugin()
	p.status = sdk.StatusDone
	p.pins = sdk.NewPinService()
	p.pins.Toggle("aws_instance.web")
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "aws_instance.web"}, Action: sdk.ActionCreate, Risk: sdk.RiskLow},
			{Resource: sdk.Resource{Address: "aws_s3_bucket.logs"}, Action: sdk.ActionCreate, Risk: sdk.RiskLow},
		},
		ToCreate: 2,
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()

	sdktest.AssertGolden(t, p.View(80, 18))
}
