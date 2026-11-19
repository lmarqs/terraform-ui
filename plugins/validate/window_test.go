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
	keyG     = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}}
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
