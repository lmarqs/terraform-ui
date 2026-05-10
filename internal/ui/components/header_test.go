package components

import (
	"strings"
	"testing"
)

func TestNewHeader_ExtractsBaseName(t *testing.T) {
	tests := []struct {
		binaryPath   string
		expectedName string
	}{
		{"/usr/local/bin/terraform", "terraform"},
		{"/usr/bin/tofu", "tofu"},
		{"terraform", "terraform"},
		{"/path/to/custom-binary", "custom-binary"},
	}

	for _, tt := range tests {
		t.Run(tt.binaryPath, func(t *testing.T) {
			h := NewHeader(".", "default", tt.binaryPath)
			if h.binaryName != tt.expectedName {
				t.Errorf("binaryName = %q, want %q", h.binaryName, tt.expectedName)
			}
		})
	}
}

func TestHeader_Render_IsThreeLines(t *testing.T) {
	h := NewHeader("/home/user/infra", "production", "terraform")
	output := h.Render(80)
	lines := strings.Split(output, "\n")
	if len(lines) != 3 {
		t.Errorf("expected 3 lines, got %d", len(lines))
	}
}

func TestHeader_Render_ContainsScope(t *testing.T) {
	h := NewHeader(".", "default", "terraform").WithScope("modules/sa-east-1")
	output := h.Render(80)
	if !strings.Contains(output, "modules/sa-east-1") {
		t.Error("should contain scope value")
	}
	if !strings.Contains(output, "Scope:") {
		t.Error("should contain Scope: label")
	}
}

func TestHeader_Render_ContainsWorkspace(t *testing.T) {
	h := NewHeader(".", "staging", "terraform")
	output := h.Render(80)
	if !strings.Contains(output, "staging") {
		t.Error("should contain workspace")
	}
	if !strings.Contains(output, "Workspace:") {
		t.Error("should contain Workspace: label")
	}
}

func TestHeader_Render_ContainsProjectAndBinary(t *testing.T) {
	h := NewHeader("/my/project", "default", "/usr/bin/tofu")
	output := h.Render(80)
	if !strings.Contains(output, "/my/project") {
		t.Error("should contain directory")
	}
	if !strings.Contains(output, "tofu") {
		t.Error("should contain binary name")
	}
	if !strings.Contains(output, "Project:") {
		t.Error("should contain Project: label")
	}
}

func TestHeader_Render_ContainsLogo(t *testing.T) {
	h := NewHeader(".", "default", "terraform")
	output := h.Render(80)
	if !strings.Contains(output, "╔╦╗") {
		t.Error("should contain ASCII logo")
	}
	if !strings.Contains(output, "╠╣") {
		t.Error("should contain ASCII logo second line")
	}
}

func TestHeader_Render_PinnedCount(t *testing.T) {
	h := NewHeader(".", "default", "terraform").WithPinnedCount(5)
	output := h.Render(80)
	if !strings.Contains(output, "5 pinned") {
		t.Error("should show pinned count")
	}
}

func TestHeader_Render_ZeroPinnedHidden(t *testing.T) {
	h := NewHeader(".", "default", "terraform").WithPinnedCount(0)
	output := h.Render(80)
	if strings.Contains(output, "pinned") {
		t.Error("should not show pinned when count is 0")
	}
}

func TestHeader_Render_NoScopeShowsDash(t *testing.T) {
	h := NewHeader(".", "default", "terraform")
	output := h.Render(80)
	lines := strings.Split(output, "\n")
	if !strings.Contains(lines[1], "-") {
		t.Error("should show dash when no scope")
	}
}

func TestHeader_Render_VariousWidths(t *testing.T) {
	h := NewHeader("/some/path", "production", "terraform").
		WithScope("prod-us-east").
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
	h := NewHeader(".", "default", "terraform").
		WithScope("ctx").
		WithPinnedCount(5)

	if h.scope != "ctx" {
		t.Error("WithScope should chain")
	}
	if h.pinnedCount != 5 {
		t.Error("WithPinnedCount should chain")
	}
}
