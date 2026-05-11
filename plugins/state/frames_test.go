package state

import (
	"io"
	"log/slog"
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
	expanded := p.tree.VisibleCount()

	// Enter filter mode — this rebuilds tree and auto-expands since filter=""
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if !p.filtering {
		t.Fatal("expected filtering mode after /")
	}

	// Expand all to ensure fully expanded
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}})
	expanded = p.tree.VisibleCount()

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
