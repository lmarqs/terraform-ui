package ui_test

import (
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/lmarqs/terraform-ui/pkg/sdk/ui"
)

func TestContentPanel_RendersRowsWithGutter(t *testing.T) {
	panel := ui.NewContentPanel()

	output := panel.Render(ui.RenderParams{
		Rows:         []string{"alpha", "bravo", "charlie"},
		Width:        20,
		Height:       3,
		TotalItems:   10,
		Cursor:       -1,
		ScrollOffset: 0,
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
	panel.HandleKey(tea.KeyMsg{Type: tea.KeyRight}) // hscroll = 10

	output := panel.Render(ui.RenderParams{
		Rows:       []string{"0123456789abcdefghij"},
		Width:      10,
		Height:     5,
		TotalItems: 1,
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
		Rows:         []string{short, long, "mid"},
		Width:        20,
		Height:       3,
		TotalItems:   10,
		Cursor:       -1,
		ScrollOffset: 0,
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
		Cursor:     -1,
	})

	stripped := ansi.Strip(output)
	joined := strings.ReplaceAll(stripped, "\n", "")
	if !strings.Contains(joined, "this-is-a-very-long-row-that-exceeds-width") {
		t.Errorf("wrap mode should preserve full content across lines, got %q", stripped)
	}

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
		Cursor:     -1,
	})

	lines := strings.Split(output, "\n")
	if len(lines) > 3 {
		t.Errorf("wrap mode should respect height budget, got %d lines", len(lines))
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
		Rows:         []string{"content"},
		Width:        20,
		Height:       1,
		TotalItems:   10,
		Cursor:       -1,
		ScrollOffset: 0,
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

func TestNeedsGutter(t *testing.T) {
	if !ui.NeedsGutter(100, 20) {
		t.Error("should need gutter when items > height")
	}
	if ui.NeedsGutter(5, 20) {
		t.Error("should not need gutter when items <= height")
	}
	if ui.NeedsGutter(20, 20) {
		t.Error("should not need gutter when items == height")
	}
}

func TestContentWidth(t *testing.T) {
	if got := ui.ContentWidth(80, true); got != 78 {
		t.Errorf("with gutter: expected 78, got %d", got)
	}
	if got := ui.ContentWidth(80, false); got != 80 {
		t.Errorf("without gutter: expected 80, got %d", got)
	}
}

func TestContentPanel_HandleKey_GivenUnrecognizedKey_ShouldReturnFalse(t *testing.T) {
	panel := ui.NewContentPanel()
	consumed := panel.HandleKey(tea.KeyMsg{Type: tea.KeyUp})
	if consumed {
		t.Error("unrecognized key should not be consumed")
	}
}

func TestContentPanel_ResetScroll_ShouldSetHScrollToZero(t *testing.T) {
	panel := ui.NewContentPanel()
	panel.HandleKey(tea.KeyMsg{Type: tea.KeyRight}) // hscroll = 10
	if panel.HScroll() != 10 {
		t.Fatalf("precondition: expected hscroll=10, got %d", panel.HScroll())
	}
	panel.ResetScroll()
	if panel.HScroll() != 0 {
		t.Errorf("ResetScroll should set hscroll to 0, got %d", panel.HScroll())
	}
}

func TestContentPanel_Render_GivenWrapModeFillingHeight_ShouldBreakAtHeightBoundary(t *testing.T) {
	panel := ui.NewContentPanel()
	panel.HandleKey(tea.KeyMsg{Type: tea.KeyCtrlW}) // enable wrap

	// First row wraps to fill entire height (3 lines from 30 chars at width 10),
	// second row should be skipped due to linesUsed >= Height.
	output := panel.Render(ui.RenderParams{
		Rows:       []string{strings.Repeat("a", 30), "second-row-should-not-appear"},
		Width:      10,
		Height:     3,
		TotalItems: 2,
		Cursor:     -1,
	})
	if strings.Contains(output, "second") {
		t.Error("second row should be skipped when first row wraps fill the height budget")
	}
	lines := strings.Split(output, "\n")
	if len(lines) != 3 {
		t.Errorf("expected exactly 3 lines, got %d", len(lines))
	}
}

func TestContentPanel_Render_GivenZeroContentWidth_ShouldNotPanic(t *testing.T) {
	panel := ui.NewContentPanel()

	// Width=2, TotalItems > Height forces gutter, contentWidth = 2-2 = 0
	output := panel.Render(ui.RenderParams{
		Rows:       []string{"hello", "world"},
		Width:      2,
		Height:     1,
		TotalItems: 5,
		Cursor:     -1,
	})
	// Should not panic; output may be minimal but should not crash
	_ = output
}

func TestContentPanel_Render_GivenWrapModeWithZeroContentWidth_ShouldNotPanic(t *testing.T) {
	panel := ui.NewContentPanel()
	panel.HandleKey(tea.KeyMsg{Type: tea.KeyCtrlW}) // enable wrap

	// Width=2, TotalItems > Height → contentWidth = 0 → wrapStyled with width=0
	output := panel.Render(ui.RenderParams{
		Rows:       []string{"hello"},
		Width:      2,
		Height:     1,
		TotalItems: 5,
		Cursor:     -1,
	})
	_ = output
}

func TestContentPanel_Render_GivenNilSelectedStyle_ShouldApplyDefaultHighlight(t *testing.T) {
	panel := ui.NewContentPanel()
	// Do NOT set SelectedStyle — leave it nil

	output := panel.Render(ui.RenderParams{
		Rows:       []string{"alpha", "bravo"},
		Width:      20,
		Height:     5,
		TotalItems: 2,
		Cursor:     0,
	})
	lines := strings.Split(output, "\n")
	// Default style applies lipgloss width — the first line should be padded to contentWidth
	if lipgloss.Width(lines[0]) != 20 {
		t.Errorf("default selected style should pad to content width, got visual width %d", lipgloss.Width(lines[0]))
	}
}

func TestContentPanel_Render_GivenNegativeContentWidth_ShouldReturnEmptyTruncation(t *testing.T) {
	panel := ui.NewContentPanel()

	// Width=1, TotalItems > Height forces gutter → contentWidth = 1-2 = -1
	output := panel.Render(ui.RenderParams{
		Rows:       []string{"hello"},
		Width:      1,
		Height:     1,
		TotalItems: 5,
		Cursor:     -1,
	})
	// truncateStyled with maxWidth=-1 returns ""
	// The result should still produce output (gutter line) or empty
	_ = output
}

func TestContentPanel_Render_GivenWrapModeCursorOnFirstWrappedLine_ShouldHighlightFirstLine(t *testing.T) {
	panel := ui.NewContentPanel()
	panel.HandleKey(tea.KeyMsg{Type: tea.KeyCtrlW}) // enable wrap
	panel.SelectedStyle = func(s string, w int) string {
		return "[SEL]" + s
	}

	output := panel.Render(ui.RenderParams{
		Rows:       []string{"this-is-a-long-row-for-wrapping"},
		Width:      10,
		Height:     10,
		TotalItems: 1,
		Cursor:     0,
	})
	lines := strings.Split(output, "\n")
	if !strings.Contains(lines[0], "[SEL]") {
		t.Errorf("first wrapped line should have selection style, got %q", lines[0])
	}
	if len(lines) > 1 && strings.Contains(lines[1], "[SEL]") {
		t.Error("only first wrapped line should have selection style")
	}
}

func TestContentPanel_Render_GivenMoreRowsThanHeight_ShouldStopAtHeightLimit(t *testing.T) {
	panel := ui.NewContentPanel()

	output := panel.Render(ui.RenderParams{
		Rows:       []string{"row0", "row1", "row2", "row3", "row4"},
		Width:      20,
		Height:     2,
		TotalItems: 5,
		Cursor:     -1,
	})
	lines := strings.Split(output, "\n")
	// Only 2 lines should appear (height=2), rows 2-4 should be truncated
	if len(lines) != 2 {
		t.Errorf("expected 2 lines (height limit), got %d", len(lines))
	}
	stripped := ansi.Strip(output)
	if strings.Contains(stripped, "row2") {
		t.Error("row2 should not appear — height budget exhausted")
	}
}

func TestContentPanel_Render_GivenZeroHeight_ShouldReturnEmpty(t *testing.T) {
	panel := ui.NewContentPanel()

	output := panel.Render(ui.RenderParams{
		Rows:       []string{"hello", "world"},
		Width:      20,
		Height:     0,
		TotalItems: 2,
		Cursor:     -1,
	})
	if output != "" {
		t.Errorf("expected empty output with zero height, got %q", output)
	}
}

func TestContentPanel_Render_GivenStyledContentWithHScroll_ShouldPreserveTrailingEscapes(t *testing.T) {
	panel := ui.NewContentPanel()
	// Scroll by 10 to exercise truncateLeft with ANSI where visibleCount < n
	panel.HandleKey(tea.KeyMsg{Type: tea.KeyRight}) // hscroll = 10

	// Use a styled string where ANSI codes appear at the start (before cut point)
	styled := "\x1b[31m" + "0123456789abcdef" + "\x1b[0m"
	output := panel.Render(ui.RenderParams{
		Rows:       []string{styled},
		Width:      6,
		Height:     5,
		TotalItems: 1,
		Cursor:     -1,
	})
	stripped := ansi.Strip(output)
	if !strings.HasPrefix(stripped, "abcdef") {
		t.Errorf("expected scrolled styled content starting with 'abcdef', got %q", stripped)
	}
}

// --- Benchmarks ---

func BenchmarkContentPanel_RenderFlat(b *testing.B) {
	panel := ui.NewContentPanel()
	panel.SelectedStyle = func(s string, w int) string {
		return lipgloss.NewStyle().Width(w).Render(s)
	}

	rows := make([]string, 20)
	for i := range rows {
		rows[i] = fmt.Sprintf("[ ] + module.vpc.aws_route53_record.api_gateway_%d [medium]", i)
	}

	params := ui.RenderParams{
		Rows:         rows,
		Width:        120,
		Height:       20,
		TotalItems:   100,
		Cursor:       5,
		ScrollOffset: 0,
	}

	for b.Loop() {
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
		Rows:         rows,
		Width:        120,
		Height:       20,
		TotalItems:   100,
		Cursor:       10,
		ScrollOffset: 0,
	}

	for b.Loop() {
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
		Rows:         rows,
		Width:        80,
		Height:       20,
		TotalItems:   100,
		Cursor:       -1,
		ScrollOffset: 0,
	}

	for b.Loop() {
		panel.Render(params)
	}
}

func wrapPanel() *ui.ContentPanel {
	panel := ui.NewContentPanel()
	panel.HandleKey(tea.KeyMsg{Type: tea.KeyCtrlW})
	panel.SelectedStyle = func(s string, w int) string { return "[SEL]" + s }
	return panel
}

// threeLineRows returns rows of 30 characters, each wrapping to three lines at width 10.
func threeLineRows(letters string) []string {
	rows := make([]string, len(letters))
	for i, r := range letters {
		rows[i] = strings.Repeat(string(r), 30)
	}
	return rows
}

func TestContentPanel_Render_GivenWrapModeCursorBelowHeight_ShouldDrawCursorRowFirstLineSelected(t *testing.T) {
	panel := wrapPanel()

	output := panel.Render(ui.RenderParams{
		Rows:       threeLineRows("abc"),
		Width:      12,
		Height:     5,
		TotalItems: 3,
		Cursor:     2,
	})

	if !strings.Contains(output, "[SEL]"+strings.Repeat("c", 10)) {
		t.Fatalf("cursor row's first wrapped line should be drawn selected, got:\n%s", output)
	}
	if lines := strings.Split(output, "\n"); len(lines) != 5 {
		t.Errorf("expected 5 lines, got %d:\n%s", len(lines), output)
	}
}

func TestContentPanel_RenderWindow_GivenWrapModeCursorOnLastRow_ShouldPutThumbAtBottomOfTrack(t *testing.T) {
	panel := wrapPanel()

	output, offset := panel.RenderWindow(ui.RenderParams{
		Rows:   threeLineRows("abcd"),
		Width:  12,
		Height: 8,
		Cursor: 3,
	})

	lines := strings.Split(output, "\n")
	if len(lines) != 8 {
		t.Fatalf("expected 8 lines, got %d:\n%s", len(lines), output)
	}
	if !strings.HasSuffix(lines[1], "│") || !strings.HasSuffix(lines[6], "┃") || !strings.HasSuffix(lines[7], "▼") {
		t.Errorf("thumb should sit at the bottom of the track, got:\n%s", output)
	}
	if !strings.Contains(output, "[SEL]"+strings.Repeat("d", 10)) {
		t.Errorf("cursor row's first wrapped line should be drawn selected, got:\n%s", output)
	}
	if offset != 1 {
		t.Errorf("offset = %d, want 1 (line 4 of 12 wrapped lines is in row b)", offset)
	}
}

func TestContentPanel_RenderWindow_GivenWrapModeCursorAboveOffset_ShouldScrollUpToCursorRow(t *testing.T) {
	panel := wrapPanel()

	output, offset := panel.RenderWindow(ui.RenderParams{
		Rows:         threeLineRows("abcd"),
		Width:        12,
		Height:       5,
		Cursor:       1,
		ScrollOffset: 3,
	})

	if offset != 1 {
		t.Errorf("offset = %d, want 1 (row b)", offset)
	}
	if first := strings.Split(output, "\n")[0]; !strings.HasPrefix(first, "[SEL]"+strings.Repeat("b", 10)) {
		t.Errorf("first line should be the cursor row, got %q", first)
	}
}

func TestContentPanel_RenderWindow_GivenWrapModeBlockTallerThanHeight_ShouldShowBlockStart(t *testing.T) {
	panel := wrapPanel()

	output, offset := panel.RenderWindow(ui.RenderParams{
		Rows:      threeLineRows("abcd"),
		Width:     12,
		Height:    4,
		Cursor:    1,
		CursorEnd: 4,
	})

	if offset != 1 {
		t.Errorf("offset = %d, want 1 (block start)", offset)
	}
	if first := strings.Split(output, "\n")[0]; !strings.HasPrefix(first, "[SEL]") {
		t.Errorf("first line should be the cursor row, got %q", first)
	}
}

func TestContentPanel_RenderWindow_GivenWrapModeRowsFitWidth_ShouldUseFullWidthWithoutGutter(t *testing.T) {
	panel := wrapPanel()

	output, offset := panel.RenderWindow(ui.RenderParams{
		Rows:   []string{strings.Repeat("a", 12), "b"},
		Width:  12,
		Height: 5,
		Cursor: -1,
	})

	if want := strings.Repeat("a", 12) + "\nb"; output != want || offset != 0 {
		t.Errorf("RenderWindow = %q, %d, want %q, 0", output, offset, want)
	}
}

func TestContentPanel_RenderWindow_GivenWrapModeOffsetPastEnd_ShouldClampToLastPage(t *testing.T) {
	panel := wrapPanel()

	output, offset := panel.RenderWindow(ui.RenderParams{
		Rows:         threeLineRows("ab"),
		Width:        12,
		Height:       5,
		Cursor:       -1,
		ScrollOffset: 9,
	})

	lines := strings.Split(output, "\n")
	if offset != 0 || len(lines) != 5 || !strings.HasPrefix(lines[1], "a") || !strings.HasPrefix(lines[2], "b") {
		t.Errorf("offset = %d, want 0 and the last page, from line 1 of row a:\n%s", offset, output)
	}
}

func TestContentPanel_RenderWindow_GivenWrapOff_ShouldWindowRowsAroundCursorBlock(t *testing.T) {
	rows := []string{"r0", "r1", "r2", "r3", "r4", "r5"}
	tests := []struct {
		name       string
		params     ui.RenderParams
		wantOffset int
		wantFirst  string
	}{
		{"cursor below window scrolls down to block end", ui.RenderParams{Rows: rows, Width: 10, Height: 3, Cursor: 3, CursorEnd: 5}, 2, "r2"},
		{"cursor above window scrolls up to cursor", ui.RenderParams{Rows: rows, Width: 10, Height: 3, Cursor: 1, ScrollOffset: 3}, 1, "[SEL]r1"},
		{"block taller than height shows its start", ui.RenderParams{Rows: rows, Width: 10, Height: 2, Cursor: 1, CursorEnd: 5}, 1, "[SEL]r1"},
		{"offset past end clamps to last page", ui.RenderParams{Rows: rows, Width: 10, Height: 3, Cursor: -1, ScrollOffset: 9}, 3, "r3"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			panel := ui.NewContentPanel()
			panel.SelectedStyle = func(s string, w int) string { return "[SEL]" + s }

			output, offset := panel.RenderWindow(tt.params)

			want := panel.Render(ui.RenderParams{
				Rows:         rows[offset:min(len(rows), offset+tt.params.Height)],
				Width:        tt.params.Width,
				Height:       tt.params.Height,
				TotalItems:   len(rows),
				Cursor:       tt.params.Cursor - offset,
				ScrollOffset: offset,
			})
			if offset != tt.wantOffset || output != want {
				t.Fatalf("RenderWindow = %q, %d, want %q, %d", output, offset, want, tt.wantOffset)
			}
			if first := strings.Split(output, "\n")[0]; !strings.HasPrefix(first, tt.wantFirst) {
				t.Errorf("first line = %q, want prefix %q", first, tt.wantFirst)
			}
		})
	}
}

func TestContentPanel_RenderWindow_GivenZeroHeight_ShouldRenderNothing(t *testing.T) {
	for _, wrap := range []bool{false, true} {
		panel := ui.NewContentPanel()
		if wrap {
			panel.HandleKey(tea.KeyMsg{Type: tea.KeyCtrlW})
		}
		if output, offset := panel.RenderWindow(ui.RenderParams{Rows: []string{"a"}, Width: 10, Cursor: 0}); output != "" || offset != 0 {
			t.Errorf("wrap=%v: RenderWindow = %q, %d, want empty, 0", wrap, output, offset)
		}
	}
}

func TestContentPanel_Render_GivenWrapModeScrolledPastLinesAbove_ShouldDrawGutter(t *testing.T) {
	panel := wrapPanel()

	output := panel.Render(ui.RenderParams{
		Rows:       threeLineRows("abc"),
		Width:      12,
		Height:     5,
		TotalItems: 3,
		Cursor:     2,
	})

	lines := strings.Split(output, "\n")
	if len(lines) != 5 || !strings.HasSuffix(lines[0], "▲") || !strings.HasSuffix(lines[4], "▼") {
		t.Fatalf("wrapped lines hidden above the cursor should draw the gutter, got:\n%s", output)
	}
}

func TestContentPanel_Render_GivenWrapModeWithoutCursor_ShouldKeepRowGutterRule(t *testing.T) {
	panel := wrapPanel()

	output := panel.Render(ui.RenderParams{
		Rows:       []string{"12345678", "second-long"},
		Width:      8,
		Height:     2,
		TotalItems: 2,
		Cursor:     -1,
	})

	if want := "12345678\nsecond-l"; output != want {
		t.Fatalf("Render = %q, want %q", output, want)
	}
}

// twoLineRows returns n rows of 16 characters, each wrapping to two lines at width 10
// once the gutter takes its two columns.
func twoLineRows(n int) []string {
	rows := make([]string, n)
	for i := range rows {
		rows[i] = fmt.Sprintf("row%02d", i) + strings.Repeat(".", 11)
	}
	return rows
}

func TestContentPanel_RenderWindow_GivenWrapToggledOnAndOff_ShouldKeepTheOffsetInRows(t *testing.T) {
	panel := ui.NewContentPanel()
	params := ui.RenderParams{Rows: twoLineRows(20), Width: 12, Height: 4, Cursor: 10}

	_, offset := panel.RenderWindow(params)
	if offset != 7 {
		t.Fatalf("wrap off offset = %d, want 7 (cursor at the bottom)", offset)
	}

	panel.HandleKey(tea.KeyMsg{Type: tea.KeyCtrlW})
	params.ScrollOffset = offset
	wrapped, offset := panel.RenderWindow(params)
	if offset != 9 || !strings.HasPrefix(ansi.Strip(wrapped), "row09") {
		t.Fatalf("wrap on offset = %d, want 9, the window's top row:\n%s", offset, wrapped)
	}

	panel.HandleKey(tea.KeyMsg{Type: tea.KeyCtrlW})
	params.ScrollOffset = offset
	output, offset := panel.RenderWindow(params)
	lines := strings.Split(ansi.Strip(output), "\n")
	if offset != 9 || !strings.HasPrefix(lines[0], "row09") || !strings.HasPrefix(lines[1], "row10") {
		t.Fatalf("wrap off again offset = %d, want 9 with the cursor on the second line:\n%s", offset, output)
	}
}

func TestContentPanel_RenderWindow_GivenWrapRoundTripWithRowsThatFit_ShouldRestoreTheWrapOffOutput(t *testing.T) {
	panel := ui.NewContentPanel()
	rows := make([]string, 20)
	for i := range rows {
		rows[i] = fmt.Sprintf("row%02d", i)
	}
	params := ui.RenderParams{Rows: rows, Width: 12, Height: 4, Cursor: 10}

	before, offset := panel.RenderWindow(params)
	for range 2 {
		panel.HandleKey(tea.KeyMsg{Type: tea.KeyCtrlW})
		params.ScrollOffset = offset
		_, offset = panel.RenderWindow(params)
	}
	params.ScrollOffset = offset
	if after, _ := panel.RenderWindow(params); after != before {
		t.Fatalf("wrap off after a round trip:\n%s\nwant:\n%s", after, before)
	}
}

func TestContentPanel_RenderWindow_GivenWidthChangeWithWrapOn_ShouldKeepTheTopRow(t *testing.T) {
	panel := wrapPanel()
	params := ui.RenderParams{Rows: twoLineRows(20), Width: 12, Height: 4, Cursor: 10, ScrollOffset: 9}

	_, offset := panel.RenderWindow(params)
	params.Width, params.ScrollOffset = 40, offset
	output, offset := panel.RenderWindow(params)

	if offset != 9 || !strings.HasPrefix(output, "row09") {
		t.Fatalf("offset after widening = %d, want 9 (same top row):\n%s", offset, output)
	}
}

func TestContentPanel_RenderWindow_GivenWrapModeOffsetPassedBack_ShouldReadItAsTheTopRow(t *testing.T) {
	panel := wrapPanel()
	params := ui.RenderParams{Rows: twoLineRows(20), Width: 12, Height: 4, Cursor: 4, ScrollOffset: 4}

	for frame := range 2 {
		output, offset := panel.RenderWindow(params)

		lines := strings.Split(ansi.Strip(output), "\n")
		if offset != 4 || len(lines) != 4 || !strings.HasPrefix(lines[0], "[SEL]row04") || !strings.HasPrefix(lines[2], "row05") {
			t.Fatalf("frame %d: offset = %d, want 4 with rows 4 and 5 on screen:\n%s", frame, offset, output)
		}
		params.ScrollOffset = offset
	}
}

func TestContentPanel_Render_GivenWideCharacters_ShouldKeepEveryCharacterAsValidUTF8(t *testing.T) {
	red := lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000"))
	tests := []struct {
		name   string
		wrap   bool
		scroll bool
		row    string
		width  int
		want   string
	}{
		{"WrapSplitsAfterTheCellsThatFit", true, false, "abcde界z", 6, "abcde界z"},
		{"WrapKeepsAWideCharacterThatStartsTheRow", true, false, "界界界", 4, "界界界"},
		{"WrapKeepsStyledWideCharacters", true, false, red.Render("abcde界z"), 6, "abcde界z"},
		{"ScrollLeftDropsWholeWideCharacters", false, true, "界界界界界界界界界界xyz", 13, "界界界界界xyz"},
		{"ScrollLeftDropsAWideCharacterStraddlingTheOffset", false, true, "a界界界界界界界界界xyz", 12, "界界界界xyz"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			panel := ui.NewContentPanel()
			if tt.wrap {
				panel.HandleKey(tea.KeyMsg{Type: tea.KeyCtrlW})
			}
			if tt.scroll {
				panel.HandleKey(tea.KeyMsg{Type: tea.KeyRight}) // hscroll = 10
			}

			output := panel.Render(ui.RenderParams{
				Rows:       []string{tt.row},
				Width:      tt.width,
				Height:     5,
				TotalItems: 1,
				Cursor:     -1,
			})

			if !utf8.ValidString(output) {
				t.Fatalf("output is not valid UTF-8: %q", output)
			}
			got := strings.ReplaceAll(ansi.Strip(output), "\n", "")
			if got != tt.want {
				t.Errorf("content = %q, want %q", got, tt.want)
			}
			for i, line := range strings.Split(output, "\n") {
				if lipgloss.Width(line) > tt.width {
					t.Errorf("line %d is %d cells wide, over %d: %q", i, lipgloss.Width(line), tt.width, line)
				}
			}
		})
	}
}

func TestContentPanel_Render_GivenPanOffsetPastARow_ShouldShowTheRowsTailOrWholeText(t *testing.T) {
	long := "0123456789abcdefghijklmnop" // 26 cells
	tests := []struct {
		name       string
		pans       int
		rows       []string
		width      int
		totalItems int
		want       []string
	}{
		{"ShortRowShowsWholeText", 1, []string{long, "short"}, 10, 2, []string{"abcdefghij", "short"}},
		{"RowWiderThanViewShowsItsTail", 1, []string{long, "abcdefghijklm"}, 10, 2, []string{"abcdefghij", "defghijklm"}},
		{"RowEndingAtTheOffsetShowsItsTail", 1, []string{long, "0123456789"}, 10, 2, []string{"abcdefghij", "0123456789"}},
		{"PanBeyondEveryRowStopsAtEachRowsEnd", 5, []string{long, "abcdefghijklm"}, 10, 2, []string{"ghijklmnop", "defghijklm"}},
		{"WideCharactersKeepTheirCells", 1, []string{long, "界界界界界界"}, 10, 2, []string{"abcdefghij", "界界界界界"}},
		{"GutterNarrowsTheView", 1, []string{long, "abcdefghijklm"}, 10, 5, []string{"abcdefgh", "fghijklm"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			panel := ui.NewContentPanel()
			for range tt.pans {
				panel.HandleKey(tea.KeyMsg{Type: tea.KeyRight})
			}

			output := panel.Render(ui.RenderParams{
				Rows:       tt.rows,
				Width:      tt.width,
				Height:     2,
				TotalItems: tt.totalItems,
				Cursor:     1,
			})

			lines := strings.Split(output, "\n")
			if len(lines) != len(tt.want) {
				t.Fatalf("got %d lines, want %d:\n%s", len(lines), len(tt.want), output)
			}
			for i, want := range tt.want {
				if got := strings.TrimRight(ansi.Strip(lines[i]), " ▲▼┃│"); got != want {
					t.Errorf("line %d = %q, want %q", i, got, want)
				}
			}
		})
	}
}
