package phantom

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
	keyDown  = tea.KeyMsg{Type: tea.KeyDown}
	keyEnter = tea.KeyMsg{Type: tea.KeyEnter}
)

const gutterGlyphs = "▲▼┃│"

func numberedPhantoms(n int) []PhantomChange {
	phantoms := make([]PhantomChange, n)
	for i := range phantoms {
		phantoms[i] = PhantomChange{
			Change:      sdk.PlanChange{Resource: sdk.Resource{Address: fmt.Sprintf("aws_s3_bucket.number%02d", i)}, IsPhantom: true},
			Explanation: "tag ordering",
			Attributes:  []sdk.AttributeDiff{{Key: "tags", OldValue: "a", NewValue: "b"}},
		}
	}
	return phantoms
}

func doneWith(t *testing.T, phantoms []PhantomChange) *Plugin {
	t.Helper()
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = sdk.StatusDone
	p.phantoms = phantoms
	p.total = len(phantoms)
	return p
}

func summaryOf(phantoms []PhantomChange) *sdk.PlanSummary {
	summary := &sdk.PlanSummary{}
	for _, pc := range phantoms {
		summary.Changes = append(summary.Changes, pc.Change)
	}
	return summary
}

func TestView_GivenMorePhantomsThanFit_ShouldKeepTheLastRowVisibleAndDrawTheGutter(t *testing.T) {
	p := doneWith(t, numberedPhantoms(30))
	p.View(80, 16)

	for range 29 {
		p.Update(keyDown)
	}
	view := ansi.Strip(p.View(80, 16))

	if !strings.Contains(view, "aws_s3_bucket.number29") {
		t.Fatalf("cursor on the last row should show it:\n%s", view)
	}
	if !strings.ContainsAny(view, gutterGlyphs) {
		t.Fatalf("more rows than fit should draw the scroll gutter:\n%s", view)
	}
	if got := len(strings.Split(view, "\n")); got > 16 {
		t.Fatalf("view is %d lines, want at most the height 16:\n%s", got, view)
	}
	if !strings.Contains(view, "Detected 30 phantom change(s)") {
		t.Fatalf("banner should stay visible:\n%s", view)
	}
}

func TestView_GivenExpandedLastPhantom_ShouldShowItsDetailsBelowIt(t *testing.T) {
	p := doneWith(t, numberedPhantoms(30))
	p.View(80, 16)

	for range 29 {
		p.Update(keyDown)
	}
	p.Update(keyEnter)
	view := ansi.Strip(p.View(80, 16))

	for _, want := range []string{"aws_s3_bucket.number29", "Reason: tag ordering"} {
		if !strings.Contains(view, want) {
			t.Fatalf("expanded last phantom should show %q:\n%s", want, view)
		}
	}
}

func TestView_GivenCtrlW_ShouldWrapLongRowsAndToggleBack(t *testing.T) {
	phantoms := numberedPhantoms(1)
	phantoms[0].Change.Resource.Address = "aws_s3_bucket." + strings.Repeat("deeply_nested_", 6) + "tailmarker"
	p := doneWith(t, phantoms)
	start := ansi.Strip(p.View(50, 12))
	if strings.Contains(strings.ReplaceAll(start, "\n", ""), "tailmarker") {
		t.Fatalf("unwrapped view should clip the long row:\n%s", start)
	}

	p.Update(keyWrap)
	if wrapped := ansi.Strip(p.View(50, 12)); !strings.Contains(strings.ReplaceAll(wrapped, "\n", ""), "tailmarker") {
		t.Fatalf("wrap on should show the end of the long row:\n%s", wrapped)
	}

	p.Update(keyWrap)
	if got := ansi.Strip(p.View(50, 12)); got != start {
		t.Fatalf("wrap off should restore the clipped view:\n%s\nwant:\n%s", got, start)
	}
}

func TestView_GivenRightAndLeft_ShouldPanTheListAndBack(t *testing.T) {
	phantoms := numberedPhantoms(1)
	phantoms[0].Change.Resource.Address = "aws_s3_bucket." + strings.Repeat("deeply_nested_", 6) + "tailmarker"
	p := doneWith(t, phantoms)
	start := ansi.Strip(p.View(50, 12))

	p.Update(keyRight)
	if panned := ansi.Strip(p.View(50, 12)); panned == start {
		t.Fatalf("right should pan the list:\n%s", panned)
	}
	p.Update(keyLeft)
	if got := ansi.Strip(p.View(50, 12)); got != start {
		t.Fatalf("left should return to the unscrolled view:\n%s\nwant:\n%s", got, start)
	}
}

func TestUpdate_GivenWrapKeysWithNothingListed_ShouldChangeNothing(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status sdk.Status
	}{
		{"idle", sdk.StatusIdle},
		{"no phantoms", sdk.StatusDone},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := doneWith(t, nil)
			p.status = tc.status

			p.Update(keyWrap)
			p.Update(keyRight)

			if p.panel.WrapMode() || p.panel.HScroll() != 0 {
				t.Fatalf("wrap = %v, hscroll = %d, want neither changed", p.panel.WrapMode(), p.panel.HScroll())
			}
		})
	}
}

func TestHints_GivenPhantoms_ShouldShowWrapWithItsMode(t *testing.T) {
	p := doneWith(t, numberedPhantoms(2))

	if got, want := p.Hints(), (sdk.HintSetInspect | sdk.HintSetWrap | sdk.HintSetQuit).Hints(); !slices.Equal(got, want) {
		t.Fatalf("Hints() = %v, want %v", got, want)
	}
	p.Update(keyWrap)
	want := (sdk.HintSetInspect | sdk.HintSetWrap | sdk.HintSetQuit).Hints(sdk.HintSetOpts{WrapMode: true})
	if got := p.Hints(); !slices.Equal(got, want) {
		t.Fatalf("Hints() after ctrl+w = %v, want %v", got, want)
	}
}

func TestHints_GivenNothingListed_ShouldNotOfferWrap(t *testing.T) {
	p := doneWith(t, nil)
	for _, hint := range p.Hints() {
		if hint.Key == "^w" {
			t.Fatalf("Hints() = %v offers wrap with nothing to wrap", p.Hints())
		}
	}
}

func TestAnalyze_GivenNewPlan_ShouldStartTheWindowAtTheTop(t *testing.T) {
	phantoms := numberedPhantoms(30)
	p := doneWith(t, phantoms)
	p.View(80, 16)
	for range 29 {
		p.Update(keyDown)
	}
	p.View(80, 16)

	p.Analyze(summaryOf(phantoms))

	if view := ansi.Strip(p.View(80, 16)); !strings.Contains(view, "aws_s3_bucket.number00") {
		t.Fatalf("a new plan should start at the first row:\n%s", view)
	}
}

func TestAnalyze_GivenPannedList_ShouldPanBackToTheStart(t *testing.T) {
	for _, tc := range []struct {
		name    string
		summary *sdk.PlanSummary
	}{
		{"phantoms", summaryOf(numberedPhantoms(3))},
		{"no changes", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := doneWith(t, numberedPhantoms(3))
			p.Update(keyRight)
			if p.panel.HScroll() == 0 {
				t.Fatal("right should pan the list")
			}

			p.Analyze(tc.summary)

			if got := p.panel.HScroll(); got != 0 {
				t.Fatalf("hscroll = %d, want 0", got)
			}
		})
	}
}

func TestView_GivenOnePhantomTallerThanTheListBudget_ShouldStillShowItsLastLine(t *testing.T) {
	// The list gives up the banner before it hides a line of the block.
	for _, attrs := range []int{7, 8} {
		t.Run(fmt.Sprintf("%d attributes", attrs), func(t *testing.T) {
			phantoms := numberedPhantoms(1)
			phantoms[0].Attributes = nil
			for i := range attrs {
				phantoms[0].Attributes = append(phantoms[0].Attributes, sdk.AttributeDiff{Key: fmt.Sprintf("attr%02d", i), OldValue: "a", NewValue: "b"})
			}
			p := doneWith(t, phantoms)
			p.Update(keyEnter)

			view := ansi.Strip(p.View(80, 12))

			if want := fmt.Sprintf("attr%02d", attrs-1); !strings.Contains(view, want) {
				t.Fatalf("view should reach %q:\n%s", want, view)
			}
			if got := len(strings.Split(view, "\n")); got > 12 {
				t.Fatalf("view is %d lines, want at most the height 12:\n%s", got, view)
			}
		})
	}
}

func TestAnalyze_GivenScrolledList_ShouldReturnTheWindowOffsetToTheTop(t *testing.T) {
	phantoms := numberedPhantoms(30)
	p := doneWith(t, phantoms)
	p.View(80, 16)
	for range 29 {
		p.Update(keyDown)
	}
	p.View(80, 16)
	if p.offset == 0 {
		t.Fatal("scrolling to the last row should move the window offset")
	}

	p.Analyze(summaryOf(phantoms))

	if p.offset != 0 {
		t.Fatalf("offset = %d, want 0 once the list is replaced", p.offset)
	}
}

func TestView_GivenLongAttributeValuesAndWrap_ShouldShowThemInFull(t *testing.T) {
	oldValue := "old-" + strings.Repeat("abcdefghij", 5) + "-oldend"
	newValue := "new-" + strings.Repeat("klmnopqrst", 5) + "-newend"
	phantoms := numberedPhantoms(1)
	phantoms[0].Attributes = []sdk.AttributeDiff{{Key: "policy", OldValue: oldValue, NewValue: newValue}}
	p := doneWith(t, phantoms)
	p.Update(keyEnter)

	p.Update(keyWrap)
	view := strings.ReplaceAll(ansi.Strip(p.View(40, 16)), "\n", "")

	for _, want := range []string{oldValue, newValue} {
		if !strings.Contains(view, want) {
			t.Fatalf("wrap on should show %q in full:\n%s", want, view)
		}
	}
}

func TestView_GivenLongAttributeValueAndRight_ShouldPanToItsEnd(t *testing.T) {
	newValue := "new-" + strings.Repeat("klmnopqrst", 5) + "-newend"
	phantoms := numberedPhantoms(1)
	phantoms[0].Attributes = []sdk.AttributeDiff{{Key: "policy", OldValue: "a", NewValue: newValue}}
	p := doneWith(t, phantoms)
	p.Update(keyEnter)
	if strings.Contains(ansi.Strip(p.View(40, 16)), "-newend") {
		t.Fatal("the unpanned view should clip the long value")
	}

	for range 6 {
		p.Update(keyRight)
	}

	if view := ansi.Strip(p.View(40, 16)); !strings.Contains(view, "-newend") {
		t.Fatalf("panning right should reveal the end of the value:\n%s", view)
	}
}
