package blastradius

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

func deposedBlastLines(t *testing.T, assessment sdk.PolicyAssessment) (updateBlock, deleteBlock string) {
	t.Helper()
	summary := &sdk.PlanSummary{Changes: []sdk.PlanChange{
		{Resource: sdk.Resource{Address: "aws_instance.web", Deposed: "00000001"}, Action: sdk.ActionDelete},
		{Resource: sdk.Resource{Address: "aws_instance.web"}, Action: sdk.ActionUpdate},
	}}
	p := activateAssessmentBlast(t, summary, assessment)
	p.ToggleExpand()
	lines := strings.Split(ansi.Strip(p.View(120, 40)), "\n")
	for i, line := range lines {
		block := line
		for _, next := range lines[i+1:] {
			if !strings.HasPrefix(next, "       ") {
				break
			}
			block += "\n" + next
		}
		switch {
		case strings.Contains(line, "~ aws_instance.web"):
			updateBlock = block
		case strings.Contains(line, "- aws_instance.web"):
			deleteBlock = block
		}
	}
	if updateBlock == "" || deleteBlock == "" {
		t.Fatalf("expanded module must list the update and the deposed delete:\n%s", strings.Join(lines, "\n"))
	}
	return updateBlock, deleteBlock
}

func TestView_GivenUpdateAndDeposedDeleteAtSameAddress_ShouldListEachWithItsOwnLevelAndReasons(t *testing.T) {
	low, high := sdk.RiskLow, sdk.RiskHigh
	updateBlock, deleteBlock := deposedBlastLines(t, sdk.PolicyAssessment{Status: sdk.AssessmentComplete, OverallLevel: &high, Resources: []sdk.PolicyResourceAssessment{
		{Address: "aws_instance.web", Deposed: "00000001", Actions: []string{"delete"}, Status: sdk.ResourceClassified, Level: &high, Reasons: []sdk.PolicyReason{{Code: "deposed-object", Message: "Deposed object cleanup."}}},
		{Address: "aws_instance.web", Actions: []string{"update"}, Status: sdk.ResourceClassified, Level: &low, Reasons: []sdk.PolicyReason{{Code: "current-object", Message: "Current object update."}}},
	}})

	if !strings.Contains(updateBlock, "[low]") || !strings.Contains(updateBlock, "Current object update.") || strings.Contains(updateBlock, "[HIGH]") || strings.Contains(updateBlock, "Deposed object cleanup.") {
		t.Errorf("update = %q, want its own [low] badge and reason", updateBlock)
	}
	if !strings.Contains(deleteBlock, "[HIGH]") || !strings.Contains(deleteBlock, "Deposed object cleanup.") || strings.Contains(deleteBlock, "[low]") || strings.Contains(deleteBlock, "Current object update.") {
		t.Errorf("deposed delete = %q, want its own [HIGH] badge and reason", deleteBlock)
	}
}

func TestView_GivenUpdateClassifiedAndDeposedDeleteFailed_ShouldMarkOnlyTheDeposedDeleteUnavailable(t *testing.T) {
	low := sdk.RiskLow
	updateBlock, deleteBlock := deposedBlastLines(t, sdk.PolicyAssessment{Status: sdk.AssessmentPartial, Resources: []sdk.PolicyResourceAssessment{
		{Address: "aws_instance.web", Deposed: "00000001", Actions: []string{"delete"}, Status: sdk.ResourceUnavailable, ErrorCode: sdk.ErrorResultInvalid},
		{Address: "aws_instance.web", Actions: []string{"update"}, Status: sdk.ResourceClassified, Level: &low},
	}})

	if !strings.Contains(updateBlock, "[low]") || strings.Contains(updateBlock, "unavailable") {
		t.Errorf("update = %q, want [low] without the deposed object's failure", updateBlock)
	}
	if !strings.Contains(deleteBlock, "risk unavailable: result_invalid") {
		t.Errorf("deposed delete = %q, want its own result_invalid failure", deleteBlock)
	}
}

func TestView_GivenUpdateClassifiedAndDeposedDeleteNotAssessed_ShouldMarkOnlyTheDeposedDeleteNotAssessed(t *testing.T) {
	low := sdk.RiskLow
	updateBlock, deleteBlock := deposedBlastLines(t, sdk.PolicyAssessment{Status: sdk.AssessmentComplete, OverallLevel: &low, Resources: []sdk.PolicyResourceAssessment{
		{Address: "aws_instance.web", Deposed: "00000001", Actions: []string{"forget"}, Status: sdk.ResourceNotAssessed, Exclusion: sdk.ExclusionForget},
		{Address: "aws_instance.web", Actions: []string{"update"}, Status: sdk.ResourceClassified, Level: &low},
	}})

	if !strings.Contains(updateBlock, "[low]") || strings.Contains(updateBlock, "not assessed") {
		t.Errorf("update = %q, want [low] without the deposed object's exclusion", updateBlock)
	}
	if !strings.Contains(deleteBlock, "Risk: not assessed") {
		t.Errorf("deposed delete = %q, want its own not-assessed status", deleteBlock)
	}
}
