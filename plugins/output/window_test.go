package output

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
	keyShift = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}}
)

const gutterGlyphs = "▲▼┃│"

func doneWith(t *testing.T, outputs []sdk.OutputValue) *Plugin {
	t.Helper()
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = sdk.StatusDone
	p.outputs = outputs
	p.filtered = outputs
	return p
}

func numberedOutputs(n int) []sdk.OutputValue {
	outputs := make([]sdk.OutputValue, n)
	for i := range outputs {
		outputs[i] = sdk.OutputValue{Name: fmt.Sprintf("output_%02d", i), Type: "string", Value: "v"}
	}
	return outputs
}

func longOutput() []sdk.OutputValue {
	return []sdk.OutputValue{{Name: "endpoint", Type: "string", Value: strings.Repeat("https://example.invalid/", 5) + "tailmarker"}}
}

func press(p *Plugin, msgs ...tea.Msg) {
	for _, msg := range msgs {
		p.stack.Update(msg)
	}
}

func TestView_GivenMoreRowsThanFit_ShouldKeepTheLastRowVisibleAndDrawTheGutter(t *testing.T) {
	p := doneWith(t, numberedOutputs(30))
	p.View(60, 14)

	press(p, keyShift)
	view := ansi.Strip(p.View(60, 14))

	if !strings.Contains(view, "output_29") {
		t.Fatalf("cursor on the last row should show it:\n%s", view)
	}
	if !strings.ContainsAny(view, gutterGlyphs) {
		t.Fatalf("more rows than fit should draw the scroll gutter:\n%s", view)
	}
	if got := len(strings.Split(view, "\n")); got > 14 {
		t.Fatalf("view is %d lines, want at most the height 14:\n%s", got, view)
	}
	if !strings.Contains(view, "30 outputs") {
		t.Fatalf("count line should stay visible:\n%s", view)
	}
}

func TestView_GivenFilterLineAndMoreRowsThanFit_ShouldKeepTheLastMatchAndTheCountVisible(t *testing.T) {
	p := doneWith(t, numberedOutputs(30))
	p.View(60, 14)
	p.SetFilter("output")
	p.filtering = false

	press(p, keyShift)
	view := ansi.Strip(p.View(60, 14))

	for _, want := range []string{"output_29", "ᗊ: output", "30 outputs"} {
		if !strings.Contains(view, want) {
			t.Fatalf("view should show %q:\n%s", want, view)
		}
	}
	if got := len(strings.Split(view, "\n")); got > 14 {
		t.Fatalf("view is %d lines, want at most the height 14:\n%s", got, view)
	}
}

func TestView_GivenMultiLineValueOnTheLastRow_ShouldShowAllItsLines(t *testing.T) {
	outputs := numberedOutputs(30)
	outputs[29].Value = "first line\nsecond line\nthird line"
	p := doneWith(t, outputs)
	p.View(60, 14)

	press(p, keyShift)
	view := ansi.Strip(p.View(60, 14))

	for _, want := range []string{"output_29", "first line", "second line", "third line"} {
		if !strings.Contains(view, want) {
			t.Fatalf("last row should show %q:\n%s", want, view)
		}
	}
}

func TestView_GivenCtrlW_ShouldWrapLongRowsAndToggleBack(t *testing.T) {
	p := doneWith(t, longOutput())
	start := ansi.Strip(p.View(40, 12))
	if strings.Contains(start, "tailmarker") {
		t.Fatalf("unwrapped view should clip the long row:\n%s", start)
	}

	press(p, keyWrap)
	if wrapped := ansi.Strip(p.View(40, 12)); !strings.Contains(strings.ReplaceAll(wrapped, "\n", ""), "tailmarker") {
		t.Fatalf("wrap on should show the end of the long row:\n%s", wrapped)
	}

	press(p, keyWrap)
	if got := ansi.Strip(p.View(40, 12)); got != start {
		t.Fatalf("wrap off should restore the clipped view:\n%s\nwant:\n%s", got, start)
	}
}

func TestView_GivenRightAndLeft_ShouldPanTheListAndBack(t *testing.T) {
	p := doneWith(t, longOutput())
	start := ansi.Strip(p.View(40, 12))

	press(p, keyRight)
	if panned := ansi.Strip(p.View(40, 12)); panned == start {
		t.Fatalf("right should pan the list:\n%s", panned)
	}
	press(p, keyLeft)
	if got := ansi.Strip(p.View(40, 12)); got != start {
		t.Fatalf("left should return to the unscrolled view:\n%s\nwant:\n%s", got, start)
	}
}

func TestUpdate_GivenWrapKeysWithNothingListed_ShouldChangeNothing(t *testing.T) {
	p := doneWith(t, nil)

	press(p, keyWrap, keyRight)

	if p.panel.WrapMode() || p.panel.HScroll() != 0 {
		t.Fatalf("wrap = %v, hscroll = %d, want neither changed", p.panel.WrapMode(), p.panel.HScroll())
	}
}

func TestHints_GivenListedOutputs_ShouldShowWrapWithItsMode(t *testing.T) {
	p := doneWith(t, numberedOutputs(2))

	if got, want := p.Stack().Hints(), (sdk.HintSetFilter | sdk.HintSetWrap | sdk.HintSetRefresh | sdk.HintSetQuit).Hints(); !slices.Equal(got, want) {
		t.Fatalf("Hints() = %v, want %v", got, want)
	}
	press(p, keyWrap)
	want := (sdk.HintSetFilter | sdk.HintSetWrap | sdk.HintSetRefresh | sdk.HintSetQuit).Hints(sdk.HintSetOpts{WrapMode: true})
	if got := p.Stack().Hints(); !slices.Equal(got, want) {
		t.Fatalf("Hints() after ctrl+w = %v, want %v", got, want)
	}
}

func TestHints_GivenNoListedOutputs_ShouldNotOfferWrap(t *testing.T) {
	p := doneWith(t, nil)
	for _, hint := range p.Stack().Hints() {
		if hint.Key == "^w" {
			t.Fatalf("Hints() = %v offers wrap with nothing to wrap", p.Stack().Hints())
		}
	}
}

func TestUpdate_GivenReload_ShouldStartTheWindowAtTheTop(t *testing.T) {
	p := doneWith(t, numberedOutputs(30))
	p.View(60, 14)
	press(p, keyShift)
	p.View(60, 14)

	p.Refresh()
	outputs := map[string]sdk.OutputValue{}
	for _, o := range numberedOutputs(30) {
		outputs[o.Name] = o
	}
	p.Update(OutputResultMsg{Outputs: outputs})

	if view := ansi.Strip(p.View(60, 14)); !strings.Contains(view, "output_00") {
		t.Fatalf("a reload should start at the first row:\n%s", view)
	}
}

func TestUpdate_GivenPannedListAndReloadOrReset_ShouldPanBackToTheStart(t *testing.T) {
	for _, tc := range []struct {
		name  string
		start func(p *Plugin)
	}{
		{"refresh", func(p *Plugin) { p.Refresh() }},
		{"reset", func(p *Plugin) { p.reset() }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := doneWith(t, numberedOutputs(3))
			press(p, keyRight)
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

func multiLineValue(lines int) string {
	parts := make([]string, lines)
	for i := range parts {
		parts[i] = fmt.Sprintf("value line %02d", i)
	}
	return strings.Join(parts, "\n")
}

func TestView_GivenOneMultiLineValueTallerThanTheListBudget_ShouldStillShowItsLastLine(t *testing.T) {
	// The list gives up the count line before it hides a line of the value, as it did before ContentPanel.
	for _, lines := range []int{11, 12} {
		t.Run(fmt.Sprintf("%d lines", lines), func(t *testing.T) {
			p := doneWith(t, []sdk.OutputValue{{Name: "only", Type: "string", Value: multiLineValue(lines)}})

			view := ansi.Strip(p.View(60, 12))

			if want := fmt.Sprintf("value line %02d", lines-1); !strings.Contains(view, want) {
				t.Fatalf("view should reach %q:\n%s", want, view)
			}
			if got := len(strings.Split(view, "\n")); got > 12 {
				t.Fatalf("view is %d lines, want at most the height 12:\n%s", got, view)
			}
		})
	}
}

func TestView_GivenOneMultiLineValueThatFits_ShouldKeepTheCountLine(t *testing.T) {
	p := doneWith(t, []sdk.OutputValue{{Name: "only", Type: "string", Value: multiLineValue(10)}})

	view := ansi.Strip(p.View(60, 12))

	for _, want := range []string{"value line 09", "1 outputs"} {
		if !strings.Contains(view, want) {
			t.Fatalf("view should show %q:\n%s", want, view)
		}
	}
}

func TestFilter_GivenPannedList_ShouldPanBackWhenTheFilterReplacesTheList(t *testing.T) {
	long := strings.Repeat("x", 40)
	for _, tc := range []struct {
		name string
		keys []tea.Msg
	}{
		{"opening the filter", []tea.Msg{tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}}},
		{"applying a query", []tea.Msg{tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := doneWith(t, []sdk.OutputValue{
				{Name: "long", Type: "string", Value: long},
				{Name: "short", Type: "string", Value: "v"},
			})
			press(p, keyRight, keyRight)
			if p.panel.HScroll() == 0 {
				t.Fatal("right should pan the list")
			}

			press(p, tc.keys...)

			if got := p.panel.HScroll(); got != 0 {
				t.Fatalf("hscroll = %d, want 0 once the filter replaces the list", got)
			}
			if view := ansi.Strip(p.View(60, 12)); !strings.Contains(view, "short") {
				t.Fatalf("view should show the list from its left edge:\n%s", view)
			}
		})
	}
}

func TestFilter_GivenScrolledList_ShouldStartTheWindowAtTheTopWhenTheFilterReplacesTheList(t *testing.T) {
	p := doneWith(t, numberedOutputs(30))
	p.View(60, 14)
	press(p, keyShift)
	p.View(60, 14)

	press(p, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})

	if view := ansi.Strip(p.View(60, 14)); !strings.Contains(view, "output_00") {
		t.Fatalf("opening the filter should show the first row:\n%s", view)
	}
}

func TestUpdate_GivenMultiLineValueTallerThanTheView_ShouldReachItsLastLineByKeyboard(t *testing.T) {
	for _, tc := range []struct {
		name     string
		down, up tea.KeyMsg
	}{
		{"j and k", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}},
		{"arrows", tea.KeyMsg{Type: tea.KeyDown}, tea.KeyMsg{Type: tea.KeyUp}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := doneWith(t, []sdk.OutputValue{
				{Name: "tall", Type: "string", Value: multiLineValue(30)},
				{Name: "next", Type: "string", Value: "v"},
			})
			view := ansi.Strip(p.View(60, 12))

			for range 30 {
				if strings.Contains(view, "value line 29") {
					break
				}
				press(p, tc.down)
				view = ansi.Strip(p.View(60, 12))
			}
			if !strings.Contains(view, "value line 29") || p.Selected() != 0 {
				t.Fatalf("down should scroll the selected value to its last line, selected = %d:\n%s", p.Selected(), view)
			}

			press(p, tc.down)
			p.View(60, 12)
			if p.Selected() != 1 {
				t.Fatalf("down with the last line in view should select the next output, selected = %d", p.Selected())
			}

			press(p, tc.up)
			view = ansi.Strip(p.View(60, 12))
			if p.Selected() != 0 || !strings.Contains(view, "tall") {
				t.Fatalf("up should select the tall value from its first line, selected = %d:\n%s", p.Selected(), view)
			}
		})
	}
}

func TestUpdate_GivenOnlyOneValueTallerThanTheView_ShouldReachItsLastLineWithJ(t *testing.T) {
	p := doneWith(t, []sdk.OutputValue{{Name: "only", Type: "string", Value: multiLineValue(30)}})
	p.View(60, 12)

	for range 30 {
		press(p, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
		p.View(60, 12)
	}

	if view := ansi.Strip(p.View(60, 12)); !strings.Contains(view, "value line 29") {
		t.Fatalf("j should reach the value's last line:\n%s", view)
	}
}

func TestUpdate_GivenGOrShiftGOnAnOversizedValue_ShouldJumpToTheListsFirstOrLastLine(t *testing.T) {
	keyJ := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	keyG := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}}
	tall := sdk.OutputValue{Name: "tall", Type: "string", Value: multiLineValue(30)}
	short := sdk.OutputValue{Name: "short", Type: "string", Value: "v"}
	for _, tc := range []struct {
		name      string
		outputs   []sdk.OutputValue
		keys      []tea.KeyMsg
		wantFirst string
		wantLast  string
	}{
		{"g on a scrolled first value", []sdk.OutputValue{tall, short}, []tea.KeyMsg{keyJ, keyJ, keyJ, keyG}, " tall", ""},
		{"G on an oversized last value", []sdk.OutputValue{short, tall}, []tea.KeyMsg{keyShift}, "", "value line 29"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := doneWith(t, tc.outputs)
			p.View(60, 12)
			for _, key := range tc.keys {
				press(p, key)
				p.View(60, 12)
			}

			view := ansi.Strip(p.View(60, 12))
			lines := strings.Split(view, "\n")
			if !strings.Contains(lines[0], tc.wantFirst) || !strings.Contains(view, tc.wantLast) {
				t.Fatalf("want first line %q and %q in view:\n%s", tc.wantFirst, tc.wantLast, view)
			}
		})
	}
}
