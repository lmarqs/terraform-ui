package blastradius

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

// Cancel abandons the pending request; its late result is then ignored.
func (e *Plugin) Cancel() {
	if e.status == sdk.StatusLoading {
		e.status = sdk.StatusError
	}
}

func (e *Plugin) Shutdown() { e.Cancel() }

func (e *Plugin) HandlePlanInvalidated(_ sdk.PlanInvalidatedEvent) tea.Cmd {
	e.Cancel()
	e.summary, e.modules = nil, nil
	e.setAssessment(nil)
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
	if e.status == sdk.StatusLoading || e.summary == nil {
		return nil
	}
	e.modules = nil
	e.setAssessment(nil)
	e.status = sdk.StatusLoading
	var cmd tea.Cmd
	e.pending, cmd = e.requestRisk(e.summary)
	return cmd
}

// HandleRiskAssessed implements sdk.RiskAssessedHandler: only the result of the
// pending request counts.
func (e *Plugin) HandleRiskAssessed(ev sdk.RiskAssessedEvent) tea.Cmd {
	if e.status != sdk.StatusLoading || ev.Request != e.pending {
		return nil
	}
	e.setAssessment(&ev.Assessment)
	e.analyzeAssessment()
	return nil
}

// setAssessment keeps rows indexing the assessment's rows.
func (e *Plugin) setAssessment(assessment *sdk.PolicyAssessment) {
	e.assessment, e.rows = assessment, nil
	if assessment != nil {
		e.rows = assessment.Rows()
	}
}

func (e *Plugin) resourceAssessment(key sdk.InstanceKey) sdk.PolicyResourceAssessment {
	if e.assessment.ErrorCode != "" {
		return sdk.PolicyResourceAssessment{Status: sdk.ResourceUnavailable, ErrorCode: e.assessment.ErrorCode}
	}
	if row, ok := e.rows[key]; ok {
		return row
	}
	return sdk.PolicyResourceAssessment{Status: sdk.ResourceUnavailable, ErrorCode: sdk.ErrorInputIncomplete}
}

func (e *Plugin) analyzeAssessment() {
	changes := sdk.ApplyAssessment(e.summary, e.rows).Changes
	e.total = len(changes)
	e.modules = nil
	for _, group := range sdk.GroupByModule(changes) {
		impact := ModuleImpact{Group: group, NotAssessed: true}
		for _, change := range group.Changes {
			row := e.resourceAssessment(change.Resource.InstanceKey())
			if row.Status != sdk.ResourceNotAssessed {
				impact.NotAssessed = false
			}
			if row.Status != sdk.ResourceNotAssessed && (row.Status != sdk.ResourceClassified || row.Level == nil) {
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
