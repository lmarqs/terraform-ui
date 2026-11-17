package blastradius

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

// Busy reports a pending assessment request, so headless output waits for it.
func (e *Plugin) Busy() bool { return e.status == sdk.StatusLoading }

// Cancel abandons the pending request; its late result is then ignored.
func (e *Plugin) Cancel() {
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
	e.assessment, e.modules = nil, nil
	e.status = sdk.StatusLoading
	return e.requestRisk(e.summary)
}

// HandleRiskAssessed implements sdk.RiskAssessedHandler. The host neither
// supersedes requests for an older plan nor stops evaluating after Cancel, so
// only a result for the reviewed plan while a request is pending counts.
func (e *Plugin) HandleRiskAssessed(ev sdk.RiskAssessedEvent) tea.Cmd {
	if !e.Busy() || ev.Summary != e.summary {
		return nil
	}
	assessment := ev.Assessment
	e.assessment = &assessment
	e.analyzeAssessment()
	return nil
}

func (e *Plugin) resourceAssessment(key sdk.InstanceKey) sdk.PolicyResourceAssessment {
	if e.assessment.ErrorCode != "" {
		return sdk.PolicyResourceAssessment{Status: sdk.ResourceUnavailable, ErrorCode: e.assessment.ErrorCode}
	}
	if row, ok := e.assessment.Resource(key); ok {
		return row
	}
	return sdk.PolicyResourceAssessment{Status: sdk.ResourceUnavailable, ErrorCode: sdk.ErrorInputIncomplete}
}

func (e *Plugin) analyzeAssessment() {
	changes := sdk.ApplyAssessment(e.summary, e.assessment).Changes
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
