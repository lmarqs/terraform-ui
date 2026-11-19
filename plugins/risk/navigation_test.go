package risk

import (
	"slices"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/sdktest"
)

const longAddress = "aws_instance.a_resource_name_long_enough_to_overflow_the_row_and_end_in_tailmarker"

var (
	keyG     = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}}
	keyShift = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}}
	keyRight = tea.KeyMsg{Type: tea.KeyRight}
	keyLeft  = tea.KeyMsg{Type: tea.KeyLeft}
	keyWrap  = tea.KeyMsg{Type: tea.KeyCtrlW}
)

func longRowPlugin(t *testing.T) *Plugin {
	t.Helper()
	return newAnalyzedPlugin(t, &sdk.PlanSummary{Changes: []sdk.PlanChange{
		{Resource: sdk.Resource{Address: longAddress}, Action: sdk.ActionCreate, Risk: sdk.RiskLow},
		{Resource: sdk.Resource{Address: "aws_instance.b"}, Action: sdk.ActionCreate, Risk: sdk.RiskLow},
		{Resource: sdk.Resource{Address: "aws_instance.c"}, Action: sdk.ActionCreate, Risk: sdk.RiskLow},
	}})
}

func plainView(p *Plugin) string { return ansi.Strip(p.View(40, 18)) }

func TestUpdate_GivenGAndShiftG_ShouldJumpToFirstAndLastMutation(t *testing.T) {
	p := longRowPlugin(t)

	p.Update(keyShift)
	if p.selected != 2 {
		t.Fatalf("selected after G = %d, want 2", p.selected)
	}
	p.Update(keyG)
	if p.selected != 0 {
		t.Fatalf("selected after g = %d, want 0", p.selected)
	}
}

func TestUpdate_GivenGAndShiftGWithoutMutations_ShouldKeepCursorAtZero(t *testing.T) {
	p, _ := newLifecyclePlugin(t, &sdktest.MockService{})
	p.outcome.kind = outcomeDone

	p.Update(keyShift)
	p.Update(keyG)

	if p.selected != 0 {
		t.Fatalf("selected = %d, want 0", p.selected)
	}
}

func TestView_GivenRightAndLeft_ShouldPanTheListAndBack(t *testing.T) {
	p := longRowPlugin(t)
	start := plainView(p)
	if !strings.Contains(start, "aws_instance.a_resource") || strings.Contains(start, "tailmarker") {
		t.Fatalf("unscrolled view should clip the long row:\n%s", start)
	}

	p.Update(keyRight)
	panned := plainView(p)
	if strings.Contains(panned, "aws_instance.a_resource") || !strings.Contains(panned, "source_name") {
		t.Fatalf("right should pan the list left by ten columns:\n%s", panned)
	}

	p.Update(keyLeft)
	if got := plainView(p); got != start {
		t.Fatalf("left should return to the unscrolled view:\n%s\nwant:\n%s", got, start)
	}
}

func TestView_GivenCtrlW_ShouldWrapLongRowsAndToggleBack(t *testing.T) {
	p := longRowPlugin(t)
	start := plainView(p)

	p.Update(keyWrap)
	if wrapped := plainView(p); !strings.Contains(strings.ReplaceAll(wrapped, "\n", ""), "tailmarker") {
		t.Fatalf("wrap on should show the end of the long row:\n%s", wrapped)
	}

	p.Update(keyWrap)
	if got := plainView(p); got != start {
		t.Fatalf("wrap off should restore the clipped view:\n%s\nwant:\n%s", got, start)
	}
}

func TestUpdate_GivenCtrlWAndRightWhileLoading_ShouldChangeNothing(t *testing.T) {
	p, _ := newLifecyclePlugin(t, &sdktest.MockService{})
	p.outcome.kind = outcomeLoading

	p.Update(keyWrap)
	p.Update(keyRight)

	if p.panel.WrapMode() || p.panel.HScroll() != 0 {
		t.Fatalf("wrap = %v, hscroll = %d while loading, want neither changed", p.panel.WrapMode(), p.panel.HScroll())
	}
}

func TestHints_GivenNavigableList_ShouldShowWrapWithItsMode(t *testing.T) {
	p := longRowPlugin(t)

	if got, want := p.Hints(), (sdk.HintSetInspect | sdk.HintSetWrap | sdk.HintSetQuit).Hints(); !slices.Equal(got, want) {
		t.Fatalf("Hints() = %v, want %v", got, want)
	}
	p.Update(keyWrap)
	want := (sdk.HintSetInspect | sdk.HintSetWrap | sdk.HintSetQuit).Hints(sdk.HintSetOpts{WrapMode: true})
	if got := p.Hints(); !slices.Equal(got, want) {
		t.Fatalf("Hints() after ctrl+w = %v, want %v", got, want)
	}
}

func TestHints_GivenErrorStateWithRows_ShouldShowWrapBetweenInspectAndRetry(t *testing.T) {
	p, _, _ := failedRisk(t, "unusable with items")

	want := (sdk.HintSetInspect | sdk.HintSetWrap | sdk.HintSetRetry | sdk.HintSetBack).Hints()
	if got := p.Hints(); !slices.Equal(got, want) {
		t.Fatalf("Hints() = %v, want %v", got, want)
	}
}

func TestHints_GivenNoListedRows_ShouldNotOfferWrap(t *testing.T) {
	for _, kind := range []string{"plan failed", "unusable"} {
		t.Run(kind, func(t *testing.T) {
			p, _, _ := failedRisk(t, kind)
			for _, hint := range p.Hints() {
				if hint.Key == "^w" {
					t.Fatalf("Hints() = %v offers wrap with nothing to wrap", p.Hints())
				}
			}
		})
	}
}

func TestView_GivenWrapOnAndShiftGInNarrowView_ShouldKeepTheLastMutationVisibleAndInspectIt(t *testing.T) {
	p := longRowPlugin(t)

	p.Update(keyWrap)
	p.Update(keyShift)
	if view := ansi.Strip(p.View(32, 8)); !strings.Contains(view, "aws_instance.c") {
		t.Fatalf("wrap on with the cursor on the third mutation should show it:\n%s", view)
	}

	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	runLifecycleCmd(p, cmd)
	if p.Stack().Peek().ID() != "inspect" {
		t.Fatal("Enter should open the inspector")
	}
	if view := ansi.Strip(p.View(32, 8)); !strings.Contains(view, "aws_instance.c") || strings.Contains(view, "tailmarker") {
		t.Fatalf("Enter should inspect the third mutation:\n%s", view)
	}
}
