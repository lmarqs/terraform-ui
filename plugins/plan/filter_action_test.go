package plan

import (
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/sdktest"
	"github.com/lmarqs/terraform-ui/pkg/sdk/ui"
)

// actionFilterChanges gives every address a hyphen, so a fuzzy match on "-"
// would keep all of them: only the action filter keeps just the delete.
func actionFilterChanges() []sdk.PlanChange {
	return []sdk.PlanChange{
		{Resource: sdk.Resource{Address: "aws_instance.web-1"}, Action: sdk.ActionCreate},
		{Resource: sdk.Resource{Address: "aws_instance.web-2"}, Action: sdk.ActionUpdate},
		{Resource: sdk.Resource{Address: "aws_instance.web-3"}, Action: sdk.ActionDelete},
		{Resource: sdk.Resource{Address: "aws_instance.db-1"}, Action: sdk.ActionDeleteThenCreate},
		{Resource: sdk.Resource{Address: "aws_instance.db-2"}, Action: sdk.ActionCreateThenDelete},
		{Resource: sdk.Resource{Address: "aws_instance.db-3"}, Action: sdk.ActionNoOp},
	}
}

func newActionFilterPlugin() *Plugin {
	p, _ := newActionFilterPluginWithHarness()
	return p
}

func newActionFilterPluginWithHarness() (*Plugin, *sdktest.PluginDepsHarness) {
	p, h := newTestPluginWithHarness(&sdktest.MockService{})
	p.summary = &sdk.PlanSummary{Changes: actionFilterChanges()}
	p.filtered = p.summary.Changes
	p.rebuildTree()
	return p, h
}

func filteredAddresses(p *Plugin) []string {
	var got []string
	for _, c := range p.filtered {
		got = append(got, c.Resource.Address)
	}
	return got
}

func TestPlugin_SetFilter_WhenTextIsAnActionSymbol_ShouldKeepOnlyThatAction(t *testing.T) {
	tests := []struct {
		filter string
		want   []string
	}{
		{"+", []string{"aws_instance.web-1"}},
		{"~", []string{"aws_instance.web-2"}},
		{"-", []string{"aws_instance.web-3"}},
		{"-/+", []string{"aws_instance.db-1", "aws_instance.db-2"}},
	}
	for _, tt := range tests {
		for _, treeMode := range []bool{false, true} {
			p := newActionFilterPlugin()
			p.treeMode = treeMode
			p.SetFilter(tt.filter)

			if got := filteredAddresses(p); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SetFilter(%q) treeMode=%v: filtered = %v, want %v", tt.filter, treeMode, got, tt.want)
			}
			if filtered, total := p.Count(); filtered != len(tt.want) || total != 6 {
				t.Errorf("SetFilter(%q) treeMode=%v: Count() = (%d, %d), want (%d, 6)", tt.filter, treeMode, filtered, total, len(tt.want))
			}
			if got := p.tree.VisibleCount(); got < len(tt.want) {
				t.Errorf("SetFilter(%q) treeMode=%v: tree shows %d rows, want at least %d", tt.filter, treeMode, got, len(tt.want))
			}
			if p.filter != tt.filter {
				t.Errorf("SetFilter(%q): filter = %q, want the typed text", tt.filter, p.filter)
			}
		}
	}
}

func TestPlugin_SetFilter_WhenTextIsAnActionSymbol_ShouldDropFuzzyScores(t *testing.T) {
	p := newActionFilterPlugin()
	p.SetFilter("instance.web")
	if p.filterScores == nil {
		t.Fatal("SetFilter(\"instance.web\"): filterScores = nil, want fuzzy scores")
	}

	p.SetFilter("-")
	if p.filterScores != nil {
		t.Errorf("SetFilter(\"-\"): filterScores = %v, want nil so no row highlights the symbol", p.filterScores)
	}

	p.SetFilter("instance.web")
	if p.filterScores == nil {
		t.Error("SetFilter(\"instance.web\") after \"-\": filterScores = nil, want fuzzy scores again")
	}
	if got := len(p.filtered); got != 3 {
		t.Errorf("SetFilter(\"instance.web\") after \"-\": %d rows, want the 3 web addresses", got)
	}
}

func TestPlugin_SetFilter_WhenTextIsNotExactlyAnActionSymbol_ShouldFuzzyMatchTheAddress(t *testing.T) {
	for _, filter := range []string{"+aws", "~web", "--", "-/", "/+", "+/-", "web-", " -", "- ", "+ ", "+-"} {
		p := newActionFilterPlugin()
		p.SetFilter(filter)

		ref := ui.NewFuzzyFilter(func(c sdk.PlanChange) string { return c.Resource.Address })
		ref.SetItems(actionFilterChanges())
		ref.SetQuery(filter)
		var want []string
		for _, c := range ref.Results() {
			want = append(want, c.Resource.Address)
		}
		if got := filteredAddresses(p); !reflect.DeepEqual(got, want) {
			t.Errorf("SetFilter(%q): filtered = %v, want fuzzy result %v", filter, got, want)
		}
		if len(want) > 0 && p.filterScores == nil {
			t.Errorf("SetFilter(%q): filterScores = nil, want fuzzy scores", filter)
		}
	}
}

func TestPlugin_SetFilter_WhenPinnedOnlyAndActionSymbol_ShouldIntersect(t *testing.T) {
	p, h := newActionFilterPluginWithHarness()
	p.pinnedOnly = true
	h.Ctx.Pins = []string{"aws_instance.web-1", "aws_instance.web-3", "aws_instance.db-1"}

	p.SetFilter("-")
	if got, want := filteredAddresses(p), []string{"aws_instance.web-3"}; !reflect.DeepEqual(got, want) {
		t.Errorf("pinned-only SetFilter(\"-\"): filtered = %v, want %v", got, want)
	}

	p.SetFilter("-/+")
	if got, want := filteredAddresses(p), []string{"aws_instance.db-1"}; !reflect.DeepEqual(got, want) {
		t.Errorf("pinned-only SetFilter(\"-/+\"): filtered = %v, want %v", got, want)
	}
}

func TestPlugin_SetFilter_WhenNoChangeHasTheAction_ShouldShowNoMatches(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.summary = &sdk.PlanSummary{Changes: []sdk.PlanChange{
		{Resource: sdk.Resource{Address: "aws_instance.web"}, Action: sdk.ActionCreate},
	}}
	p.status = sdk.StatusDone
	p.filtered = p.summary.Changes
	p.rebuildTree()

	p.SetFilter("~")
	if len(p.filtered) != 0 {
		t.Fatalf("SetFilter(\"~\"): filtered = %v, want none", filteredAddresses(p))
	}
	view := p.View(80, 20)
	if !strings.Contains(view, "No matching changes.") || !strings.Contains(view, "~") {
		t.Errorf("View() = %q, want the filter indicator and \"No matching changes.\"", view)
	}
}

func TestPlugin_FilterFrame_WhenTypingActionSymbolKeyByKey_ShouldSwitchBetweenActionAndFuzzy(t *testing.T) {
	p := newActionFilterPlugin()
	p.status = sdk.StatusDone
	key := func(r rune) { p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}) }

	key('/')
	for _, step := range []struct {
		typed rune
		want  []string
	}{
		{'-', []string{"aws_instance.web-3"}},
		{'/', nil},
		{'+', []string{"aws_instance.db-1", "aws_instance.db-2"}},
	} {
		key(step.typed)
		if step.want != nil {
			if got := filteredAddresses(p); !reflect.DeepEqual(got, step.want) {
				t.Fatalf("after typing %q (filter %q): filtered = %v, want %v", step.typed, p.filter, got, step.want)
			}
			continue
		}
		if p.filter != "-/" || len(p.filtered) != 0 {
			t.Fatalf("after typing %q: filter = %q, filtered = %v, want \"-/\" matching no address", step.typed, p.filter, filteredAddresses(p))
		}
	}

	p.stack.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	p.stack.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if got := filteredAddresses(p); !reflect.DeepEqual(got, []string{"aws_instance.web-3"}) {
		t.Errorf("after backspacing to %q: filtered = %v, want the delete", p.filter, got)
	}

	// Esc leaves filter mode and keeps the filter applied, as for a fuzzy filter.
	p.stack.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if p.filtering || p.filter != "-" || !reflect.DeepEqual(filteredAddresses(p), []string{"aws_instance.web-3"}) {
		t.Errorf("after esc: filtering = %v, filter = %q, filtered = %v, want the delete kept", p.filtering, p.filter, filteredAddresses(p))
	}

	key('/')
	if p.filter != "" || len(p.filtered) != 6 || p.filterScores != nil {
		t.Errorf("after reopening the filter: filter = %q, rows = %d, scores = %v, want an empty filter and all 6 rows", p.filter, len(p.filtered), p.filterScores)
	}
}

func TestPlugin_FilterFrame_WhenTextArrivesAsOneEvent_ShouldApplyTheQueryOnce(t *testing.T) {
	tests := []struct {
		name string
		msg  tea.KeyMsg
		want string
		rows []string
	}{
		{"pasted address", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("aws_instance.db-1"), Paste: true}, "aws_instance.db-1", []string{"aws_instance.db-1"}},
		{"pasted action symbols", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("-/+"), Paste: true}, "-/+", []string{"aws_instance.db-1", "aws_instance.db-2"}},
		{"grouped action symbols", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("-/+")}, "-/+", []string{"aws_instance.db-1", "aws_instance.db-2"}},
		{"grouped text named like a frame shortcut", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("right")}, "right", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newActionFilterPlugin()
			p.status = sdk.StatusDone
			p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})

			p.stack.Update(tt.msg)

			if p.filter != tt.want {
				t.Fatalf("filter = %q, want %q", p.filter, tt.want)
			}
			if got := filteredAddresses(p); tt.rows != nil && !reflect.DeepEqual(got, tt.rows) {
				t.Errorf("filtered = %v, want %v", got, tt.rows)
			}
		})
	}
}
