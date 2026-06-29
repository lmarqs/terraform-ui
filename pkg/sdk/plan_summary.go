package sdk

import (
	"fmt"
	"strings"
)

// PlanSummary holds the full set of resource changes from a terraform plan
// along with aggregate counts by action type.
type PlanSummary struct {
	Changes   []PlanChange
	ToCreate  int
	ToUpdate  int
	ToDelete  int
	ToReplace int
	ToRead    int
}

// SummaryLine renders the action counts as a single styled line, e.g.
// "Plan: 1 to add, 2 to change", omitting any action with a zero count. It
// returns a faint "Plan: no changes" when nothing will change.
func (s *PlanSummary) SummaryLine() string {
	parts := []string{}
	if s.ToCreate > 0 {
		parts = append(parts, StyleCreate.Render(fmt.Sprintf("%d to add", s.ToCreate)))
	}
	if s.ToUpdate > 0 {
		parts = append(parts, StyleUpdate.Render(fmt.Sprintf("%d to change", s.ToUpdate)))
	}
	if s.ToDelete > 0 {
		parts = append(parts, StyleDelete.Render(fmt.Sprintf("%d to destroy", s.ToDelete)))
	}
	if s.ToReplace > 0 {
		parts = append(parts, StyleReplace.Render(fmt.Sprintf("%d to replace", s.ToReplace)))
	}
	if len(parts) == 0 {
		return StyleFaint.Render("Plan: no changes")
	}
	return "Plan: " + strings.Join(parts, ", ")
}

// ModuleGroup represents a set of plan changes that belong to the same terraform
// module, along with an action summary for quick aggregate display.
type ModuleGroup struct {
	Module  string
	Summary ActionSummary
	Changes []PlanChange
}

// ActionSummary holds counts of changes grouped by action type within a module.
type ActionSummary struct {
	Add     int
	Change  int
	Destroy int
	Replace int
}

// PhantomResult holds the outcome of phantom change detection, including counts
// and the addresses of resources identified as phantom (cosmetic-only) changes.
type PhantomResult struct {
	PhantomCount     int
	RealCount        int
	PhantomAddresses []string
}
