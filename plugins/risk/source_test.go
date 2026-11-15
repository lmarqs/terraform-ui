package risk

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/sdktest"
)

func TestAnalyze_GivenSelectedSourceFailure_ShouldTerminateWithoutFallbackClassification(t *testing.T) {
	svc := &sdktest.MockService{}
	harness := sdktest.NewDeps(svc)
	diagnostic := ""
	harness.Deps.RiskPolicyDiagnostic = func() string { return diagnostic }
	p := New(svc).(*Plugin)
	if cmd := p.Init(harness.Deps); cmd != nil {
		p.Update(cmd())
	}
	summary := &sdk.PlanSummary{Changes: []sdk.PlanChange{{Resource: sdk.Resource{Address: "aws_db_instance.database"}, Action: sdk.ActionDelete, Risk: sdk.RiskCritical}}}
	p.Analyze(summary)
	if !strings.Contains(p.View(100, 30), "aws_db_instance.database") {
		t.Fatal("initial classification missing")
	}
	diagnostic = "source_unreadable"
	p.Analyze(summary)
	if p.Status() != sdk.StatusError {
		t.Fatalf("source failure status = %v; want terminal error", p.Status())
	}
	view := p.View(100, 30)
	if !strings.Contains(view, "risk unavailable: source_unreadable") {
		t.Fatalf("view = %q, want fixed source failure", view)
	}
	if strings.Contains(view, "CRITICAL") || len(p.groups) != 0 {
		t.Fatalf("source failure retained prior successful classification: %q", view)
	}
	diagnostic = ""
	p.Analyze(summary)
	if p.Status() != sdk.StatusDone || !strings.Contains(p.View(100, 30), "aws_db_instance.database") {
		t.Fatal("restored source did not recover on next assessment")
	}
}

func TestActivate_GivenSourceDeletedAfterAnalysis_ShouldReplacePreviousClassificationWithUnavailable(t *testing.T) {
	svc := &sdktest.MockService{}
	harness := sdktest.NewDeps(svc)
	diagnostic := ""
	harness.Deps.RiskPolicyDiagnostic = func() string { return diagnostic }
	p := New(svc).(*Plugin)
	if cmd := p.Init(harness.Deps); cmd != nil {
		p.Update(cmd())
	}
	p.Analyze(&sdk.PlanSummary{Changes: []sdk.PlanChange{{Resource: sdk.Resource{Address: "aws_instance.web"}, Action: sdk.ActionCreate, Risk: sdk.RiskLow}}})
	diagnostic = "source_unreadable"
	var execute func(tea.Cmd)
	execute = func(cmd tea.Cmd) {
		if cmd == nil {
			return
		}
		msg := cmd()
		if batch, ok := msg.(tea.BatchMsg); ok {
			for _, sub := range batch {
				execute(sub)
			}
			return
		}
		_, next := p.Update(msg)
		execute(next)
	}
	execute(p.Activate())
	if p.Status() != sdk.StatusError {
		t.Fatalf("activation status = %v; want terminal error", p.Status())
	}
	if got := p.View(100, 30); !strings.Contains(got, "risk unavailable: source_unreadable") || strings.Contains(got, "Low risk") {
		t.Fatalf("reactivated Risk view = %q", got)
	}
}
