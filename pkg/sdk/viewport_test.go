package sdk

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewViewport(t *testing.T) {
	vp := NewViewport(80, 24)

	if vp.Width != 80 {
		t.Fatalf("expected width 80, got %d", vp.Width)
	}
	if vp.Height != 24 {
		t.Fatalf("expected height 24, got %d", vp.Height)
	}
	if vp.ScrollX != 0 {
		t.Fatalf("expected ScrollX 0, got %d", vp.ScrollX)
	}
	if vp.ScrollY != 0 {
		t.Fatalf("expected ScrollY 0, got %d", vp.ScrollY)
	}
	if vp.WrapEnabled {
		t.Fatal("expected WrapEnabled false")
	}
	if len(vp.Lines) != 0 {
		t.Fatalf("expected empty Lines, got %d", len(vp.Lines))
	}
}

func TestSetContent(t *testing.T) {
	vp := NewViewport(80, 24)
	vp.ScrollY = 5
	vp.ScrollX = 3

	lines := []string{"line1", "line2", "line3"}
	vp.SetContent(lines)

	if len(vp.Lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(vp.Lines))
	}
	if vp.Lines[0] != "line1" {
		t.Fatalf("expected line1, got %s", vp.Lines[0])
	}
	if vp.ScrollY != 0 {
		t.Fatalf("expected ScrollY reset to 0, got %d", vp.ScrollY)
	}
	if vp.ScrollX != 0 {
		t.Fatalf("expected ScrollX reset to 0, got %d", vp.ScrollX)
	}
}

func TestSetContentString(t *testing.T) {
	vp := NewViewport(80, 24)
	vp.ScrollY = 10
	vp.ScrollX = 5

	vp.SetContentString("alpha\nbeta\ngamma")

	if len(vp.Lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(vp.Lines))
	}
	if vp.Lines[1] != "beta" {
		t.Fatalf("expected beta, got %s", vp.Lines[1])
	}
	if vp.ScrollY != 0 {
		t.Fatalf("expected ScrollY reset to 0, got %d", vp.ScrollY)
	}
	if vp.ScrollX != 0 {
		t.Fatalf("expected ScrollX reset to 0, got %d", vp.ScrollX)
	}
}

func TestSetSize(t *testing.T) {
	vp := NewViewport(80, 24)
	vp.SetSize(120, 40)

	if vp.Width != 120 {
		t.Fatalf("expected width 120, got %d", vp.Width)
	}
	if vp.Height != 40 {
		t.Fatalf("expected height 40, got %d", vp.Height)
	}
}

func TestRender_ContentShorterThanViewport(t *testing.T) {
	vp := NewViewport(80, 10)
	vp.SetContent([]string{"line1", "line2", "line3"})

	result := vp.Render()
	expected := "line1\nline2\nline3"
	if result != expected {
		t.Fatalf("expected %q, got %q", expected, result)
	}
}

func TestRender_ContentTallerThanViewport(t *testing.T) {
	vp := NewViewport(80, 3)
	lines := make([]string, 10)
	for i := range lines {
		lines[i] = strings.Repeat("x", i+1)
	}
	vp.SetContent(lines)

	// At scroll 0, should see first 3 lines
	result := vp.Render()
	expected := "x\nxx\nxxx"
	if result != expected {
		t.Fatalf("expected %q, got %q", expected, result)
	}

	// Scroll down
	vp.ScrollY = 2
	result = vp.Render()
	expected = "xxx\nxxxx\nxxxxx"
	if result != expected {
		t.Fatalf("expected %q, got %q", expected, result)
	}
}

func TestRender_HorizontalScroll_Truncation(t *testing.T) {
	vp := NewViewport(5, 3)
	vp.SetContent([]string{"abcdefghij", "1234567890", "short"})

	// No horizontal scroll — truncates to width
	result := vp.Render()
	lines := strings.Split(result, "\n")
	if lines[0] != "abcde" {
		t.Fatalf("expected abcde, got %q", lines[0])
	}
	if lines[1] != "12345" {
		t.Fatalf("expected 12345, got %q", lines[1])
	}
	if lines[2] != "short" {
		t.Fatalf("expected short, got %q", lines[2])
	}
}

func TestRender_HorizontalScroll_Panned(t *testing.T) {
	vp := NewViewport(5, 3)
	vp.SetContent([]string{"abcdefghij", "1234567890", "short"})

	// With horizontal scroll (set before first render)
	vp.ScrollX = 3
	result := vp.Render()
	lines := strings.Split(result, "\n")
	if lines[0] != "defgh" {
		t.Fatalf("expected defgh, got %q", lines[0])
	}
	if lines[1] != "45678" {
		t.Fatalf("expected 45678, got %q", lines[1])
	}
	if lines[2] != "rt" {
		t.Fatalf("expected rt, got %q", lines[2])
	}
}

func TestRender_HorizontalScroll_PastLineLength(t *testing.T) {
	vp := NewViewport(10, 2)
	vp.SetContent([]string{"hi", "hello world"})

	// Scroll past the short line length
	vp.ScrollX = 5
	result := vp.Render()
	lines := strings.Split(result, "\n")
	// "hi" has len 2, ScrollX 5 >= len, so it becomes ""
	if lines[0] != "" {
		t.Fatalf("expected empty for short line, got %q", lines[0])
	}
	// "hello world" at offset 5 = " world", truncated to 10 = " world"
	if lines[1] != " world" {
		t.Fatalf("expected ' world', got %q", lines[1])
	}
}

func TestRender_WrapEnabled(t *testing.T) {
	vp := NewViewport(5, 10)
	vp.WrapEnabled = true
	vp.SetContent([]string{"abcdefghij", "short"})

	result := vp.Render()
	lines := strings.Split(result, "\n")

	// "abcdefghij" wraps to "abcde", "fghij"
	// "short" fits in 5
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines after wrap, got %d: %v", len(lines), lines)
	}
	if lines[0] != "abcde" {
		t.Fatalf("expected abcde, got %q", lines[0])
	}
	if lines[1] != "fghij" {
		t.Fatalf("expected fghij, got %q", lines[1])
	}
	if lines[2] != "short" {
		t.Fatalf("expected short, got %q", lines[2])
	}
}

func TestRender_EmptyContent(t *testing.T) {
	vp := NewViewport(80, 24)
	result := vp.Render()
	if result != "" {
		t.Fatalf("expected empty string, got %q", result)
	}
}

func TestRender_ZeroSizeViewport(t *testing.T) {
	vp := NewViewport(0, 0)
	vp.SetContent([]string{"hello", "world"})

	// Height 0 means endIdx = 0, visible is empty slice
	result := vp.Render()
	if result != "" {
		t.Fatalf("expected empty string for zero-height viewport, got %q", result)
	}
}

func keyMsg(key string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
}

func specialKeyMsg(keyType tea.KeyType) tea.KeyMsg {
	return tea.KeyMsg{Type: keyType}
}

func TestHandleKey_UpDown(t *testing.T) {
	vp := NewViewport(80, 3)
	lines := make([]string, 10)
	for i := range lines {
		lines[i] = "line"
	}
	vp.SetContent(lines)

	// Down
	consumed := vp.HandleKey(specialKeyMsg(tea.KeyDown))
	if !consumed {
		t.Fatal("expected down key to be consumed")
	}
	if vp.ScrollY != 1 {
		t.Fatalf("expected ScrollY 1, got %d", vp.ScrollY)
	}

	// Up
	consumed = vp.HandleKey(specialKeyMsg(tea.KeyUp))
	if !consumed {
		t.Fatal("expected up key to be consumed")
	}
	if vp.ScrollY != 0 {
		t.Fatalf("expected ScrollY 0, got %d", vp.ScrollY)
	}

	// Up at top stays at 0
	consumed = vp.HandleKey(specialKeyMsg(tea.KeyUp))
	if !consumed {
		t.Fatal("expected up key to be consumed")
	}
	if vp.ScrollY != 0 {
		t.Fatalf("expected ScrollY 0, got %d", vp.ScrollY)
	}
}

func TestHandleKey_DownClamp(t *testing.T) {
	vp := NewViewport(80, 3)
	lines := make([]string, 5)
	for i := range lines {
		lines[i] = "line"
	}
	vp.SetContent(lines)

	// Max scroll = 5 - 3 = 2
	for i := 0; i < 10; i++ {
		vp.HandleKey(specialKeyMsg(tea.KeyDown))
	}
	if vp.ScrollY != 2 {
		t.Fatalf("expected ScrollY clamped to 2, got %d", vp.ScrollY)
	}
}

func TestHandleKey_LeftRight(t *testing.T) {
	vp := NewViewport(10, 3)
	vp.SetContent([]string{strings.Repeat("x", 50)})

	// Right increases by 10
	consumed := vp.HandleKey(specialKeyMsg(tea.KeyRight))
	if !consumed {
		t.Fatal("expected right key to be consumed")
	}
	if vp.ScrollX != 10 {
		t.Fatalf("expected ScrollX 10, got %d", vp.ScrollX)
	}

	// Left decreases by 10
	consumed = vp.HandleKey(specialKeyMsg(tea.KeyLeft))
	if !consumed {
		t.Fatal("expected left key to be consumed")
	}
	if vp.ScrollX != 0 {
		t.Fatalf("expected ScrollX 0, got %d", vp.ScrollX)
	}

	// Left at 0 stays at 0
	vp.HandleKey(specialKeyMsg(tea.KeyLeft))
	if vp.ScrollX != 0 {
		t.Fatalf("expected ScrollX 0, got %d", vp.ScrollX)
	}
}

func TestHandleKey_RightClamp(t *testing.T) {
	vp := NewViewport(10, 3)
	vp.SetContent([]string{strings.Repeat("x", 25)})

	// Max scroll = 25 - 10 = 15
	for i := 0; i < 10; i++ {
		vp.HandleKey(specialKeyMsg(tea.KeyRight))
	}
	if vp.ScrollX != 15 {
		t.Fatalf("expected ScrollX clamped to 15, got %d", vp.ScrollX)
	}
}

func TestHandleKey_GoTop(t *testing.T) {
	vp := NewViewport(80, 3)
	lines := make([]string, 20)
	for i := range lines {
		lines[i] = "line"
	}
	vp.SetContent(lines)
	vp.ScrollY = 10

	consumed := vp.HandleKey(keyMsg("g"))
	if !consumed {
		t.Fatal("expected g key to be consumed")
	}
	if vp.ScrollY != 0 {
		t.Fatalf("expected ScrollY 0, got %d", vp.ScrollY)
	}
}

func TestHandleKey_GoBottom(t *testing.T) {
	vp := NewViewport(80, 3)
	lines := make([]string, 20)
	for i := range lines {
		lines[i] = "line"
	}
	vp.SetContent(lines)

	consumed := vp.HandleKey(keyMsg("G"))
	if !consumed {
		t.Fatal("expected G key to be consumed")
	}
	// Max scroll = 20 - 3 = 17
	if vp.ScrollY != 17 {
		t.Fatalf("expected ScrollY 17, got %d", vp.ScrollY)
	}
}

func TestHandleKey_ToggleWrap(t *testing.T) {
	vp := NewViewport(80, 10)
	vp.SetContent([]string{"hello"})
	vp.ScrollY = 5
	vp.ScrollX = 3

	consumed := vp.HandleKey(keyMsg("w"))
	if !consumed {
		t.Fatal("expected w key to be consumed")
	}
	if !vp.WrapEnabled {
		t.Fatal("expected WrapEnabled true")
	}
	if vp.ScrollY != 0 {
		t.Fatalf("expected ScrollY reset to 0, got %d", vp.ScrollY)
	}
	if vp.ScrollX != 0 {
		t.Fatalf("expected ScrollX reset to 0, got %d", vp.ScrollX)
	}

	// Toggle again
	vp.HandleKey(keyMsg("w"))
	if vp.WrapEnabled {
		t.Fatal("expected WrapEnabled false after second toggle")
	}
}

func TestHandleKey_UnrecognizedKey(t *testing.T) {
	vp := NewViewport(80, 24)
	vp.SetContent([]string{"hello"})

	consumed := vp.HandleKey(keyMsg("z"))
	if consumed {
		t.Fatal("expected unrecognized key to return false")
	}

	consumed = vp.HandleKey(keyMsg("a"))
	if consumed {
		t.Fatal("expected unrecognized key to return false")
	}
}

func TestScrollInfo_NoScrollNeeded(t *testing.T) {
	vp := NewViewport(80, 10)
	vp.SetContent([]string{"line1", "line2", "line3"})

	info := vp.ScrollInfo()
	if info != "" {
		t.Fatalf("expected empty scroll info, got %q", info)
	}
}

func TestScrollInfo_ScrollNeeded(t *testing.T) {
	vp := NewViewport(80, 3)
	lines := make([]string, 10)
	for i := range lines {
		lines[i] = "line"
	}
	vp.SetContent(lines)

	// At top: position 1 of (10-3+1)=8
	info := vp.ScrollInfo()
	if info != "[1/8]" {
		t.Fatalf("expected [1/8], got %q", info)
	}

	vp.ScrollY = 3
	info = vp.ScrollInfo()
	if info != "[4/8]" {
		t.Fatalf("expected [4/8], got %q", info)
	}
}

func TestAtTop(t *testing.T) {
	vp := NewViewport(80, 3)
	lines := make([]string, 10)
	for i := range lines {
		lines[i] = "line"
	}
	vp.SetContent(lines)

	if !vp.AtTop() {
		t.Fatal("expected AtTop true at scroll 0")
	}

	vp.ScrollY = 1
	if vp.AtTop() {
		t.Fatal("expected AtTop false at scroll 1")
	}
}

func TestAtBottom(t *testing.T) {
	vp := NewViewport(80, 3)
	lines := make([]string, 10)
	for i := range lines {
		lines[i] = "line"
	}
	vp.SetContent(lines)

	if vp.AtBottom() {
		t.Fatal("expected AtBottom false at scroll 0")
	}

	// Max scroll = 10 - 3 = 7
	vp.ScrollY = 7
	if !vp.AtBottom() {
		t.Fatal("expected AtBottom true at max scroll")
	}
}

func TestAtBottom_ContentFitsViewport(t *testing.T) {
	vp := NewViewport(80, 10)
	vp.SetContent([]string{"line1", "line2"})

	// When content fits, always at bottom
	if !vp.AtBottom() {
		t.Fatal("expected AtBottom true when content fits in viewport")
	}
}

func TestScrollClamp_ScrollYPastEnd(t *testing.T) {
	vp := NewViewport(80, 5)
	lines := make([]string, 8)
	for i := range lines {
		lines[i] = "line"
	}
	vp.SetContent(lines)

	// Manually set scroll past max (max = 8 - 5 = 3)
	vp.ScrollY = 100
	result := vp.Render()
	// Should clamp to max and show last 5 lines
	resultLines := strings.Split(result, "\n")
	if len(resultLines) != 5 {
		t.Fatalf("expected 5 visible lines after clamp, got %d", len(resultLines))
	}
	// ScrollY should be clamped
	if vp.ScrollY != 3 {
		t.Fatalf("expected ScrollY clamped to 3, got %d", vp.ScrollY)
	}
}

func TestTotalLines_NoWrap(t *testing.T) {
	vp := NewViewport(80, 10)
	vp.SetContent([]string{"a", "b", "c"})
	if vp.TotalLines() != 3 {
		t.Fatalf("expected 3, got %d", vp.TotalLines())
	}
}

func TestTotalLines_WithWrap(t *testing.T) {
	vp := NewViewport(5, 10)
	vp.WrapEnabled = true
	// "abcdefghij" wraps to 2 lines at width 5, "hi" stays 1 line
	vp.SetContent([]string{"abcdefghij", "hi"})
	if vp.TotalLines() != 3 {
		t.Fatalf("expected 3, got %d", vp.TotalLines())
	}
}

func TestRender_WrapWithZeroWidth(t *testing.T) {
	vp := NewViewport(0, 10)
	vp.WrapEnabled = true
	vp.SetContent([]string{"hello", "world"})

	// wrapLines with width 0 returns lines as-is
	result := vp.Render()
	if result != "hello\nworld" {
		t.Fatalf("expected hello\\nworld, got %q", result)
	}
}

func TestHandleKey_CtrlW_ShouldToggleWrap(t *testing.T) {
	vp := NewViewport(80, 10)
	vp.SetContent([]string{"hello"})
	vp.ScrollY = 5
	vp.ScrollX = 3

	consumed := vp.HandleKey(tea.KeyMsg{Type: tea.KeyCtrlW})
	if !consumed {
		t.Fatal("expected ctrl+w key to be consumed")
	}
	if !vp.WrapEnabled {
		t.Fatal("expected WrapEnabled true after ctrl+w")
	}
	if vp.ScrollY != 0 {
		t.Fatalf("expected ScrollY reset to 0, got %d", vp.ScrollY)
	}
	if vp.ScrollX != 0 {
		t.Fatalf("expected ScrollX reset to 0, got %d", vp.ScrollX)
	}
}

func TestScrollInfo_WhenMaxScrollIsZero_ShouldReturnEmpty(t *testing.T) {
	vp := NewViewport(80, 10)
	vp.SetContent([]string{"line1", "line2", "line3", "line4", "line5",
		"line6", "line7", "line8", "line9", "line10"})

	// total (10) - height (10) = 0
	info := vp.ScrollInfo()
	if info != "" {
		t.Fatalf("expected empty scroll info when maxScroll=0, got %q", info)
	}
}

func TestHandleKey_DownWhenContentFitsViewport_ShouldNotScroll(t *testing.T) {
	vp := NewViewport(80, 10)
	vp.SetContent([]string{"line1", "line2"})

	vp.HandleKey(specialKeyMsg(tea.KeyDown))
	if vp.ScrollY != 0 {
		t.Fatalf("expected ScrollY 0 when content fits, got %d", vp.ScrollY)
	}
}

func TestHandleKey_GoBottom_WhenContentFits_ShouldStayAtZero(t *testing.T) {
	vp := NewViewport(80, 10)
	vp.SetContent([]string{"line1", "line2"})

	vp.HandleKey(keyMsg("G"))
	if vp.ScrollY != 0 {
		t.Fatalf("expected ScrollY 0 when content fits viewport, got %d", vp.ScrollY)
	}
}

func TestHandleKey_RightWhenContentNarrowerThanViewport_ShouldNotScroll(t *testing.T) {
	vp := NewViewport(80, 5)
	vp.SetContent([]string{"short"})

	vp.HandleKey(specialKeyMsg(tea.KeyRight))
	if vp.ScrollX != 0 {
		t.Fatalf("expected ScrollX 0 when content narrower than viewport, got %d", vp.ScrollX)
	}
}
