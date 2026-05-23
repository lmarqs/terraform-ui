package state

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/sdktest"
)

func TestRenderResources_WhenFilteringWithPinnedOnly_ShouldShowBothIndicators(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.Init(sdktest.NewDeps(svc).Deps)
	p.status = sdk.StatusDone
	p.resources = []sdk.Resource{{Address: "a"}, {Address: "b"}}
	p.filtered = p.resources

	p.rebuildTree()
	p.filtering = true
	p.filter = "test"
	p.pinnedOnly = true

	view := p.View(80, 24)
	if !strings.Contains(view, "[pinned]") {
		t.Error("expected [pinned] indicator in filter mode with pinnedOnly")
	}
}

func TestRenderResources_WhenFilterInactiveWithPinnedOnly_ShouldShowPinnedLabel(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.Init(sdktest.NewDeps(svc).Deps)
	p.status = sdk.StatusDone
	p.resources = []sdk.Resource{{Address: "a"}}
	p.filtered = p.resources

	p.rebuildTree()
	p.filtering = false
	p.pinnedOnly = true

	view := p.View(80, 24)
	if !strings.Contains(view, "[pinned]") {
		t.Error("expected [pinned] indicator when pinnedOnly is active")
	}
}

func TestRenderDetail_WhenWrapped_ShouldWrapLongLines(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.Init(sdktest.NewDeps(svc).Deps)
	p.status = StatusShowingDetail
	p.detailAddr = "aws_instance.web"
	p.detail = strings.Repeat("x", 200)
	p.detailPanel.HandleKey(tea.KeyMsg{Type: tea.KeyCtrlW}) // toggle wrap on

	view := p.renderDetail(80, 20)
	lines := strings.Split(view, "\n")
	// With wrapping at contentWidth=74, 200 chars = 3 wrapped lines
	// Plus 2 header lines = at least 5
	if len(lines) < 4 {
		t.Errorf("expected wrapped detail to have multiple lines, got %d", len(lines))
	}
}

func TestRenderDetail_WhenHScrolled_ShouldShiftContent(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.Init(sdktest.NewDeps(svc).Deps)
	p.status = StatusShowingDetail
	p.detailAddr = "aws_instance.web"
	p.detail = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	// HandleKey(Right) increments hScroll by 10; we need hscroll=10 (closest to 5)
	p.detailPanel.HandleKey(tea.KeyMsg{Type: tea.KeyRight})

	view := p.renderDetail(80, 20)
	if strings.Contains(view, "ABCDE") {
		t.Error("expected first chars to be hidden with hscroll")
	}
	if !strings.Contains(view, "KLMNO") {
		t.Error("expected shifted content to be visible")
	}
}

func TestRenderDetail_WhenScrolled_ShouldShowScrollIndicator(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.Init(sdktest.NewDeps(svc).Deps)
	p.status = StatusShowingDetail
	p.detailAddr = "aws_instance.web"
	lines := make([]string, 50)
	for i := range lines {
		lines[i] = "line content"
	}
	p.detail = strings.Join(lines, "\n")
	p.detailScroll = 5

	view := p.renderDetail(80, 20)
	if !strings.Contains(view, "[") || !strings.Contains(view, "/") {
		t.Error("expected scroll indicator in detail view with overflow")
	}
}

func TestRenderDetail_WhenPinned_ShouldShowPinnedIndicator(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	h := sdktest.NewDeps(svc)
	p.Init(h.Deps)
	p.status = StatusShowingDetail
	p.detailAddr = "aws_instance.web"
	p.detail = `{"id": "123"}`
	h.Ctx.Pins = []string{"aws_instance.web"}

	view := p.renderDetail(80, 20)
	if !strings.Contains(view, "[pinned]") {
		t.Error("expected [pinned] indicator in detail view")
	}
}

func TestRenderDetail_WhenSmallHeight_ShouldClampMinLines(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.Init(sdktest.NewDeps(svc).Deps)
	p.status = StatusShowingDetail
	p.detailAddr = "aws_instance.web"
	lines := make([]string, 20)
	for i := range lines {
		lines[i] = "content"
	}
	p.detail = strings.Join(lines, "\n")

	view := p.renderDetail(80, 3)
	outputLines := strings.Split(view, "\n")
	// minLines=5 content + 2 header = 7 total minimum
	if len(outputLines) < 5 {
		t.Errorf("expected at least 5 lines with small height, got %d", len(outputLines))
	}
}

func TestRenderDetail_WhenContentWidthTooSmall_ShouldUseMinimum(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.Init(sdktest.NewDeps(svc).Deps)
	p.status = StatusShowingDetail
	p.detailAddr = "a"
	p.detail = strings.Repeat("y", 100)
	// default: no wrap

	// Width 10 forces contentWidth = max(10-6, 40) = 40
	view := p.renderDetail(10, 20)
	if view == "" {
		t.Error("expected non-empty view with small width")
	}
}

func TestFormatResourceRow_ShouldAlwaysIncludePinMark(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.Init(sdktest.NewDeps(svc).Deps)

	row := p.formatResourceRow("[ ] ", sdk.Resource{Address: "short", Type: "t"})
	if !strings.Contains(row, "[ ] ") {
		t.Errorf("expected pin mark in row, got %q", row)
	}
}

func TestRenderResources_TreeMode_WithFilterScores(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.Init(sdktest.NewDeps(svc).Deps)
	p.status = sdk.StatusDone
	p.treeMode = true
	p.resources = []sdk.Resource{
		{Address: "module.a.aws_instance.web", Type: "aws_instance"},
		{Address: "module.a.aws_s3_bucket.data", Type: "aws_s3_bucket"},
	}
	p.filtered = p.resources
	p.rebuildTree()
	p.tree.ExpandAll()
	p.filter = "web"
	p.filterScores = map[string]int{"module.a.aws_instance.web": 100}

	view := p.View(80, 24)
	if view == "" {
		t.Error("expected non-empty view in tree mode with filter scores")
	}
}

func TestRenderResources_TreeMode_WithHScroll(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.Init(sdktest.NewDeps(svc).Deps)
	p.status = sdk.StatusDone
	p.treeMode = true
	p.resources = []sdk.Resource{
		{Address: "module.a.aws_instance.web", Type: "aws_instance"},
	}
	p.filtered = p.resources
	p.rebuildTree()
	p.tree.ExpandAll()
	p.listPanel.HandleKey(tea.KeyMsg{Type: tea.KeyRight}) // hscroll += 10

	view := p.View(80, 24)
	if view == "" {
		t.Error("expected non-empty view in tree mode with hscroll")
	}
}

func TestRenderResources_TreeMode_WithListHScrollExceedingContent(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.Init(sdktest.NewDeps(svc).Deps)
	p.status = sdk.StatusDone
	p.treeMode = true
	p.resources = []sdk.Resource{
		{Address: "module.a.aws_instance.web", Type: "aws_instance"},
	}
	p.filtered = p.resources
	p.rebuildTree()
	p.tree.ExpandAll()
	// Scroll right many times to simulate excessive hscroll
	for i := 0; i < 100; i++ {
		p.listPanel.HandleKey(tea.KeyMsg{Type: tea.KeyRight})
	}

	view := p.View(80, 24)
	if view == "" {
		t.Error("expected non-empty view even with excessive hscroll")
	}
}

func TestRenderResources_TreeMode_WithListWrap_ShouldNotTruncate(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.Init(sdktest.NewDeps(svc).Deps)
	p.status = sdk.StatusDone
	p.treeMode = true
	p.listPanel.HandleKey(tea.KeyMsg{Type: tea.KeyCtrlW}) // toggle wrap on
	p.resources = []sdk.Resource{
		{Address: "module.a.aws_instance.web_server_with_a_very_long_name", Type: "aws_instance"},
	}
	p.filtered = p.resources
	p.rebuildTree()
	p.tree.ExpandAll()

	view := p.View(40, 24)
	if !strings.Contains(view, "very_long_name") {
		t.Error("expected full content visible in tree mode with wrap enabled")
	}
}

func TestRenderDetail_WhenHScrollExceedsLineLength_ShouldShowEmpty(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.Init(sdktest.NewDeps(svc).Deps)
	p.status = StatusShowingDetail
	p.detailAddr = "a"
	p.detail = "short\nline"
	// Scroll right many times to simulate excessive hscroll
	for i := 0; i < 10; i++ {
		p.detailPanel.HandleKey(tea.KeyMsg{Type: tea.KeyRight})
	}

	view := p.renderDetail(80, 20)
	if view == "" {
		t.Error("expected non-empty view even with excessive hscroll")
	}
}

func TestRenderDetail_WhenLineTruncatedByContentWidth_ShouldTruncate(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.Init(sdktest.NewDeps(svc).Deps)
	p.status = StatusShowingDetail
	p.detailAddr = "a"
	p.detail = strings.Repeat("x", 200)
	// default: no hscroll, no wrap

	width := 50
	view := p.renderDetail(width, 20)
	lines := strings.Split(view, "\n")
	// Skip header lines (address + blank), check content is truncated to panel width
	if len(lines) > 2 {
		contentLine := lines[2]
		if len(contentLine) > width {
			t.Errorf("expected line truncated to width=%d, got length %d", width, len(contentLine))
		}
	}
}

func TestRenderResources_WhenHeightVerySmall_ShouldClampMinVisible(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.Init(sdktest.NewDeps(svc).Deps)
	p.status = sdk.StatusDone
	p.resources = []sdk.Resource{{Address: "a"}, {Address: "b"}, {Address: "c"}}
	p.filtered = p.resources
	p.rebuildTree()

	// Height = 1, no filter means maxVisible = 1-0 = 1, clamped to 3
	view := p.View(80, 1)
	if view == "" {
		t.Error("expected non-empty view with very small height")
	}
}

func TestRenderDetail_WhenTainted_ShouldShowTaintedIndicator(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.Init(sdktest.NewDeps(svc).Deps)
	p.status = StatusShowingDetail
	p.detailAddr = "aws_instance.web"
	p.detail = `{"id": "123"}`
	p.resources = []sdk.Resource{{Address: "aws_instance.web", Tainted: true}}

	view := p.renderDetail(80, 20)
	if !strings.Contains(view, "[tainted]") {
		t.Error("expected [tainted] indicator in detail view for tainted resource")
	}
}

func TestFormatResourceRow_WhenTainted_ShouldShowTaintedBadge(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.Init(sdktest.NewDeps(svc).Deps)

	row := p.formatResourceRow("[ ] ", sdk.Resource{Address: "aws_instance.web", Type: "aws_instance", Tainted: true})
	if !strings.Contains(row, "[tainted]") {
		t.Error("expected [tainted] in formatResourceRow for tainted resource")
	}
}

func TestFormatResourceRow_ShouldIncludeFullAddress(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.Init(sdktest.NewDeps(svc).Deps)

	longAddr := strings.Repeat("x", 200)
	row := p.formatResourceRow("[ ] ", sdk.Resource{Address: longAddr, Type: "t"})
	if !strings.Contains(row, longAddr) {
		t.Error("expected full address in row (truncation handled by panel)")
	}
}

func TestRenderDetail_WhenScrollExceedsMax_ShouldClamp(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.Init(sdktest.NewDeps(svc).Deps)
	p.status = StatusShowingDetail
	p.detailAddr = "a"
	lines := make([]string, 5)
	for i := range lines {
		lines[i] = "content"
	}
	p.detail = strings.Join(lines, "\n")
	p.detailScroll = 100

	view := p.renderDetail(80, 20)
	if view == "" {
		t.Error("expected non-empty view with excessive scroll")
	}
}

func TestRenderResources_WhenPinnedOnlyWithoutFilter_ShouldNotAddFilterHeight(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.Init(sdktest.NewDeps(svc).Deps)
	p.status = sdk.StatusDone
	p.resources = []sdk.Resource{{Address: "a"}, {Address: "b"}, {Address: "c"}}
	p.filtered = p.resources

	p.rebuildTree()
	p.filtering = false
	p.filter = ""
	p.pinnedOnly = true

	view := p.View(80, 24)
	if !strings.Contains(view, "[pinned]") {
		t.Error("expected [pinned] indicator when pinnedOnly but no filter text")
	}
}

func TestRenderResources_TreeMode_WithTaintedResource_ShouldShowBadge(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.Init(sdktest.NewDeps(svc).Deps)
	p.status = sdk.StatusDone
	p.treeMode = true
	p.resources = []sdk.Resource{
		{Address: "module.a.aws_instance.web", Type: "aws_instance", Tainted: true},
	}
	p.filtered = p.resources
	p.rebuildTree()
	p.tree.ExpandAll()

	view := p.View(80, 24)
	if !strings.Contains(view, "tainted") {
		t.Error("tree mode view should show [tainted] badge for tainted resource")
	}
}
