package plan

import (
	"strings"
	"testing"

	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/sdktest"
)

func TestPolicyFailure_GivenReviewNavigation_ShouldRemainVisibleAndSuppressClassifications(t *testing.T) {
	for _, mode := range []string{"empty", "filtered", "tree", "flat"} {
		t.Run(mode, func(t *testing.T) {
			svc := &sdktest.MockService{}
			harness := sdktest.NewDeps(svc)
			harness.Deps.RiskPolicyDiagnostic = func() string { return "source_invalid" }
			p := New(svc).(*Plugin)
			p.Init(harness.Deps)
			defer p.Shutdown()
			summary := &sdk.PlanSummary{Changes: []sdk.PlanChange{{Resource: sdk.Resource{Address: "aws_db_instance.database"}, Action: sdk.ActionDelete, Risk: sdk.RiskCritical}}}
			if mode == "empty" {
				summary.Changes = nil
			}
			p.Update(PlanResultMsg{Summary: summary})
			switch mode {
			case "filtered":
				p.SetFilter("no-match")
			case "tree":
				p.treeMode = true
				p.SetFilter("")
			case "flat":
				p.treeMode = false
				p.SetFilter("")
			}
			view := sdktest.StripANSI(p.View(100, 30))
			if !strings.Contains(view, "risk unavailable: source_invalid") || strings.Contains(view, "CRITICAL") {
				t.Fatalf("review = %q", view)
			}
		})
	}
}

func TestActivate_GivenSourceChangedAfterPlan_ShouldReloadWithoutAcquiringAnotherPlan(t *testing.T) {
	svc := &sdktest.MockService{}
	harness := sdktest.NewDeps(svc)
	diagnostic := ""
	harness.Deps.RiskPolicyDiagnostic = func() string { return diagnostic }
	p := New(svc).(*Plugin)
	p.Init(harness.Deps)
	defer p.Shutdown()
	p.Update(PlanResultMsg{Summary: &sdk.PlanSummary{Changes: []sdk.PlanChange{{Resource: sdk.Resource{Address: "aws_instance.web"}, Action: sdk.ActionCreate, Risk: sdk.RiskLow}}}})
	diagnostic = "source_unreadable"
	if cmd := p.Activate(Input{}); cmd != nil {
		t.Fatal("cached review started another plan")
	}
	if !strings.Contains(p.View(100, 30), "risk unavailable: source_unreadable") {
		t.Fatal("activation retained risk from deleted source")
	}
	diagnostic = ""
	p.Activate(Input{})
	if len(p.Stderr()) != 0 || strings.Contains(p.View(100, 30), "risk unavailable") {
		t.Fatal("next boundary retained resolved source failure")
	}
}
