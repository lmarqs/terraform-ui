package components

import (
	"strings"
	"testing"
)

func TestHeader_Render_IsThreeLines(t *testing.T) {
	h := NewHeader("/home/user/infra", "production")
	output := h.Render(80)
	lines := strings.Split(output, "\n")
	if len(lines) != 3 {
		t.Errorf("expected 3 lines, got %d", len(lines))
	}
}

func TestHeader_Render_ContainsChdir(t *testing.T) {
	h := NewHeader(".", "default").WithChdir("modules/sa-east-1")
	output := h.Render(80)
	if !strings.Contains(output, "modules/sa-east-1") {
		t.Error("should contain chdir value")
	}
	if !strings.Contains(output, "Chdir:") {
		t.Error("should contain Chdir: label")
	}
}

func TestHeader_Render_ContainsWorkspace(t *testing.T) {
	h := NewHeader(".", "staging")
	output := h.Render(80)
	if !strings.Contains(output, "staging") {
		t.Error("should contain workspace")
	}
	if !strings.Contains(output, "Workspace:") {
		t.Error("should contain Workspace: label")
	}
}

func TestHeader_Render_ContainsProject(t *testing.T) {
	h := NewHeader("/my/project", "default")
	output := h.Render(80)
	if !strings.Contains(output, "project") {
		t.Error("should contain directory basename")
	}
	if strings.Contains(output, "/my/project") {
		t.Error("should show only basename, not full path")
	}
	if !strings.Contains(output, "Project:") {
		t.Error("should contain Project: label")
	}
}

func TestHeader_Render_ContainsLogo(t *testing.T) {
	h := NewHeader(".", "default")
	output := h.Render(80)
	if !strings.Contains(output, "╔╦╗") {
		t.Error("should contain ASCII logo")
	}
	if !strings.Contains(output, "╠╣") {
		t.Error("should contain ASCII logo second line")
	}
}

func TestHeader_Render_PinnedCount(t *testing.T) {
	h := NewHeader(".", "default").WithPinnedCount(5)
	output := h.Render(80)
	if !strings.Contains(output, "5 pinned") {
		t.Error("should show pinned count")
	}
}

func TestHeader_Render_ZeroPinnedHidden(t *testing.T) {
	h := NewHeader(".", "default").WithPinnedCount(0)
	output := h.Render(80)
	if strings.Contains(output, "pinned") {
		t.Error("should not show pinned when count is 0")
	}
}

func TestHeader_Render_NoChdirShowsDash(t *testing.T) {
	h := NewHeader(".", "default")
	output := h.Render(80)
	lines := strings.Split(output, "\n")
	if !strings.Contains(lines[1], "-") {
		t.Error("should show dash when no chdir")
	}
}

func TestHeader_Render_VariousWidths(t *testing.T) {
	h := NewHeader("/some/path", "production").
		WithChdir("prod-us-east").
		WithPinnedCount(3)

	widths := []int{40, 80, 120, 200}
	for _, w := range widths {
		output := h.Render(w)
		if output == "" {
			t.Errorf("Render(%d) returned empty string", w)
		}
		lines := strings.Split(output, "\n")
		if len(lines) != 3 {
			t.Errorf("Render(%d) should produce 3 lines, got %d", w, len(lines))
		}
	}
}

func TestHeader_Chainable(t *testing.T) {
	h := NewHeader(".", "default").
		WithChdir("ctx").
		WithPinnedCount(5)

	if h.chdir != "ctx" {
		t.Error("WithChdir should chain")
	}
	if h.pinnedCount != 5 {
		t.Error("WithPinnedCount should chain")
	}
}

func TestHeader_WithWorkspace(t *testing.T) {
	h := NewHeader("/project", "default").WithChdir("modules/vpc").WithPinnedCount(3)
	h = h.WithWorkspace("staging")

	if h.workspace != "staging" {
		t.Errorf("WithWorkspace: workspace = %q, want %q", h.workspace, "staging")
	}
	if h.chdir != "modules/vpc" {
		t.Errorf("WithWorkspace should preserve chdir, got %q", h.chdir)
	}
	if h.pinnedCount != 3 {
		t.Errorf("WithWorkspace should preserve pinnedCount, got %d", h.pinnedCount)
	}

	output := h.Render(80)
	if !strings.Contains(output, "staging") {
		t.Error("Render after WithWorkspace should contain new workspace name")
	}
	if !strings.Contains(output, "modules/vpc") {
		t.Error("Render after WithWorkspace should still contain chdir")
	}
}

func TestHeader_Render_VeryNarrowWidth(t *testing.T) {
	h := NewHeader("/some/project", "production").
		WithChdir("modules/vpc").
		WithPinnedCount(3)
	output := h.Render(10)
	if output == "" {
		t.Error("Render with narrow width should still produce output")
	}
	lines := strings.Split(output, "\n")
	if len(lines) != 3 {
		t.Errorf("expected 3 lines even with narrow width, got %d", len(lines))
	}
}
