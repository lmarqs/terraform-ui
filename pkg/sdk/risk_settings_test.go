package sdk

import "testing"

func TestRiskSettingsError_GivenSupportedSettings_ShouldAcceptConfiguration(t *testing.T) {
	for _, tc := range []struct {
		name     string
		settings map[string]interface{}
	}{
		{"absent", nil},
		{"empty", map[string]interface{}{}},
		{"unrelated", map[string]interface{}{"enabled": false}},
		{"off", map[string]interface{}{"threshold": "off"}},
		{"none", map[string]interface{}{"threshold": "none"}},
		{"low", map[string]interface{}{"threshold": "low"}},
		{"medium", map[string]interface{}{"threshold": "medium"}},
		{"high", map[string]interface{}{"threshold": "high"}},
		{"critical", map[string]interface{}{"threshold": "critical"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := RiskSettingsError(tc.settings); err != nil {
				t.Fatalf("supported Risk settings %v rejected: %v", tc.settings, err)
			}
		})
	}
}

func TestRiskSettingsError_GivenInvalidThreshold_ShouldReturnFixedConfigurationDiagnostic(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value interface{}
	}{
		{"unknown", "urgent"}, {"empty", ""}, {"uppercase", "HIGH"}, {"whitespace", " high "},
		{"boolean", true}, {"number", 3}, {"null", nil}, {"list", []string{"high"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := RiskSettingsError(map[string]interface{}{"threshold": tc.value})
			if err == nil || err.Error() != "configuration_invalid" {
				t.Fatalf("invalid Risk threshold %v error = %v, want configuration_invalid", tc.value, err)
			}
		})
	}
}

func TestRiskSettingsError_GivenLegacyLevel_ShouldReturnMigrationHintRegardlessOfValue(t *testing.T) {
	for _, tc := range []struct {
		name     string
		settings map[string]interface{}
	}{
		{"legacy", map[string]interface{}{"level": "high"}},
		{"null", map[string]interface{}{"level": nil}},
		{"valid threshold does not rescue legacy", map[string]interface{}{"level": "high", "threshold": "off"}},
		{"invalid threshold retains hint", map[string]interface{}{"level": "high", "threshold": "urgent"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := RiskSettingsError(tc.settings)
			want := "configuration_invalid\n\nhint: risk setting level is unsupported; use threshold"
			if err == nil || err.Error() != want {
				t.Fatalf("legacy Risk setting error = %v, want %q", err, want)
			}
		})
	}
}

func TestDiagnosticError_GivenCodeAndOptionalHint_ShouldRenderAndExposeBoth(t *testing.T) {
	plain := DiagnosticError{Code: ErrorSourceInvalid}
	if plain.Error() != "source_invalid" || DiagnosticHint(plain) != "" {
		t.Fatalf("hintless diagnostic = %q hint %q, want the code alone", plain.Error(), DiagnosticHint(plain))
	}
	hinted := RiskSettingsError(map[string]interface{}{"level": "high"})
	if got := DiagnosticHint(hinted); got == "" || (DiagnosticError{Code: ErrorConfigurationInvalid, Hint: got}).Error() != hinted.Error() {
		t.Fatalf("hint %q does not rebuild %q", got, hinted.Error())
	}
	if got := DiagnosticHint(ErrorConfigurationInvalid); got != "" {
		t.Fatalf("bare code hint = %q, want none", got)
	}
}
