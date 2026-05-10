package state

import (
	"fmt"
	"io"
	"log/slog"
	"testing"

	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/sdktest"
)

func newGoldenPlugin() *Plugin {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.log = slog.New(slog.NewTextHandler(io.Discard, nil))
	return p
}

func testResources() []sdk.Resource {
	return []sdk.Resource{
		{Address: "aws_instance.web", Type: "aws_instance", Name: "web"},
		{Address: "aws_s3_bucket.data", Type: "aws_s3_bucket", Name: "data"},
		{Address: "module.vpc.aws_subnet.public", Type: "aws_subnet", Name: "public", Module: "module.vpc"},
		{Address: "aws_security_group.allow_ssh", Type: "aws_security_group", Name: "allow_ssh"},
		{Address: "aws_iam_role.lambda_exec", Type: "aws_iam_role", Name: "lambda_exec"},
	}
}

func TestView_Given_Idle_ShouldRender_LoadingPlaceholder(t *testing.T) {
	p := newGoldenPlugin()
	p.status = StatusIdle

	sdktest.AssertGolden(t, p.View(80, 18))
}

func TestView_Given_Loading_ShouldRender_LoadingMessage(t *testing.T) {
	p := newGoldenPlugin()
	p.status = StatusLoading

	sdktest.AssertGolden(t, p.View(80, 18))
}

func TestView_Given_Error_ShouldRender_ErrorMessage(t *testing.T) {
	p := newGoldenPlugin()
	p.status = StatusError
	p.errMsg = "Failed to read state: no state file found"

	sdktest.AssertGolden(t, p.View(80, 18))
}

func TestView_Given_ErrorWithLock_ShouldRender_LockPanel(t *testing.T) {
	p := newGoldenPlugin()
	p.status = StatusError
	p.lockInfo = &sdk.StateLock{
		ID:        "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
		Who:       "user@machine",
		Operation: "OperationTypePlan",
	}

	sdktest.AssertGolden(t, p.View(80, 18))
}

func TestView_Given_ResourceList_ShouldRender_AllResources(t *testing.T) {
	p := newGoldenPlugin()
	p.status = StatusDone
	p.resources = testResources()
	p.filtered = testResources()

	sdktest.AssertGolden(t, p.View(80, 18))
}

func TestView_Given_ResourceList_WithSelection_ShouldRender_HighlightedRow(t *testing.T) {
	p := newGoldenPlugin()
	p.status = StatusDone
	p.resources = testResources()
	p.filtered = testResources()
	p.selected = 2

	sdktest.AssertGolden(t, p.View(80, 18))
}

func TestView_Given_FilterActive_ShouldRender_FilterInput(t *testing.T) {
	p := newGoldenPlugin()
	p.status = StatusDone
	p.resources = testResources()
	p.filtered = []sdk.Resource{
		{Address: "aws_s3_bucket.data", Type: "aws_s3_bucket", Name: "data"},
	}
	p.filter = "s3"
	p.filtering = true

	sdktest.AssertGolden(t, p.View(80, 18))
}

func TestView_Given_FilterInactive_ShouldRender_FilterLabel(t *testing.T) {
	p := newGoldenPlugin()
	p.status = StatusDone
	p.resources = testResources()
	p.filtered = []sdk.Resource{
		{Address: "aws_s3_bucket.data", Type: "aws_s3_bucket", Name: "data"},
	}
	p.filter = "s3"
	p.filtering = false

	sdktest.AssertGolden(t, p.View(80, 18))
}

func TestView_Given_EmptyResourceList_ShouldRender_NoResourcesMessage(t *testing.T) {
	p := newGoldenPlugin()
	p.status = StatusDone
	p.resources = []sdk.Resource{}
	p.filtered = []sdk.Resource{}

	sdktest.AssertGolden(t, p.View(80, 18))
}

func TestView_Given_PinnedResources_ShouldRender_PinMarkers(t *testing.T) {
	p := newGoldenPlugin()
	p.status = StatusDone
	p.resources = testResources()
	p.filtered = testResources()
	p.session = sdk.NewSession()
	p.session.Set("terraform.pinned", []string{"aws_instance.web", "aws_s3_bucket.data"})

	sdktest.AssertGolden(t, p.View(80, 18))
}

func TestView_Given_DetailView_ShouldRender_ExpandedAttributes(t *testing.T) {
	p := newGoldenPlugin()
	p.status = StatusShowingDetail
	p.detailAddr = "aws_instance.web"
	p.detail = `{
  "id": "i-0abc123def456",
  "instance_type": "t3.micro",
  "ami": "ami-12345678",
  "tags": {
    "Name": "web-server",
    "Environment": "production"
  }
}`

	sdktest.AssertGolden(t, p.View(80, 18))
}

func TestView_Given_DetailView_WithPinned_ShouldRender_PinnedIndicator(t *testing.T) {
	p := newGoldenPlugin()
	p.status = StatusShowingDetail
	p.detailAddr = "aws_instance.web"
	p.detail = `{"id": "i-0abc123def456"}`
	p.session = sdk.NewSession()
	p.session.Set("terraform.pinned", []string{"aws_instance.web"})

	sdktest.AssertGolden(t, p.View(80, 18))
}

func TestView_Given_ManyResources_ShouldRender_ScrolledWindow(t *testing.T) {
	resources := make([]sdk.Resource, 30)
	for i := range resources {
		resources[i] = sdk.Resource{
			Address: fmt.Sprintf("aws_instance.server_%02d", i),
			Type:    "aws_instance",
			Name:    fmt.Sprintf("server_%02d", i),
		}
	}

	p := newGoldenPlugin()
	p.status = StatusDone
	p.resources = resources
	p.filtered = resources
	p.selected = 15

	sdktest.AssertGolden(t, p.View(80, 18))
}
