package validate

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/sdktest"
)

var (
	keyWrap  = tea.KeyMsg{Type: tea.KeyCtrlW}
	keyLeft  = tea.KeyMsg{Type: tea.KeyLeft}
	keyRight = tea.KeyMsg{Type: tea.KeyRight}
	keyEnter = tea.KeyMsg{Type: tea.KeyEnter}
	keyShift = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}}
)

const gutterGlyphs = "▲▼┃│"

func doneWith(t *testing.T, diags []sdk.Diagnostic) *Plugin {
	t.Helper()
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = sdk.StatusDone
	p.diagnostics = diags
	return p
}

func numberedErrors(n int) []sdk.Diagnostic {
	diags := make([]sdk.Diagnostic, n)
	for i := range diags {
		diags[i] = sdk.Diagnostic{Severity: "error", Summary: fmt.Sprintf("error number %02d", i)}
	}
	return diags
}

func TestView_GivenMoreRowsThanFit_ShouldKeepTheLastRowVisibleAndDrawTheGutter(t *testing.T) {
	p := doneWith(t, numberedErrors(30))
	p.View(60, 14)

	p.Update(keyShift)
	view := ansi.Strip(p.View(60, 14))

	if !strings.Contains(view, "error number 29") {
		t.Fatalf("cursor on the last row should show it:\n%s", view)
	}
	if !strings.ContainsAny(view, gutterGlyphs) {
		t.Fatalf("more rows than fit should draw the scroll gutter:\n%s", view)
	}
	if got := len(strings.Split(view, "\n")); got > 14 {
		t.Fatalf("view is %d lines, want at most the height 14:\n%s", got, view)
	}
	if !strings.Contains(view, "30 error(s)") {
		t.Fatalf("summary line should stay visible:\n%s", view)
	}
}

func TestView_GivenExpandedLastRow_ShouldShowItsDetailBelowIt(t *testing.T) {
	diags := numberedErrors(30)
	diags[29].Detail = "first detail line\nsecond detail line"
	p := doneWith(t, diags)
	p.View(60, 14)

	p.Update(keyShift)
	p.Update(keyEnter)
	view := ansi.Strip(p.View(60, 14))

	for _, want := range []string{"error number 29", "first detail line", "second detail line"} {
		if !strings.Contains(view, want) {
			t.Fatalf("expanded last row should show %q:\n%s", want, view)
		}
	}
}

func TestView_GivenCtrlW_ShouldWrapLongRowsAndToggleBack(t *testing.T) {
	long := "an unsupported argument was given " + strings.Repeat("and then some more words ", 4) + "tailmarker"
	p := doneWith(t, []sdk.Diagnostic{{Severity: "error", Summary: long}})
	start := ansi.Strip(p.View(40, 12))
	if strings.Contains(start, "tailmarker") {
		t.Fatalf("unwrapped view should clip the long row:\n%s", start)
	}

	p.Update(keyWrap)
	if wrapped := ansi.Strip(p.View(40, 12)); !strings.Contains(strings.ReplaceAll(wrapped, "\n", ""), "tailmarker") {
		t.Fatalf("wrap on should show the end of the long row:\n%s", wrapped)
	}

	p.Update(keyWrap)
	if got := ansi.Strip(p.View(40, 12)); got != start {
		t.Fatalf("wrap off should restore the clipped view:\n%s\nwant:\n%s", got, start)
	}
}

func TestView_GivenRightAndLeft_ShouldPanTheListAndBack(t *testing.T) {
	long := "an unsupported argument was given " + strings.Repeat("and then some more words ", 4) + "tailmarker"
	p := doneWith(t, []sdk.Diagnostic{{Severity: "error", Summary: long}})
	start := ansi.Strip(p.View(40, 12))

	p.Update(keyRight)
	if panned := ansi.Strip(p.View(40, 12)); panned == start {
		t.Fatalf("right should pan the list:\n%s", panned)
	}
	p.Update(keyLeft)
	if got := ansi.Strip(p.View(40, 12)); got != start {
		t.Fatalf("left should return to the unscrolled view:\n%s\nwant:\n%s", got, start)
	}
}

func TestUpdate_GivenWrapKeysWithNothingListed_ShouldChangeNothing(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status sdk.Status
		diags  []sdk.Diagnostic
	}{
		{"loading", sdk.StatusLoading, nil},
		{"valid", sdk.StatusDone, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := doneWith(t, tc.diags)
			p.status = tc.status

			p.Update(keyWrap)
			p.Update(keyRight)

			if p.panel.WrapMode() || p.panel.HScroll() != 0 {
				t.Fatalf("wrap = %v, hscroll = %d, want neither changed", p.panel.WrapMode(), p.panel.HScroll())
			}
		})
	}
}

func TestHints_GivenDiagnostics_ShouldShowWrapWithItsMode(t *testing.T) {
	p := doneWith(t, numberedErrors(2))

	if got, want := p.Hints(), (sdk.HintSetInspect | sdk.HintSetWrap | sdk.HintSetRefresh | sdk.HintSetQuit).Hints(); !slices.Equal(got, want) {
		t.Fatalf("Hints() = %v, want %v", got, want)
	}
	p.Update(keyWrap)
	want := (sdk.HintSetInspect | sdk.HintSetWrap | sdk.HintSetRefresh | sdk.HintSetQuit).Hints(sdk.HintSetOpts{WrapMode: true})
	if got := p.Hints(); !slices.Equal(got, want) {
		t.Fatalf("Hints() after ctrl+w = %v, want %v", got, want)
	}
}

func TestHints_GivenValidConfiguration_ShouldNotOfferWrap(t *testing.T) {
	p := doneWith(t, nil)
	for _, hint := range p.Hints() {
		if hint.Key == "^w" {
			t.Fatalf("Hints() = %v offers wrap with nothing to wrap", p.Hints())
		}
	}
}

func TestUpdate_GivenNewRun_ShouldStartTheWindowAtTheTop(t *testing.T) {
	p := doneWith(t, numberedErrors(30))
	p.View(60, 14)
	p.Update(keyShift)
	p.View(60, 14)

	p.Refresh()
	p.Update(ValidateResultMsg{Diagnostics: numberedErrors(30)})

	if view := ansi.Strip(p.View(60, 14)); !strings.Contains(view, "error number 00") {
		t.Fatalf("a new run should start at the first row:\n%s", view)
	}
}

func TestUpdate_GivenPannedListAndNewRunOrReset_ShouldPanBackToTheStart(t *testing.T) {
	for _, tc := range []struct {
		name  string
		start func(p *Plugin)
	}{
		{"refresh", func(p *Plugin) { p.Refresh() }},
		{"reset", func(p *Plugin) { p.reset() }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := doneWith(t, numberedErrors(3))
			p.Update(keyRight)
			if p.panel.HScroll() == 0 {
				t.Fatal("right should pan the list")
			}

			tc.start(p)

			if got := p.panel.HScroll(); got != 0 {
				t.Fatalf("hscroll = %d, want 0", got)
			}
		})
	}
}

func detailLines(n int) string {
	lines := make([]string, n)
	for i := range lines {
		lines[i] = fmt.Sprintf("detail line %02d", i)
	}
	return strings.Join(lines, "\n")
}

func TestView_GivenOneExpandedDiagnosticTallerThanTheListBudget_ShouldStillShowItsLastLine(t *testing.T) {
	// The list gives up the summary line before it hides a line of the block, as it did before ContentPanel.
	for _, lines := range []int{10, 11} {
		t.Run(fmt.Sprintf("%d detail lines", lines), func(t *testing.T) {
			p := doneWith(t, []sdk.Diagnostic{{Severity: "error", Summary: "only one", Detail: detailLines(lines)}})
			p.Update(keyEnter)

			view := ansi.Strip(p.View(60, 12))

			if want := fmt.Sprintf("detail line %02d", lines-1); !strings.Contains(view, want) {
				t.Fatalf("view should reach %q:\n%s", want, view)
			}
			if got := len(strings.Split(view, "\n")); got > 12 {
				t.Fatalf("view is %d lines, want at most the height 12:\n%s", got, view)
			}
		})
	}
}

func TestView_GivenOneExpandedDiagnosticThatFits_ShouldKeepTheSummaryLine(t *testing.T) {
	p := doneWith(t, []sdk.Diagnostic{{Severity: "error", Summary: "only one", Detail: detailLines(9)}})
	p.Update(keyEnter)

	view := ansi.Strip(p.View(60, 12))

	for _, want := range []string{"detail line 08", "1 error(s)"} {
		if !strings.Contains(view, want) {
			t.Fatalf("view should show %q:\n%s", want, view)
		}
	}
}

func TestUpdate_GivenExpandedDiagnosticTallerThanTheView_ShouldReachItsLastLineByKeyboard(t *testing.T) {
	for _, tc := range []struct {
		name     string
		down, up tea.KeyMsg
	}{
		{"j and k", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}},
		{"arrows", tea.KeyMsg{Type: tea.KeyDown}, tea.KeyMsg{Type: tea.KeyUp}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := doneWith(t, []sdk.Diagnostic{
				{Severity: "error", Summary: "tall one", Detail: detailLines(30)},
				{Severity: "error", Summary: "next one"},
			})
			p.Update(keyEnter)
			view := ansi.Strip(p.View(60, 12))

			for range 30 {
				if strings.Contains(view, "detail line 29") {
					break
				}
				p.Update(tc.down)
				view = ansi.Strip(p.View(60, 12))
			}
			if !strings.Contains(view, "detail line 29") || p.Selected() != 0 {
				t.Fatalf("down should scroll the selected diagnostic to its last line, selected = %d:\n%s", p.Selected(), view)
			}

			p.Update(tc.down)
			p.View(60, 12)
			if p.Selected() != 1 {
				t.Fatalf("down with the last line in view should select the next diagnostic, selected = %d", p.Selected())
			}

			p.Update(tc.up)
			view = ansi.Strip(p.View(60, 12))
			if p.Selected() != 0 || !strings.Contains(view, "tall one") {
				t.Fatalf("up should select the tall diagnostic from its first line, selected = %d:\n%s", p.Selected(), view)
			}
		})
	}
}

func TestUpdate_GivenOnlyOneDiagnosticTallerThanTheView_ShouldReachItsLastLineWithJ(t *testing.T) {
	p := doneWith(t, []sdk.Diagnostic{{Severity: "error", Summary: "only one", Detail: detailLines(30)}})
	p.Update(keyEnter)
	p.View(60, 12)

	for range 30 {
		p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
		p.View(60, 12)
	}

	if view := ansi.Strip(p.View(60, 12)); !strings.Contains(view, "detail line 29") {
		t.Fatalf("j should reach the diagnostic's last line:\n%s", view)
	}
}
