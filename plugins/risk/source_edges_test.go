package risk

import (
	"testing"

	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/sdktest"
)

func TestActivate_GivenSourceRestored_ShouldClearFailureWithoutRevivingOldClassification(t *testing.T) {
	svc := &sdktest.MockService{}
	harness := sdktest.NewDeps(svc)
	diagnostic := "source_unreadable"
	harness.Deps.RiskPolicyDiagnostic = func() string { return diagnostic }
	p := New(svc).(*Plugin)
	runLifecycleCmd(p, p.Init(harness.Deps))
	runLifecycleCmd(p, p.Activate())
	if p.ExitCode() != 1 || string(p.Stderr()) != "risk unavailable: source_unreadable\n" {
		t.Fatalf("source failure: exit %d, stderr %q", p.ExitCode(), p.Stderr())
	}
	diagnostic = ""
	runLifecycleCmd(p, p.Activate())
	if p.Status() != sdk.StatusDone || p.ExitCode() != 0 || len(p.Stderr()) != 0 || len(p.groups) != 0 {
		t.Fatalf("recovery revived assessment: status %v, stderr %q", p.Status(), p.Stderr())
	}
	runLifecycleCmd(p, p.Activate())
	if p.Status() != sdk.StatusDone {
		t.Fatalf("unchanged activation: status %v", p.Status())
	}
}
