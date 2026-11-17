package sdk_test

import (
	"errors"
	"testing"

	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

func TestPolicyConstants_GivenEachTypedConstant_ShouldKeepTheWireString(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want string
	}{
		{"AssessmentComplete", string(sdk.AssessmentComplete), "complete"},
		{"AssessmentEmpty", string(sdk.AssessmentEmpty), "empty"},
		{"AssessmentPartial", string(sdk.AssessmentPartial), "partial"},
		{"AssessmentUnavailable", string(sdk.AssessmentUnavailable), "unavailable"},
		{"AssessmentCancelled", string(sdk.AssessmentCancelled), "cancelled"},

		{"ResourceClassified", string(sdk.ResourceClassified), "classified"},
		{"ResourceNotAssessed", string(sdk.ResourceNotAssessed), "not_assessed"},
		{"ResourceUnavailable", string(sdk.ResourceUnavailable), "unavailable"},

		{"ExclusionRead", string(sdk.ExclusionRead), "read"},
		{"ExclusionNoOp", string(sdk.ExclusionNoOp), "no_op"},
		{"ExclusionForget", string(sdk.ExclusionForget), "forget"},
		{"ExclusionPureImport", string(sdk.ExclusionPureImport), "pure_import"},
		{"ExclusionPureMove", string(sdk.ExclusionPureMove), "pure_move"},
		{"ExclusionOutputOnly", string(sdk.ExclusionOutputOnly), "output_only"},

		{"ErrorConfigurationInvalid", string(sdk.ErrorConfigurationInvalid), "configuration_invalid"},
		{"ErrorSourceUnreadable", string(sdk.ErrorSourceUnreadable), "source_unreadable"},
		{"ErrorSourceInvalid", string(sdk.ErrorSourceInvalid), "source_invalid"},
		{"ErrorSourceChanged", string(sdk.ErrorSourceChanged), "source_changed"},
		{"ErrorInputIncomplete", string(sdk.ErrorInputIncomplete), "input_incomplete"},
		{"ErrorInputInvalid", string(sdk.ErrorInputInvalid), "input_invalid"},
		{"ErrorCompilationFailed", string(sdk.ErrorCompilationFailed), "compilation_failed"},
		{"ErrorResultUndefined", string(sdk.ErrorResultUndefined), "result_undefined"},
		{"ErrorResultInvalid", string(sdk.ErrorResultInvalid), "result_invalid"},
		{"ErrorResultCardinality", string(sdk.ErrorResultCardinality), "result_cardinality"},
		{"ErrorEvaluationConflict", string(sdk.ErrorEvaluationConflict), "evaluation_conflict"},
		{"ErrorEvaluationFailed", string(sdk.ErrorEvaluationFailed), "evaluation_failed"},
		{"ErrorCancelled", string(sdk.ErrorCancelled), "cancelled"},

		{"KindAbsent", string(sdk.KindAbsent), "absent"},
		{"KindNull", string(sdk.KindNull), "null"},
		{"KindString", string(sdk.KindString), "string"},
		{"KindBoolean", string(sdk.KindBoolean), "boolean"},
		{"KindNumber", string(sdk.KindNumber), "number"},
		{"KindArray", string(sdk.KindArray), "array"},
		{"KindObject", string(sdk.KindObject), "object"},
		{"KindUnknown", string(sdk.KindUnknown), "unknown"},
		{"KindSensitive", string(sdk.KindSensitive), "sensitive"},
		{"KindSensitiveUnknown", string(sdk.KindSensitiveUnknown), "sensitive_unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Fatalf("%s = %q, want %q", tt.name, tt.got, tt.want)
			}
		})
	}
}

func TestPolicyAssessment_GivenTypedFields_ShouldCarryThemOnEveryRecord(t *testing.T) {
	assessment := sdk.PolicyAssessment{
		Status:    sdk.AssessmentUnavailable,
		ErrorCode: sdk.ErrorInputIncomplete,
		Resources: []sdk.PolicyResourceAssessment{{
			Status:    sdk.ResourceNotAssessed,
			Exclusion: sdk.ExclusionForget,
			ErrorCode: sdk.ErrorCode(""),
		}},
	}
	evidence := sdk.PolicyResourceEvidence{Exclusion: sdk.ExclusionPureMove}
	value := sdk.PolicyValue{Kind: sdk.KindSensitiveUnknown}
	if assessment.Status != sdk.AssessmentUnavailable || assessment.Resources[0].Exclusion != sdk.ExclusionForget ||
		evidence.Exclusion != sdk.ExclusionPureMove || value.Kind != sdk.KindSensitiveUnknown {
		t.Fatalf("typed fields lost: %+v %+v %+v", assessment, evidence, value)
	}
}

func TestErrorCode_GivenWrappedCode_ShouldBeRecoverableWithoutParsingText(t *testing.T) {
	err := &wrapped{err: sdk.ErrorSourceInvalid}
	var code sdk.ErrorCode
	if !errors.As(err, &code) || code != sdk.ErrorSourceInvalid {
		t.Fatalf("errors.As code=%q, want %q", code, sdk.ErrorSourceInvalid)
	}
	if !errors.Is(err, sdk.ErrorSourceInvalid) || errors.Is(err, sdk.ErrorSourceChanged) {
		t.Fatalf("errors.Is must match only the carried code: %v", err)
	}
	if got := sdk.ErrorSourceInvalid.Error(); got != "source_invalid" {
		t.Fatalf("ErrorCode.Error() = %q, want the code text unchanged", got)
	}
}

func TestRiskSettingsError_GivenInvalidSetting_ShouldCarryConfigurationInvalidCode(t *testing.T) {
	tests := []struct {
		name     string
		settings map[string]interface{}
		text     string
	}{
		{"threshold", map[string]interface{}{"threshold": "bogus"}, "configuration_invalid"},
		{"level", map[string]interface{}{"level": "high"}, "configuration_invalid\n\nhint: risk setting level is unsupported; use threshold"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := sdk.RiskSettingsError(tt.settings)
			if !errors.Is(err, sdk.ErrorConfigurationInvalid) {
				t.Fatalf("error %v does not carry %q", err, sdk.ErrorConfigurationInvalid)
			}
			if err.Error() != tt.text {
				t.Fatalf("text = %q, want %q", err.Error(), tt.text)
			}
		})
	}
}

type wrapped struct{ err error }

func (w *wrapped) Error() string { return "wrapped: " + w.err.Error() }
func (w *wrapped) Unwrap() error { return w.err }
