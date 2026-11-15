package blastradius

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

type blastRadiusAssessedMsg struct {
	generation uint64
	assessment *sdk.PolicyAssessment
}

func (e *Plugin) Busy() bool { return e.status == sdk.StatusLoading }

func (e *Plugin) Cancel() {
	if e.cancel != nil {
		e.cancel()
		e.cancel = nil
	}
	e.generation++
	if e.Busy() {
		e.status = sdk.StatusError
	}
}

func (e *Plugin) Shutdown() { e.Cancel() }

func (e *Plugin) HandlePlanInvalidated(_ sdk.PlanInvalidatedEvent) tea.Cmd {
	e.Cancel()
	e.summary, e.assessment, e.modules = nil, nil, nil
	e.status, e.total, e.selected = sdk.StatusIdle, 0, 0
	return nil
}

func (e *Plugin) HandleContextChanged(ev sdk.ContextChangedEvent) tea.Cmd {
	if e.HandleContextChangedDefault(ev) {
		return e.HandlePlanInvalidated(sdk.PlanInvalidatedEvent{})
	}
	return nil
}

func (e *Plugin) HandlePlanCompleted(ev sdk.PlanCompletedEvent) tea.Cmd {
	e.HandlePlanInvalidated(sdk.PlanInvalidatedEvent{})
	e.summary = ev.Summary
	return nil
}

func (e *Plugin) Activate() tea.Cmd {
	if e.Busy() || e.summary == nil {
		return nil
	}
	e.Cancel()
	e.assessment, e.modules = nil, nil
	e.status = sdk.StatusLoading
	ctx, cancel := context.WithCancel(context.Background())
	e.cancel = cancel
	generation, current, summary, assess := e.generation, e.GetCtx(), e.summary, e.assess
	return func() tea.Msg {
		var result *sdk.PolicyAssessment
		if assess != nil {
			assessment := assess(ctx, current, summary)
			result = &assessment
		}
		return blastRadiusAssessedMsg{generation: generation, assessment: result}
	}
}

func (e *Plugin) resourceAssessment(address string) sdk.PolicyResourceAssessment {
	if e.assessment.ErrorCode != "" {
		return sdk.PolicyResourceAssessment{Status: "unavailable", ErrorCode: e.assessment.ErrorCode}
	}
	for _, row := range e.assessment.Resources {
		if row.Address == address {
			return row
		}
	}
	return sdk.PolicyResourceAssessment{Status: "unavailable", ErrorCode: "input_incomplete"}
}

func (e *Plugin) analyzeAssessment() {
	if e.assessment == nil {
		e.Analyze(e.summary)
		return
	}
	changes := append([]sdk.PlanChange(nil), e.summary.Changes...)
	for i := range changes {
		changes[i].Risk = sdk.RiskNone
		row := e.resourceAssessment(changes[i].Resource.Address)
		if row.Status == "classified" && row.Level != nil {
			changes[i].Risk = *row.Level
		}
	}
	e.total = len(changes)
	e.modules = nil
	for _, group := range sdk.GroupByModule(changes) {
		impact := ModuleImpact{Group: group, NotAssessed: true}
		for _, change := range group.Changes {
			row := e.resourceAssessment(change.Resource.Address)
			if row.Status != "not_assessed" {
				impact.NotAssessed = false
			}
			if row.Status != "not_assessed" && (row.Status != "classified" || row.Level == nil) {
				impact.Unavailable = true
			}
		}
		if !impact.Unavailable {
			impact.Score = calculateImpact(group)
		}
		e.modules = append(e.modules, impact)
	}
	sortByImpact(e.modules)
	e.status, e.selected = sdk.StatusDone, 0
	e.expanded = make(map[int]bool)
}
