package components

import (
	"strings"
	"testing"
)

func TestNewHeader(t *testing.T) {
	h := NewHeader("/home/user/infra", "production", "/usr/bin/terraform", 42)

	if h.dir != "/home/user/infra" {
		t.Errorf("Header.dir = %q, want %q", h.dir, "/home/user/infra")
	}
	if h.workspace != "production" {
		t.Errorf("Header.workspace = %q, want %q", h.workspace, "production")
	}
	if h.binaryName != "terraform" {
		t.Errorf("Header.binaryName = %q, want %q", h.binaryName, "terraform")
	}
	if h.resourceCount != 42 {
		t.Errorf("Header.resourceCount = %d, want %d", h.resourceCount, 42)
	}
}

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
			h := NewHeader(".", "default", tt.binaryPath, 0)
			if h.binaryName != tt.expectedName {
				t.Errorf("NewHeader binary name = %q, want %q", h.binaryName, tt.expectedName)
			}
		})
	}
}

func TestHeader_Render_ReturnsNonEmpty(t *testing.T) {
	h := NewHeader("/home/user/infra", "production", "terraform", 10)

	output := h.Render(80)
	if output == "" {
		t.Fatal("Render(80) returned empty string")
	}
}

func TestHeader_Render_ContainsWorkspace(t *testing.T) {
	h := NewHeader("/home/user/infra", "staging", "terraform", 5)

	output := h.Render(120)
	if !strings.Contains(output, "staging") {
		t.Error("Render() should contain workspace name 'staging'")
	}
}

func TestHeader_Render_ContainsDir(t *testing.T) {
	h := NewHeader("/my/project", "default", "terraform", 0)

	output := h.Render(120)
	if !strings.Contains(output, "/my/project") {
		t.Error("Render() should contain directory path")
	}
}

func TestHeader_Render_ContainsBinaryName(t *testing.T) {
	h := NewHeader(".", "default", "/usr/bin/tofu", 0)

	output := h.Render(120)
	if !strings.Contains(output, "tofu") {
		t.Error("Render() should contain binary name 'tofu'")
	}
}

func TestHeader_Render_ContainsResourceCount(t *testing.T) {
	h := NewHeader(".", "default", "terraform", 99)

	output := h.Render(120)
	if !strings.Contains(output, "99") {
		t.Error("Render() should contain resource count '99'")
	}
}

func TestHeader_Render_VariousWidths(t *testing.T) {
	h := NewHeader("/some/path", "production", "terraform", 25)

	widths := []int{20, 40, 80, 120, 200}
	for _, w := range widths {
		output := h.Render(w)
		if output == "" {
			t.Errorf("Render(%d) returned empty string", w)
		}
	}
}

func TestHeader_Render_ZeroResourceCount(t *testing.T) {
	h := NewHeader(".", "default", "terraform", 0)

	output := h.Render(80)
	if output == "" {
		t.Fatal("Render() with zero resources returned empty string")
	}
	if !strings.Contains(output, "0") {
		t.Error("Render() should contain '0' for zero resources")
	}
}
