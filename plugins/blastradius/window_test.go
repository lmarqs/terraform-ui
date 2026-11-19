package blastradius

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

func numberedModules(n int) []ModuleImpact {
	modules := make([]ModuleImpact, n)
	for i := range modules {
		name := fmt.Sprintf("module.number%02d", i)
		modules[i] = ModuleImpact{
			Group: sdk.ModuleGroup{
				Module:  name,
				Changes: []sdk.PlanChange{{Resource: sdk.Resource{Address: name + ".aws_subnet.a", Module: name}, Action: sdk.ActionCreate}},
				Summary: sdk.ActionSummary{Add: 1},
			},
			Score: ImpactMinimal,
		}
	}
	return modules
}

func doneWith(t *testing.T, modules []ModuleImpact) *Plugin {
	t.Helper()
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = sdk.StatusDone
	p.modules = modules
	p.total = len(modules)
	p.setAssessment(classifiedLow(modules))
	return p
}

func TestView_GivenMoreModulesThanFit_ShouldKeepTheLastRowVisibleAndDrawTheGutter(t *testing.T) {
	p := doneWith(t, numberedModules(30))
	p.View(80, 16)

	for range 29 {
		p.Update(keyDown)
	}
	view := ansi.Strip(p.View(80, 16))

	if !strings.Contains(view, "module.number29") {
		t.Fatalf("cursor on the last row should show it:\n%s", view)
	}
	if !strings.ContainsAny(view, gutterGlyphs) {
		t.Fatalf("more rows than fit should draw the scroll gutter:\n%s", view)
	}
	if got := len(strings.Split(view, "\n")); got > 16 {
		t.Fatalf("view is %d lines, want at most the height 16:\n%s", got, view)
	}
	if !strings.Contains(view, "module(s) affected") {
		t.Fatalf("overall summary should stay visible:\n%s", view)
	}
}

func TestView_GivenExpandedLastModule_ShouldShowItsChangesBelowIt(t *testing.T) {
	p := doneWith(t, numberedModules(30))
	p.View(80, 16)

	for range 29 {
		p.Update(keyDown)
	}
	p.Update(keyEnter)
	view := ansi.Strip(p.View(80, 16))

	for _, want := range []string{"module.number29", "aws_subnet.a"} {
		if !strings.Contains(view, want) {
			t.Fatalf("expanded last module should show %q:\n%s", want, view)
		}
	}
}

func TestView_GivenCtrlW_ShouldWrapLongRowsAndToggleBack(t *testing.T) {
	long := "module." + strings.Repeat("deeply_nested_", 6) + "tailmarker"
	modules := numberedModules(1)
	modules[0].Group.Module = long
	p := doneWith(t, modules)
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
	long := "module." + strings.Repeat("deeply_nested_", 6) + "tailmarker"
	modules := numberedModules(1)
	modules[0].Group.Module = long
	p := doneWith(t, modules)
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
		name    string
		status  sdk.Status
		modules []ModuleImpact
	}{
		{"loading", sdk.StatusLoading, nil},
		{"idle", sdk.StatusIdle, nil},
		{"no modules", sdk.StatusDone, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := doneWith(t, tc.modules)
			p.status = tc.status

			p.Update(keyWrap)
			p.Update(keyRight)

			if p.panel.WrapMode() || p.panel.HScroll() != 0 {
				t.Fatalf("wrap = %v, hscroll = %d, want neither changed", p.panel.WrapMode(), p.panel.HScroll())
			}
		})
	}
}

func TestHints_GivenModules_ShouldShowWrapWithItsMode(t *testing.T) {
	p := doneWith(t, numberedModules(2))

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

func TestHandleRiskAssessed_GivenNewAssessment_ShouldStartTheWindowAtTheTop(t *testing.T) {
	summary := &sdk.PlanSummary{}
	for _, m := range numberedModules(30) {
		summary.Changes = append(summary.Changes, m.Group.Changes...)
	}
	assess := func(p *Plugin) {
		p.status = sdk.StatusLoading
		p.HandleRiskAssessed(classifiedEvent(summary, sdk.RiskLow, "ok"))
	}
	p := doneWith(t, nil)
	p.summary = summary
	assess(p)
	p.View(80, 16)
	for range 29 {
		p.Update(keyDown)
	}
	p.View(80, 16)

	assess(p)

	if view := ansi.Strip(p.View(80, 16)); !strings.Contains(view, "module.number00") {
		t.Fatalf("a new assessment should start at the first row:\n%s", view)
	}
}

func TestSetAssessment_GivenPannedList_ShouldPanBackToTheStart(t *testing.T) {
	for _, tc := range []struct {
		name  string
		start func(p *Plugin)
	}{
		{"new assessment", func(p *Plugin) {
			p.status = sdk.StatusLoading
			p.HandleRiskAssessed(classifiedEvent(p.summary, sdk.RiskLow, "ok"))
		}},
		{"plan invalidated", func(p *Plugin) { p.HandlePlanInvalidated(sdk.PlanInvalidatedEvent{}) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := doneWith(t, numberedModules(3))
			p.summary = &sdk.PlanSummary{}
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

func TestView_GivenOneExpandedModuleTallerThanTheListBudget_ShouldStillShowItsLastChange(t *testing.T) {
	// The list gives up the overall summary before it hides a line of the block.
	for _, changes := range []int{10, 11} {
		t.Run(fmt.Sprintf("%d changes", changes), func(t *testing.T) {
			modules := numberedModules(1)
			modules[0].Group.Changes = nil
			for i := range changes {
				modules[0].Group.Changes = append(modules[0].Group.Changes, sdk.PlanChange{Resource: sdk.Resource{Address: fmt.Sprintf("module.number00.aws_subnet.s%02d", i), Module: "module.number00"}, Action: sdk.ActionCreate})
			}
			p := doneWith(t, modules)
			p.Update(keyEnter)

			view := ansi.Strip(p.View(80, 12))

			if want := fmt.Sprintf("aws_subnet.s%02d", changes-1); !strings.Contains(view, want) {
				t.Fatalf("view should reach %q:\n%s", want, view)
			}
			if got := len(strings.Split(view, "\n")); got > 12 {
				t.Fatalf("view is %d lines, want at most the height 12:\n%s", got, view)
			}
		})
	}
}
