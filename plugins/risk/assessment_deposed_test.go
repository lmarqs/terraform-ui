package risk

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/sdktest"
)

func deposedRiskPlugin(t *testing.T, assessment sdk.PolicyAssessment) *Plugin {
	t.Helper()
	p, h := newLifecyclePlugin(t, &sdktest.MockService{})
	h.Deps.AssessRisk = func(context.Context, *sdk.Context, *sdk.PlanSummary) sdk.PolicyAssessment { return assessment }
	runLifecycleCmd(p, p.Init(h.Deps))
	summary := &sdk.PlanSummary{Changes: []sdk.PlanChange{
		{Resource: sdk.Resource{Address: "aws_instance.web", Deposed: "00000001"}, Action: sdk.ActionDelete},
		{Resource: sdk.Resource{Address: "aws_instance.web"}, Action: sdk.ActionUpdate},
	}}
	runLifecycleCmd(p, p.HandlePlanCompleted(sdk.PlanCompletedEvent{Summary: summary}))
	runLifecycleCmd(p, p.Activate())
	return p
}

func sectionOf(t *testing.T, view, heading string) string {
	t.Helper()
	start := strings.Index(view, heading)
	if start < 0 {
		t.Fatalf("view has no %q group:\n%s", heading, view)
	}
	section := strings.TrimPrefix(view[start+len(heading):], " ---")
	if end := strings.Index(section, "---"); end >= 0 {
		section = section[:end]
	}
	return section
}

func TestView_GivenUpdateAndDeposedDeleteAtSameAddress_ShouldGroupEachInstanceUnderItsOwnLevelAndReasons(t *testing.T) {
	low, high := sdk.RiskLow, sdk.RiskHigh
	p := deposedRiskPlugin(t, sdk.PolicyAssessment{Status: sdk.AssessmentComplete, OverallLevel: &high, Resources: []sdk.PolicyResourceAssessment{
		{Address: "aws_instance.web", Deposed: "00000001", Actions: []string{"delete"}, Status: sdk.ResourceClassified, Level: &high, Reasons: []sdk.PolicyReason{{Code: "deposed-object", Message: "Deposed object cleanup."}}},
		{Address: "aws_instance.web", Actions: []string{"update"}, Status: sdk.ResourceClassified, Level: &low, Reasons: []sdk.PolicyReason{{Code: "current-object", Message: "Current object update."}}},
	}})

	view := ansi.Strip(p.View(120, 30))

	highSection, lowSection := sectionOf(t, view, "HIGH (1)"), sectionOf(t, view, "LOW (1)")
	if !strings.Contains(highSection, "- aws_instance.web") || !strings.Contains(highSection, "Deposed object cleanup.") || strings.Contains(highSection, "Current object update.") {
		t.Errorf("HIGH group = %q, want the deposed delete with its own reason only", highSection)
	}
	if !strings.Contains(lowSection, "~ aws_instance.web") || !strings.Contains(lowSection, "Current object update.") || strings.Contains(lowSection, "Deposed object cleanup.") {
		t.Errorf("LOW group = %q, want the update with its own reason only", lowSection)
	}

	p.MoveDown()
	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	runLifecycleCmd(p, cmd)
	inspection := ansi.Strip(p.View(120, 30))
	if !strings.Contains(inspection, "Current object update.") || strings.Contains(inspection, "Deposed object cleanup.") {
		t.Errorf("inspecting the update must show only its own reason:\n%s", inspection)
	}
}

func TestView_GivenUpdateClassifiedAndDeposedDeleteFailed_ShouldMarkOnlyTheDeposedInstanceUnavailable(t *testing.T) {
	low := sdk.RiskLow
	p := deposedRiskPlugin(t, sdk.PolicyAssessment{Status: sdk.AssessmentPartial, ErrorCode: sdk.ErrorResultInvalid, Resources: []sdk.PolicyResourceAssessment{
		{Address: "aws_instance.web", Deposed: "00000001", Actions: []string{"delete"}, Status: sdk.ResourceUnavailable, ErrorCode: sdk.ErrorResultInvalid},
		{Address: "aws_instance.web", Actions: []string{"update"}, Status: sdk.ResourceClassified, Level: &low, Reasons: []sdk.PolicyReason{{Code: "current-object", Message: "Current object update."}}},
	}})

	view := ansi.Strip(p.View(120, 30))

	unavailable, lowSection := sectionOf(t, view, "Risk unavailable (1)"), sectionOf(t, view, "LOW (1)")
	if !strings.Contains(unavailable, "- aws_instance.web") || !strings.Contains(unavailable, "risk unavailable: result_invalid") || strings.Contains(unavailable, "~ aws_instance.web") {
		t.Errorf("unavailable group = %q, want only the deposed delete", unavailable)
	}
	if !strings.Contains(lowSection, "~ aws_instance.web") || strings.Contains(lowSection, "risk unavailable") {
		t.Errorf("LOW group = %q, want the update without the deposed object's failure", lowSection)
	}
}

func TestReport_GivenDeposedRowsAtSameAddress_ShouldSortCurrentFirstAndLabelDeposedKey(t *testing.T) {
	low, high := sdk.RiskLow, sdk.RiskHigh
	p := newReportPlugin(t, sdk.PolicyAssessment{
		Status: sdk.AssessmentComplete, PlanID: "sha256:reviewed", OverallLevel: &high,
		Policy: sdk.PolicyIdentity{Kind: "custom", ID: "custom/sha256:policy", Digest: "sha256:policy"},
		Resources: []sdk.PolicyResourceAssessment{
			{Address: "aws_instance.web", Deposed: "00000002", Actions: []string{"delete"}, Status: sdk.ResourceNotAssessed, Exclusion: sdk.ExclusionNoOp},
			{Address: "aws_instance.web", Deposed: "00000001", Actions: []string{"delete"}, Status: sdk.ResourceClassified, Level: &high},
			{Address: "aws_instance.web", Actions: []string{"update"}, Status: sdk.ResourceClassified, Level: &low},
		},
	})
	runLifecycleCmd(p, p.ActivateInput(Input{JSON: true}))

	out, err := reportStdout(t, p)
	if err != nil {
		t.Fatal(err)
	}
	var report struct{ Resources []map[string]any }
	if err := json.Unmarshal(out, &report); err != nil {
		t.Fatal(err)
	}
	var deposed []any
	for _, row := range report.Resources {
		deposed = append(deposed, row["deposed"])
	}
	if !reflect.DeepEqual(deposed, []any{nil, "00000001", "00000002"}) {
		t.Errorf("row deposed keys = %v, want current object first, then 00000001, 00000002", deposed)
	}

	p.input.JSON = false
	text, _ := reportStdout(t, p)
	for _, want := range []string{"update aws_instance.web: low\n", "delete aws_instance.web (deposed 00000001): high\n", "delete aws_instance.web (deposed 00000002): not assessed (no_op)\n"} {
		if !strings.Contains(string(text), want) {
			t.Errorf("text = %q, want line %q", text, want)
		}
	}
}

func TestResourceAssessment_GivenAssessmentWithoutMatchingRow_ShouldReturnNil(t *testing.T) {
	p := &Plugin{assessment: &sdk.PolicyAssessment{Resources: []sdk.PolicyResourceAssessment{{Address: "aws_instance.web", Deposed: "00000001"}}}}

	if row := p.resourceAssessment(sdk.InstanceKey{Address: "aws_instance.web"}); row != nil {
		t.Errorf("resourceAssessment(current object) = %+v, want nil when only the deposed row exists", row)
	}
}
