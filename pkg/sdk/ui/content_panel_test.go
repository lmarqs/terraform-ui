package ui_test

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/lmarqs/terraform-ui/pkg/sdk/ui"
)

func TestContentPanel_RendersRowsWithGutter(t *testing.T) {
	panel := ui.NewContentPanel()

	output := panel.Render(ui.RenderParams{
		Rows:       []string{"alpha", "bravo", "charlie"},
		Width:      20,
		Height:     3,
		TotalItems: 10,
		ViewOffset: 0,
		Cursor:     -1,
	})
	lines := strings.Split(output, "\n")

	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	if !strings.HasSuffix(lines[0], "▲") {
		t.Errorf("first line should end with ▲, got %q", lines[0])
	}
	if !strings.HasSuffix(lines[len(lines)-1], "▼") {
		t.Errorf("last line should end with ▼, got %q", lines[len(lines)-1])
	}
}

func TestContentPanel_NoGutterWhenNoOverflow(t *testing.T) {
	panel := ui.NewContentPanel()

	output := panel.Render(ui.RenderParams{
		Rows:       []string{"alpha", "bravo"},
		Width:      20,
		Height:     5,
		TotalItems: 2,
		ViewOffset: 0,
		Cursor:     -1,
	})
	lines := strings.Split(output, "\n")

	for i, line := range lines {
		if strings.ContainsAny(line, "▲▼┃│") {
			t.Errorf("line %d should have no gutter, got %q", i, line)
		}
	}
}

func TestContentPanel_TruncatesStyledContent(t *testing.T) {
	panel := ui.NewContentPanel()
	styled := lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000")).Render("very-long-resource-address-that-exceeds-width")

	output := panel.Render(ui.RenderParams{
		Rows:       []string{styled},
		Width:      20,
		Height:     5,
		TotalItems: 1,
		ViewOffset: 0,
		Cursor:     -1,
	})
	lines := strings.Split(output, "\n")
	visWidth := lipgloss.Width(lines[0])

	if visWidth > 20 {
		t.Errorf("expected visual width <= 20, got %d", visWidth)
	}
}

func TestContentPanel_DoesNotBreakAnsiSequences(t *testing.T) {
	panel := ui.NewContentPanel()
	styled := lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000")).Render("styled-content-that-is-long-enough-to-be-truncated")

	output := panel.Render(ui.RenderParams{
		Rows:       []string{styled},
		Width:      15,
		Height:     5,
		TotalItems: 1,
		ViewOffset: 0,
		Cursor:     -1,
	})
	stripped := ansi.Strip(output)
	if len(stripped) == 0 {
		t.Error("stripped output should not be empty")
	}
	if lipgloss.Width(output) > 15 {
		t.Errorf("visual width should not exceed panel width")
	}
}

func TestContentPanel_HorizontalScrollDropsLeadingChars(t *testing.T) {
	panel := ui.NewContentPanel()
	panel.HandleKey(tea.KeyMsg{Type: tea.KeyRight}) // hscroll += 10
	// We need hscroll=5, but HandleKey increments by 10. Let's reset and test via Render.
	// Actually let's just test the rendering with a fresh panel at hscroll 5.
	// We'll press right and check the output skips chars.

	panel2 := ui.NewContentPanel()
	panel2.HandleKey(tea.KeyMsg{Type: tea.KeyRight}) // hscroll = 10

	output := panel2.Render(ui.RenderParams{
		Rows:       []string{"0123456789abcdefghij"},
		Width:      10,
		Height:     5,
		TotalItems: 1,
		ViewOffset: 0,
		Cursor:     -1,
	})
	stripped := ansi.Strip(output)
	if !strings.HasPrefix(stripped, "abcdefghij") {
		t.Errorf("expected content scrolled by 10, got %q", stripped)
	}
}

func TestContentPanel_HorizontalScrollWithStyledContent(t *testing.T) {
	panel := ui.NewContentPanel()
	panel.HandleKey(tea.KeyMsg{Type: tea.KeyRight}) // hscroll = 10

	styled := lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000")).Render("0123456789abcdefghij")
	output := panel.Render(ui.RenderParams{
		Rows:       []string{styled},
		Width:      10,
		Height:     5,
		TotalItems: 1,
		ViewOffset: 0,
		Cursor:     -1,
	})
	stripped := ansi.Strip(output)
	if !strings.HasPrefix(stripped, "abcdefghij") {
		t.Errorf("expected styled content scrolled by 10, got %q", stripped)
	}
	if lipgloss.Width(output) > 10 {
		t.Errorf("visual width should not exceed panel width, got %d", lipgloss.Width(output))
	}
}

func TestContentPanel_CursorHighlightsRow(t *testing.T) {
	panel := ui.NewContentPanel()
	panel.SelectedStyle = func(s string, w int) string {
		return "[SEL]" + s
	}

	output := panel.Render(ui.RenderParams{
		Rows:       []string{"first", "second", "third"},
		Width:      20,
		Height:     5,
		TotalItems: 3,
		ViewOffset: 0,
		Cursor:     1,
	})
	lines := strings.Split(output, "\n")

	if !strings.Contains(lines[1], "[SEL]") {
		t.Errorf("cursor row should have selected style applied, got %q", lines[1])
	}
	if strings.Contains(lines[0], "[SEL]") {
		t.Error("non-cursor row should not have selected style")
	}
}

func TestContentPanel_GutterAlignedWithStyledRows(t *testing.T) {
	panel := ui.NewContentPanel()
	short := "ab"
	long := lipgloss.NewStyle().Foreground(lipgloss.Color("#00ff00")).Render("a-much-longer-content")

	output := panel.Render(ui.RenderParams{
		Rows:       []string{short, long, "mid"},
		Width:      20,
		Height:     3,
		TotalItems: 10,
		ViewOffset: 0,
		Cursor:     -1,
	})
	lines := strings.Split(output, "\n")

	widths := make([]int, len(lines))
	for i, line := range lines {
		widths[i] = lipgloss.Width(line)
	}

	for i := 1; i < len(widths); i++ {
		if widths[i] != widths[0] {
			t.Errorf("all lines should have same visual width, line 0=%d, line %d=%d", widths[0], i, widths[i])
		}
	}
}

func TestContentPanel_WrapModeWrapsLongContent(t *testing.T) {
	panel := ui.NewContentPanel()
	panel.HandleKey(tea.KeyMsg{Type: tea.KeyCtrlW})

	output := panel.Render(ui.RenderParams{
		Rows:       []string{"short", "this-is-a-very-long-row-that-exceeds-width"},
		Width:      15,
		Height:     10,
		TotalItems: 2,
		ViewOffset: 0,
		Cursor:     -1,
	})

	// Content should be preserved across wrapped lines
	stripped := ansi.Strip(output)
	joined := strings.ReplaceAll(stripped, "\n", "")
	if !strings.Contains(joined, "this-is-a-very-long-row-that-exceeds-width") {
		t.Errorf("wrap mode should preserve full content across lines, got %q", stripped)
	}

	// Each line should not exceed width
	for i, line := range strings.Split(output, "\n") {
		if lipgloss.Width(line) > 15 {
			t.Errorf("line %d exceeds width in wrap mode: %d > 15", i, lipgloss.Width(line))
		}
	}
}

func TestContentPanel_WrapModeRespectsHeight(t *testing.T) {
	panel := ui.NewContentPanel()
	panel.HandleKey(tea.KeyMsg{Type: tea.KeyCtrlW})

	output := panel.Render(ui.RenderParams{
		Rows:       []string{strings.Repeat("x", 100)},
		Width:      10,
		Height:     3,
		TotalItems: 1,
		ViewOffset: 0,
		Cursor:     -1,
	})

	lines := strings.Split(output, "\n")
	if len(lines) > 3 {
		t.Errorf("wrap mode should respect height budget, got %d lines", len(lines))
	}
}

func TestContentPanel_ContentWidthWithOverflow(t *testing.T) {
	panel := ui.NewContentPanel()
	if got := panel.ContentWidth(80, 5, 100); got != 80-ui.GutterWidth {
		t.Errorf("expected content width %d, got %d", 80-ui.GutterWidth, got)
	}
}

func TestContentPanel_ContentWidthWithoutOverflow(t *testing.T) {
	panel := ui.NewContentPanel()
	if got := panel.ContentWidth(80, 10, 5); got != 80 {
		t.Errorf("expected full width 80 when no overflow, got %d", got)
	}
}

func TestContentPanel_CursorWithNoOverflow(t *testing.T) {
	panel := ui.NewContentPanel()
	panel.SelectedStyle = func(s string, w int) string {
		return ">" + s
	}

	output := panel.Render(ui.RenderParams{
		Rows:       []string{"alpha", "bravo", "charlie"},
		Width:      20,
		Height:     5,
		TotalItems: 3,
		ViewOffset: 0,
		Cursor:     1,
	})
	lines := strings.Split(output, "\n")

	if !strings.HasPrefix(lines[1], ">") {
		t.Errorf("cursor should be applied without gutter, got %q", lines[1])
	}
	for _, line := range lines {
		if strings.ContainsAny(line, "▲▼┃│") {
			t.Error("should not have gutter when no overflow")
		}
	}
}

func TestContentPanel_EmptyReturnsEmpty(t *testing.T) {
	panel := ui.NewContentPanel()
	output := panel.Render(ui.RenderParams{
		Width:      20,
		Height:     5,
		TotalItems: 0,
	})
	if output != "" {
		t.Error("empty panel should render empty")
	}
}

func TestContentPanel_GutterMarginIsOneSpace(t *testing.T) {
	panel := ui.NewContentPanel()

	output := panel.Render(ui.RenderParams{
		Rows:       []string{"content"},
		Width:      20,
		Height:     1,
		TotalItems: 10,
		ViewOffset: 0,
		Cursor:     -1,
	})
	lines := strings.Split(output, "\n")
	stripped := ansi.Strip(lines[0])

	runes := []rune(stripped)
	gutterIdx := len(runes) - 1
	if gutterIdx > 0 && runes[gutterIdx-1] != ' ' {
		t.Errorf("expected one space before gutter char, got %q", string(runes[gutterIdx-1]))
	}

	if lipgloss.Width(lines[0]) != 20 {
		t.Errorf("expected visual width = 20, got %d", lipgloss.Width(lines[0]))
	}
}

func TestContentPanel_BuildRowGenerator(t *testing.T) {
	panel := ui.NewContentPanel()
	items := []string{"zero", "one", "two", "three", "four", "five"}

	output := panel.Render(ui.RenderParams{
		Width:      20,
		Height:     3,
		TotalItems: len(items),
		ViewOffset: 2,
		Cursor:     -1,
		BuildRow: func(index int) string {
			return items[index]
		},
	})
	lines := strings.Split(output, "\n")

	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	stripped := ansi.Strip(lines[0])
	if !strings.HasPrefix(stripped, "two") {
		t.Errorf("first line should be item at ViewOffset=2, got %q", stripped)
	}
}

func TestContentPanel_BuildRowStopsAtHeight(t *testing.T) {
	panel := ui.NewContentPanel()
	callCount := 0

	panel.Render(ui.RenderParams{
		Width:      20,
		Height:     3,
		TotalItems: 100,
		ViewOffset: 0,
		Cursor:     -1,
		BuildRow: func(index int) string {
			callCount++
			return "row"
		},
	})
	if callCount != 3 {
		t.Errorf("expected BuildRow called 3 times (Height), got %d", callCount)
	}
}

func TestContentPanel_HandleKeyLeft(t *testing.T) {
	panel := ui.NewContentPanel()
	panel.HandleKey(tea.KeyMsg{Type: tea.KeyRight})
	if panel.HScroll() != 10 {
		t.Fatalf("expected hscroll=10 after right, got %d", panel.HScroll())
	}
	panel.HandleKey(tea.KeyMsg{Type: tea.KeyLeft})
	if panel.HScroll() != 0 {
		t.Errorf("expected hscroll=0 after left, got %d", panel.HScroll())
	}
}

func TestContentPanel_HandleKeyLeftClampsToZero(t *testing.T) {
	panel := ui.NewContentPanel()
	panel.HandleKey(tea.KeyMsg{Type: tea.KeyLeft})
	if panel.HScroll() != 0 {
		t.Errorf("hscroll should not go negative, got %d", panel.HScroll())
	}
}

func TestContentPanel_HandleKeyWrapToggle(t *testing.T) {
	panel := ui.NewContentPanel()
	panel.HandleKey(tea.KeyMsg{Type: tea.KeyRight}) // hscroll = 10
	panel.HandleKey(tea.KeyMsg{Type: tea.KeyCtrlW})
	if !panel.WrapMode() {
		t.Error("ctrl+w should enable wrap mode")
	}
	if panel.HScroll() != 0 {
		t.Error("wrap toggle should reset hscroll")
	}
}

func TestContentPanel_RightIgnoredInWrapMode(t *testing.T) {
	panel := ui.NewContentPanel()
	panel.HandleKey(tea.KeyMsg{Type: tea.KeyCtrlW}) // wrap on
	panel.HandleKey(tea.KeyMsg{Type: tea.KeyRight})
	if panel.HScroll() != 0 {
		t.Error("right key should be ignored in wrap mode")
	}
}

// --- Benchmarks ---

func BenchmarkContentPanel_RenderFlat(b *testing.B) {
	panel := ui.NewContentPanel()
	panel.SelectedStyle = func(s string, w int) string {
		return lipgloss.NewStyle().Width(w).Render(s)
	}

	rows := make([]string, 100)
	for i := range rows {
		rows[i] = fmt.Sprintf("[ ] + module.vpc.aws_route53_record.api_gateway_%d [medium]", i)
	}

	params := ui.RenderParams{
		Rows:       rows[:20],
		Width:      120,
		Height:     20,
		TotalItems: 100,
		ViewOffset: 0,
		Cursor:     5,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		panel.Render(params)
	}
}

func BenchmarkContentPanel_RenderWithStyledContent(b *testing.B) {
	panel := ui.NewContentPanel()
	panel.SelectedStyle = func(s string, w int) string {
		return lipgloss.NewStyle().Width(w).Render(s)
	}

	green := lipgloss.NewStyle().Foreground(lipgloss.Color("#50fa7b"))
	yellow := lipgloss.NewStyle().Foreground(lipgloss.Color("#f1fa8c"))
	rows := make([]string, 20)
	for i := range rows {
		sym := green.Render("+")
		risk := yellow.Render("[medium]")
		rows[i] = fmt.Sprintf("[ ] %s module.vpc.aws_route53_record.api_gateway_%d %s", sym, i, risk)
	}

	params := ui.RenderParams{
		Rows:       rows,
		Width:      120,
		Height:     20,
		TotalItems: 100,
		ViewOffset: 0,
		Cursor:     10,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		panel.Render(params)
	}
}

func BenchmarkContentPanel_RenderWithHScroll(b *testing.B) {
	panel := ui.NewContentPanel()
	panel.HandleKey(tea.KeyMsg{Type: tea.KeyRight}) // hscroll = 10

	rows := make([]string, 20)
	for i := range rows {
		rows[i] = fmt.Sprintf("[ ] + module.very_long_module_name.module.another_module.aws_cloudwatch_metric_alarm.resource_%d", i)
	}

	params := ui.RenderParams{
		Rows:       rows,
		Width:      80,
		Height:     20,
		TotalItems: 100,
		ViewOffset: 0,
		Cursor:     -1,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		panel.Render(params)
	}
}

func BenchmarkContentPanel_BuildRowGenerator(b *testing.B) {
	panel := ui.NewContentPanel()
	panel.SelectedStyle = func(s string, w int) string {
		return lipgloss.NewStyle().Width(w).Render(s)
	}

	params := ui.RenderParams{
		Width:      120,
		Height:     20,
		TotalItems: 1000,
		ViewOffset: 500,
		Cursor:     510,
		BuildRow: func(index int) string {
			return fmt.Sprintf("[ ] + aws_instance.server_%d  aws_instance", index)
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		panel.Render(params)
	}
}
