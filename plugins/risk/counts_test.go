package risk

import (
	"testing"

	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

func TestCountRows_GivenAssessment_ShouldCountRowsByStatus(t *testing.T) {
	tests := []struct {
		name       string
		assessment *sdk.PolicyAssessment
		want       rowCounts
	}{
		{"NoAssessment", nil, rowCounts{}},
		{"NoRows", &sdk.PolicyAssessment{Status: sdk.AssessmentEmpty}, rowCounts{}},
		{"EveryStatus", &sdk.PolicyAssessment{Resources: []sdk.PolicyResourceAssessment{
			{Address: "a.one", Status: sdk.ResourceClassified},
			{Address: "a.two", Status: sdk.ResourceClassified},
			{Address: "a.three", Status: sdk.ResourceUnavailable},
			{Address: "a.four", Status: sdk.ResourceNotAssessed},
			{Address: "a.five", Status: sdk.ResourceNotAssessed},
			{Address: "a.six", Status: sdk.ResourceNotAssessed},
		}}, rowCounts{classified: 2, unavailable: 1, notAssessed: 3}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := countRows(tt.assessment); got != tt.want {
				t.Errorf("countRows() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
