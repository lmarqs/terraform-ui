package plan

import (
	"context"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/sdktest"
)

func deposedPlanLines(t *testing.T, assessment sdk.PolicyAssessment) (updateLine, deleteLine string) {
	t.Helper()
	summary := &sdk.PlanSummary{ToUpdate: 1, ToDelete: 1, Changes: []sdk.PlanChange{
		{Resource: sdk.Resource{Address: "aws_instance.web", Deposed: "00000001"}, Action: sdk.ActionDelete},
		{Resource: sdk.Resource{Address: "aws_instance.web"}, Action: sdk.ActionUpdate},
	}}
	svc := &sdktest.MockService{PlanFn: func(context.Context, sdk.PlanOptions) (*sdk.PlanSummary, error) { return summary, nil }}
	h := sdktest.NewDeps(svc)
	h.Deps.AssessRisk = func(context.Context, *sdk.Context, *sdk.PlanSummary) sdk.PolicyAssessment { return assessment }
	p := New(svc).(*Plugin)
	p.Init(h.Deps)
	defer p.Shutdown()
	drainAssessmentPlan(t, p, p.Activate(Input{}), nil)
	for _, line := range strings.Split(ansi.Strip(p.View(120, 40)), "\n") {
		switch {
		case strings.Contains(line, "~ aws_instance.web"):
			updateLine = line
		case strings.Contains(line, "- aws_instance.web"):
			deleteLine = line
		}
	}
	if updateLine == "" || deleteLine == "" {
		t.Fatalf("view must list the update and the deposed delete as separate rows:\n%s", ansi.Strip(p.View(120, 40)))
	}
	return updateLine, deleteLine
}

func TestView_GivenUpdateAndDeposedDeleteAtSameAddress_ShouldBadgeEachRowWithItsOwnLevel(t *testing.T) {
	low, high := sdk.RiskLow, sdk.RiskHigh
	updateLine, deleteLine := deposedPlanLines(t, sdk.PolicyAssessment{Status: sdk.AssessmentComplete, OverallLevel: &high, Resources: []sdk.PolicyResourceAssessment{
		{Address: "aws_instance.web", Deposed: "00000001", Actions: []string{"delete"}, Status: sdk.ResourceClassified, Level: &high},
		{Address: "aws_instance.web", Actions: []string{"update"}, Status: sdk.ResourceClassified, Level: &low},
	}})

	if !strings.Contains(updateLine, "[low]") || strings.Contains(updateLine, "[HIGH]") {
		t.Errorf("update row = %q, want its own [low] badge", updateLine)
	}
	if !strings.Contains(deleteLine, "[HIGH]") || strings.Contains(deleteLine, "[low]") {
		t.Errorf("deposed delete row = %q, want its own [HIGH] badge", deleteLine)
	}
}

func TestView_GivenUpdateClassifiedAndDeposedDeleteFailed_ShouldMarkOnlyTheDeposedRowUnavailable(t *testing.T) {
	low := sdk.RiskLow
	updateLine, deleteLine := deposedPlanLines(t, sdk.PolicyAssessment{Status: sdk.AssessmentPartial, Resources: []sdk.PolicyResourceAssessment{
		{Address: "aws_instance.web", Deposed: "00000001", Actions: []string{"delete"}, Status: sdk.ResourceUnavailable, ErrorCode: sdk.ErrorResultInvalid},
		{Address: "aws_instance.web", Actions: []string{"update"}, Status: sdk.ResourceClassified, Level: &low},
	}})

	if !strings.Contains(updateLine, "[low]") || strings.Contains(updateLine, "unavailable") {
		t.Errorf("update row = %q, want [low] without the deposed object's failure", updateLine)
	}
	if !strings.Contains(deleteLine, "risk unavailable: result_invalid") {
		t.Errorf("deposed delete row = %q, want its own result_invalid failure", deleteLine)
	}
}

func TestSetAssessment_GivenDetailOpenOnDeposedDelete_ShouldRebuildDetailForThatInstance(t *testing.T) {
	low, high := sdk.RiskLow, sdk.RiskHigh
	p := New(&sdktest.MockService{}).(*Plugin)
	p.rawSummary = &sdk.PlanSummary{Changes: []sdk.PlanChange{
		{Resource: sdk.Resource{Address: "aws_instance.web"}, Action: sdk.ActionUpdate},
		{Resource: sdk.Resource{Address: "aws_instance.web", Deposed: "00000001"}, Action: sdk.ActionDelete},
	}}
	p.detailAddr, p.detailDeposed = "aws_instance.web", "00000001"

	p.setAssessment(&sdk.PolicyAssessment{Status: sdk.AssessmentComplete, Resources: []sdk.PolicyResourceAssessment{
		{Address: "aws_instance.web", Status: sdk.ResourceClassified, Level: &low},
		{Address: "aws_instance.web", Deposed: "00000001", Status: sdk.ResourceClassified, Level: &high},
	}})

	detail := ansi.Strip(p.detail)
	if !strings.Contains(detail, "delete") || !strings.Contains(detail, "[HIGH]") || strings.Contains(detail, "update") {
		t.Errorf("detail = %q, want the deposed delete with its own [HIGH] risk", detail)
	}
}

func TestBuildInspectContent_GivenUpdateAndDeposedDelete_ShouldShowOnlyTheOwnReasons(t *testing.T) {
	low, high := sdk.RiskLow, sdk.RiskHigh
	p := New(&sdktest.MockService{}).(*Plugin)
	p.assessment = &sdk.PolicyAssessment{Status: sdk.AssessmentComplete, Resources: []sdk.PolicyResourceAssessment{
		{Address: "aws_instance.web", Status: sdk.ResourceClassified, Level: &low, Reasons: []sdk.PolicyReason{{Code: "current-object", Message: "Current object update."}}},
		{Address: "aws_instance.web", Deposed: "00000001", Status: sdk.ResourceClassified, Level: &high, Reasons: []sdk.PolicyReason{{Code: "deposed-object", Message: "Deposed object cleanup."}}},
	}}

	detail := p.buildInspectContent(&sdk.PlanChange{Resource: sdk.Resource{Address: "aws_instance.web", Deposed: "00000001"}, Action: sdk.ActionDelete})

	if !strings.Contains(detail, "Deposed object cleanup.") || strings.Contains(detail, "Current object update.") {
		t.Errorf("detail = %q, want only the deposed object's reason", detail)
	}
}

func TestSetAssessment_GivenFailuresAtSameAddress_ShouldPickCurrentObjectFailureFirst(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)

	p.setAssessment(&sdk.PolicyAssessment{Status: sdk.AssessmentPartial, Resources: []sdk.PolicyResourceAssessment{
		{Address: "aws_instance.web", Deposed: "00000001", Status: sdk.ResourceUnavailable, ErrorCode: sdk.ErrorResultInvalid},
		{Address: "aws_instance.web", Status: sdk.ResourceUnavailable, ErrorCode: sdk.ErrorEvaluationFailed},
	}})

	if p.riskDiagnostic != string(sdk.ErrorEvaluationFailed) {
		t.Errorf("riskDiagnostic = %q, want the current object's %q (address, then deposed key order)", p.riskDiagnostic, sdk.ErrorEvaluationFailed)
	}
}
