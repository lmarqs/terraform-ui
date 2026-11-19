package risk

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

func deposedRiskPlugin(t *testing.T, assessment sdk.PolicyAssessment) *Plugin {
	t.Helper()
	summary := &sdk.PlanSummary{Changes: []sdk.PlanChange{
		{Resource: sdk.Resource{Address: "aws_instance.web", Deposed: "00000001"}, Action: sdk.ActionDelete},
		{Resource: sdk.Resource{Address: "aws_instance.web"}, Action: sdk.ActionUpdate},
	}}
	p, _, _ := newAssessedPlugin(t, summary, assessment)
	runLifecycleCmd(p, p.Activate(Input{}))
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

	p.moveDown()
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
	runLifecycleCmd(p, p.Activate(Input{JSON: true}))

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

func TestAnalyze_GivenOnlyDeposedRowAtAddress_ShouldNotJoinItToTheCurrentObject(t *testing.T) {
	high := sdk.RiskHigh
	summary := &sdk.PlanSummary{Changes: []sdk.PlanChange{{Resource: sdk.Resource{Address: "aws_instance.web"}, Action: sdk.ActionUpdate}}}
	p, _, _ := newAssessedPlugin(t, summary, sdk.PolicyAssessment{Status: sdk.AssessmentComplete, OverallLevel: &high, Resources: []sdk.PolicyResourceAssessment{
		{Address: "aws_instance.web", Deposed: "00000001", Actions: []string{"delete"}, Status: sdk.ResourceClassified, Level: &high},
	}})
	runLifecycleCmd(p, p.Activate(Input{}))

	if len(p.items) != 0 {
		t.Errorf("items = %+v, want none when only the deposed row exists", p.items)
	}
}

func TestReport_GivenControlCharacterInDeposedKey_ShouldEscapeItInText(t *testing.T) {
	high := sdk.RiskHigh
	p := newReportPlugin(t, sdk.PolicyAssessment{
		Status: sdk.AssessmentComplete, PlanID: "sha256:reviewed", OverallLevel: &high,
		Policy: sdk.PolicyIdentity{Kind: "custom", ID: "custom/sha256:policy", Digest: "sha256:policy"},
		Resources: []sdk.PolicyResourceAssessment{
			{Address: "aws_instance.web", Deposed: "0001\x1b[31m", Actions: []string{"delete"}, Status: sdk.ResourceClassified, Level: &high},
			{Address: "aws_instance.web", Deposed: "0002\x1b[31m", Actions: []string{"delete"}, Status: sdk.ResourceNotAssessed, Exclusion: sdk.ExclusionNoOp},
		},
	})
	runLifecycleCmd(p, p.Activate(Input{}))

	out, err := reportStdout(t, p)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`delete aws_instance.web (deposed "0001\x1b[31m"): high`, `delete aws_instance.web (deposed "0002\x1b[31m"): not assessed (no_op)`} {
		if !strings.Contains(string(out), want) {
			t.Errorf("text = %q, want line containing %q", out, want)
		}
	}
	if strings.ContainsRune(string(out), '\x1b') {
		t.Errorf("text report contains an unescaped escape byte: %q", out)
	}
}

func TestStderr_GivenDeposedInstanceFailed_ShouldNameItsDeposedKey(t *testing.T) {
	low := sdk.RiskLow
	for _, tc := range []struct {
		name, deposed, want string
	}{
		{"plain key", "00000001", "risk unavailable: result_invalid for aws_instance.web (deposed 00000001)\n"},
		{"key with terminal controls", "0000\x1b[31m", "risk unavailable: result_invalid for aws_instance.web (deposed \"0000\\x1b[31m\")\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := deposedRiskPlugin(t, sdk.PolicyAssessment{Status: sdk.AssessmentPartial, Resources: []sdk.PolicyResourceAssessment{
				{Address: "aws_instance.web", Deposed: tc.deposed, Actions: []string{"delete"}, Status: sdk.ResourceUnavailable, ErrorCode: sdk.ErrorResultInvalid},
				{Address: "aws_instance.web", Actions: []string{"update"}, Status: sdk.ResourceClassified, Level: &low, Reasons: []sdk.PolicyReason{{Code: "current-object", Message: "Current object update."}}},
			}})

			if got := string(p.Stderr()); got != tc.want {
				t.Fatalf("stderr = %q, want %q", got, tc.want)
			}
		})
	}
}

func twoDeposedDeletesPlugin(t *testing.T) *Plugin {
	t.Helper()
	high := sdk.RiskHigh
	summary := &sdk.PlanSummary{Changes: []sdk.PlanChange{
		{Resource: sdk.Resource{Address: "aws_instance.web", Deposed: "00000001"}, Action: sdk.ActionDelete},
		{Resource: sdk.Resource{Address: "aws_instance.web", Deposed: "00000002"}, Action: sdk.ActionDelete},
	}}
	p, _, _ := newAssessedPlugin(t, summary, sdk.PolicyAssessment{Status: sdk.AssessmentComplete, OverallLevel: &high, Resources: []sdk.PolicyResourceAssessment{
		{Address: "aws_instance.web", Deposed: "00000001", Actions: []string{"delete"}, Status: sdk.ResourceClassified, Level: &high},
		{Address: "aws_instance.web", Deposed: "00000002", Actions: []string{"delete"}, Status: sdk.ResourceClassified, Level: &high},
	}})
	runLifecycleCmd(p, p.Activate(Input{}))
	return p
}

func TestView_GivenTwoDeposedDeletesAtSameAddress_ShouldLabelEachRowWithItsDeposedKey(t *testing.T) {
	view := ansi.Strip(twoDeposedDeletesPlugin(t).View(120, 30))

	for _, want := range []string{"aws_instance.web (deposed 00000001)", "aws_instance.web (deposed 00000002)"} {
		if !strings.Contains(view, want) {
			t.Errorf("view missing %q:\n%s", want, view)
		}
	}
}

func TestInspect_GivenTwoDeposedDeletesAtSameAddress_ShouldShowTheSelectedRowsDeposedKey(t *testing.T) {
	for i, key := range []string{"00000001", "00000002"} {
		p := twoDeposedDeletesPlugin(t)
		for range i {
			p.moveDown()
		}
		_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEnter})
		runLifecycleCmd(p, cmd)

		want := "aws_instance.web (deposed " + key + ")"
		if view := ansi.Strip(p.View(120, 30)); !strings.Contains(view, "Address:  "+want) {
			t.Errorf("row %d inspector missing %q in the Address line:\n%s", i, want, view)
		}
		frame, ok := p.Stack().Peek().(*inspectFrame)
		if !ok {
			t.Fatalf("row %d: top frame = %T, want *inspectFrame", i, p.Stack().Peek())
		}
		if frame.Address != want {
			t.Errorf("row %d frame address = %q, want %q", i, frame.Address, want)
		}
	}
}
