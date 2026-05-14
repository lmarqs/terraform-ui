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

func TestAIPanel_SetSize(t *testing.T) {
	panel := NewAIPanel()
	panel = panel.SetSize(100, 50)
	if panel.width != 100 {
		t.Errorf("expected width=100, got %d", panel.width)
	}
	if panel.height != 50 {
		t.Errorf("expected height=50, got %d", panel.height)
	}
}

func TestAIPanel_Render_LoadingWithPartialContent(t *testing.T) {
	panel := NewAIPanel().Show("streaming")
	panel = panel.AppendContent("Partial response so far")
	result := panel.Render(80, 24)
	if !strings.Contains(result, "Partial response so far") {
		t.Error("expected render to contain partial content")
	}
	if !strings.Contains(result, "▍") {
		t.Error("expected render to contain loading cursor when loading with content")
	}
}

func TestAIPanel_Render_LongLinesWrapped(t *testing.T) {
	panel := NewAIPanel().Show("wrap test")
	longLine := strings.Repeat("A", 200)
	panel = panel.AppendContent(longLine)
	panel = panel.SetDone()
	result := panel.Render(80, 24)
	if !strings.Contains(result, "AAAA") {
		t.Error("expected render to contain wrapped content")
	}
}

func TestAIPanel_Render_ScrollClamping(t *testing.T) {
	panel := NewAIPanel().Show("scroll test")
	panel = panel.AppendContent("line1\nline2\nline3")
	panel = panel.SetDone()
	// Scroll way past the content
	for i := 0; i < 50; i++ {
		panel = panel.ScrollDown()
	}
	result := panel.Render(80, 24)
	if result == "" {
		t.Error("expected non-empty render even with excessive scroll")
	}
}

func TestAIPanel_Render_NarrowWidth(t *testing.T) {
	panel := NewAIPanel().Show("narrow")
	panel = panel.AppendContent("content")
	panel = panel.SetDone()
	result := panel.Render(30, 24)
	if result == "" {
		t.Error("expected non-empty render with narrow width")
	}
}

func TestAIPanel_Render_ShortHeight(t *testing.T) {
	panel := NewAIPanel().Show("short")
	panel = panel.AppendContent("content")
	panel = panel.SetDone()
	result := panel.Render(80, 8)
	if result == "" {
		t.Error("expected non-empty render with short height")
	}
}

func TestAIPanel_Render_WideWidth(t *testing.T) {
	panel := NewAIPanel().Show("wide")
	panel = panel.AppendContent("content")
	panel = panel.SetDone()
	result := panel.Render(200, 24)
	if result == "" {
		t.Error("expected non-empty render with wide width")
	}
}
