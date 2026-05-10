package frames

import (
	"testing"

	"github.com/lmarqs/terraform-ui/pkg/sdk/sdktest"
)

func TestFilterView_Given_EmptyQuery_ShouldRender_Cursor(t *testing.T) {
	f := NewFilterFrame(FilterOpts{})

	sdktest.AssertGolden(t, f.View(80, 1))
}

func TestFilterView_Given_Query_ShouldRender_QueryWithCursor(t *testing.T) {
	f := NewFilterFrame(FilterOpts{})
	f.Query = "aws_instance"

	sdktest.AssertGolden(t, f.View(80, 1))
}

func TestConfirmView_Given_Prompt_ShouldRender_PromptWithOptions(t *testing.T) {
	f := NewConfirmFrame("Delete aws_instance.web?", nil, nil)

	sdktest.AssertGolden(t, f.View(80, 1))
}

func TestInspectView_Given_Content_ShouldRender_ViewportContent(t *testing.T) {
	f := NewInspectFrame(InspectOpts{
		Title:   "Resource Detail",
		Address: "aws_instance.web",
		Content: `{
  "id": "i-0abc123def456",
  "instance_type": "t3.micro",
  "ami": "ami-12345678"
}`,
	})

	sdktest.AssertGolden(t, f.View(80, 10))
}
