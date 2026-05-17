package components

import (
	"strings"
	"testing"
	"time"

	"github.com/lmarqs/terraform-ui/pkg/sdk"
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

func TestHeader_WithLockInfo_ShowsLockedBadge(t *testing.T) {
	lock := &sdk.StateLock{
		ID:  "abc-123",
		Who: "user@host",
	}
	h := NewHeader("/project", "default").WithLockInfo(lock)
	output := h.Render(120)
	if !strings.Contains(output, "locked") {
		t.Error("should show 'locked' badge when lock info is set")
	}
	if !strings.Contains(output, "user@host") {
		t.Error("should show lock owner in badge")
	}
}

func TestHeader_WithLockInfo_ShowsAge(t *testing.T) {
	lock := &sdk.StateLock{
		ID:      "abc-123",
		Who:     "user@host",
		Created: time.Now().Add(-10 * time.Minute),
	}
	h := NewHeader("/project", "default").WithLockInfo(lock)
	output := h.Render(120)
	if !strings.Contains(output, "10m ago") {
		t.Errorf("should show age in badge, got: %s", output)
	}
}

func TestHeader_WithLockInfo_Nil_HidesBadge(t *testing.T) {
	h := NewHeader("/project", "default").WithLockInfo(nil)
	output := h.Render(80)
	if strings.Contains(output, "locked") {
		t.Error("should not show locked badge when lock info is nil")
	}
}

func TestHeader_WithStale_ShowsStaleBadge(t *testing.T) {
	h := NewHeader("/project", "default").WithStale(true)
	output := h.Render(80)
	if !strings.Contains(output, "stale") {
		t.Error("should show 'stale' badge when stale is true")
	}
}

func TestHeader_WithStale_False_HidesBadge(t *testing.T) {
	h := NewHeader("/project", "default").WithStale(false)
	output := h.Render(80)
	if strings.Contains(output, "stale") {
		t.Error("should not show stale badge when stale is false")
	}
}

func TestFormatBadgeAge_WhenSeconds_ShouldShowSecondsAgo(t *testing.T) {
	result := formatBadgeAge(30 * time.Second)
	if result != "30s ago" {
		t.Errorf("formatBadgeAge(30s) = %q, want %q", result, "30s ago")
	}
}

func TestFormatBadgeAge_WhenMinutes_ShouldShowMinutesAgo(t *testing.T) {
	result := formatBadgeAge(5 * time.Minute)
	if result != "5m ago" {
		t.Errorf("formatBadgeAge(5m) = %q, want %q", result, "5m ago")
	}
}

func TestFormatBadgeAge_WhenHours_ShouldShowHoursAgo(t *testing.T) {
	result := formatBadgeAge(3 * time.Hour)
	if result != "3h ago" {
		t.Errorf("formatBadgeAge(3h) = %q, want %q", result, "3h ago")
	}
}

func TestFormatBadgeAge_WhenDays_ShouldShowDaysAgo(t *testing.T) {
	result := formatBadgeAge(48 * time.Hour)
	if result != "2d ago" {
		t.Errorf("formatBadgeAge(48h) = %q, want %q", result, "2d ago")
	}
}

func TestFormatBadgeAge_WhenExactlyOneMinute_ShouldShowMinutes(t *testing.T) {
	result := formatBadgeAge(60 * time.Second)
	if result != "1m ago" {
		t.Errorf("formatBadgeAge(60s) = %q, want %q", result, "1m ago")
	}
}

func TestFormatBadgeAge_WhenExactly24Hours_ShouldShowDays(t *testing.T) {
	result := formatBadgeAge(24 * time.Hour)
	if result != "1d ago" {
		t.Errorf("formatBadgeAge(24h) = %q, want %q", result, "1d ago")
	}
}

func TestFormatLockBadge_WhenOnlyCreated_ShouldShowAgeWithoutWho(t *testing.T) {
	lock := &sdk.StateLock{
		ID:      "abc-123",
		Created: time.Now().Add(-30 * time.Second),
	}
	result := formatLockBadge(lock)
	if !strings.Contains(result, "locked") {
		t.Error("should contain 'locked'")
	}
	if !strings.Contains(result, "s ago") {
		t.Errorf("should contain age, got: %q", result)
	}
	if strings.Contains(result, "  ") {
		t.Errorf("should not contain double-space from empty Who, got: %q", result)
	}
}

func TestFormatLockBadge_WhenNoWhoNoCreated_ShouldShowOnlyLocked(t *testing.T) {
	lock := &sdk.StateLock{
		ID: "abc-123",
	}
	result := formatLockBadge(lock)
	if result != "locked" {
		t.Errorf("formatLockBadge with no details = %q, want %q", result, "locked")
	}
}

func TestHeader_WithLockInfo_PreservesOtherFields(t *testing.T) {
	lock := &sdk.StateLock{ID: "x", Who: "user"}
	h := NewHeader("/project", "prod").
		WithChdir("modules/vpc").
		WithPinnedCount(3).
		WithLockInfo(lock).
		WithStale(true)

	if h.chdir != "modules/vpc" {
		t.Error("WithLockInfo should preserve chdir")
	}
	if h.pinnedCount != 3 {
		t.Error("WithLockInfo should preserve pinnedCount")
	}
	if h.workspace != "prod" {
		t.Error("WithLockInfo should preserve workspace")
	}
	if !h.stale {
		t.Error("WithStale should preserve stale")
	}
}
