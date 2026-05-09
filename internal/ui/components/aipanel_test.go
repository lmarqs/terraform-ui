package components

import (
	"strings"
	"testing"
)

func TestNewAIPanel(t *testing.T) {
	panel := NewAIPanel()
	if panel.IsVisible() {
		t.Error("expected new panel to be hidden")
	}
}

func TestAIPanel_Show(t *testing.T) {
	panel := NewAIPanel().Show("aws_instance.web")
	if !panel.IsVisible() {
		t.Error("expected panel to be visible after Show")
	}
	if !panel.loading {
		t.Error("expected panel to be in loading state after Show")
	}
	if panel.title != "aws_instance.web" {
		t.Errorf("expected title %q, got %q", "aws_instance.web", panel.title)
	}
	if panel.content != "" {
		t.Error("expected content to be empty after Show")
	}
}

func TestAIPanel_AppendContent(t *testing.T) {
	panel := NewAIPanel().Show("test")
	panel = panel.AppendContent("Hello ")
	panel = panel.AppendContent("World")
	if panel.content != "Hello World" {
		t.Errorf("expected content %q, got %q", "Hello World", panel.content)
	}
}

func TestAIPanel_Hide(t *testing.T) {
	panel := NewAIPanel().Show("test")
	panel = panel.AppendContent("some content")
	panel = panel.Hide()
	if panel.IsVisible() {
		t.Error("expected panel to be hidden after Hide")
	}
	if panel.content != "" {
		t.Error("expected content to be cleared after Hide")
	}
	if panel.loading {
		t.Error("expected loading to be false after Hide")
	}
}

func TestAIPanel_Render_HiddenReturnsEmpty(t *testing.T) {
	panel := NewAIPanel()
	result := panel.Render(80, 24)
	if result != "" {
		t.Errorf("expected empty string for hidden panel, got %q", result)
	}
}

func TestAIPanel_Render_VisibleWithContent(t *testing.T) {
	panel := NewAIPanel().Show("aws_s3_bucket.data")
	panel = panel.AppendContent("This bucket stores application data.")
	panel = panel.SetDone()
	result := panel.Render(80, 24)
	if result == "" {
		t.Error("expected non-empty render for visible panel with content")
	}
	if !strings.Contains(result, "aws_s3_bucket.data") {
		t.Error("expected render to contain the title")
	}
	if !strings.Contains(result, "This bucket stores application data.") {
		t.Error("expected render to contain the content")
	}
}

func TestAIPanel_Render_Loading(t *testing.T) {
	panel := NewAIPanel().Show("loading test")
	result := panel.Render(80, 24)
	if !strings.Contains(result, "Thinking...") {
		t.Error("expected render to contain 'Thinking...' when loading with no content")
	}
}

func TestAIPanel_SetError(t *testing.T) {
	panel := NewAIPanel().Show("test")
	panel = panel.SetError("connection failed")
	if panel.loading {
		t.Error("expected loading to be false after SetError")
	}
	if panel.content != "Error: connection failed" {
		t.Errorf("expected error content, got %q", panel.content)
	}
}

func TestAIPanel_SetDone(t *testing.T) {
	panel := NewAIPanel().Show("test")
	panel = panel.AppendContent("done content")
	panel = panel.SetDone()
	if panel.loading {
		t.Error("expected loading to be false after SetDone")
	}
}

func TestAIPanel_ScrollDown(t *testing.T) {
	panel := NewAIPanel().Show("test")
	panel = panel.ScrollDown()
	if panel.scrollY != 1 {
		t.Errorf("expected scrollY=1 after ScrollDown, got %d", panel.scrollY)
	}
}

func TestAIPanel_ScrollUp(t *testing.T) {
	panel := NewAIPanel().Show("test")
	panel = panel.ScrollDown().ScrollDown().ScrollUp()
	if panel.scrollY != 1 {
		t.Errorf("expected scrollY=1 after Down+Down+Up, got %d", panel.scrollY)
	}
}

func TestAIPanel_ScrollUp_AtZero(t *testing.T) {
	panel := NewAIPanel().Show("test")
	panel = panel.ScrollUp()
	if panel.scrollY != 0 {
		t.Errorf("expected scrollY=0 when scrolling up at top, got %d", panel.scrollY)
	}
}
