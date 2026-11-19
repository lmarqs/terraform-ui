package plan

import (
	"reflect"
	"strings"
	"testing"

	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/sdktest"
	"github.com/lmarqs/terraform-ui/pkg/sdk/ui"
)

// actionFilterChanges gives every address a hyphen and a plus-free name so a
// fuzzy match on "-" would keep creates and updates too: only the action filter
// keeps just the deletes.
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
