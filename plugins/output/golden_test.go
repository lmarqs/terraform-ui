package output

import (
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
	p.errMsg = "Failed to read outputs"

	sdktest.AssertGolden(t, p.View(80, 18))
}

func TestView_Given_OutputList_ShouldRender_AllOutputs(t *testing.T) {
	p := newGoldenPlugin()
	p.status = StatusDone
	p.outputs = []sdk.OutputValue{
		{Name: "vpc_id", Value: "vpc-abc123", Type: "string"},
		{Name: "subnet_ids", Value: []interface{}{"subnet-1", "subnet-2"}, Type: "list"},
		{Name: "db_password", Value: "secret", Type: "string", Sensitive: true},
	}
	p.filtered = p.outputs

	sdktest.AssertGolden(t, p.View(80, 18))
}

func TestView_Given_OutputList_WithSelection_ShouldRender_HighlightedRow(t *testing.T) {
	p := newGoldenPlugin()
	p.status = StatusDone
	p.outputs = []sdk.OutputValue{
		{Name: "vpc_id", Value: "vpc-abc123", Type: "string"},
		{Name: "subnet_ids", Value: []interface{}{"subnet-1", "subnet-2"}, Type: "list"},
		{Name: "db_password", Value: "secret", Type: "string", Sensitive: true},
	}
	p.filtered = p.outputs
	p.selected = 1

	sdktest.AssertGolden(t, p.View(80, 18))
}

func TestView_Given_FilterActive_ShouldRender_FilterInput(t *testing.T) {
	p := newGoldenPlugin()
	p.status = StatusDone
	p.outputs = []sdk.OutputValue{
		{Name: "vpc_id", Value: "vpc-abc123", Type: "string"},
		{Name: "subnet_ids", Value: []interface{}{"subnet-1", "subnet-2"}, Type: "list"},
	}
	p.filtered = []sdk.OutputValue{
		{Name: "vpc_id", Value: "vpc-abc123", Type: "string"},
	}
	p.filter = "vpc"
	p.filtering = true

	sdktest.AssertGolden(t, p.View(80, 18))
}

func TestView_Given_EmptyOutputList_ShouldRender_NoOutputsMessage(t *testing.T) {
	p := newGoldenPlugin()
	p.status = StatusDone
	p.outputs = []sdk.OutputValue{}
	p.filtered = []sdk.OutputValue{}

	sdktest.AssertGolden(t, p.View(80, 18))
}
