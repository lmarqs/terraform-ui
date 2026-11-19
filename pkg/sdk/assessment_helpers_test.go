package sdk

import (
	"fmt"
	"reflect"
	"testing"
)

func TestParseRiskLevel_WhenInput_ShouldMatchExactLowercaseNames(t *testing.T) {
	tests := []struct {
		input  string
		want   RiskLevel
		wantOK bool
	}{
		{"none", RiskNone, true},
		{"low", RiskLow, true},
		{"medium", RiskMedium, true},
		{"high", RiskHigh, true},
		{"critical", RiskCritical, true},
		{"off", RiskNone, false},
		{"", RiskNone, false},
		{"LOW", RiskNone, false},
		{"Critical", RiskNone, false},
		{" low", RiskNone, false},
		{"low ", RiskNone, false},
		{"urgent", RiskNone, false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, ok := ParseRiskLevel(tt.input)
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("ParseRiskLevel(%q) = (%v, %v), want (%v, %v)", tt.input, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

func TestParseRiskLevel_WhenRoundTripped_ShouldInvertString(t *testing.T) {
	for _, level := range []RiskLevel{RiskNone, RiskLow, RiskMedium, RiskHigh, RiskCritical} {
		got, ok := ParseRiskLevel(level.String())
		if !ok || got != level {
			t.Errorf("ParseRiskLevel(%q) = (%v, %v), want (%v, true)", level.String(), got, ok, level)
		}
	}
}

func TestPolicyAssessment_Usable_WhenStatus_ShouldBeTrueOnlyForCompleteAndEmpty(t *testing.T) {
	tests := []struct {
		status AssessmentStatus
		want   bool
	}{
		{AssessmentComplete, true},
		{AssessmentEmpty, true},
		{AssessmentPartial, false},
		{AssessmentUnavailable, false},
		{AssessmentCancelled, false},
		{AssessmentStatus(""), false},
	}
	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			a := PolicyAssessment{Status: tt.status}
			if got := a.Usable(); got != tt.want {
				t.Errorf("PolicyAssessment{Status: %q}.Usable() = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}

func levelPtr(l RiskLevel) *RiskLevel { return &l }

func TestPolicyAssessment_Rows_WhenKeysShareAddress_ShouldIndexByDeposedKey(t *testing.T) {
	a := PolicyAssessment{Resources: []PolicyResourceAssessment{
		{Address: "aws_instance.web", Status: ResourceClassified, Level: levelPtr(RiskLow)},
		{Address: "aws_instance.web", Deposed: "0a1b2c3d", Status: ResourceClassified, Level: levelPtr(RiskCritical)},
		{Address: "aws_instance.db", Status: ResourceNotAssessed, Exclusion: ExclusionNoOp},
	}}

	rows := a.Rows()

	current, ok := rows[InstanceKey{Address: "aws_instance.web"}]
	if !ok || current.Deposed != "" || current.Level == nil || *current.Level != RiskLow {
		t.Errorf("current object row = (%+v, %v), want the low row with empty Deposed", current, ok)
	}
	deposed, ok := rows[InstanceKey{Address: "aws_instance.web", Deposed: "0a1b2c3d"}]
	if !ok || deposed.Deposed != "0a1b2c3d" || deposed.Level == nil || *deposed.Level != RiskCritical {
		t.Errorf("deposed object row = (%+v, %v), want the critical row", deposed, ok)
	}
	other, ok := rows[InstanceKey{Address: "aws_instance.db"}]
	if !ok || other.Status != ResourceNotAssessed || other.Exclusion != ExclusionNoOp {
		t.Errorf("db row = (%+v, %v), want the not_assessed no_op row", other, ok)
	}
}

func TestPolicyAssessment_Rows_WhenNoRowMatches_ShouldReportNotFound(t *testing.T) {
	a := PolicyAssessment{Resources: []PolicyResourceAssessment{
		{Address: "aws_instance.web", Deposed: "0a1b2c3d", Status: ResourceClassified},
	}}
	rows := a.Rows()
	tests := []struct {
		name string
		key  InstanceKey
	}{
		{"UnknownAddress", InstanceKey{Address: "aws_instance.missing"}},
		{"CurrentKeyWhenOnlyDeposedRowExists", InstanceKey{Address: "aws_instance.web"}},
		{"DifferentDeposedKey", InstanceKey{Address: "aws_instance.web", Deposed: "ffffffff"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			row, ok := rows[tt.key]
			if ok || !reflect.DeepEqual(row, PolicyResourceAssessment{}) {
				t.Errorf("rows[%+v] = (%+v, %v), want zero row and false", tt.key, row, ok)
			}
		})
	}
}

func TestPolicyAssessment_Rows_WhenNoResources_ShouldReportNotFound(t *testing.T) {
	var a PolicyAssessment
	if _, ok := a.Rows()[InstanceKey{Address: "aws_instance.web"}]; ok {
		t.Error("Rows on an empty assessment found a row")
	}
}

func planChange(address, deposed string, risk RiskLevel) PlanChange {
	return PlanChange{
		Resource:       Resource{Address: address, Deposed: deposed},
		Action:         ActionDelete,
		AttributeDiffs: []AttributeDiff{{Key: "ami", OldValue: "a", NewValue: "b"}},
		Risk:           risk,
	}
}

func TestApplyAssessment_WhenRowsClassified_ShouldSetRiskByAddressAndDeposedKey(t *testing.T) {
	input := &PlanSummary{
		ToCreate: 1, ToDelete: 3,
		Changes: []PlanChange{
			planChange("aws_instance.web", "", RiskCritical),
			planChange("aws_instance.web", "0a1b2c3d", RiskCritical),
			planChange("aws_instance.db", "", RiskCritical),
			planChange("aws_instance.cache", "", RiskCritical),
		},
	}
	assessment := &PolicyAssessment{Status: AssessmentComplete, Resources: []PolicyResourceAssessment{
		{Address: "aws_instance.web", Status: ResourceClassified, Level: levelPtr(RiskLow)},
		{Address: "aws_instance.web", Deposed: "0a1b2c3d", Status: ResourceClassified, Level: levelPtr(RiskHigh)},
		{Address: "aws_instance.db", Status: ResourceNotAssessed, Exclusion: ExclusionNoOp, Level: levelPtr(RiskHigh)},
	}}

	got := ApplyAssessment(input, assessment.Rows())

	want := []RiskLevel{RiskLow, RiskHigh, RiskNone, RiskNone}
	if got == nil || len(got.Changes) != len(want) {
		t.Fatalf("ApplyAssessment returned %+v, want %d changes", got, len(want))
	}
	for i, risk := range want {
		if got.Changes[i].Risk != risk {
			t.Errorf("Changes[%d] (%s deposed=%q) Risk = %v, want %v",
				i, got.Changes[i].Resource.Address, got.Changes[i].Resource.Deposed, got.Changes[i].Risk, risk)
		}
	}
	if got.ToCreate != 1 || got.ToDelete != 3 {
		t.Errorf("counts = create %d delete %d, want 1 and 3", got.ToCreate, got.ToDelete)
	}
}

func TestApplyAssessment_WhenClassifiedRowHasNoLevel_ShouldSetRiskNone(t *testing.T) {
	input := &PlanSummary{Changes: []PlanChange{planChange("aws_instance.web", "", RiskCritical)}}
	assessment := &PolicyAssessment{Resources: []PolicyResourceAssessment{
		{Address: "aws_instance.web", Status: ResourceClassified},
	}}

	got := ApplyAssessment(input, assessment.Rows())

	if got.Changes[0].Risk != RiskNone {
		t.Errorf("Risk = %v, want %v", got.Changes[0].Risk, RiskNone)
	}
}

func TestApplyAssessment_WhenCalled_ShouldNotMutateInput(t *testing.T) {
	input := &PlanSummary{
		PolicyEvidence: &PolicyEvidence{},
		ToDelete:       2,
		Changes: []PlanChange{
			planChange("aws_instance.web", "", RiskCritical),
			planChange("aws_instance.web", "0a1b2c3d", RiskCritical),
		},
	}
	snapshot := &PlanSummary{
		PolicyEvidence: &PolicyEvidence{},
		ToDelete:       2,
		Changes: []PlanChange{
			planChange("aws_instance.web", "", RiskCritical),
			planChange("aws_instance.web", "0a1b2c3d", RiskCritical),
		},
	}
	originalChanges := input.Changes
	assessment := &PolicyAssessment{Resources: []PolicyResourceAssessment{
		{Address: "aws_instance.web", Status: ResourceClassified, Level: levelPtr(RiskLow)},
		{Address: "aws_instance.web", Deposed: "0a1b2c3d", Status: ResourceClassified, Level: levelPtr(RiskMedium)},
	}}

	got := ApplyAssessment(input, assessment.Rows())

	if got == input {
		t.Fatal("ApplyAssessment returned the input pointer, want a copy")
	}
	if !reflect.DeepEqual(input, snapshot) {
		t.Errorf("input mutated: got %+v, want %+v", input, snapshot)
	}
	for i := range originalChanges {
		if originalChanges[i].Risk != RiskCritical {
			t.Errorf("input Changes[%d].Risk = %v, want unchanged %v", i, originalChanges[i].Risk, RiskCritical)
		}
	}
	got.Changes[0].Risk = RiskHigh
	got.Changes[0].Resource.Address = "changed"
	if input.Changes[0].Risk != RiskCritical || input.Changes[0].Resource.Address != "aws_instance.web" {
		t.Errorf("writing to the copy changed the input: %+v", input.Changes[0])
	}
}

func TestApplyAssessment_WhenRowsNil_ShouldKeepExistingRisk(t *testing.T) {
	input := &PlanSummary{Changes: []PlanChange{
		planChange("aws_instance.web", "", RiskHigh),
		planChange("aws_instance.web", "0a1b2c3d", RiskLow),
	}}
	snapshot := &PlanSummary{Changes: []PlanChange{
		planChange("aws_instance.web", "", RiskHigh),
		planChange("aws_instance.web", "0a1b2c3d", RiskLow),
	}}

	got := ApplyAssessment(input, nil)

	if !reflect.DeepEqual(got, snapshot) {
		t.Errorf("ApplyAssessment(summary, nil) = %+v, want an unchanged equivalent %+v", got, snapshot)
	}
	if !reflect.DeepEqual(input, snapshot) {
		t.Errorf("input mutated: %+v", input)
	}
}

func TestApplyAssessment_WhenSummaryNil_ShouldReturnNil(t *testing.T) {
	if got := ApplyAssessment(nil, AssessmentRows{}); got != nil {
		t.Errorf("ApplyAssessment(nil, assessment) = %+v, want nil", got)
	}
	if got := ApplyAssessment(nil, nil); got != nil {
		t.Errorf("ApplyAssessment(nil, nil) = %+v, want nil", got)
	}
}

func TestResource_InstanceKey_WhenDeposed_ShouldCarryAddressAndDeposedKey(t *testing.T) {
	current := Resource{Address: "aws_instance.web", Type: "aws_instance"}
	deposed := Resource{Address: "aws_instance.web", Deposed: "0a1b2c3d"}

	if got := current.InstanceKey(); got != (InstanceKey{Address: "aws_instance.web"}) {
		t.Errorf("current InstanceKey = %+v, want address only", got)
	}
	if got := deposed.InstanceKey(); got != (InstanceKey{Address: "aws_instance.web", Deposed: "0a1b2c3d"}) {
		t.Errorf("deposed InstanceKey = %+v, want address and deposed key", got)
	}
}

func BenchmarkApplyAssessment(b *testing.B) {
	const n = 10000
	summary := &PlanSummary{Changes: make([]PlanChange, 0, n)}
	assessment := &PolicyAssessment{Status: AssessmentComplete, Resources: make([]PolicyResourceAssessment, 0, n)}
	for i := range n {
		address := fmt.Sprintf("aws_instance.web[%d]", i)
		summary.Changes = append(summary.Changes, planChange(address, "", RiskCritical))
		assessment.Resources = append(assessment.Resources, PolicyResourceAssessment{Address: address, Status: ResourceClassified, Level: levelPtr(RiskLow)})
	}
	rows := assessment.Rows()
	b.ResetTimer()
	for range b.N {
		ApplyAssessment(summary, rows)
	}
}
