package workspaces

import (
	"testing"

	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/sdktest"
)

func newGoldenPlugin() *Plugin {
	svc := &mockService{workspace: "default", workspaceList: []string{"default"}}
	p := New(svc).(*Plugin)
	return p
}

func TestView_Given_Loading_ShouldRender_LoadingMessage(t *testing.T) {
	p := newGoldenPlugin()
	p.status = sdk.StatusLoading

	sdktest.AssertGolden(t, p.View(80, 18))
}

func TestView_Given_Error_ShouldRender_ErrorMessage(t *testing.T) {
	p := newGoldenPlugin()
	p.status = sdk.StatusError
	p.errMsg = "Failed to list workspaces"

	sdktest.AssertGolden(t, p.View(80, 18))
}

func TestView_Given_WorkspaceList_ShouldRender_AllWorkspaces(t *testing.T) {
	p := newGoldenPlugin()
	p.status = sdk.StatusDone
	p.workspaces = []string{"default", "staging", "production"}
	p.current = "default"

	sdktest.AssertGolden(t, p.View(80, 18))
}

func TestView_Given_WorkspaceList_WithSelection_ShouldRender_HighlightedRow(t *testing.T) {
	p := newGoldenPlugin()
	p.status = sdk.StatusDone
	p.workspaces = []string{"default", "staging", "production"}
	p.current = "default"
	p.selected = 2

	sdktest.AssertGolden(t, p.View(80, 18))
}

func TestView_Given_CreatingWorkspace_ShouldRender_InputPrompt(t *testing.T) {
	p := newGoldenPlugin()
	p.status = sdk.StatusDone
	p.workspaces = []string{"default", "staging"}
	p.current = "default"
	p.creating = true
	p.newName = "feature-x"

	sdktest.AssertGolden(t, p.View(80, 18))
}
