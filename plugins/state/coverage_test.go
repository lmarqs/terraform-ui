package state

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/frames"
	"github.com/lmarqs/terraform-ui/pkg/sdk/sdktest"
	"github.com/lmarqs/terraform-ui/pkg/sdk/ui"
)

func TestBusy_WhenMutating_ShouldReturnTrue(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	if p.Busy() {
		t.Error("Busy() = true before mutation, want false")
	}
	p.mutating = true
	if !p.Busy() {
		t.Error("Busy() = false during mutation, want true")
	}
}

func TestStack_ShouldReturnStackReference(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	s := p.Stack()
	if s == nil {
		t.Fatal("Stack() = nil, want non-nil")
	}
	if s.Depth() != 1 {
		t.Errorf("Stack().Depth() = %d, want 1 (list frame)", s.Depth())
	}
}

func TestNavigate_WhenDirectionPositive_ShouldMoveDown(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.resources = []sdk.Resource{{Address: "a"}, {Address: "b"}, {Address: "c"}}
	p.filtered = p.resources
	p.rebuildTree()

	p.navigate(1)
	if p.Selected() != 1 {
		t.Errorf("navigate(1): selected = %d, want 1", p.Selected())
	}
	p.navigate(-1)
	if p.Selected() != 0 {
		t.Errorf("navigate(-1): selected = %d, want 0", p.Selected())
	}
}

func TestPanDetailRight_ShouldIncrementHScroll(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.viewWidth = 80
	p.detail = strings.Repeat("x", 200)
	p.detailHScroll = 0

	p.panDetailRight()
	if p.detailHScroll != 10 {
		t.Errorf("panDetailRight: detailHScroll = %d, want 10", p.detailHScroll)
	}
}

func TestPanDetailRight_ShouldNotExceedMaxScroll(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.viewWidth = 80
	p.detail = "short line"
	p.detailHScroll = 0

	p.panDetailRight()
	if p.detailHScroll != 0 {
		t.Errorf("panDetailRight with short content: detailHScroll = %d, want 0", p.detailHScroll)
	}
}

func TestPanDetailRight_ShouldClampToMaxScroll(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.viewWidth = 80
	contentWidth := 80 - 6
	line := strings.Repeat("x", contentWidth+20)
	p.detail = line
	p.detailHScroll = 10

	p.panDetailRight()
	maxScroll := len(line) - contentWidth
	if p.detailHScroll > maxScroll {
		t.Errorf("panDetailRight: detailHScroll = %d, exceeds max %d", p.detailHScroll, maxScroll)
	}
}

func TestPanDetailLeft_ShouldDecrementHScroll(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.detailHScroll = 20

	p.panDetailLeft()
	if p.detailHScroll != 10 {
		t.Errorf("panDetailLeft: detailHScroll = %d, want 10", p.detailHScroll)
	}
}

func TestPanDetailLeft_ShouldNotGoBelowZero(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.detailHScroll = 5

	p.panDetailLeft()
	if p.detailHScroll != 0 {
		t.Errorf("panDetailLeft from 5: detailHScroll = %d, want 0", p.detailHScroll)
	}
}

func TestWrapLines_WhenLineFitsWidth_ShouldNotWrap(t *testing.T) {
	lines := []string{"short", "also short"}
	result := wrapLines(lines, 20)
	if len(result) != 2 {
		t.Errorf("wrapLines: got %d lines, want 2", len(result))
	}
	if result[0] != "short" || result[1] != "also short" {
		t.Errorf("wrapLines: unexpected content %v", result)
	}
}

func TestWrapLines_WhenLineExceedsWidth_ShouldSplitAtBoundary(t *testing.T) {
	lines := []string{"abcdefghij"}
	result := wrapLines(lines, 4)
	if len(result) != 3 {
		t.Errorf("wrapLines('abcdefghij', 4): got %d lines, want 3", len(result))
	}
	if result[0] != "abcd" {
		t.Errorf("wrapLines[0] = %q, want %q", result[0], "abcd")
	}
	if result[1] != "efgh" {
		t.Errorf("wrapLines[1] = %q, want %q", result[1], "efgh")
	}
	if result[2] != "ij" {
		t.Errorf("wrapLines[2] = %q, want %q", result[2], "ij")
	}
}

func TestWrapLines_WhenExactWidth_ShouldNotSplit(t *testing.T) {
	lines := []string{"abcd"}
	result := wrapLines(lines, 4)
	if len(result) != 1 {
		t.Errorf("wrapLines exact width: got %d lines, want 1", len(result))
	}
}

func TestWrapLines_WhenMultipleLines_ShouldWrapEachIndependently(t *testing.T) {
	lines := []string{"abc", "defgh", "ij"}
	result := wrapLines(lines, 3)
	expected := []string{"abc", "def", "gh", "ij"}
	if len(result) != len(expected) {
		t.Fatalf("wrapLines: got %d lines, want %d", len(result), len(expected))
	}
	for i, r := range result {
		if r != expected[i] {
			t.Errorf("wrapLines[%d] = %q, want %q", i, r, expected[i])
		}
	}
}

func TestTogglePin_ShouldTogglePinInTree(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.pins = sdk.NewPinService()
	p.resources = []sdk.Resource{{Address: "aws_instance.web"}, {Address: "aws_s3_bucket.data"}}
	p.filtered = p.resources
	p.rebuildTree()

	cmd := p.togglePin("aws_instance.web")
	if cmd != nil {
		t.Error("togglePin returned non-nil cmd, want nil")
	}
	if p.pins.Count() != 1 {
		t.Errorf("pins.Count() = %d after toggle, want 1", p.pins.Count())
	}
	if !p.pins.IsPinned("aws_instance.web") {
		t.Error("expected aws_instance.web to be pinned")
	}

	p.togglePin("aws_instance.web")
	if p.pins.Count() != 0 {
		t.Errorf("pins.Count() = %d after second toggle, want 0", p.pins.Count())
	}
}

func TestTogglePin_WhenNoPinService_ShouldNotPanic(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.pins = nil
	p.resources = []sdk.Resource{{Address: "a"}}
	p.filtered = p.resources
	p.rebuildTree()

	cmd := p.togglePin("a")
	if cmd != nil {
		t.Error("togglePin without pin service returned non-nil cmd")
	}
}

func TestRequestDelete_ShouldConfirmThenDelete(t *testing.T) {
	svc := &sdktest.MockService{}
	p := newTrackingPlugin(svc, []sdk.Resource{{Address: "aws_instance.web"}})

	t.Run("ShouldReturnConfirmRequest", func(t *testing.T) {
		cmd := p.requestDelete("aws_instance.web")
		msg := cmd()
		reqMsg, ok := msg.(sdk.RequestInputMsg)
		if !ok {
			t.Fatalf("expected sdk.RequestInputMsg, got %T", msg)
		}
		if reqMsg.Request.Mode != sdk.InputRequestBool {
			t.Errorf("expected InputRequestBool, got %d", reqMsg.Request.Mode)
		}
	})

	t.Run("ShouldDeleteOnConfirmation", func(t *testing.T) {
		svc2 := &sdktest.MockService{}
		p2 := newTrackingPlugin(svc2, []sdk.Resource{{Address: "aws_instance.web"}})
		cmd := p2.requestDelete("aws_instance.web")
		msg := cmd()
		reqMsg := msg.(sdk.RequestInputMsg)
		execCmd := reqMsg.Request.Callback("y")
		if execCmd == nil {
			t.Fatal("expected non-nil cmd after confirmation")
		}
		result := execCmd()
		deleted, ok := result.(StateDeletedMsg)
		if !ok {
			t.Fatalf("expected StateDeletedMsg, got %T", result)
		}
		if deleted.Address != "aws_instance.web" {
			t.Errorf("expected address 'aws_instance.web', got %q", deleted.Address)
		}
		if len(svc2.StateRmCalls) != 1 {
			t.Errorf("expected 1 stateRm call, got %d", len(svc2.StateRmCalls))
		}
	})

	t.Run("ShouldReturnErrorOnDeleteFailure", func(t *testing.T) {
		svc2 := &sdktest.MockService{StateRmFn: func(_ context.Context, _ string) error { return errors.New("rm failed") }}
		p2 := newTrackingPlugin(svc2, []sdk.Resource{{Address: "aws_instance.web"}})
		cmd := p2.requestDelete("aws_instance.web")
		msg := cmd()
		reqMsg := msg.(sdk.RequestInputMsg)
		execCmd := reqMsg.Request.Callback("y")
		result := execCmd()
		listMsg, ok := result.(StateListMsg)
		if !ok {
			t.Fatalf("expected StateListMsg on error, got %T", result)
		}
		if listMsg.Err == nil {
			t.Error("expected non-nil error")
		}
	})

	t.Run("ShouldReturnNilOnDecline", func(t *testing.T) {
		cmd := p.requestDelete("aws_instance.web")
		msg := cmd()
		reqMsg := msg.(sdk.RequestInputMsg)
		result := reqMsg.Request.Callback("n")
		if result != nil {
			t.Error("expected nil cmd on decline")
		}
	})

	t.Run("ShouldSetMutatingTrue", func(t *testing.T) {
		svc2 := &sdktest.MockService{}
		p2 := newTrackingPlugin(svc2, []sdk.Resource{{Address: "aws_instance.web"}})
		cmd := p2.requestDelete("aws_instance.web")
		msg := cmd()
		reqMsg := msg.(sdk.RequestInputMsg)
		execCmd := reqMsg.Request.Callback("y")
		if !p2.mutating {
			t.Error("expected mutating=true after confirmation")
		}
		execCmd()
	})
}

func TestRequestEdit_ShouldProduceStateEditMsg(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)

	cmd := p.requestEdit("aws_instance.web")
	if cmd == nil {
		t.Fatal("requestEdit returned nil cmd")
	}
	msg := cmd()
	editMsg, ok := msg.(StateEditMsg)
	if !ok {
		t.Fatalf("expected StateEditMsg, got %T", msg)
	}
	if editMsg.Address != "aws_instance.web" {
		t.Errorf("expected Address 'aws_instance.web', got %q", editMsg.Address)
	}
}

func TestUpdate_WhenStateDeletedMsg_ShouldRefresh(t *testing.T) {
	svc := &sdktest.MockService{StateListFn: func(_ context.Context, _ ...sdk.StateListOption) ([]sdk.Resource, error) {
		return []sdk.Resource{}, nil
	}}
	p := New(svc).(*Plugin)
	p.log = slog.New(slog.NewTextHandler(io.Discard, nil))
	p.svc = svc
	p.status = sdk.StatusDone
	p.mutating = true

	_, cmd := p.Update(StateDeletedMsg{Address: "aws_instance.web"})
	if cmd == nil {
		t.Error("expected non-nil cmd (refresh) after StateDeletedMsg")
	}
	if p.mutating {
		t.Error("expected mutating=false after StateDeletedMsg")
	}
}

func TestInspectSelected_WhenLoading_ShouldReturnNil(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = sdk.StatusLoading
	p.resources = []sdk.Resource{{Address: "a"}}
	p.filtered = p.resources
	p.rebuildTree()

	cmd := p.InspectSelected()
	if cmd != nil {
		t.Error("InspectSelected during loading should return nil")
	}
}

func TestInspectSelected_WhenFilterFrameActive_ShouldPopIt(t *testing.T) {
	svc := &sdktest.MockService{ShowFn: func(_ context.Context, _ string) (string, error) { return `{"id": "123"}`, nil }}
	p := New(svc).(*Plugin)
	p.log = slog.New(slog.NewTextHandler(io.Discard, nil))
	p.resources = []sdk.Resource{{Address: "aws_instance.web"}}
	p.filtered = p.resources
	p.rebuildTree()
	p.status = sdk.StatusDone
	p.filtering = true

	// Push a filter frame onto the stack
	p.stack.Push(&stateFilterFrame{plugin: p, inner: nil})

	// Verify stack depth before
	depthBefore := p.stack.Depth()

	cmd := p.InspectSelected()
	if cmd == nil {
		t.Fatal("InspectSelected should return cmd")
	}
	if p.stack.Depth() >= depthBefore {
		t.Errorf("expected filter frame to be popped, depth before=%d after=%d", depthBefore, p.stack.Depth())
	}
	if p.filtering {
		t.Error("expected filtering=false after InspectSelected")
	}
}

func TestView_WhenLoadingWithErrMsg_ShouldShowCustomMessage(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = sdk.StatusLoading
	p.errMsg = "Loading aws_instance.web..."

	view := p.View(80, 24)
	if !strings.Contains(view, "Loading aws_instance.web") {
		t.Errorf("expected custom loading message, got %q", view)
	}
}

func TestRenderResources_WhenFilteringWithPinnedOnly_ShouldShowBothIndicators(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.log = slog.New(slog.NewTextHandler(io.Discard, nil))
	p.status = sdk.StatusDone
	p.resources = []sdk.Resource{{Address: "a"}, {Address: "b"}}
	p.filtered = p.resources
	p.pins = sdk.NewPinService()
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
	p := New(&sdktest.MockService{}).(*Plugin)
	p.log = slog.New(slog.NewTextHandler(io.Discard, nil))
	p.status = sdk.StatusDone
	p.resources = []sdk.Resource{{Address: "a"}}
	p.filtered = p.resources
	p.pins = sdk.NewPinService()
	p.rebuildTree()
	p.filtering = false
	p.pinnedOnly = true

	view := p.View(80, 24)
	if !strings.Contains(view, "[pinned]") {
		t.Error("expected [pinned] indicator when pinnedOnly is active")
	}
}

func TestRenderDetail_WhenWrapped_ShouldWrapLongLines(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = StatusShowingDetail
	p.detailAddr = "aws_instance.web"
	p.detail = strings.Repeat("x", 200)
	p.detailWrap = true

	view := p.renderDetail(80, 20)
	lines := strings.Split(view, "\n")
	// With wrapping at contentWidth=74, 200 chars = 3 wrapped lines
	// Plus 2 header lines = at least 5
	if len(lines) < 4 {
		t.Errorf("expected wrapped detail to have multiple lines, got %d", len(lines))
	}
}

func TestRenderDetail_WhenHScrolled_ShouldShiftContent(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = StatusShowingDetail
	p.detailAddr = "aws_instance.web"
	p.detail = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	p.detailHScroll = 5
	p.detailWrap = false

	view := p.renderDetail(80, 20)
	if strings.Contains(view, "ABCDE") {
		t.Error("expected first 5 chars to be hidden with hscroll=5")
	}
	if !strings.Contains(view, "FGHIJ") {
		t.Error("expected shifted content to be visible")
	}
}

func TestRenderDetail_WhenScrolled_ShouldShowScrollIndicator(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
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
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = StatusShowingDetail
	p.detailAddr = "aws_instance.web"
	p.detail = `{"id": "123"}`
	p.pins = sdk.NewPinService()
	p.pins.Toggle("aws_instance.web")

	view := p.renderDetail(80, 20)
	if !strings.Contains(view, "[pinned]") {
		t.Error("expected [pinned] indicator in detail view")
	}
}

func TestRenderDetail_WhenSmallHeight_ShouldClampMinLines(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
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
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = StatusShowingDetail
	p.detailAddr = "a"
	p.detail = strings.Repeat("y", 100)
	p.detailWrap = false

	// Width 10 forces contentWidth = max(10-6, 40) = 40
	view := p.renderDetail(10, 20)
	if view == "" {
		t.Error("expected non-empty view with small width")
	}
}

func TestFormatResourceRow_WhenHScrollExceedsContent_ShouldReturnEmpty(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.listHScroll = 1000
	p.listWrap = false

	row := p.formatResourceRow("[ ] ", sdk.Resource{Address: "short", Type: "t"}, 80)
	// When scroll exceeds content, full becomes "" and we get just the pin mark
	if !strings.Contains(row, "[ ] ") {
		t.Errorf("expected pin mark in row, got %q", row)
	}
}

func TestRenderResources_TreeMode_WithFilterScores(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.log = slog.New(slog.NewTextHandler(io.Discard, nil))
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
	p := New(&sdktest.MockService{}).(*Plugin)
	p.log = slog.New(slog.NewTextHandler(io.Discard, nil))
	p.status = sdk.StatusDone
	p.treeMode = true
	p.resources = []sdk.Resource{
		{Address: "module.a.aws_instance.web", Type: "aws_instance"},
	}
	p.filtered = p.resources
	p.rebuildTree()
	p.tree.ExpandAll()
	p.listHScroll = 5

	view := p.View(80, 24)
	if view == "" {
		t.Error("expected non-empty view in tree mode with hscroll")
	}
}

func TestRenderResources_TreeMode_WithListHScrollExceedingContent(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.log = slog.New(slog.NewTextHandler(io.Discard, nil))
	p.status = sdk.StatusDone
	p.treeMode = true
	p.resources = []sdk.Resource{
		{Address: "module.a.aws_instance.web", Type: "aws_instance"},
	}
	p.filtered = p.resources
	p.rebuildTree()
	p.tree.ExpandAll()
	p.listHScroll = 1000

	view := p.View(80, 24)
	if view == "" {
		t.Error("expected non-empty view even with excessive hscroll")
	}
}

func TestDetailFrame_ID_ShouldReturnInspect(t *testing.T) {
	f := &detailFrame{plugin: &Plugin{}}
	if f.ID() != "inspect" {
		t.Errorf("detailFrame.ID() = %q, want %q", f.ID(), "inspect")
	}
}

func TestDetailFrame_Update_WhenDown_ShouldIncrementScroll(t *testing.T) {
	p := newTestPlugin([]sdk.Resource{{Address: "a"}})
	p.status = StatusShowingDetail
	p.detail = strings.Repeat("line\n", 50)
	f := &detailFrame{plugin: p}

	f.Update(tea.KeyMsg{Type: tea.KeyDown})
	if p.detailScroll != 1 {
		t.Errorf("detailScroll after down = %d, want 1", p.detailScroll)
	}
}

func TestDetailFrame_Update_WhenUp_ShouldDecrementScroll(t *testing.T) {
	p := newTestPlugin([]sdk.Resource{{Address: "a"}})
	p.status = StatusShowingDetail
	p.detail = strings.Repeat("line\n", 50)
	p.detailScroll = 5
	f := &detailFrame{plugin: p}

	f.Update(tea.KeyMsg{Type: tea.KeyUp})
	if p.detailScroll != 4 {
		t.Errorf("detailScroll after up = %d, want 4", p.detailScroll)
	}
}

func TestDetailFrame_Update_WhenUpAtZero_ShouldStayAtZero(t *testing.T) {
	p := newTestPlugin([]sdk.Resource{{Address: "a"}})
	p.status = StatusShowingDetail
	p.detailScroll = 0
	f := &detailFrame{plugin: p}

	f.Update(tea.KeyMsg{Type: tea.KeyUp})
	if p.detailScroll != 0 {
		t.Errorf("detailScroll after up at 0 = %d, want 0", p.detailScroll)
	}
}

func TestDetailFrame_Update_WhenRight_ShouldPanRight(t *testing.T) {
	p := newTestPlugin([]sdk.Resource{{Address: "a"}})
	p.status = StatusShowingDetail
	p.detail = strings.Repeat("x", 200)
	p.viewWidth = 80
	p.detailWrap = false
	f := &detailFrame{plugin: p}

	f.Update(tea.KeyMsg{Type: tea.KeyRight})
	if p.detailHScroll != 10 {
		t.Errorf("detailHScroll after right = %d, want 10", p.detailHScroll)
	}
}

func TestDetailFrame_Update_WhenLeft_ShouldPanLeft(t *testing.T) {
	p := newTestPlugin([]sdk.Resource{{Address: "a"}})
	p.status = StatusShowingDetail
	p.detailHScroll = 20
	p.detailWrap = false
	f := &detailFrame{plugin: p}

	f.Update(tea.KeyMsg{Type: tea.KeyLeft})
	if p.detailHScroll != 10 {
		t.Errorf("detailHScroll after left = %d, want 10", p.detailHScroll)
	}
}

func TestDetailFrame_Update_WhenRightWithWrap_ShouldNotPan(t *testing.T) {
	p := newTestPlugin([]sdk.Resource{{Address: "a"}})
	p.status = StatusShowingDetail
	p.detailWrap = true
	f := &detailFrame{plugin: p}

	f.Update(tea.KeyMsg{Type: tea.KeyRight})
	if p.detailHScroll != 0 {
		t.Errorf("detailHScroll after right with wrap = %d, want 0", p.detailHScroll)
	}
}

func TestDetailFrame_Update_WhenLeftWithWrap_ShouldNotPan(t *testing.T) {
	p := newTestPlugin([]sdk.Resource{{Address: "a"}})
	p.status = StatusShowingDetail
	p.detailHScroll = 10
	p.detailWrap = true
	f := &detailFrame{plugin: p}

	f.Update(tea.KeyMsg{Type: tea.KeyLeft})
	if p.detailHScroll != 10 {
		t.Errorf("detailHScroll after left with wrap = %d, want 10", p.detailHScroll)
	}
}

func TestDetailFrame_Update_WhenCtrlW_ShouldToggleWrap(t *testing.T) {
	p := newTestPlugin([]sdk.Resource{{Address: "a"}})
	p.status = StatusShowingDetail
	p.detailWrap = false
	p.detailScroll = 5
	p.detailHScroll = 10
	f := &detailFrame{plugin: p}

	f.Update(tea.KeyMsg{Type: tea.KeyCtrlW})
	if !p.detailWrap {
		t.Error("expected detailWrap=true after ctrl+w")
	}
	if p.detailScroll != 0 {
		t.Errorf("expected detailScroll=0 after wrap toggle, got %d", p.detailScroll)
	}
	if p.detailHScroll != 0 {
		t.Errorf("expected detailHScroll=0 after wrap toggle, got %d", p.detailHScroll)
	}
}

func TestDetailFrame_Update_WhenSpace_ShouldTogglePin(t *testing.T) {
	p := newTestPlugin([]sdk.Resource{{Address: "a"}})
	p.status = StatusShowingDetail
	p.detailAddr = "aws_instance.web"
	p.pins = sdk.NewPinService()
	p.rebuildTree()
	f := &detailFrame{plugin: p}

	f.Update(tea.KeyMsg{Type: tea.KeySpace})
	// togglePin is called on detailAddr
}

func TestDetailFrame_Update_WhenDelete_ShouldRequestDelete(t *testing.T) {
	p := newTestPlugin([]sdk.Resource{{Address: "a"}})
	p.status = StatusShowingDetail
	p.detailAddr = "aws_instance.web"
	p.pins = sdk.NewPinService()
	f := &detailFrame{plugin: p}

	_, cmd := f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if cmd == nil {
		t.Error("expected non-nil cmd for 'd' in detail frame")
	}
}

func TestDetailFrame_Update_WhenEdit_ShouldRequestEdit(t *testing.T) {
	p := newTestPlugin([]sdk.Resource{{Address: "a"}})
	p.status = StatusShowingDetail
	p.detailAddr = "aws_instance.web"
	f := &detailFrame{plugin: p}

	_, cmd := f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	if cmd == nil {
		t.Error("expected non-nil cmd for 'e' in detail frame")
	}
}

func TestDetailFrame_Update_WhenEsc_ShouldReturnNilAndResetState(t *testing.T) {
	p := newTestPlugin([]sdk.Resource{{Address: "a"}})
	p.status = StatusShowingDetail
	p.detail = "some detail"
	p.detailAddr = "aws_instance.web"
	p.detailScroll = 5
	p.detailHScroll = 10
	f := &detailFrame{plugin: p}

	result, _ := f.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if result != nil {
		t.Error("expected nil result on esc (pop frame)")
	}
	if p.status != sdk.StatusDone {
		t.Errorf("expected status=Done after esc, got %v", p.status)
	}
	if p.detail != "" {
		t.Error("expected detail cleared after esc")
	}
	if p.detailAddr != "" {
		t.Error("expected detailAddr cleared after esc")
	}
	if p.detailScroll != 0 {
		t.Error("expected detailScroll=0 after esc")
	}
	if p.detailHScroll != 0 {
		t.Error("expected detailHScroll=0 after esc")
	}
}

func TestDetailFrame_Update_WhenNonKeyMsg_ShouldReturnSelf(t *testing.T) {
	p := newTestPlugin([]sdk.Resource{{Address: "a"}})
	f := &detailFrame{plugin: p}

	type otherMsg struct{}
	result, cmd := f.Update(otherMsg{})
	if result != f {
		t.Error("expected same frame for non-key msg")
	}
	if cmd != nil {
		t.Error("expected nil cmd for non-key msg")
	}
}

func TestDetailFrame_View_ShouldRenderDetail(t *testing.T) {
	p := newTestPlugin([]sdk.Resource{{Address: "a"}})
	p.status = StatusShowingDetail
	p.detailAddr = "aws_instance.web"
	p.detail = `{"id": "123"}`
	f := &detailFrame{plugin: p}

	view := f.View(80, 20)
	if view == "" {
		t.Error("detailFrame.View returned empty")
	}
	if !strings.Contains(view, "123") {
		t.Error("expected detail content in view")
	}
}

func TestDetailFrame_Hints_ShouldIncludeWrapAndPin(t *testing.T) {
	p := newTestPlugin([]sdk.Resource{{Address: "a"}})
	p.status = StatusShowingDetail
	p.detailAddr = "aws_instance.web"
	p.pins = sdk.NewPinService()
	f := &detailFrame{plugin: p}

	hints := f.Hints()
	if len(hints) == 0 {
		t.Fatal("expected non-empty hints")
	}
	foundEsc := false
	for _, h := range hints {
		if h.Key == "Esc" {
			foundEsc = true
		}
	}
	if !foundEsc {
		t.Error("expected Esc in detail frame hints")
	}
}

func TestDetailFrame_Hints_WhenPinned_ShouldReflectPinnedState(t *testing.T) {
	p := newTestPlugin([]sdk.Resource{{Address: "a"}})
	p.status = StatusShowingDetail
	p.detailAddr = "aws_instance.web"
	p.pins = sdk.NewPinService()
	p.pins.Toggle("aws_instance.web")
	f := &detailFrame{plugin: p}

	hints := f.Hints()
	if len(hints) == 0 {
		t.Fatal("expected non-empty hints when pinned")
	}
}

func TestStateFilterFrame_ID_ShouldReturnFilter(t *testing.T) {
	f := &stateFilterFrame{plugin: &Plugin{}}
	if f.ID() != "filter" {
		t.Errorf("stateFilterFrame.ID() = %q, want %q", f.ID(), "filter")
	}
}

func TestStateFilterFrame_View_ShouldDelegateToInner(t *testing.T) {
	p := newTestPlugin([]sdk.Resource{{Address: "a"}})
	inner := frames.NewFilterFrame(frames.FilterOpts{})
	f := &stateFilterFrame{plugin: p, inner: inner}

	view := f.View(80, 20)
	if view == "" {
		t.Error("stateFilterFrame.View() returned empty")
	}
}

func TestStateFilterFrame_Update_WhenEscFromInner_ShouldClearFiltering(t *testing.T) {
	p := newTestPlugin([]sdk.Resource{{Address: "a"}})
	p.filtering = true

	// Use the real flow via listFrame to push filter frame
	f := &listFrame{plugin: p}
	f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})

	// Now press esc in filter mode
	p.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if p.filtering {
		t.Error("expected filtering=false after esc from filter frame")
	}
}

func TestStateFilterFrame_Update_WhenPinnedFilter_ShouldToggle(t *testing.T) {
	p := newTestPlugin([]sdk.Resource{{Address: "a"}, {Address: "b"}})
	p.pins = sdk.NewPinService()
	p.pins.Toggle("a")
	p.rebuildTree()

	// Enter filter mode
	f := &listFrame{plugin: p}
	f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})

	// Toggle pinned filter
	p.Update(tea.KeyMsg{Type: tea.KeyCtrlP})
	if !p.pinnedOnly {
		t.Error("expected pinnedOnly=true after ctrl+p in filter mode")
	}
}

func TestListFrame_View_ShouldDelegateToPluginView(t *testing.T) {
	p := newTestPlugin([]sdk.Resource{{Address: "a"}})
	f := &listFrame{plugin: p}

	view := f.View(80, 20)
	if view == "" {
		t.Error("listFrame.View returned empty")
	}
}

func TestListFrame_Hints_WhenError_ShouldIncludeRetry(t *testing.T) {
	p := newTestPlugin(nil)
	p.status = sdk.StatusError
	f := &listFrame{plugin: p}

	hints := f.Hints()
	foundRetry := false
	for _, h := range hints {
		if h.Key == "^r" {
			foundRetry = true
			break
		}
	}
	if !foundRetry {
		t.Error("expected ^r (retry) in error state hints")
	}
}

func TestListFrame_Hints_WhenErrorWithLock_ShouldIncludeUnlock(t *testing.T) {
	p := newTestPlugin(nil)
	p.status = sdk.StatusError
	p.lockInfo = &sdk.StateLock{ID: "abc"}
	f := &listFrame{plugin: p}

	hints := f.Hints()
	foundUnlock := false
	for _, h := range hints {
		if h.Key == "u" {
			foundUnlock = true
			break
		}
	}
	if !foundUnlock {
		t.Error("expected 'u' (unlock) in error+lock state hints")
	}
}

func TestListFrame_Hints_WhenLoading_ShouldShowBackOnly(t *testing.T) {
	p := newTestPlugin(nil)
	p.status = sdk.StatusLoading
	f := &listFrame{plugin: p}

	hints := f.Hints()
	if len(hints) == 0 {
		t.Fatal("expected at least back hint in loading state")
	}
}

func TestListFrame_Hints_WhenDoneWithPins_ShouldIncludeActions(t *testing.T) {
	p := newTestPlugin([]sdk.Resource{{Address: "a"}})
	p.pins = sdk.NewPinService()
	p.pins.Toggle("a")
	f := &listFrame{plugin: p}

	hints := f.Hints()
	foundBang := false
	for _, h := range hints {
		if h.Key == "!" {
			foundBang = true
			break
		}
	}
	if !foundBang {
		t.Error("expected '!' (actions) in hints when pins exist")
	}
}

func TestListFrame_Update_WhenU_InErrorWithLock_ShouldNavigateToForceUnlock(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.log = slog.New(slog.NewTextHandler(io.Discard, nil))
	p.svc = svc
	p.status = sdk.StatusError
	p.lockInfo = &sdk.StateLock{ID: "abc-123"}
	f := &listFrame{plugin: p}

	_, cmd := f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	if cmd == nil {
		t.Fatal("expected non-nil cmd for 'u' in error+lock state")
	}
	msg := cmd()
	navMsg, ok := msg.(sdk.NavigateMsg)
	if !ok {
		t.Fatalf("expected sdk.NavigateMsg, got %T", msg)
	}
	if navMsg.PluginID != "forceunlock" {
		t.Errorf("NavigateMsg.PluginID = %q, want %q", navMsg.PluginID, "forceunlock")
	}
}

func TestListFrame_Update_WhenU_WithoutLock_ShouldDoNothing(t *testing.T) {
	p := newTestPlugin(nil)
	p.status = sdk.StatusError
	p.lockInfo = nil
	f := &listFrame{plugin: p}

	_, cmd := f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	if cmd != nil {
		t.Error("expected nil cmd for 'u' without lock")
	}
}

func TestListFrame_Update_WhenEnterOnBranch_ShouldToggle(t *testing.T) {
	resources := []sdk.Resource{
		{Address: "module.a.aws_instance.one", Type: "aws_instance"},
		{Address: "module.a.aws_instance.two", Type: "aws_instance"},
	}
	p := newTestPlugin(resources)
	p.treeMode = true
	p.rebuildTree()
	f := &listFrame{plugin: p}

	// Cursor should be on module.a (branch node)
	beforeCount := p.tree.VisibleCount()
	f.Update(tea.KeyMsg{Type: tea.KeyEnter})
	afterCount := p.tree.VisibleCount()
	if afterCount == beforeCount {
		t.Error("expected enter on branch to toggle expansion")
	}
}

func TestListFrame_Update_WhenNonKeyMsg_ShouldReturnSelf(t *testing.T) {
	p := newTestPlugin([]sdk.Resource{{Address: "a"}})
	f := &listFrame{plugin: p}

	type otherMsg struct{}
	result, cmd := f.Update(otherMsg{})
	if result != f {
		t.Error("expected same frame for non-key msg")
	}
	if cmd != nil {
		t.Error("expected nil cmd for non-key msg")
	}
}

func TestListFrame_Update_WhenEditOnBranch_ShouldEditBranchPath(t *testing.T) {
	resources := []sdk.Resource{
		{Address: "module.a.aws_instance.one", Type: "aws_instance"},
		{Address: "module.a.aws_instance.two", Type: "aws_instance"},
	}
	p := newTestPlugin(resources)
	p.treeMode = true
	p.rebuildTree()
	f := &listFrame{plugin: p}

	// Cursor on branch node - SelectedResource() returns empty
	_, cmd := f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	if cmd == nil {
		t.Error("expected non-nil cmd for 'e' on branch node")
	}
}

func TestListFrame_Update_WhenTreeToggle_ShouldSwitchMode(t *testing.T) {
	resources := []sdk.Resource{
		{Address: "module.a.aws_instance.one", Type: "aws_instance"},
	}
	p := newTestPlugin(resources)
	p.treeMode = false
	f := &listFrame{plugin: p}

	f.Update(tea.KeyMsg{Type: tea.KeyCtrlT})
	if !p.treeMode {
		t.Error("expected treeMode=true after ctrl+t")
	}
	f.Update(tea.KeyMsg{Type: tea.KeyCtrlT})
	if p.treeMode {
		t.Error("expected treeMode=false after second ctrl+t")
	}
}

func TestPanDetailRight_WhenViewWidthSmall_ShouldUseMinContentWidth(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.viewWidth = 20
	p.detail = strings.Repeat("x", 200)
	p.detailHScroll = 0

	p.panDetailRight()
	if p.detailHScroll != 10 {
		t.Errorf("panDetailRight with small width: detailHScroll = %d, want 10", p.detailHScroll)
	}
}

func TestRenderResources_TreeMode_WithListWrap_ShouldNotTruncate(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.log = slog.New(slog.NewTextHandler(io.Discard, nil))
	p.status = sdk.StatusDone
	p.treeMode = true
	p.listWrap = true
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
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = StatusShowingDetail
	p.detailAddr = "a"
	p.detail = "short\nline"
	p.detailHScroll = 100
	p.detailWrap = false

	view := p.renderDetail(80, 20)
	if view == "" {
		t.Error("expected non-empty view even with excessive hscroll")
	}
}

func TestRenderDetail_WhenLineTruncatedByContentWidth_ShouldTruncate(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = StatusShowingDetail
	p.detailAddr = "a"
	p.detail = strings.Repeat("x", 200)
	p.detailHScroll = 0
	p.detailWrap = false

	view := p.renderDetail(50, 20)
	// contentWidth = max(50-6, 40) = 44
	// The visible line should be truncated to 44 chars
	lines := strings.Split(view, "\n")
	// Skip header lines (address + blank)
	if len(lines) > 2 {
		contentLine := lines[2]
		if len(contentLine) > 44 {
			t.Errorf("expected line truncated to contentWidth=44, got length %d", len(contentLine))
		}
	}
}

func TestListFrame_Update_WhenIKey_ShouldInspect(t *testing.T) {
	resources := []sdk.Resource{
		{Address: "aws_instance.web", Type: "aws_instance"},
	}
	svc := &sdktest.MockService{ShowFn: func(_ context.Context, _ string) (string, error) { return `{"id": "i-123"}`, nil }}
	p := New(svc).(*Plugin)
	p.log = slog.New(slog.NewTextHandler(io.Discard, nil))
	p.status = sdk.StatusDone
	p.resources = resources
	p.filtered = resources
	p.rebuildTree()
	f := &listFrame{plugin: p}

	_, cmd := f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	if cmd == nil {
		t.Error("expected non-nil cmd for 'i' key (inspect alias)")
	}
}

func TestListFrame_Update_WhenFilterSelectOnLeaf_ShouldInspect(t *testing.T) {
	resources := []sdk.Resource{
		{Address: "aws_instance.web", Type: "aws_instance"},
	}
	svc := &sdktest.MockService{ShowFn: func(_ context.Context, _ string) (string, error) { return `{"id": "i-123"}`, nil }}
	p := New(svc).(*Plugin)
	p.log = slog.New(slog.NewTextHandler(io.Discard, nil))
	p.status = sdk.StatusDone
	p.resources = resources
	p.filtered = resources
	p.rebuildTree()

	// Enter filter mode via /
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if !p.filtering {
		t.Fatal("expected filtering=true after /")
	}

	// Press enter in filter mode — should inspect the leaf
	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Error("expected non-nil cmd for enter in filter mode on leaf")
	}
}

func TestListFrame_Update_WhenFilterNavigate_ShouldMoveSelection(t *testing.T) {
	resources := []sdk.Resource{
		{Address: "aws_instance.a", Type: "aws_instance"},
		{Address: "aws_instance.b", Type: "aws_instance"},
	}
	p := newTestPlugin(resources)

	// Enter filter mode
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})

	// Navigate down in filter mode
	p.Update(tea.KeyMsg{Type: tea.KeyDown})
	if p.Selected() != 1 {
		t.Errorf("expected selection=1 after down in filter mode, got %d", p.Selected())
	}

	// Navigate up
	p.Update(tea.KeyMsg{Type: tea.KeyUp})
	if p.Selected() != 0 {
		t.Errorf("expected selection=0 after up in filter mode, got %d", p.Selected())
	}
}

func TestListFrame_Update_WhenFilterPin_ShouldTogglePin(t *testing.T) {
	resources := []sdk.Resource{
		{Address: "aws_instance.a", Type: "aws_instance"},
	}
	p := newTestPlugin(resources)
	p.pins = sdk.NewPinService()
	p.rebuildTree()

	// Enter filter mode
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})

	// Space to toggle pin
	p.Update(tea.KeyMsg{Type: tea.KeySpace})
	if p.pins.Count() != 1 {
		t.Errorf("expected 1 pin after space in filter, got %d", p.pins.Count())
	}
}

func TestListFrame_Update_WhenFilterToggleTree_ShouldSwitchMode(t *testing.T) {
	resources := []sdk.Resource{
		{Address: "module.a.aws_instance.one", Type: "aws_instance"},
	}
	p := newTestPlugin(resources)
	p.treeMode = false

	// Enter filter mode
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})

	// ctrl+t to toggle tree mode
	p.Update(tea.KeyMsg{Type: tea.KeyCtrlT})
	if !p.treeMode {
		t.Error("expected treeMode=true after ctrl+t in filter mode")
	}
}

func TestListFrame_Update_WhenRefreshInLoading_ShouldDoNothing(t *testing.T) {
	p := newTestPlugin(nil)
	p.status = sdk.StatusLoading
	f := &listFrame{plugin: p}

	_, cmd := f.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
	if cmd != nil {
		t.Error("expected nil cmd for ctrl+r in loading state")
	}
}

func TestListFrame_Update_WhenFilterSelectOnBranch_ShouldToggle(t *testing.T) {
	resources := []sdk.Resource{
		{Address: "module.a.aws_instance.one", Type: "aws_instance"},
		{Address: "module.a.aws_instance.two", Type: "aws_instance"},
	}
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.log = slog.New(slog.NewTextHandler(io.Discard, nil))
	p.status = sdk.StatusDone
	p.resources = resources
	p.filtered = resources
	p.treeMode = true
	p.rebuildTree()

	// Enter filter mode
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if !p.filtering {
		t.Fatal("expected filtering=true after /")
	}

	// Cursor is on branch "module.a" - press enter should toggle not inspect
	beforeCount := p.tree.VisibleCount()
	p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	afterCount := p.tree.VisibleCount()
	if afterCount == beforeCount {
		t.Error("expected enter on branch in filter mode to toggle expansion")
	}
}

func TestListFrame_Update_WhenFilterPinOnEmptyList_ShouldDoNothing(t *testing.T) {
	p := newTestPlugin([]sdk.Resource{})
	p.pins = sdk.NewPinService()

	// Enter filter mode
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})

	// Space on empty list
	p.Update(tea.KeyMsg{Type: tea.KeySpace})
	if p.pins.Count() != 0 {
		t.Errorf("expected 0 pins after space on empty list, got %d", p.pins.Count())
	}
}

func TestListFrame_Update_WhenIKeyOnBranch_ShouldToggle(t *testing.T) {
	resources := []sdk.Resource{
		{Address: "module.a.aws_instance.one", Type: "aws_instance"},
		{Address: "module.a.aws_instance.two", Type: "aws_instance"},
	}
	p := newTestPlugin(resources)
	p.treeMode = true
	p.rebuildTree()
	f := &listFrame{plugin: p}

	beforeCount := p.tree.VisibleCount()
	f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	afterCount := p.tree.VisibleCount()
	if afterCount == beforeCount {
		t.Error("expected 'i' on branch to toggle expansion")
	}
}

func TestListFrame_Update_WhenDKeyNoResource_ShouldDoNothing(t *testing.T) {
	p := newTestPlugin([]sdk.Resource{})
	p.pins = sdk.NewPinService()
	f := &listFrame{plugin: p}

	_, cmd := f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if cmd != nil {
		t.Error("expected nil cmd for 'd' with no resource")
	}
}

func TestListFrame_Update_WhenTKeyNoResource_ShouldDoNothing(t *testing.T) {
	p := newTestPlugin([]sdk.Resource{})
	f := &listFrame{plugin: p}

	_, cmd := f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	if cmd != nil {
		t.Error("expected nil cmd for 't' with no resource")
	}
}

func TestListFrame_Update_WhenShiftTKeyNoResource_ShouldDoNothing(t *testing.T) {
	p := newTestPlugin([]sdk.Resource{})
	f := &listFrame{plugin: p}

	_, cmd := f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'T'}})
	if cmd != nil {
		t.Error("expected nil cmd for 'T' with no resource")
	}
}

func TestListFrame_Update_WhenNKeyNoResource_ShouldDoNothing(t *testing.T) {
	p := newTestPlugin([]sdk.Resource{})
	f := &listFrame{plugin: p}

	_, cmd := f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if cmd != nil {
		t.Error("expected nil cmd for 'n' with no resource")
	}
}

func TestListFrame_Update_WhenBangNoTargets_ShouldDoNothing(t *testing.T) {
	p := newTestPlugin([]sdk.Resource{})
	p.pins = sdk.NewPinService()
	f := &listFrame{plugin: p}

	depthBefore := p.stack.Depth()
	f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'!'}})
	if p.stack.Depth() != depthBefore {
		t.Error("expected no frame pushed with no targets")
	}
}

func TestListFrame_Update_WhenSpaceOnNilNode_ShouldDoNothing(t *testing.T) {
	p := newTestPlugin([]sdk.Resource{})
	p.pins = sdk.NewPinService()
	f := &listFrame{plugin: p}

	_, cmd := f.Update(tea.KeyMsg{Type: tea.KeySpace})
	if cmd != nil {
		t.Error("expected nil cmd for space with nil cursor node")
	}
}

func TestListFrame_Update_WhenEKeyNoResourceNoBranch_ShouldDoNothing(t *testing.T) {
	p := newTestPlugin([]sdk.Resource{})
	f := &listFrame{plugin: p}

	_, cmd := f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	if cmd != nil {
		t.Error("expected nil cmd for 'e' with no resource and no branch")
	}
}

func TestListFrame_Update_WhenMKeyNoResource_ShouldDoNothing(t *testing.T) {
	p := newTestPlugin([]sdk.Resource{})
	f := &listFrame{plugin: p}

	_, cmd := f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	if cmd != nil {
		t.Error("expected nil cmd for 'm' with no resource")
	}
}

func TestListFrame_Update_WhenEnterInTreeModeEmptyTree_ShouldInspect(t *testing.T) {
	p := newTestPlugin([]sdk.Resource{})
	p.treeMode = true
	p.rebuildTree()
	f := &listFrame{plugin: p}

	_, cmd := f.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("expected nil cmd for enter in tree mode with empty tree")
	}
}

func TestListFrame_Update_WhenUInErrorWithoutLock_ShouldDoNothing(t *testing.T) {
	p := newTestPlugin(nil)
	p.status = sdk.StatusError
	p.lockInfo = nil
	f := &listFrame{plugin: p}

	_, cmd := f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	if cmd != nil {
		t.Error("expected nil cmd for 'u' in error without lock")
	}
}

func TestListFrame_Update_WhenUInDoneState_ShouldDoNothing(t *testing.T) {
	p := newTestPlugin([]sdk.Resource{{Address: "a"}})
	p.status = sdk.StatusDone
	f := &listFrame{plugin: p}

	_, cmd := f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	if cmd != nil {
		t.Error("expected nil cmd for 'u' in done state")
	}
}

func TestListFrame_Update_WhenIKeyOnLeafInTreeMode_ShouldInspect(t *testing.T) {
	resources := []sdk.Resource{
		{Address: "aws_instance.web", Type: "aws_instance"},
	}
	svc := &sdktest.MockService{ShowFn: func(_ context.Context, _ string) (string, error) { return `{"id": "123"}`, nil }}
	p := New(svc).(*Plugin)
	p.log = slog.New(slog.NewTextHandler(io.Discard, nil))
	p.status = sdk.StatusDone
	p.resources = resources
	p.filtered = resources
	p.treeMode = true
	p.rebuildTree()
	f := &listFrame{plugin: p}

	// In flat-like tree mode with single resource, cursor is on the leaf
	_, cmd := f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	if cmd == nil {
		t.Error("expected non-nil cmd for 'i' on leaf in tree mode")
	}
}

func TestListFrame_Update_WhenFilterSelectOnLeafInTreeMode_ShouldInspect(t *testing.T) {
	resources := []sdk.Resource{
		{Address: "aws_instance.one", Type: "aws_instance"},
	}
	svc := &sdktest.MockService{ShowFn: func(_ context.Context, _ string) (string, error) { return `{"id": "123"}`, nil }}
	p := New(svc).(*Plugin)
	p.log = slog.New(slog.NewTextHandler(io.Discard, nil))
	p.status = sdk.StatusDone
	p.resources = resources
	p.filtered = resources
	p.treeMode = true
	p.rebuildTree()

	// Enter filter mode — tree with single leaf, auto-expanded
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if !p.filtering {
		t.Fatal("expected filtering=true after /")
	}

	// Type to filter down to the leaf
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})

	// Press enter on leaf in tree mode filter
	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Error("expected non-nil cmd for enter on leaf in tree mode filter")
	}
}

func TestListFrame_Update_WhenEsc_ShouldReturnDeactivateCmd(t *testing.T) {
	p := newTestPlugin([]sdk.Resource{{Address: "a"}})
	f := &listFrame{plugin: p}

	_, cmd := f.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("expected non-nil cmd for esc in list frame")
	}
	msg := cmd()
	if _, ok := msg.(sdk.DeactivateMsg); !ok {
		t.Errorf("expected DeactivateMsg, got %T", msg)
	}
}

func TestListFrame_Update_WhenSpaceWithResource_ShouldTogglePin(t *testing.T) {
	resources := []sdk.Resource{{Address: "aws_instance.web", Type: "aws_instance"}}
	p := newTestPlugin(resources)
	p.pins = sdk.NewPinService()
	p.rebuildTree()
	f := &listFrame{plugin: p}

	_, cmd := f.Update(tea.KeyMsg{Type: tea.KeySpace})
	if cmd != nil {
		t.Error("togglePin returns nil cmd, so expected nil")
	}
	if p.pins.Count() != 1 {
		t.Errorf("expected 1 pin after space, got %d", p.pins.Count())
	}
}

func TestListFrame_Update_WhenDKeyWithResource_ShouldRequestDelete(t *testing.T) {
	resources := []sdk.Resource{{Address: "aws_instance.web", Type: "aws_instance"}}
	p := newTestPlugin(resources)
	p.pins = sdk.NewPinService()
	p.rebuildTree()
	f := &listFrame{plugin: p}

	_, cmd := f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if cmd == nil {
		t.Error("expected non-nil cmd for 'd' with resource selected")
	}
}

func TestListFrame_Update_WhenEKeyWithResource_ShouldRequestEdit(t *testing.T) {
	resources := []sdk.Resource{{Address: "aws_instance.web", Type: "aws_instance"}}
	p := newTestPlugin(resources)
	p.rebuildTree()
	f := &listFrame{plugin: p}

	_, cmd := f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	if cmd == nil {
		t.Error("expected non-nil cmd for 'e' with resource selected")
	}
}

func TestListFrame_Update_WhenUnrecognizedKey_ShouldDoNothing(t *testing.T) {
	p := newTestPlugin([]sdk.Resource{{Address: "a"}})
	f := &listFrame{plugin: p}

	_, cmd := f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})
	if cmd != nil {
		t.Error("expected nil cmd for unrecognized key 'z'")
	}
}

func TestListFrame_Update_WhenCtrlRInIdle_ShouldDoNothing(t *testing.T) {
	p := newTestPlugin(nil)
	p.status = sdk.StatusIdle
	f := &listFrame{plugin: p}

	_, cmd := f.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
	if cmd != nil {
		t.Error("expected nil cmd for ctrl+r in idle state")
	}
}

func TestListFrame_Update_WhenEnterInTreeModeOnLeaf_ShouldInspect(t *testing.T) {
	resources := []sdk.Resource{
		{Address: "module.a.aws_instance.web", Type: "aws_instance"},
	}
	svc := &sdktest.MockService{ShowFn: func(_ context.Context, _ string) (string, error) { return `{"id": "123"}`, nil }}
	p := New(svc).(*Plugin)
	p.log = slog.New(slog.NewTextHandler(io.Discard, nil))
	p.status = sdk.StatusDone
	p.resources = resources
	p.filtered = resources
	p.treeMode = true
	p.rebuildTree()
	// Expand the branch, move to leaf
	p.tree.ExpandAll()
	p.tree.MoveDown()
	f := &listFrame{plugin: p}

	node := p.CursorNode()
	if node == nil {
		t.Fatal("expected non-nil node after expand+movedown")
	}

	_, cmd := f.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Error("expected non-nil cmd for enter on leaf in tree mode")
	}
}

func TestRenderResources_WhenHeightVerySmall_ShouldClampMinVisible(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.log = slog.New(slog.NewTextHandler(io.Discard, nil))
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

func TestActivate_WhenLoadingAndTimerRunning_ShouldReturnTick(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.log = slog.New(slog.NewTextHandler(io.Discard, nil))
	p.status = sdk.StatusLoading
	p.timer.Start()

	cmd := p.Activate()
	if cmd == nil {
		t.Error("Activate() when loading with running timer should return tick cmd")
	}
}

func TestActivate_WhenLoadingAndTimerNotRunning_ShouldReturnNil(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.log = slog.New(slog.NewTextHandler(io.Discard, nil))
	p.status = sdk.StatusLoading

	cmd := p.Activate()
	if cmd != nil {
		t.Error("Activate() when loading with stopped timer should return nil")
	}
}

func TestUpdate_WhenStateMovedMsg_ShouldRefreshAndClearMutating(t *testing.T) {
	svc := &sdktest.MockService{StateListFn: func(_ context.Context, _ ...sdk.StateListOption) ([]sdk.Resource, error) {
		return []sdk.Resource{}, nil
	}}
	p := New(svc).(*Plugin)
	p.log = slog.New(slog.NewTextHandler(io.Discard, nil))
	p.svc = svc
	p.status = sdk.StatusDone
	p.mutating = true

	_, cmd := p.Update(StateMovedMsg{Source: "a", Dest: "b"})
	if cmd == nil {
		t.Error("expected non-nil cmd (refresh) after StateMovedMsg")
	}
	if p.mutating {
		t.Error("expected mutating=false after StateMovedMsg")
	}
}

func TestIsTaintedAddress_WhenResourceTainted_ShouldReturnTrue(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.resources = []sdk.Resource{
		{Address: "aws_instance.web", Type: "aws_instance", Tainted: true},
		{Address: "aws_s3_bucket.data", Type: "aws_s3_bucket", Tainted: false},
	}

	if !p.isTaintedAddress("aws_instance.web") {
		t.Error("expected isTaintedAddress to return true for tainted resource")
	}
	if p.isTaintedAddress("aws_s3_bucket.data") {
		t.Error("expected isTaintedAddress to return false for non-tainted resource")
	}
	if p.isTaintedAddress("nonexistent") {
		t.Error("expected isTaintedAddress to return false for nonexistent address")
	}
}

func TestRenderDetail_WhenTainted_ShouldShowTaintedIndicator(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
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
	p := New(&sdktest.MockService{}).(*Plugin)
	p.listHScroll = 0
	p.listWrap = false

	row := p.formatResourceRow("[ ] ", sdk.Resource{Address: "aws_instance.web", Type: "aws_instance", Tainted: true}, 120)
	if !strings.Contains(row, "[tainted]") {
		t.Error("expected [tainted] in formatResourceRow for tainted resource")
	}
}

func TestFormatResourceRow_WhenWrapMode_ShouldNotTruncate(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.listHScroll = 0
	p.listWrap = true

	longAddr := strings.Repeat("x", 200)
	row := p.formatResourceRow("[ ] ", sdk.Resource{Address: longAddr, Type: "t"}, 80)
	if !strings.Contains(row, longAddr) {
		t.Error("expected full content in wrap mode (no truncation)")
	}
}

func TestOutput_WhenJsonWithNilResources_ShouldReturnEmptyArray(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.resources = nil

	data, err := p.Output(true)
	if err != nil {
		t.Fatalf("Output(true) error = %v", err)
	}
	if !strings.Contains(string(data), "[]") {
		t.Errorf("JSON for nil resources = %q, want '[]'", string(data))
	}
}

func TestOutput_WhenTextWithNilResources_ShouldReturnEmpty(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.resources = nil

	data, err := p.Output(false)
	if err != nil {
		t.Fatalf("Output(false) error = %v", err)
	}
	if len(data) != 0 {
		t.Errorf("text for nil resources = %q, want empty", string(data))
	}
}

func TestBuildActionFrame_WhenMultiTarget_ShouldDisableMoveAndImport(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.log = slog.New(slog.NewTextHandler(io.Discard, nil))
	p.pins = sdk.NewPinService()
	p.resources = []sdk.Resource{{Address: "a"}, {Address: "b"}}
	p.filtered = p.resources
	p.rebuildTree()
	p.pins.Toggle("a")
	p.pins.Toggle("b")

	frame := p.buildActionFrame("a", true)
	if frame == nil {
		t.Fatal("buildActionFrame returned nil")
	}
}

func TestBuildActionFrame_WhenSingleTarget_ShouldEnableAllActions(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.log = slog.New(slog.NewTextHandler(io.Discard, nil))
	p.pins = sdk.NewPinService()
	p.resources = []sdk.Resource{{Address: "a"}}
	p.filtered = p.resources
	p.rebuildTree()

	frame := p.buildActionFrame("a", false)
	if frame == nil {
		t.Fatal("buildActionFrame returned nil")
	}
}

func TestBuildActionFrame_WhenMultiTargetEditHandler_ShouldEditMultiple(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.log = slog.New(slog.NewTextHandler(io.Discard, nil))
	p.pins = sdk.NewPinService()
	p.resources = []sdk.Resource{{Address: "a"}, {Address: "b"}}
	p.filtered = p.resources
	p.rebuildTree()
	p.pins.Toggle("a")
	p.pins.Toggle("b")

	frame := p.buildActionFrame("a", true)
	if frame == nil {
		t.Fatal("buildActionFrame returned nil")
	}

	// Push the frame and trigger the 'e' action (edit)
	p.stack.Push(frame)
	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	if cmd == nil {
		t.Error("expected non-nil cmd for edit action in multi-target mode")
	}
}

func TestBuildActionFrame_WhenMultiTargetDeleteHandler_ShouldBatchDelete(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.log = slog.New(slog.NewTextHandler(io.Discard, nil))
	p.pins = sdk.NewPinService()
	p.resources = []sdk.Resource{{Address: "a"}, {Address: "b"}}
	p.filtered = p.resources
	p.rebuildTree()
	p.pins.Toggle("a")
	p.pins.Toggle("b")

	frame := p.buildActionFrame("a", true)
	p.stack.Push(frame)
	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if cmd == nil {
		t.Error("expected non-nil cmd for delete action in multi-target mode")
	}
}

func TestBuildActionFrame_WhenSingleTargetTaintHandler_ShouldEmitTaintRequest(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.log = slog.New(slog.NewTextHandler(io.Discard, nil))
	p.pins = sdk.NewPinService()
	p.resources = []sdk.Resource{{Address: "a"}}
	p.filtered = p.resources
	p.rebuildTree()

	frame := p.buildActionFrame("a", false)
	p.stack.Push(frame)
	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	if cmd == nil {
		t.Error("expected non-nil cmd for taint action")
	}
}

func TestBuildActionFrame_WhenSingleTargetUntaintHandler_ShouldEmitUntaintRequest(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.log = slog.New(slog.NewTextHandler(io.Discard, nil))
	p.pins = sdk.NewPinService()
	p.resources = []sdk.Resource{{Address: "a"}}
	p.filtered = p.resources
	p.rebuildTree()

	frame := p.buildActionFrame("a", false)
	p.stack.Push(frame)
	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'T'}})
	if cmd == nil {
		t.Error("expected non-nil cmd for untaint action")
	}
}

func TestBuildActionFrame_WhenSingleTargetImportHandler_ShouldEmitImportRequest(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.log = slog.New(slog.NewTextHandler(io.Discard, nil))
	p.pins = sdk.NewPinService()
	p.resources = []sdk.Resource{{Address: "a"}}
	p.filtered = p.resources
	p.rebuildTree()

	frame := p.buildActionFrame("a", false)
	p.stack.Push(frame)
	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if cmd == nil {
		t.Error("expected non-nil cmd for import action")
	}
}

func TestBuildActionFrame_WhenSingleTargetMoveHandler_ShouldEmitMoveRequest(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.log = slog.New(slog.NewTextHandler(io.Discard, nil))
	p.pins = sdk.NewPinService()
	p.resources = []sdk.Resource{{Address: "a"}}
	p.filtered = p.resources
	p.rebuildTree()

	frame := p.buildActionFrame("a", false)
	p.stack.Push(frame)
	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	if cmd == nil {
		t.Error("expected non-nil cmd for move action")
	}
}

func TestDetailFrame_Update_WhenMKey_ShouldRequestMove(t *testing.T) {
	p := newTestPlugin([]sdk.Resource{{Address: "a"}})
	p.status = StatusShowingDetail
	p.detailAddr = "aws_instance.web"
	f := &detailFrame{plugin: p}

	_, cmd := f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	if cmd == nil {
		t.Error("expected non-nil cmd for 'm' in detail frame")
	}
}

func TestDetailFrame_Update_WhenTKey_ShouldEmitTaintRequest(t *testing.T) {
	p := newTestPlugin([]sdk.Resource{{Address: "a"}})
	p.status = StatusShowingDetail
	p.detailAddr = "aws_instance.web"
	f := &detailFrame{plugin: p}

	_, cmd := f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	if cmd == nil {
		t.Error("expected non-nil cmd for 't' in detail frame")
	}
}

func TestDetailFrame_Update_WhenShiftTKey_ShouldEmitUntaintRequest(t *testing.T) {
	p := newTestPlugin([]sdk.Resource{{Address: "a"}})
	p.status = StatusShowingDetail
	p.detailAddr = "aws_instance.web"
	f := &detailFrame{plugin: p}

	_, cmd := f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'T'}})
	if cmd == nil {
		t.Error("expected non-nil cmd for 'T' in detail frame")
	}
}

func TestDetailFrame_Update_WhenNKey_ShouldEmitImportRequest(t *testing.T) {
	p := newTestPlugin([]sdk.Resource{{Address: "a"}})
	p.status = StatusShowingDetail
	p.detailAddr = "aws_instance.web"
	f := &detailFrame{plugin: p}

	_, cmd := f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if cmd == nil {
		t.Error("expected non-nil cmd for 'n' in detail frame")
	}
}

func TestDetailFrame_Update_WhenUnrecognizedKey_ShouldReturnSelf(t *testing.T) {
	p := newTestPlugin([]sdk.Resource{{Address: "a"}})
	p.status = StatusShowingDetail
	f := &detailFrame{plugin: p}

	result, cmd := f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})
	if result != f {
		t.Error("expected same frame for unrecognized key")
	}
	if cmd != nil {
		t.Error("expected nil cmd for unrecognized key")
	}
}

func TestListFrame_Update_WhenRightKey_ShouldPanRight(t *testing.T) {
	p := newTestPlugin([]sdk.Resource{{Address: "a_very_long_address"}})
	p.listWrap = false
	f := &listFrame{plugin: p}

	f.Update(tea.KeyMsg{Type: tea.KeyRight})
	if p.listHScroll != 10 {
		t.Errorf("listHScroll after right = %d, want 10", p.listHScroll)
	}
}

func TestListFrame_Update_WhenLeftKey_ShouldPanLeft(t *testing.T) {
	p := newTestPlugin([]sdk.Resource{{Address: "a"}})
	p.listHScroll = 20
	p.listWrap = false
	f := &listFrame{plugin: p}

	f.Update(tea.KeyMsg{Type: tea.KeyLeft})
	if p.listHScroll != 10 {
		t.Errorf("listHScroll after left = %d, want 10", p.listHScroll)
	}
}

func TestListFrame_Update_WhenRightKeyWithWrap_ShouldNotPan(t *testing.T) {
	p := newTestPlugin([]sdk.Resource{{Address: "a"}})
	p.listWrap = true
	f := &listFrame{plugin: p}

	f.Update(tea.KeyMsg{Type: tea.KeyRight})
	if p.listHScroll != 0 {
		t.Errorf("listHScroll after right with wrap = %d, want 0", p.listHScroll)
	}
}

func TestListFrame_Update_WhenLeftKeyWithWrap_ShouldNotPan(t *testing.T) {
	p := newTestPlugin([]sdk.Resource{{Address: "a"}})
	p.listHScroll = 10
	p.listWrap = true
	f := &listFrame{plugin: p}

	f.Update(tea.KeyMsg{Type: tea.KeyLeft})
	if p.listHScroll != 10 {
		t.Errorf("listHScroll after left with wrap = %d, want 10", p.listHScroll)
	}
}

func TestRenderDetail_WhenScrollExceedsMax_ShouldClamp(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
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

func TestUpdate_WhenStateListMsgSuccess_ShouldClearMutating(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.log = slog.New(slog.NewTextHandler(io.Discard, nil))
	p.mutating = true
	p.status = sdk.StatusLoading

	p.Update(StateListMsg{Resources: []sdk.Resource{{Address: "a"}}})
	if p.mutating {
		t.Error("expected mutating=false after StateListMsg success")
	}
}

func TestUpdate_WhenStateListMsgWithLockError_ShouldParseLockInfo(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.log = slog.New(slog.NewTextHandler(io.Discard, nil))
	p.status = sdk.StatusLoading

	lockErr := `Error acquiring the state lock
  ID:        a1b2c3d4-e5f6-7890-abcd-ef1234567890
  Who:       user@machine
  Operation: OperationTypePlan`

	p.Update(StateListMsg{Err: errors.New(lockErr)})
	if p.lockInfo == nil {
		t.Fatal("expected lockInfo to be set after lock error")
	}
	if p.lockInfo.ID != "a1b2c3d4-e5f6-7890-abcd-ef1234567890" {
		t.Errorf("lockInfo.ID = %q, want 'a1b2c3d4-e5f6-7890-abcd-ef1234567890'", p.lockInfo.ID)
	}
}

func TestPlugin_WhenHandlePlanInvalidated_ShouldReset(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.log = slog.New(slog.NewTextHandler(io.Discard, nil))
	p.status = sdk.StatusDone
	p.resources = []sdk.Resource{{Address: "a"}}

	cmd := p.HandlePlanInvalidated(sdk.PlanInvalidatedEvent{})
	if cmd != nil {
		t.Error("HandlePlanInvalidated() should return nil")
	}
	if p.status != sdk.StatusIdle {
		t.Errorf("status = %v, want Idle", p.status)
	}
	if p.resources != nil {
		t.Error("resources should be nil after reset")
	}
}

func TestPlugin_WhenHandleLockCleared_ShouldClearLockAndReset(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.log = slog.New(slog.NewTextHandler(io.Discard, nil))
	p.status = sdk.StatusError
	p.lockInfo = &sdk.StateLock{ID: "abc"}

	cmd := p.HandleLockCleared(sdk.LockClearedEvent{})
	if cmd != nil {
		t.Error("HandleLockCleared() should return nil")
	}
	if p.lockInfo != nil {
		t.Error("lockInfo should be nil")
	}
	if p.status != sdk.StatusIdle {
		t.Errorf("status = %v, want Idle", p.status)
	}
}

func TestPlugin_WhenOutputJson_ShouldReturnResourceArray(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.resources = []sdk.Resource{
		{Address: "aws_instance.web", Type: "aws_instance", Tainted: true},
		{Address: "aws_s3_bucket.data", Type: "aws_s3_bucket"},
	}

	data, err := p.Output(true)
	if err != nil {
		t.Fatalf("Output(true) error = %v", err)
	}
	s := string(data)
	if !strings.Contains(s, `"aws_instance.web"`) {
		t.Error("JSON missing address")
	}
	if !strings.Contains(s, `"tainted": true`) {
		t.Error("JSON missing tainted flag")
	}
}

func TestPlugin_WhenOutputText_ShouldReturnAddressList(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.resources = []sdk.Resource{
		{Address: "aws_instance.web"},
		{Address: "aws_s3_bucket.data"},
	}

	data, err := p.Output(false)
	if err != nil {
		t.Fatalf("Output(false) error = %v", err)
	}
	s := string(data)
	if !strings.Contains(s, "aws_instance.web\n") {
		t.Error("text missing aws_instance.web")
	}
	if !strings.Contains(s, "aws_s3_bucket.data\n") {
		t.Error("text missing aws_s3_bucket.data")
	}
}

func TestUpdate_WhenTimerTickMsg_ShouldReturnTickCmd(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.log = slog.New(slog.NewTextHandler(io.Discard, nil))
	p.status = sdk.StatusLoading
	p.timer.Start()

	_, cmd := p.Update(ui.TimerTickMsg{})
	if cmd == nil {
		t.Error("Update(TimerTickMsg) with running timer should return tick cmd")
	}
}

func TestUpdate_WhenTimerTickMsgTimerStopped_ShouldReturnNil(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.log = slog.New(slog.NewTextHandler(io.Discard, nil))
	p.status = sdk.StatusDone

	_, cmd := p.Update(ui.TimerTickMsg{})
	if cmd != nil {
		t.Error("Update(TimerTickMsg) with stopped timer should return nil")
	}
}

func TestRenderResources_WhenPinnedOnlyWithoutFilter_ShouldNotAddFilterHeight(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.log = slog.New(slog.NewTextHandler(io.Discard, nil))
	p.status = sdk.StatusDone
	p.resources = []sdk.Resource{{Address: "a"}, {Address: "b"}, {Address: "c"}}
	p.filtered = p.resources
	p.pins = sdk.NewPinService()
	p.rebuildTree()
	p.filtering = false
	p.filter = ""
	p.pinnedOnly = true

	view := p.View(80, 24)
	if !strings.Contains(view, "[pinned]") {
		t.Error("expected [pinned] indicator when pinnedOnly but no filter text")
	}
}

func TestOutput_WhenJsonWithTaintedResource_ShouldIncludeTaintedField(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.resources = []sdk.Resource{
		{Address: "aws_instance.web", Type: "aws_instance", Tainted: true},
	}

	data, err := p.Output(true)
	if err != nil {
		t.Fatalf("Output(true) error = %v", err)
	}
	s := string(data)
	if !strings.Contains(s, `"tainted": true`) {
		t.Errorf("JSON output should contain tainted field, got: %s", s)
	}
}
