package ui

import (
	"context"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/lmarqs/terraform-ui/internal/config"
	"github.com/lmarqs/terraform-ui/internal/plugin"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	tfuiforceunlock "github.com/lmarqs/terraform-ui/plugins/forceunlock"
	tfuiinit "github.com/lmarqs/terraform-ui/plugins/init"
	tfuirisk "github.com/lmarqs/terraform-ui/plugins/risk"
	tfuistate "github.com/lmarqs/terraform-ui/plugins/state"
)

func TestPolicySourceFailure_ShouldAllowNavigationToOtherOperations(t *testing.T) {
	root := t.TempDir()
	writePolicyFile(t, root, "tfui.hcl", `defaults { risk_policy = "missing.rego" }`)
	rootCfg, err := config.LoadRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	svc := newMockService("default", nil)
	registry := plugin.NewRegistry()
	registry.RegisterFactory("risk", tfuirisk.New, plugin.PluginMeta{})
	registry.RegisterFactory("state", tfuistate.New, plugin.PluginMeta{})
	registry.RegisterFactory("init", tfuiinit.New, plugin.PluginMeta{})
	registry.RegisterFactory("forceunlock", tfuiforceunlock.New, plugin.PluginMeta{})
	registry.Build(svc, nil)
	app := NewApp(config.Config{Dir: root}, svc, registry, rootCfg)
	app.Init()
	for _, id := range []string{"state", "init", "forceunlock"} {
		model, activate := app.Update(sdk.NavigateMsg{PluginID: "risk"})
		app = model.(App)
		drainPolicyAppCommand(&app, activate)
		if !strings.Contains(app.ActivePlugin().View(100, 30), "risk unavailable: source_unreadable") {
			t.Fatal("Risk source error missing")
		}
		model, cmd := app.Update(sdk.NavigateMsg{PluginID: id})
		app = model.(App)
		if app.ActivePlugin().ID() != id {
			t.Fatalf("source error blocked %s navigation", id)
		}
		switch id {
		case "state":
			if cmd == nil || app.ActivePlugin().(sdk.Statusable).Status() != sdk.StatusLoading {
				t.Fatal("state read did not start")
			}
			drainPolicyAppCommand(&app, cmd)
			if app.ActivePlugin().(sdk.Statusable).Status() != sdk.StatusDone {
				t.Fatal("state read did not complete")
			}
		case "init":
			if !strings.Contains(app.ActivePlugin().View(100, 30), "backend") {
				t.Fatal("init form unavailable after source failure")
			}
		case "forceunlock":
			if cmd == nil {
				t.Fatal("manual unlock prompt unavailable")
			}
			request, ok := cmd().(sdk.RequestInputMsg)
			if !ok || request.Request.Mode != sdk.InputRequestBool {
				t.Fatal("manual unlock must keep confirmation")
			}
		}
	}
}

func TestRiskReopen_GivenHostPlanCancelledByEsc_ShouldShowTheNewAssessmentNotTheCancellation(t *testing.T) {
	svc := newMockService("default", nil)
	svc.PlanFn = func(ctx context.Context, _ sdk.PlanOptions) (*sdk.PlanSummary, error) {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		planned := reviewedRiskSummary()
		planned.Changes = []sdk.PlanChange{{Resource: sdk.Resource{Address: "aws_db_instance.database"}, Action: sdk.ActionDelete}}
		return planned, nil
	}
	registry := plugin.NewRegistry()
	registry.RegisterFactory("risk", tfuirisk.New, plugin.PluginMeta{})
	registry.Build(svc, nil)
	app := NewApp(config.Config{Dir: t.TempDir()}, svc, registry, nil)
	app.Init()

	model, cancelledOpen := app.Update(sdk.NavigateMsg{PluginID: "risk"})
	app = model.(App)
	model, home := app.Update(sdk.DeactivateMsg{})
	app = model.(App)
	drainPolicyAppCommand(&app, home)
	model, reopen := app.Update(sdk.NavigateMsg{PluginID: "risk"})
	app = model.(App)

	drainPolicyAppCommand(&app, cancelledOpen)
	drainPolicyAppCommand(&app, reopen)

	view := ansi.Strip(app.ActivePlugin().View(100, 30))
	if strings.Contains(view, "context canceled") || !strings.Contains(view, "aws_db_instance.database") {
		t.Fatalf("reopened Risk view = %q, want the reopened request's assessment, not the cancelled plan's error", view)
	}
}
