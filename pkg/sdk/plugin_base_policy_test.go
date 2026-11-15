package sdk_test

import (
	"testing"

	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/sdktest"
)

func TestPluginBasePolicyDiagnostic(t *testing.T) {
	harness := sdktest.NewDeps(&sdktest.MockService{})
	base := sdk.NewPluginBase("review", "Review", "")
	base.InitBase(harness.Deps)
	if got := base.PolicyDiagnostic(); got != "" {
		t.Fatalf("host without policy loader: %q", got)
	}
	diagnostic := "source_unreadable"
	harness.Deps.RiskPolicyDiagnostic = func() string { return diagnostic }
	base.InitBase(harness.Deps)
	if got := base.PolicyDiagnostic(); got != diagnostic {
		t.Fatalf("loader failure: %q", got)
	}
	diagnostic = ""
	if got := base.PolicyDiagnostic(); got != "" {
		t.Fatalf("loader recovery: %q", got)
	}
}
