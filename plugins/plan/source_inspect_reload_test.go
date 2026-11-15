package plan

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/sdktest"
)

func TestActivate_GivenInspectedPlanAndDeletedPolicySource_ShouldReplaceCachedRiskBadge(t *testing.T) {
	svc := &sdktest.MockService{}
	harness := sdktest.NewDeps(svc)
	diagnostic := ""
	harness.Deps.RiskPolicyDiagnostic = func() string { return diagnostic }
	p := New(svc).(*Plugin)
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
	execute(p.Init(harness.Deps))
	_, cmd := p.Update(PlanResultMsg{Summary: &sdk.PlanSummary{ToDelete: 1, Changes: []sdk.PlanChange{{Resource: sdk.Resource{Address: "aws_db_instance.database"}, Action: sdk.ActionDelete, Risk: sdk.RiskCritical}}}})
	execute(cmd)
	cmd = p.stack.Update(tea.KeyMsg{Type: tea.KeyEnter})
	execute(cmd)
	if p.stack.Peek().ID() != "inspect" {
		t.Fatalf("top frame = %q, want inspect", p.stack.Peek().ID())
	}
	if view := p.View(100, 30); !strings.Contains(view, "CRITICAL") {
		t.Fatalf("initial inspection = %q, want existing risk badge", view)
	}

	diagnostic = "source_unreadable"
	execute(p.Activate(Input{}))

	view := p.View(100, 30)
	if !strings.Contains(view, "risk unavailable: source_unreadable") || strings.Contains(view, "CRITICAL") {
		t.Fatalf("reactivated inspection = %q, want unavailable diagnostic without previous risk badge", view)
	}
	diagnostic = ""
	execute(p.Activate(Input{}))
	recovered := p.View(100, 30)
	if !strings.Contains(recovered, "CRITICAL") || strings.Contains(recovered, "risk unavailable:") {
		t.Fatalf("inspection after source recovery = %q, want current risk badge", recovered)
	}
	if len(svc.PlanCalls) != 0 {
		t.Fatalf("cached Plan reactivation acquired %d plans", len(svc.PlanCalls))
	}
}
