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
	p.rebuildTree()
	sdktest.AssertGolden(t, p.View(80, 18))
}

func TestView_Given_ResourceList_WithSelection_ShouldRender_HighlightedRow(t *testing.T) {
	p := newGoldenPlugin()
	p.status = StatusDone
	p.resources = testResources()
	p.filtered = testResources()
	p.rebuildTree()
	p.tree.MoveDown()
	p.tree.MoveDown()
	sdktest.AssertGolden(t, p.View(80, 18))
}

func TestView_Given_FilterActive_ShouldRender_FilterInput(t *testing.T) {
	p := newGoldenPlugin()
	p.status = StatusDone
	p.resources = testResources()
	p.filtered = []sdk.Resource{
		{Address: "aws_s3_bucket.data", Type: "aws_s3_bucket", Name: "data"},
	}
	p.rebuildTree()
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
	p.rebuildTree()
	p.filter = "s3"
	p.filtering = false
	sdktest.AssertGolden(t, p.View(80, 18))
}

func TestView_Given_EmptyResourceList_ShouldRender_NoResourcesMessage(t *testing.T) {
	p := newGoldenPlugin()
	p.status = StatusDone
	p.resources = []sdk.Resource{}
	p.filtered = []sdk.Resource{}
	p.rebuildTree()
	sdktest.AssertGolden(t, p.View(80, 18))
}

func TestView_Given_PinnedResources_ShouldRender_PinMarkers(t *testing.T) {
	p := newGoldenPlugin()
	p.status = StatusDone
	p.resources = testResources()
	p.filtered = testResources()
	p.session = sdk.NewSession()
	p.session.Set("terraform.pinned", []string{"aws_instance.web", "aws_s3_bucket.data"})
	p.rebuildTree()
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

func realisticResources() []sdk.Resource {
	return []sdk.Resource{
		{Address: "module.medprev_online.module.postgresql_proxy.aws_db_proxy.this[0]", Type: "aws_db_proxy", Module: "module.medprev_online.module.postgresql_proxy"},
		{Address: "module.medprev_online.module.postgresql_proxy.aws_db_proxy_default_target_group.this[0]", Type: "aws_db_proxy_default_target_group", Module: "module.medprev_online.module.postgresql_proxy"},
		{Address: "module.medprev_online.module.postgresql_proxy.aws_db_proxy_endpoint.read_only[0]", Type: "aws_db_proxy_endpoint", Module: "module.medprev_online.module.postgresql_proxy"},
		{Address: "module.medprev_online.module.medprev_api.aws_lambda_function.api", Type: "aws_lambda_function", Module: "module.medprev_online.module.medprev_api"},
		{Address: "module.medprev_online.module.medprev_api.aws_lambda_function.worker", Type: "aws_lambda_function", Module: "module.medprev_online.module.medprev_api"},
		{Address: "module.medprev_online.module.medprev_api.aws_api_gateway_rest_api.this", Type: "aws_api_gateway_rest_api", Module: "module.medprev_online.module.medprev_api"},
		{Address: "module.cloudwatch.aws_cloudwatch_metric_alarm.cpu_high", Type: "aws_cloudwatch_metric_alarm", Module: "module.cloudwatch"},
		{Address: "module.cloudwatch.aws_cloudwatch_metric_alarm.memory_high", Type: "aws_cloudwatch_metric_alarm", Module: "module.cloudwatch"},
		{Address: "module.cloudwatch.aws_cloudwatch_dashboard.main", Type: "aws_cloudwatch_dashboard", Module: "module.cloudwatch"},
		{Address: "aws_s3_bucket.terraform_state", Type: "aws_s3_bucket"},
		{Address: "aws_dynamodb_table.terraform_locks", Type: "aws_dynamodb_table"},
	}
}

func TestView_Given_Tree_AllCollapsed_ShouldRender_ModuleGroups(t *testing.T) {
	p := newGoldenPlugin()
	p.status = StatusDone
	p.resources = realisticResources()
	p.filtered = realisticResources()
	p.treeMode = true
	p.rebuildTree()
	sdktest.AssertGolden(t, p.View(80, 18))
}

func TestView_Given_Tree_OneModuleExpanded_ShouldRender_Children(t *testing.T) {
	p := newGoldenPlugin()
	p.status = StatusDone
	p.resources = realisticResources()
	p.filtered = realisticResources()
	p.treeMode = true
	p.rebuildTree()
	// Expand module.cloudwatch (first in alphabetical order)
	p.tree.Toggle()
	sdktest.AssertGolden(t, p.View(80, 18))
}

func TestView_Given_Tree_NestedExpanded_ShouldRender_FullHierarchy(t *testing.T) {
	p := newGoldenPlugin()
	p.status = StatusDone
	p.resources = realisticResources()
	p.filtered = realisticResources()
	p.treeMode = true
	p.rebuildTree()
	// Expand all to show full tree
	p.tree.ExpandAll()
	sdktest.AssertGolden(t, p.View(80, 24))
}

func TestView_Given_Tree_PinnedModule_ShouldRender_PinOnGroup(t *testing.T) {
	p := newGoldenPlugin()
	p.status = StatusDone
	p.resources = realisticResources()
	p.filtered = realisticResources()
	p.session = sdk.NewSession()
	p.session.Set("terraform.pinned", []string{"module.cloudwatch"})
	p.rebuildTree()
	sdktest.AssertGolden(t, p.View(80, 18))
}

func TestView_Given_Tree_PartialExpand_ShouldRender_MixedState(t *testing.T) {
	p := newGoldenPlugin()
	p.status = StatusDone
	p.resources = realisticResources()
	p.filtered = realisticResources()
	p.treeMode = true
	p.rebuildTree()
	// Expand medprev_online but keep its children collapsed
	p.tree.MoveDown() // move to module.medprev_online
	p.tree.Toggle()   // expand it — shows sub-modules collapsed
	sdktest.AssertGolden(t, p.View(80, 18))
}

func TestView_Given_Tree_DeepExpand_ShouldRender_TreeConnectors(t *testing.T) {
	p := newGoldenPlugin()
	p.status = StatusDone
	p.resources = realisticResources()
	p.filtered = realisticResources()
	p.treeMode = true
	p.rebuildTree()
	// Expand medprev_online and then postgresql_proxy
	p.tree.MoveDown() // module.medprev_online
	p.tree.Toggle()   // expand
	p.tree.MoveDown() // module.medprev_api (first child)
	p.tree.Toggle()   // expand medprev_api
	sdktest.AssertGolden(t, p.View(80, 24))
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
	p.rebuildTree()
	for i := 0; i < 15; i++ {
		p.tree.MoveDown()
	}
	sdktest.AssertGolden(t, p.View(80, 18))
}
