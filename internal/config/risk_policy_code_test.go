package config

import (
	"errors"
	"testing"

	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

func TestResolveRiskPolicy_GivenInvalidConfiguration_ShouldCarryErrorCodeWithoutParsingText(t *testing.T) {
	project := t.TempDir()
	writeHCL(t, project, "defaults {}\ndefaults {}")
	_, err := ResolveRiskPolicy(project, project, "prod")
	var code sdk.ErrorCode
	if !errors.As(err, &code) || code != sdk.ErrorConfigurationInvalid {
		t.Fatalf("code = %q (error %v), want %q", code, err, sdk.ErrorConfigurationInvalid)
	}
	if err.Error() != "configuration_invalid" {
		t.Fatalf("text = %q, want configuration_invalid unchanged", err.Error())
	}
}
