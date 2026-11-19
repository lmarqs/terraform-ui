package ui

import (
	"reflect"
	"testing"

	"github.com/lmarqs/terraform-ui/internal/config"
	"github.com/lmarqs/terraform-ui/internal/plugin"
	"github.com/lmarqs/terraform-ui/internal/terraform"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/sdktest"
)

func newStartupPinsApp(t *testing.T, cfg config.Config) (App, *sdktest.MockService) {
	t.Helper()
	cfg.Dir = t.TempDir()
	cfg.Terraform = config.TerraformConfig{Bin: "terraform"}
	svc := newMockService("default", nil)
	registry := plugin.NewRegistry()
	registry.RegisterFactory("chdir", func(_ terraform.Service) plugin.Plugin {
		return &mockPlugin{id: "chdir", name: "Chdir", viewOutput: "chdir view"}
	}, plugin.PluginMeta{MenuVisible: false})
	registry.RegisterFactory("plan", func(_ terraform.Service) plugin.Plugin {
		return &mockPlugin{id: "plan", name: "Plan", viewOutput: "plan view"}
	}, plugin.PluginMeta{Keybinding: "p", MenuVisible: true})
	registry.Build(nil, nil)
	app := NewApp(cfg, svc, registry, nil)
	app.Init()
	return app, svc
}

func update(t *testing.T, app App, msg any) App {
	t.Helper()
	model, _ := app.Update(msg)
	return model.(App)
}

func TestStartupContext_GivenTargets_ShouldCarryThemAsPins(t *testing.T) {
	tests := []struct {
		name    string
		targets []string
		want    sdk.Pins
	}{
		{name: "no targets", targets: nil, want: nil},
		{name: "one target", targets: []string{"aws_instance.web"}, want: sdk.Pins{"aws_instance.web"}},
		{
			name:    "repeated targets keep their order once each",
			targets: []string{"aws_instance.web", `module.a["x y"].aws_s3_bucket.old`, "aws_instance.web"},
			want:    sdk.Pins{"aws_instance.web", `module.a["x y"].aws_s3_bucket.old`},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, _ := newStartupPinsApp(t, config.Config{Pins: tt.targets})
			if got := app.holder.current.Pins; !reflect.DeepEqual(got, tt.want) {
				t.Errorf("startup Context Pins = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestStartupPins_GivenTheStartupContextSwitch_ShouldCarryIntoTheFirstContextOnly(t *testing.T) {
	app, _ := newStartupPinsApp(t, config.Config{Pins: []string{"aws_instance.web"}})

	app = update(t, app, sdk.ContextSwitchRequestMsg{Chdir: "modules/vpc", Workspace: sdk.WorkspaceDefault})
	if got, want := app.holder.current.Pins, (sdk.Pins{"aws_instance.web"}); !reflect.DeepEqual(got, want) {
		t.Fatalf("Pins after the startup switch = %#v, want %#v", got, want)
	}

	app = update(t, app, sdk.ContextSwitchRequestMsg{Chdir: "modules/ecs", Workspace: sdk.WorkspaceDefault})
	if got := app.holder.current.Pins; got != nil {
		t.Errorf("Pins after a later switch = %#v, want none: pins die on a context switch", got)
	}
}

func TestStartupPins_GivenPreloadedData_ShouldStayOnTheStartupContextAndNotFollowALaterSwitch(t *testing.T) {
	app, _ := newStartupPinsApp(t, config.Config{Pins: []string{"aws_instance.web"}, PreloadedData: true})

	app = update(t, app, openContextOnStartupMsg{})
	if got, want := app.holder.current.Pins, (sdk.Pins{"aws_instance.web"}); !reflect.DeepEqual(got, want) {
		t.Fatalf("Pins after startup = %#v, want %#v", got, want)
	}

	app = update(t, app, sdk.ContextSwitchRequestMsg{Chdir: "modules/vpc", Workspace: sdk.WorkspaceDefault})
	if got := app.holder.current.Pins; got != nil {
		t.Errorf("Pins after a later switch = %#v, want none", got)
	}
}

func TestStartupPins_GivenAPinChangeBeforeTheSwitch_ShouldNotReturnOnTheSwitch(t *testing.T) {
	app, _ := newStartupPinsApp(t, config.Config{Pins: []string{"aws_instance.web", "aws_s3_bucket.old"}})

	app = update(t, app, sdk.PinClearRequestMsg{})
	app = update(t, app, sdk.ContextSwitchRequestMsg{Chdir: "modules/vpc", Workspace: sdk.WorkspaceDefault})
	if got := app.holder.current.Pins; got != nil {
		t.Errorf("Pins after clearing then switching = %#v, want none", got)
	}
}

func TestStartupPins_GivenStartup_ShouldRunNoTerraformPlan(t *testing.T) {
	app, svc := newStartupPinsApp(t, config.Config{Pins: []string{"aws_instance.web"}})

	app = update(t, app, openContextOnStartupMsg{})
	update(t, app, sdk.ContextSwitchRequestMsg{Chdir: "modules/vpc", Workspace: sdk.WorkspaceDefault})
	if len(svc.PlanCalls) != 0 || len(svc.ApplyCalls) != 0 {
		t.Errorf("startup ran plan %d and apply %d times, want none: pins reach terraform only when the user plans",
			len(svc.PlanCalls), len(svc.ApplyCalls))
	}
}

func TestStartupPins_GivenPinToggles_ShouldAddAndRemoveLikeHandSetPins(t *testing.T) {
	app, _ := newStartupPinsApp(t, config.Config{Pins: []string{"aws_instance.web"}})
	app = update(t, app, sdk.ContextSwitchRequestMsg{Chdir: "modules/vpc", Workspace: sdk.WorkspaceDefault})

	app = update(t, app, sdk.PinToggleRequestMsg{Addresses: []string{"aws_s3_bucket.data"}})
	if got, want := app.holder.current.Pins, (sdk.Pins{"aws_instance.web", "aws_s3_bucket.data"}); !reflect.DeepEqual(got, want) {
		t.Fatalf("Pins after pinning another address = %#v, want %#v", got, want)
	}
	app = update(t, app, sdk.PinToggleRequestMsg{Addresses: []string{"aws_instance.web"}})
	if got, want := app.holder.current.Pins, (sdk.Pins{"aws_s3_bucket.data"}); !reflect.DeepEqual(got, want) {
		t.Errorf("Pins after unpinning the -target address = %#v, want %#v", got, want)
	}
}

func newStartupPinsAppWithout(t *testing.T, chdirPlugin bool) App {
	t.Helper()
	cfg := config.Config{Dir: t.TempDir(), Terraform: config.TerraformConfig{Bin: "terraform"}, Pins: []string{"aws_instance.web"}}
	registry := plugin.NewRegistry()
	if chdirPlugin {
		registry.RegisterFactory("chdir", func(_ terraform.Service) plugin.Plugin {
			return &mockPlugin{id: "chdir", name: "Chdir", viewOutput: "chdir view"}
		}, plugin.PluginMeta{MenuVisible: false})
	}
	registry.RegisterFactory("workspace", func(_ terraform.Service) plugin.Plugin {
		return &mockPlugin{id: "workspace", name: "Workspace", viewOutput: "workspace view"}
	}, plugin.PluginMeta{MenuVisible: false})
	registry.Build(nil, nil)
	app := NewApp(cfg, newMockService("default", nil), registry, nil)
	app.Init()
	return update(t, app, openContextOnStartupMsg{})
}

func TestStartupPins_GivenChdirDisabled_ShouldStayOnTheStartupContextAndNotFollowAWorkspaceSwitch(t *testing.T) {
	for _, workspace := range []sdk.Workspace{sdk.NewWorkspace("staging"), sdk.WorkspaceDefault} {
		t.Run(workspace.String(), func(t *testing.T) {
			app := newStartupPinsAppWithout(t, false)
			if got, want := app.holder.current.Pins, (sdk.Pins{"aws_instance.web"}); !reflect.DeepEqual(got, want) {
				t.Fatalf("Pins after startup = %#v, want %#v", got, want)
			}
			app = update(t, app, sdk.ContextSwitchRequestMsg{Chdir: app.holder.current.Chdir, Workspace: workspace})
			if got := app.holder.current.Pins; got != nil {
				t.Errorf("Pins after a workspace switch = %#v, want none: pins die on a context switch", got)
			}
		})
	}
}

func TestStartupPins_GivenThePickerAbandoned_ShouldNotFollowALaterMemberPick(t *testing.T) {
	app := newStartupPinsAppWithout(t, true)
	if app.activePlugin == nil || app.activePlugin.ID() != "chdir" {
		t.Fatalf("startup did not open the chdir picker")
	}

	app = update(t, app, sdk.NavigateMsg{PluginID: "workspace"})
	app = update(t, app, sdk.ContextSwitchRequestMsg{Chdir: "modules/vpc", Workspace: sdk.WorkspaceDefault})
	if got := app.holder.current.Pins; got != nil {
		t.Errorf("Pins after leaving the picker and picking a member = %#v, want none", got)
	}
}

func TestStartupPins_GivenThePickerPicksAMember_ShouldCarryThemIntoThatMember(t *testing.T) {
	app := newStartupPinsAppWithout(t, true)

	app = update(t, app, sdk.ContextSwitchRequestMsg{Chdir: "modules/vpc", Workspace: sdk.WorkspaceDefault})
	if got, want := app.holder.current.Pins, (sdk.Pins{"aws_instance.web"}); !reflect.DeepEqual(got, want) {
		t.Errorf("Pins after the startup pick = %#v, want %#v", got, want)
	}
}
