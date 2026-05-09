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

func TestHeader_Render_CompactDefault(t *testing.T) {
	h := NewHeader("/home/user/infra", "production", "terraform", 10)

	output := h.Render(80)
	if output == "" {
		t.Fatal("Render(80) returned empty string")
	}
}

func TestHeader_Render_CompactContainsWorkspace(t *testing.T) {
	h := NewHeader("/home/user/infra", "staging", "terraform", 5)

	output := h.Render(120)
	if !strings.Contains(output, "staging") {
		t.Error("Render() should contain workspace name 'staging'")
	}
}

func TestHeader_Render_CompactContainsDir(t *testing.T) {
	h := NewHeader("/my/project", "default", "terraform", 0)

	output := h.Render(120)
	if !strings.Contains(output, "/my/project") {
		t.Error("Render() should contain directory path")
	}
}

func TestHeader_Render_CompactContainsBinaryName(t *testing.T) {
	h := NewHeader(".", "default", "/usr/bin/tofu", 0)

	output := h.Render(120)
	if !strings.Contains(output, "tofu") {
		t.Error("Render() should contain binary name 'tofu'")
	}
}

func TestHeader_Render_CompactContainsResourceCount(t *testing.T) {
	h := NewHeader(".", "default", "terraform", 99)

	output := h.Render(120)
	if !strings.Contains(output, "99") {
		t.Error("Render() should contain resource count '99'")
	}
}

func TestHeader_Render_CompactWithActiveView(t *testing.T) {
	h := NewHeader(".", "default", "terraform", 10).
		WithActiveView("State Browser")

	output := h.Render(120)
	if !strings.Contains(output, "State Browser") {
		t.Error("Render() should contain active view name 'State Browser'")
	}
	// Should NOT show resource count when active view is set
	if strings.Contains(output, "resources:") {
		t.Error("Render() should not show resources: when active view is set")
	}
}

func TestHeader_Render_CompactWithoutActiveView(t *testing.T) {
	h := NewHeader(".", "default", "terraform", 15)

	output := h.Render(120)
	if !strings.Contains(output, "15") {
		t.Error("Render() should show resource count when no active view")
	}
}

func TestHeader_Render_CompactWithPlanSummary(t *testing.T) {
	h := NewHeader("./infra/vpc", "staging", "/usr/bin/tofu", 0).
		WithPlanSummary(3, 1, 2, 0).
		WithActiveView("State Browser")

	output := h.Render(150)
	if !strings.Contains(output, "Plan:") {
		t.Error("Render() should contain 'Plan:' section")
	}
	if !strings.Contains(output, "+3") {
		t.Error("Render() should contain '+3' for creates")
	}
	if !strings.Contains(output, "~1") {
		t.Error("Render() should contain '~1' for updates")
	}
	if !strings.Contains(output, "-2") {
		t.Error("Render() should contain '-2' for deletes")
	}
}

func TestHeader_Render_CompactWithPlanSummaryReplace(t *testing.T) {
	h := NewHeader(".", "default", "terraform", 0).
		WithPlanSummary(0, 0, 0, 5)

	output := h.Render(120)
	if !strings.Contains(output, "±5") {
		t.Error("Render() should contain replace count")
	}
}

func TestHeader_Render_CompactWithPinnedCount(t *testing.T) {
	h := NewHeader("./infra/vpc", "staging", "tofu", 0).
		WithPinnedCount(4).
		WithActiveView("State Browser")

	output := h.Render(150)
	if !strings.Contains(output, "4 pinned") {
		t.Error("Render() should contain '4 pinned'")
	}
}

func TestHeader_Render_CompactZeroPinnedHidden(t *testing.T) {
	h := NewHeader(".", "default", "terraform", 0).
		WithPinnedCount(0)

	output := h.Render(120)
	if strings.Contains(output, "pinned") {
		t.Error("Render() should not contain pinned section when count is 0")
	}
}

func TestHeader_Render_CompactAllFields(t *testing.T) {
	h := NewHeader("./infra/vpc", "staging", "/usr/bin/tofu", 10).
		WithPlanSummary(3, 1, 2, 0).
		WithPinnedCount(4).
		WithActiveView("State Browser")

	output := h.Render(200)
	if !strings.Contains(output, "staging") {
		t.Error("should contain workspace")
	}
	if !strings.Contains(output, "./infra/vpc") {
		t.Error("should contain dir")
	}
	if !strings.Contains(output, "tofu") {
		t.Error("should contain binary name")
	}
	if !strings.Contains(output, "+3") {
		t.Error("should contain create count")
	}
	if !strings.Contains(output, "4 pinned") {
		t.Error("should contain pinned count")
	}
	if !strings.Contains(output, "State Browser") {
		t.Error("should contain active view")
	}
}

func TestHeader_Render_CompactNoPlanSummary(t *testing.T) {
	h := NewHeader(".", "default", "terraform", 5)

	output := h.Render(120)
	if strings.Contains(output, "Plan:") {
		t.Error("Render() should not show Plan: when no plan summary is set")
	}
}

func TestHeader_Render_ExpandedMode(t *testing.T) {
	h := NewHeader("./infra/vpc", "staging", "/usr/bin/tofu", 0).
		WithPlanSummary(3, 1, 2, 0).
		WithPinnedCount(4).
		WithActiveView("State Browser").
		WithExpanded(true)

	output := h.Render(120)
	if !strings.Contains(output, "workspace:") {
		t.Error("Expanded render should contain 'workspace:' label")
	}
	if !strings.Contains(output, "staging") {
		t.Error("Expanded render should contain workspace name")
	}
	if !strings.Contains(output, "dir:") {
		t.Error("Expanded render should contain 'dir:' label")
	}
	if !strings.Contains(output, "./infra/vpc") {
		t.Error("Expanded render should contain dir path")
	}
	if !strings.Contains(output, "binary:") {
		t.Error("Expanded render should contain 'binary:' label")
	}
	if !strings.Contains(output, "tofu") {
		t.Error("Expanded render should contain binary name")
	}
	if !strings.Contains(output, "3 to add") {
		t.Error("Expanded render should contain '3 to add'")
	}
	if !strings.Contains(output, "1 to change") {
		t.Error("Expanded render should contain '1 to change'")
	}
	if !strings.Contains(output, "2 to destroy") {
		t.Error("Expanded render should contain '2 to destroy'")
	}
	if !strings.Contains(output, "4 targets") {
		t.Error("Expanded render should contain pinned targets")
	}
	if !strings.Contains(output, "State Browser") {
		t.Error("Expanded render should contain active view on separator line")
	}
	if !strings.Contains(output, "───") {
		t.Error("Expanded render should contain separator line")
	}
}

func TestHeader_Render_ExpandedWithReplace(t *testing.T) {
	h := NewHeader(".", "default", "terraform", 0).
		WithPlanSummary(1, 0, 0, 2).
		WithExpanded(true)

	output := h.Render(120)
	if !strings.Contains(output, "2 to replace") {
		t.Error("Expanded render should contain '2 to replace'")
	}
}

func TestHeader_Render_ExpandedNoPlan(t *testing.T) {
	h := NewHeader(".", "default", "terraform", 5).
		WithExpanded(true)

	output := h.Render(120)
	// Should not have Plan line when no plan summary
	if strings.Contains(output, "Plan:") {
		t.Error("Expanded render without plan should not contain 'Plan:'")
	}
}

func TestHeader_Toggle(t *testing.T) {
	h := NewHeader(".", "default", "terraform", 0)

	if h.expanded {
		t.Fatal("new Header should not be expanded")
	}

	h2 := h.Toggle()
	if !h2.expanded {
		t.Error("Toggle should set expanded to true")
	}

	h3 := h2.Toggle()
	if h3.expanded {
		t.Error("Toggle again should set expanded to false")
	}
}

func TestHeader_WithExpanded(t *testing.T) {
	h := NewHeader(".", "default", "terraform", 0)

	h2 := h.WithExpanded(true)
	if !h2.expanded {
		t.Error("WithExpanded(true) should set expanded to true")
	}

	h3 := h2.WithExpanded(false)
	if h3.expanded {
		t.Error("WithExpanded(false) should set expanded to false")
	}
}

func TestHeader_WithPlanSummary(t *testing.T) {
	h := NewHeader(".", "default", "terraform", 0).
		WithPlanSummary(5, 3, 1, 2)

	if h.planSummary == nil {
		t.Fatal("planSummary should not be nil")
	}
	if h.planSummary.Create != 5 {
		t.Errorf("Create = %d, want 5", h.planSummary.Create)
	}
	if h.planSummary.Update != 3 {
		t.Errorf("Update = %d, want 3", h.planSummary.Update)
	}
	if h.planSummary.Delete != 1 {
		t.Errorf("Delete = %d, want 1", h.planSummary.Delete)
	}
	if h.planSummary.Replace != 2 {
		t.Errorf("Replace = %d, want 2", h.planSummary.Replace)
	}
}

func TestHeader_WithPinnedCount(t *testing.T) {
	h := NewHeader(".", "default", "terraform", 0).
		WithPinnedCount(7)

	if h.pinnedCount != 7 {
		t.Errorf("pinnedCount = %d, want 7", h.pinnedCount)
	}
}

func TestHeader_Render_VariousWidths(t *testing.T) {
	h := NewHeader("/some/path", "production", "terraform", 25).
		WithPlanSummary(2, 1, 0, 0).
		WithPinnedCount(3).
		WithActiveView("Plan")

	widths := []int{20, 40, 80, 120, 200}
	for _, w := range widths {
		output := h.Render(w)
		if output == "" {
			t.Errorf("Render(%d) returned empty string", w)
		}
	}
}

func TestHeader_Render_ExpandedVariousWidths(t *testing.T) {
	h := NewHeader("/some/path", "production", "terraform", 25).
		WithPlanSummary(2, 1, 0, 0).
		WithPinnedCount(3).
		WithActiveView("Plan").
		WithExpanded(true)

	widths := []int{20, 40, 80, 120, 200}
	for _, w := range widths {
		output := h.Render(w)
		if output == "" {
			t.Errorf("Expanded Render(%d) returned empty string", w)
		}
	}
}

func TestHeader_Render_CompactWithContext(t *testing.T) {
	h := NewHeader(".", "default", "terraform", 0).
		WithContext("prod-us-east")

	output := h.Render(150)
	if !strings.Contains(output, "prod-us-east") {
		t.Error("Render() should contain context")
	}
}

func TestHeader_Render_ExpandedWithContext(t *testing.T) {
	h := NewHeader(".", "default", "terraform", 0).
		WithContext("prod-us-east").
		WithExpanded(true)

	output := h.Render(150)
	if !strings.Contains(output, "context:") {
		t.Error("Expanded render should contain 'context:' label")
	}
	if !strings.Contains(output, "prod-us-east") {
		t.Error("Expanded render should contain context value")
	}
}

func TestHeader_Render_CompactZeroPlanCounts(t *testing.T) {
	h := NewHeader(".", "default", "terraform", 0).
		WithPlanSummary(0, 0, 0, 0)

	output := h.Render(120)
	// With all zeros, plan section should not show individual counts
	if strings.Contains(output, "+0") {
		t.Error("Should not show +0 for zero creates")
	}
	if strings.Contains(output, "~0") {
		t.Error("Should not show ~0 for zero updates")
	}
	if strings.Contains(output, "-0") {
		t.Error("Should not show -0 for zero deletes")
	}
}

func TestHeader_Chainable(t *testing.T) {
	// Verify all setters are chainable
	h := NewHeader(".", "default", "terraform", 0).
		WithContext("ctx").
		WithActiveView("View").
		WithPlanSummary(1, 2, 3, 4).
		WithPinnedCount(5).
		WithExpanded(true)

	if h.context != "ctx" {
		t.Error("WithContext should chain")
	}
	if h.activeView != "View" {
		t.Error("WithActiveView should chain")
	}
	if h.planSummary == nil {
		t.Error("WithPlanSummary should chain")
	}
	if h.pinnedCount != 5 {
		t.Error("WithPinnedCount should chain")
	}
	if !h.expanded {
		t.Error("WithExpanded should chain")
	}
}
