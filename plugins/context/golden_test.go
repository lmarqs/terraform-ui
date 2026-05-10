package context

import (
	"testing"

	"github.com/lmarqs/terraform-ui/pkg/sdk/sdktest"
)

func newGoldenPlugin() *Plugin {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	return p
}

func TestView_Given_Loading_ShouldRender_LoadingMessage(t *testing.T) {
	p := newGoldenPlugin()
	p.status = StatusLoading

	sdktest.AssertGolden(t, p.View(80, 18))
}

func TestView_Given_Error_ShouldRender_ErrorMessage(t *testing.T) {
	p := newGoldenPlugin()
	p.status = StatusError
	p.errMsg = "failed to discover scopes: permission denied"

	sdktest.AssertGolden(t, p.View(80, 18))
}

func TestView_Given_NoScopes_ShouldRender_Placeholder(t *testing.T) {
	p := newGoldenPlugin()
	p.status = StatusDone
	p.scopes = []Scope{}

	sdktest.AssertGolden(t, p.View(80, 18))
}

func TestView_Given_ScopeList_ShouldRender_AllScopes(t *testing.T) {
	p := newGoldenPlugin()
	p.status = StatusDone
	p.scopes = []Scope{
		{Path: "modules/networking", Name: "networking", AbsPath: "/repo/modules/networking"},
		{Path: "modules/compute", Name: "compute", AbsPath: "/repo/modules/compute"},
		{Path: "envs/production", Name: "production", AbsPath: "/repo/envs/production"},
	}

	sdktest.AssertGolden(t, p.View(80, 18))
}

func TestView_Given_ScopeList_WithSelection_ShouldRender_HighlightedRow(t *testing.T) {
	p := newGoldenPlugin()
	p.status = StatusDone
	p.scopes = []Scope{
		{Path: "modules/networking", Name: "networking", AbsPath: "/repo/modules/networking"},
		{Path: "modules/compute", Name: "compute", AbsPath: "/repo/modules/compute"},
		{Path: "envs/production", Name: "production", AbsPath: "/repo/envs/production"},
	}
	p.selected = 1

	sdktest.AssertGolden(t, p.View(80, 18))
}

func TestView_Given_ScopeList_WithActiveScope_ShouldRender_ActiveIndicator(t *testing.T) {
	p := newGoldenPlugin()
	p.status = StatusDone
	p.scopes = []Scope{
		{Path: "modules/networking", Name: "networking", AbsPath: "/repo/modules/networking"},
		{Path: "modules/compute", Name: "compute", AbsPath: "/repo/modules/compute"},
		{Path: "envs/production", Name: "production", AbsPath: "/repo/envs/production"},
	}
	p.active = 2

	sdktest.AssertGolden(t, p.View(80, 18))
}
