package frames

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

func TestConfirmFrame_YesCallsHandler(t *testing.T) {
	called := false
	f := NewConfirmFrame("Delete?", func() tea.Cmd {
		called = true
		return nil
	}, nil)

	result, _ := f.Update(keyMsg("y"))
	if result != nil {
		t.Fatal("y should pop confirm frame")
	}
	if !called {
		t.Fatal("y should call onYes")
	}
}

func TestConfirmFrame_UpperY(t *testing.T) {
	called := false
	f := NewConfirmFrame("Delete?", func() tea.Cmd {
		called = true
		return nil
	}, nil)

	result, _ := f.Update(keyMsg("Y"))
	if result != nil {
		t.Fatal("Y should pop confirm frame")
	}
	if !called {
		t.Fatal("Y should call onYes")
	}
}

func TestConfirmFrame_NoCallsHandler(t *testing.T) {
	noCalled := false
	f := NewConfirmFrame("Delete?", func() tea.Cmd { return nil }, func() tea.Cmd {
		noCalled = true
		return nil
	})

	result, _ := f.Update(keyMsg("n"))
	if result != nil {
		t.Fatal("n should pop confirm frame")
	}
	if !noCalled {
		t.Fatal("n should call onNo")
	}
}

func TestConfirmFrame_EscCancels(t *testing.T) {
	noCalled := false
	f := NewConfirmFrame("Delete?", func() tea.Cmd { return nil }, func() tea.Cmd {
		noCalled = true
		return nil
	})

	result, _ := f.Update(keyMsg("esc"))
	if result != nil {
		t.Fatal("esc should pop confirm frame")
	}
	if !noCalled {
		t.Fatal("esc should call onNo")
	}
}

func TestConfirmFrame_OtherKeysIgnored(t *testing.T) {
	f := NewConfirmFrame("Delete?", func() tea.Cmd { return nil }, nil)

	otherKeys := []string{"a", "i", "d", "enter", " ", "r", "q"}
	for _, key := range otherKeys {
		result, _ := f.Update(keyMsg(key))
		if result == nil {
			t.Fatalf("key %q should be ignored, not cause pop", key)
		}
	}
}

func TestConfirmFrame_NilOnNo(t *testing.T) {
	f := NewConfirmFrame("Delete?", func() tea.Cmd { return nil }, nil)

	// Should not panic with nil onNo
	result, _ := f.Update(keyMsg("n"))
	if result != nil {
		t.Fatal("n should still pop with nil onNo")
	}
}

func TestConfirmFrame_View(t *testing.T) {
	f := NewConfirmFrame("Delete resource?", func() tea.Cmd { return nil }, nil)
	view := f.View(80, 24)
	if view != "Delete resource? (y/n)" {
		t.Fatalf("expected 'Delete resource? (y/n)', got %q", view)
	}
}

func TestConfirmFrame_Hints(t *testing.T) {
	f := NewConfirmFrame("Delete?", func() tea.Cmd { return nil }, nil)
	hints := f.Hints()
	if len(hints) != 2 {
		t.Fatalf("expected 2 hints, got %d", len(hints))
	}
	if hints[0].Key != "y" || hints[1].Key != "n" {
		t.Fatalf("expected y/n hints, got %v", hints)
	}
}

func TestConfirmFrame_ID(t *testing.T) {
	f := NewConfirmFrame("", nil, nil)
	if f.ID() != "confirm" {
		t.Fatalf("expected ID 'confirm', got %q", f.ID())
	}
}

// Verify ConfirmFrame satisfies the Frame interface at compile time.
var _ sdk.Frame = (*ConfirmFrame)(nil)
