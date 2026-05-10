package scope

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

func TestNew(t *testing.T) {
	p := New(nil).(*Plugin)
	if p.ID() != "scope" {
		t.Errorf("ID() = %q, want %q", p.ID(), "scope")
	}
	if p.Name() != "Scope" {
		t.Errorf("Name() = %q, want %q", p.Name(), "Scope")
	}
}

func TestReady_Idle(t *testing.T) {
	p := New(nil).(*Plugin)
	if p.Ready() {
		t.Error("should not be ready when idle")
	}
}

func TestReady_Done(t *testing.T) {
	p := New(nil).(*Plugin)
	p.status = StatusDone
	if !p.Ready() {
		t.Error("should be ready when done")
	}
}

func TestUpdate_ScopeDiscoveredMsg_Success(t *testing.T) {
	p := New(nil).(*Plugin)
	p.session = sdk.NewSession()

	scopes := []Scope{
		{Path: "modules/a", Name: "a", AbsPath: "/repo/modules/a"},
		{Path: "modules/b", Name: "b", AbsPath: "/repo/modules/b"},
	}

	p.Update(ScopeDiscoveredMsg{Scopes: scopes})

	if p.status != StatusDone {
		t.Errorf("status = %d, want StatusDone", p.status)
	}
	if len(p.scopes) != 2 {
		t.Errorf("scopes count = %d, want 2", len(p.scopes))
	}
}

func TestUpdate_ScopeDiscoveredMsg_Error(t *testing.T) {
	p := New(nil).(*Plugin)
	p.session = sdk.NewSession()

	p.Update(ScopeDiscoveredMsg{Err: fmt.Errorf("fail")})

	if p.status != StatusError {
		t.Errorf("status = %d, want StatusError", p.status)
	}
	if p.errMsg != "fail" {
		t.Errorf("errMsg = %q, want %q", p.errMsg, "fail")
	}
}

func TestUpdate_Navigation(t *testing.T) {
	p := New(nil).(*Plugin)
	p.status = StatusDone
	p.scopes = []Scope{
		{Path: "a"}, {Path: "b"}, {Path: "c"},
	}
	p.session = sdk.NewSession()

	// Move down
	p.Update(tea.KeyMsg{Type: tea.KeyDown})
	if p.selected != 1 {
		t.Errorf("selected = %d, want 1", p.selected)
	}

	// Move down again
	p.Update(tea.KeyMsg{Type: tea.KeyDown})
	if p.selected != 2 {
		t.Errorf("selected = %d, want 2", p.selected)
	}

	// Move down at bottom (stays)
	p.Update(tea.KeyMsg{Type: tea.KeyDown})
	if p.selected != 2 {
		t.Errorf("selected = %d, want 2 (at bottom)", p.selected)
	}

	// Move up
	p.Update(tea.KeyMsg{Type: tea.KeyUp})
	if p.selected != 1 {
		t.Errorf("selected = %d, want 1", p.selected)
	}
}

func TestUpdate_Enter_SetsActiveAndDeactivates(t *testing.T) {
	p := New(nil).(*Plugin)
	p.status = StatusDone
	p.scopes = []Scope{
		{Path: "a", AbsPath: "/abs/a"},
		{Path: "b", AbsPath: "/abs/b"},
	}
	p.session = sdk.NewSession()
	p.selected = 1

	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected deactivate command")
	}

	if p.active != 1 {
		t.Errorf("active = %d, want 1", p.active)
	}

	// Session should be updated
	v, _ := sdk.GetTyped[string](p.session, sdk.SessionKeyActiveScope)
	if v != "b" {
		t.Errorf("session scope = %q, want %q", v, "b")
	}
}

func TestScopeCount(t *testing.T) {
	p := New(nil).(*Plugin)
	p.scopes = []Scope{{}, {}, {}}
	if p.ScopeCount() != 3 {
		t.Errorf("ScopeCount() = %d, want 3", p.ScopeCount())
	}
}

func TestView_Loading(t *testing.T) {
	p := New(nil).(*Plugin)
	p.status = StatusLoading
	output := p.View(80, 20)
	if output == "" {
		t.Error("should render loading message")
	}
}

func TestView_Error(t *testing.T) {
	p := New(nil).(*Plugin)
	p.status = StatusError
	p.errMsg = "something broke"
	output := p.View(80, 20)
	if output == "" {
		t.Error("should render error message")
	}
}

func TestView_Done_WithScopes(t *testing.T) {
	p := New(nil).(*Plugin)
	p.status = StatusDone
	p.scopes = []Scope{
		{Path: "modules/a", Name: "a"},
		{Path: "modules/b", Name: "b"},
	}
	output := p.View(80, 20)
	if output == "" {
		t.Error("should render scope list")
	}
}

func TestView_Done_Empty(t *testing.T) {
	p := New(nil).(*Plugin)
	p.status = StatusDone
	p.scopes = []Scope{}
	output := p.View(80, 20)
	if output == "" {
		t.Error("should render placeholder")
	}
}

func TestActiveScope(t *testing.T) {
	p := New(nil).(*Plugin)
	p.scopes = []Scope{{Path: "a"}, {Path: "b"}}
	p.active = 1

	s := p.ActiveScope()
	if s == nil || s.Path != "b" {
		t.Errorf("ActiveScope() = %v, want scope 'b'", s)
	}
}

func TestActiveScope_NoSelection(t *testing.T) {
	p := New(nil).(*Plugin)
	p.scopes = []Scope{{Path: "a"}}
	p.active = -1

	if p.ActiveScope() != nil {
		t.Error("should return nil when no selection")
	}
}
