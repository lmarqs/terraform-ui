package state

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/frames"
	"github.com/lmarqs/terraform-ui/pkg/sdk/sdktest"
)

func newTestPlugin(resources []sdk.Resource) *Plugin {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusDone
	p.resources = resources
	p.filtered = resources
	p.log = slog.New(slog.NewTextHandler(io.Discard, nil))
	p.rebuildTree()
	return p
}

func TestListFrame_ExpandAll(t *testing.T) {
	resources := []sdk.Resource{
		{Address: "module.a.aws_instance.one", Type: "aws_instance"},
		{Address: "module.a.aws_instance.two", Type: "aws_instance"},
		{Address: "module.b.aws_s3_bucket.data", Type: "aws_s3_bucket"},
	}
	p := newTestPlugin(resources)
	p.treeMode = true
	p.rebuildTree()
	f := &listFrame{plugin: p}

	collapsed := p.tree.VisibleCount()
	f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}})
	expanded := p.tree.VisibleCount()

	if expanded <= collapsed {
		t.Errorf("ExpandAll: visible count did not increase (before=%d, after=%d)", collapsed, expanded)
	}
}

func TestListFrame_CollapseAll(t *testing.T) {
	resources := []sdk.Resource{
		{Address: "module.a.aws_instance.one", Type: "aws_instance"},
		{Address: "module.a.aws_instance.two", Type: "aws_instance"},
		{Address: "module.b.aws_s3_bucket.data", Type: "aws_s3_bucket"},
	}
	p := newTestPlugin(resources)
	p.treeMode = true
	p.rebuildTree()
	p.tree.ExpandAll()
	f := &listFrame{plugin: p}

	expanded := p.tree.VisibleCount()
	f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'['}})
	collapsed := p.tree.VisibleCount()

	if collapsed >= expanded {
		t.Errorf("CollapseAll: visible count did not decrease (before=%d, after=%d)", expanded, collapsed)
	}
}

func TestListFrame_ExpandCollapse_FlatMode_NoOp(t *testing.T) {
	resources := []sdk.Resource{
		{Address: "module.a.aws_instance.one", Type: "aws_instance"},
		{Address: "module.b.aws_s3_bucket.data", Type: "aws_s3_bucket"},
	}
	p := newTestPlugin(resources)
	p.treeMode = false
	p.rebuildTree()
	f := &listFrame{plugin: p}

	before := p.tree.VisibleCount()
	f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}})
	after := p.tree.VisibleCount()
	if after != before {
		t.Errorf("] in flat mode should be no-op (before=%d, after=%d)", before, after)
	}

	f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'['}})
	after = p.tree.VisibleCount()
	if after != before {
		t.Errorf("[ in flat mode should be no-op (before=%d, after=%d)", before, after)
	}
}

func TestStateFilterFrame_ExpandAll(t *testing.T) {
	resources := []sdk.Resource{
		{Address: "module.a.aws_instance.one", Type: "aws_instance"},
		{Address: "module.a.aws_instance.two", Type: "aws_instance"},
		{Address: "module.b.aws_s3_bucket.data", Type: "aws_s3_bucket"},
	}
	p := newTestPlugin(resources)
	p.treeMode = true
	p.rebuildTree()

	// Enter filter mode
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if !p.filtering {
		t.Fatal("expected filtering mode after /")
	}

	// Collapse first to verify expand works in filter mode
	p.tree.CollapseAll()
	collapsed := p.tree.VisibleCount()

	// Press ] while in filter mode
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}})
	expanded := p.tree.VisibleCount()

	if expanded <= collapsed {
		t.Errorf("ExpandAll in filter mode: visible count did not increase (before=%d, after=%d)", collapsed, expanded)
	}
}

func TestStateFilterFrame_CollapseAll(t *testing.T) {
	resources := []sdk.Resource{
		{Address: "module.a.module.inner.aws_instance.one", Type: "aws_instance"},
		{Address: "module.a.module.inner.aws_instance.two", Type: "aws_instance"},
		{Address: "module.a.aws_subnet.x", Type: "aws_subnet"},
		{Address: "module.b.aws_s3_bucket.data", Type: "aws_s3_bucket"},
	}
	p := newTestPlugin(resources)
	p.treeMode = true
	p.rebuildTree()
	p.tree.ExpandAll()

	// Enter filter mode — this rebuilds tree and auto-expands since filter=""
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if !p.filtering {
		t.Fatal("expected filtering mode after /")
	}

	// Expand all to ensure fully expanded
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}})
	expanded := p.tree.VisibleCount()

	// Press [ while in filter mode
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'['}})
	collapsed := p.tree.VisibleCount()

	if collapsed >= expanded {
		t.Errorf("CollapseAll in filter mode: visible count did not decrease (before=%d, after=%d)", expanded, collapsed)
	}
}

func TestStateFilterFrame_ExpandCollapse_FlatMode_NoOp(t *testing.T) {
	resources := []sdk.Resource{
		{Address: "module.a.aws_instance.one", Type: "aws_instance"},
		{Address: "module.b.aws_s3_bucket.data", Type: "aws_s3_bucket"},
	}
	p := newTestPlugin(resources)
	p.treeMode = false
	p.rebuildTree()

	// Enter filter mode
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})

	before := p.tree.VisibleCount()
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}})
	after := p.tree.VisibleCount()
	if after != before {
		t.Errorf("] in filter+flat mode should be no-op (before=%d, after=%d)", before, after)
	}

	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'['}})
	after = p.tree.VisibleCount()
	if after != before {
		t.Errorf("[ in filter+flat mode should be no-op (before=%d, after=%d)", before, after)
	}
}

func TestStateFilterFrame_BracketNotAddedToFilter(t *testing.T) {
	resources := []sdk.Resource{
		{Address: "module.a.aws_instance.one", Type: "aws_instance"},
	}
	p := newTestPlugin(resources)
	p.treeMode = true
	p.rebuildTree()

	// Enter filter mode and type some text
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})

	// Press ] — should not appear in filter text
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}})
	if p.filter != "a" {
		t.Errorf("filter = %q, want %q (] should not be appended)", p.filter, "a")
	}

	// Press [ — should not appear in filter text
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'['}})
	if p.filter != "a" {
		t.Errorf("filter = %q, want %q ([ should not be appended)", p.filter, "a")
	}
}

func TestStateFilterFrame_ExpandAfterSearch(t *testing.T) {
	resources := []sdk.Resource{
		{Address: "module.vpc.aws_subnet.a", Type: "aws_subnet"},
		{Address: "module.vpc.aws_subnet.b", Type: "aws_subnet"},
		{Address: "module.eks.aws_eks_cluster.this", Type: "aws_eks_cluster"},
		{Address: "module.rds.aws_db_instance.main", Type: "aws_db_instance"},
	}
	p := newTestPlugin(resources)
	p.treeMode = true
	p.rebuildTree()

	// Enter filter mode, search, then collapse and expand
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})

	// Collapse
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'['}})
	collapsed := p.tree.VisibleCount()

	// Expand
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}})
	expanded := p.tree.VisibleCount()

	if expanded <= collapsed {
		t.Errorf("ExpandAll after search+collapse: visible count did not increase (collapsed=%d, expanded=%d)", collapsed, expanded)
	}
}

func TestStateFilterFrame_Hints_TreeMode(t *testing.T) {
	resources := []sdk.Resource{
		{Address: "module.a.aws_instance.one", Type: "aws_instance"},
	}
	p := newTestPlugin(resources)
	p.treeMode = true
	p.rebuildTree()

	// Enter filter mode
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})

	// Get hints from the top of the stack (filter frame)
	hints := p.stack.Hints()
	foundCollapse := false
	foundExpand := false
	for _, h := range hints {
		if h.Key == "[" {
			foundCollapse = true
		}
		if h.Key == "]" {
			foundExpand = true
		}
	}
	if !foundCollapse || !foundExpand {
		t.Error("filter frame hints should include collapse/expand keys in tree mode")
	}
}

func TestStateFilterFrame_Hints_FlatMode(t *testing.T) {
	resources := []sdk.Resource{
		{Address: "module.a.aws_instance.one", Type: "aws_instance"},
	}
	p := newTestPlugin(resources)
	p.treeMode = false
	p.rebuildTree()

	// Enter filter mode
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})

	hints := p.stack.Hints()
	for _, h := range hints {
		if h.Key == "[" || h.Key == "]" {
			t.Error("filter frame hints should NOT include collapse/expand keys in flat mode")
			break
		}
	}
}

func TestListFrame_Hints_TreeMode(t *testing.T) {
	resources := []sdk.Resource{
		{Address: "module.a.aws_instance.one", Type: "aws_instance"},
	}
	p := newTestPlugin(resources)
	p.treeMode = true
	p.rebuildTree()

	hints := p.stack.Hints()
	foundCollapse := false
	foundExpand := false
	for _, h := range hints {
		if h.Key == "[" {
			foundCollapse = true
		}
		if h.Key == "]" {
			foundExpand = true
		}
	}
	if !foundCollapse || !foundExpand {
		t.Error("list frame hints should include collapse/expand keys in tree mode")
	}
}

func TestFlatMode_SelectionMatchesCursorPosition(t *testing.T) {
	resources := []sdk.Resource{
		{Address: "module.z.aws_instance.web", Type: "aws_instance"},
		{Address: "module.a.aws_s3_bucket.data", Type: "aws_s3_bucket"},
		{Address: "module.m.aws_lambda_function.api", Type: "aws_lambda_function"},
	}
	p := newTestPlugin(resources)
	p.treeMode = false
	p.rebuildTree()

	tests := []struct {
		name     string
		moves    int
		expected string
	}{
		{"ShouldSelectFirstItem", 0, "module.z.aws_instance.web"},
		{"ShouldSelectSecondItem", 1, "module.a.aws_s3_bucket.data"},
		{"ShouldSelectThirdItem", 2, "module.m.aws_lambda_function.api"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p.treeMode = false
			p.rebuildTree()
			for i := 0; i < tt.moves; i++ {
				p.MoveDown()
			}
			r := p.SelectedResource()
			if r.Address != tt.expected {
				t.Errorf("after %d moves, selected %q, want %q", tt.moves, r.Address, tt.expected)
			}
		})
	}
}

func TestFlatMode_SelectionMatchesFuzzyFilterOrder(t *testing.T) {
	resources := []sdk.Resource{
		{Address: "module.vpc.aws_subnet.main", Type: "aws_subnet"},
		{Address: "module.rds.aws_db_instance.primary", Type: "aws_db_instance"},
		{Address: "module.lambda.aws_lambda_function.api", Type: "aws_lambda_function"},
		{Address: "module.s3.aws_s3_bucket.data", Type: "aws_s3_bucket"},
	}
	p := newTestPlugin(resources)
	p.treeMode = false
	p.rebuildTree()

	p.SetFilter("lambda")

	if len(p.filtered) == 0 {
		t.Fatal("expected filter to match at least one resource")
	}

	p.MoveToStart()
	r := p.SelectedResource()
	if r.Address != p.filtered[0].Address {
		t.Errorf("cursor at 0 selected %q, want %q (first filtered result)", r.Address, p.filtered[0].Address)
	}

	if len(p.filtered) > 1 {
		p.MoveDown()
		r = p.SelectedResource()
		if r.Address != p.filtered[1].Address {
			t.Errorf("cursor at 1 selected %q, want %q (second filtered result)", r.Address, p.filtered[1].Address)
		}
	}
}

func TestFlatMode_PinTargetsSelectedResource(t *testing.T) {
	resources := []sdk.Resource{
		{Address: "module.z.aws_instance.web", Type: "aws_instance"},
		{Address: "module.a.aws_s3_bucket.data", Type: "aws_s3_bucket"},
		{Address: "module.m.aws_lambda_function.api", Type: "aws_lambda_function"},
	}
	p := newTestPlugin(resources)
	p.treeMode = false
	p.pins = sdk.NewPinService()
	p.rebuildTree()

	p.MoveDown()
	p.MoveDown()

	node := p.CursorNode()
	if node == nil {
		t.Fatal("expected non-nil cursor node")
	}
	if node.Path != "module.m.aws_lambda_function.api" {
		t.Fatalf("expected cursor at module.m.aws_lambda_function.api, got %q", node.Path)
	}
}

func TestStateFilterFrame_PanRightLeft(t *testing.T) {
	resources := []sdk.Resource{
		{Address: "module.very_long_name.aws_instance.server", Type: "aws_instance"},
	}
	p := newTestPlugin(resources)
	p.rebuildTree()

	// Enter filter mode
	f := &listFrame{plugin: p}
	f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if !p.filtering {
		t.Fatal("expected filtering mode after /")
	}

	t.Run("ShouldPanRightInFilterMode", func(t *testing.T) {
		p.listHScroll = 0
		p.Update(tea.KeyMsg{Type: tea.KeyRight})
		if p.listHScroll != 10 {
			t.Errorf("expected listHScroll=10 after right, got %d", p.listHScroll)
		}
	})

	t.Run("ShouldPanLeftInFilterMode", func(t *testing.T) {
		p.listHScroll = 10
		p.Update(tea.KeyMsg{Type: tea.KeyLeft})
		if p.listHScroll != 0 {
			t.Errorf("expected listHScroll=0 after left, got %d", p.listHScroll)
		}
	})

	t.Run("ShouldNotAddArrowsToFilter", func(t *testing.T) {
		p.listHScroll = 0
		p.Update(tea.KeyMsg{Type: tea.KeyRight})
		p.Update(tea.KeyMsg{Type: tea.KeyLeft})
		if p.filter != "" {
			t.Errorf("expected filter to remain empty, got %q", p.filter)
		}
	})
}

func TestListFrame_WrapToggle(t *testing.T) {
	resources := []sdk.Resource{
		{Address: "aws_instance.short", Type: "aws_instance"},
		{Address: "module.very_long_module_name.module.another_module.aws_instance.server_with_long_name", Type: "aws_instance"},
	}
	p := newTestPlugin(resources)
	p.rebuildTree()
	f := &listFrame{plugin: p}

	t.Run("ShouldToggleWrapOnCtrlW", func(t *testing.T) {
		p.listWrap = false
		f.Update(tea.KeyMsg{Type: tea.KeyCtrlW})
		if !p.listWrap {
			t.Error("expected listWrap=true after ctrl+w")
		}
	})

	t.Run("ShouldResetHScrollOnWrap", func(t *testing.T) {
		p.listHScroll = 20
		p.listWrap = false
		f.Update(tea.KeyMsg{Type: tea.KeyCtrlW})
		if p.listHScroll != 0 {
			t.Errorf("expected listHScroll=0 after wrap toggle, got %d", p.listHScroll)
		}
	})

	t.Run("ShouldNotTruncateWhenWrapped", func(t *testing.T) {
		p.listWrap = true
		output := p.View(40, 10)
		if !strings.Contains(output, "server_with_long_name") {
			t.Error("expected full address visible when wrap is on")
		}
	})

	t.Run("ShouldTruncateWhenNotWrapped", func(t *testing.T) {
		p.listWrap = false
		output := p.View(40, 10)
		if strings.Contains(output, "server_with_long_name") {
			t.Error("expected address to be truncated when wrap is off")
		}
	})

	t.Run("WKeyShouldNotToggleWrap", func(t *testing.T) {
		p.listWrap = false
		f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
		if p.listWrap {
			t.Error("expected 'w' to not toggle wrap in list frame")
		}
	})
}

func TestListFrame_PinnedFilter(t *testing.T) {
	resources := []sdk.Resource{
		{Address: "aws_instance.a", Type: "aws_instance"},
		{Address: "aws_instance.b", Type: "aws_instance"},
		{Address: "aws_instance.c", Type: "aws_instance"},
	}
	p := newTestPlugin(resources)
	p.pins = sdk.NewPinService()
	p.rebuildTree()
	f := &listFrame{plugin: p}

	// Pin one resource
	p.pins.Toggle("aws_instance.b")
	p.syncPinnedToTree()

	t.Run("ShouldFilterToPinnedOnly", func(t *testing.T) {
		f.Update(tea.KeyMsg{Type: tea.KeyCtrlP})
		if !p.pinnedOnly {
			t.Fatal("expected pinnedOnly=true after ctrl+p")
		}
		if len(p.filtered) != 1 {
			t.Errorf("expected 1 filtered resource, got %d", len(p.filtered))
		}
		if p.filtered[0].Address != "aws_instance.b" {
			t.Errorf("expected pinned resource, got %q", p.filtered[0].Address)
		}
	})

	t.Run("ShouldToggleBackToAll", func(t *testing.T) {
		f.Update(tea.KeyMsg{Type: tea.KeyCtrlP})
		if p.pinnedOnly {
			t.Fatal("expected pinnedOnly=false after second ctrl+p")
		}
		if len(p.filtered) != 3 {
			t.Errorf("expected 3 filtered resources, got %d", len(p.filtered))
		}
	})
}

func TestListFrame_ClearAllPins(t *testing.T) {
	resources := []sdk.Resource{
		{Address: "aws_instance.a", Type: "aws_instance"},
		{Address: "aws_instance.b", Type: "aws_instance"},
	}
	p := newTestPlugin(resources)
	p.pins = sdk.NewPinService()
	p.rebuildTree()
	f := &listFrame{plugin: p}

	p.pins.Toggle("aws_instance.a")
	p.pins.Toggle("aws_instance.b")
	p.syncPinnedToTree()

	if p.PinnedCount() != 2 {
		t.Fatalf("expected 2 pinned, got %d", p.PinnedCount())
	}

	f.Update(tea.KeyMsg{Type: tea.KeyCtrlU})

	if p.PinnedCount() != 0 {
		t.Errorf("expected 0 pinned after ctrl+u, got %d", p.PinnedCount())
	}
}

func TestListFrame_ClearAllPins_ExitsPinnedFilter(t *testing.T) {
	resources := []sdk.Resource{
		{Address: "aws_instance.a", Type: "aws_instance"},
	}
	p := newTestPlugin(resources)
	p.pins = sdk.NewPinService()
	p.rebuildTree()
	f := &listFrame{plugin: p}

	p.pins.Toggle("aws_instance.a")
	p.syncPinnedToTree()
	p.pinnedOnly = true
	p.SetFilter("")

	f.Update(tea.KeyMsg{Type: tea.KeyCtrlU})

	if p.pinnedOnly {
		t.Error("expected pinnedOnly=false after clearing all pins")
	}
}

func TestListFrame_PanDisabledWhenWrapOn(t *testing.T) {
	resources := []sdk.Resource{
		{Address: "module.very_long_name.aws_instance.server", Type: "aws_instance"},
	}
	p := newTestPlugin(resources)
	p.rebuildTree()
	p.listWrap = true
	f := &listFrame{plugin: p}

	f.Update(tea.KeyMsg{Type: tea.KeyRight})
	if p.listHScroll != 0 {
		t.Errorf("expected pan disabled when wrap is on, got listHScroll=%d", p.listHScroll)
	}
}

func TestStateFilterFrame_WrapToggle(t *testing.T) {
	resources := []sdk.Resource{
		{Address: "module.a.aws_instance.one", Type: "aws_instance"},
	}
	p := newTestPlugin(resources)
	p.rebuildTree()

	f := &listFrame{plugin: p}
	f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})

	p.listWrap = false
	p.Update(tea.KeyMsg{Type: tea.KeyCtrlW})
	if !p.listWrap {
		t.Error("expected ctrl+w to toggle wrap in filter mode")
	}
}

func TestStateFilterFrame_PanDisabledWhenWrapOn(t *testing.T) {
	resources := []sdk.Resource{
		{Address: "module.very_long_name.aws_instance.server", Type: "aws_instance"},
	}
	p := newTestPlugin(resources)
	p.rebuildTree()

	f := &listFrame{plugin: p}
	f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})

	p.listWrap = true
	p.listHScroll = 0
	p.Update(tea.KeyMsg{Type: tea.KeyRight})
	if p.listHScroll != 0 {
		t.Errorf("expected pan disabled in filter mode when wrap is on, got listHScroll=%d", p.listHScroll)
	}
}

func TestListFrame_PanRightLeft(t *testing.T) {
	resources := []sdk.Resource{
		{Address: "module.very_long_name.aws_instance.server", Type: "aws_instance"},
	}
	p := newTestPlugin(resources)
	p.rebuildTree()
	f := &listFrame{plugin: p}

	t.Run("ShouldPanRight", func(t *testing.T) {
		p.listHScroll = 0
		f.Update(tea.KeyMsg{Type: tea.KeyRight})
		if p.listHScroll != 10 {
			t.Errorf("expected listHScroll=10 after right, got %d", p.listHScroll)
		}
	})

	t.Run("ShouldPanLeft", func(t *testing.T) {
		p.listHScroll = 10
		f.Update(tea.KeyMsg{Type: tea.KeyLeft})
		if p.listHScroll != 0 {
			t.Errorf("expected listHScroll=0 after left, got %d", p.listHScroll)
		}
	})

	t.Run("ShouldNotPanBelowZero", func(t *testing.T) {
		p.listHScroll = 0
		f.Update(tea.KeyMsg{Type: tea.KeyLeft})
		if p.listHScroll != 0 {
			t.Errorf("expected listHScroll=0, got %d", p.listHScroll)
		}
	})
}

func TestListFrame_Hints_FlatMode(t *testing.T) {
	resources := []sdk.Resource{
		{Address: "module.a.aws_instance.one", Type: "aws_instance"},
	}
	p := newTestPlugin(resources)
	p.treeMode = false
	p.rebuildTree()

	hints := p.stack.Hints()
	for _, h := range hints {
		if h.Key == "[" || h.Key == "]" {
			t.Error("list frame hints should NOT include collapse/expand keys in flat mode")
			break
		}
	}
}

func TestListFrame_Hints_IncludesRefresh(t *testing.T) {
	resources := []sdk.Resource{
		{Address: "aws_instance.web", Type: "aws_instance"},
	}
	p := newTestPlugin(resources)

	hints := p.stack.Hints()
	found := false
	for _, h := range hints {
		if h.Key == "^r" && h.Description == "refresh" {
			found = true
			break
		}
	}
	if !found {
		t.Error("list frame hints should include ^r refresh when status is Done")
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

func TestDetailFrame_WhenTKeyCmdExecuted_ShouldProduceTaintRequestMsg(t *testing.T) {
	p := newTestPlugin([]sdk.Resource{{Address: "a"}})
	p.status = StatusShowingDetail
	p.detailAddr = "aws_instance.web"
	f := &detailFrame{plugin: p}

	_, cmd := f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}
	msg := cmd()
	if msg == nil {
		t.Fatal("cmd() returned nil")
	}
}

func TestDetailFrame_WhenShiftTKeyCmdExecuted_ShouldProduceUntaintRequestMsg(t *testing.T) {
	p := newTestPlugin([]sdk.Resource{{Address: "a"}})
	p.status = StatusShowingDetail
	p.detailAddr = "aws_instance.web"
	f := &detailFrame{plugin: p}

	_, cmd := f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'T'}})
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}
	msg := cmd()
	if msg == nil {
		t.Fatal("cmd() returned nil")
	}
}

func TestDetailFrame_WhenNKeyCmdExecuted_ShouldProduceImportRequestMsg(t *testing.T) {
	p := newTestPlugin([]sdk.Resource{{Address: "a"}})
	p.status = StatusShowingDetail
	p.detailAddr = "aws_instance.web"
	f := &detailFrame{plugin: p}

	_, cmd := f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}
	msg := cmd()
	if msg == nil {
		t.Fatal("cmd() returned nil")
	}
}

func TestListFrame_WhenTKeyCmdExecuted_ShouldProduceTaintRequestMsg(t *testing.T) {
	resources := []sdk.Resource{{Address: "aws_instance.web", Type: "aws_instance"}}
	p := newTestPlugin(resources)
	p.rebuildTree()
	f := &listFrame{plugin: p}

	_, cmd := f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}
	msg := cmd()
	if msg == nil {
		t.Fatal("cmd() returned nil")
	}
}

func TestListFrame_WhenShiftTKeyCmdExecuted_ShouldProduceUntaintRequestMsg(t *testing.T) {
	resources := []sdk.Resource{{Address: "aws_instance.web", Type: "aws_instance"}}
	p := newTestPlugin(resources)
	p.rebuildTree()
	f := &listFrame{plugin: p}

	_, cmd := f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'T'}})
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}
	msg := cmd()
	if msg == nil {
		t.Fatal("cmd() returned nil")
	}
}

func TestListFrame_WhenNKeyCmdExecuted_ShouldProduceImportRequestMsg(t *testing.T) {
	resources := []sdk.Resource{{Address: "aws_instance.web", Type: "aws_instance"}}
	p := newTestPlugin(resources)
	p.rebuildTree()
	f := &listFrame{plugin: p}

	_, cmd := f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}
	msg := cmd()
	if msg == nil {
		t.Fatal("cmd() returned nil")
	}
}
