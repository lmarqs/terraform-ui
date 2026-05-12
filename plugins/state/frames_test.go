package state

import (
	"io"
	"log/slog"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

func newTestPlugin(resources []sdk.Resource) *Plugin {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
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
	found := false
	for _, h := range hints {
		if h.Key == "[/]" {
			found = true
			break
		}
	}
	if !found {
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
		if h.Key == "[/]" {
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
	found := false
	for _, h := range hints {
		if h.Key == "[/]" {
			found = true
			break
		}
	}
	if !found {
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
	p.session = sdk.NewSession()
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
	p.session = sdk.NewSession()
	p.pins = sdk.NewPinService(p.session)
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
	p.session = sdk.NewSession()
	p.pins = sdk.NewPinService(p.session)
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
	p.session = sdk.NewSession()
	p.pins = sdk.NewPinService(p.session)
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
		if h.Key == "[/]" {
			t.Error("list frame hints should NOT include collapse/expand keys in flat mode")
			break
		}
	}
}
