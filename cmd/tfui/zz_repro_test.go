package main

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/internal/config"
	"github.com/lmarqs/terraform-ui/internal/macro"
	"github.com/lmarqs/terraform-ui/internal/ui"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/sdktest"
	"github.com/lmarqs/terraform-ui/plugins/apply"
)

func TestRepro_StandaloneApplyTTY(t *testing.T) {
	svc := &sdktest.MockService{}
	cfg := config.Config{}
	registry := buildRegistry(svc, cfg, nil)
	standalone := &ui.StandaloneConfig{
		PluginID: "apply",
		Activate: func(p sdk.Plugin) tea.Cmd {
			return p.(*apply.Plugin).Activate(apply.Input{AutoApprove: false})
		},
	}
	app := ui.NewApp(cfg, svc, registry, nil, standalone)

	// Drive like the TTY path: run the model loop via the macro driver
	// (the driver processes Init + all queued cmds synchronously).
	d := macro.NewDriver(app, 80, 24)
	d.Init()

	// give async cmds a beat to settle
	deadline := time.Now().Add(2 * time.Second)
	var view string
	for time.Now().Before(deadline) {
		view = d.View()
		if strings.Contains(view, "Are you sure") || len(svc.ApplyCalls) > 0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	p, _ := registry.ByID("apply")
	ap := p.(*apply.Plugin)
	t.Logf("apply status = %v (Confirming=%v Loading=%v Done=%v)", ap.Status(), apply.StatusConfirming, sdk.StatusLoading, sdk.StatusDone)
	t.Logf("ApplyCalls = %d", len(svc.ApplyCalls))
	t.Logf("view contains 'Are you sure' = %v", strings.Contains(view, "Are you sure"))
	t.Logf("---- VIEW ----\n%s\n----", view)

	if len(svc.ApplyCalls) > 0 && !strings.Contains(view, "Are you sure") {
		t.Errorf("BUG REPRODUCED: apply executed without confirmation prompt")
	}
}
