package risk

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

// Busy reports a pending assessment request, so headless output waits for it.
func (e *Plugin) Busy() bool { return e.status == sdk.StatusLoading }

// Assessment returns the current assessment, or nil after invalidation.
func (e *Plugin) Assessment() *sdk.PolicyAssessment { return e.assessment }

func (e *Plugin) resetDerived() {
	e.Cancel()
	e.stack.Clear()
	e.offset = 0
	e.panel.ResetScroll()
	e.status = sdk.StatusIdle
	e.setAssessment(nil)
	e.groups = nil
	e.overall = sdk.RiskNone
	e.selected, e.total = 0, 0
	e.riskDiagnostic, e.planError = "", ""
}

// Cancel abandons the pending request; its late result is then ignored.
func (e *Plugin) Cancel() {
	if e.status == sdk.StatusLoading {
		e.status = sdk.StatusError
		e.riskDiagnostic = string(sdk.ErrorCancelled)
	}
}

func (e *Plugin) Shutdown() { e.Cancel() }

func (e *Plugin) HandlePlanInvalidated(_ sdk.PlanInvalidatedEvent) tea.Cmd {
	e.resetDerived()
	e.summary = nil
	return nil
}

func (e *Plugin) HandleContextChanged(ev sdk.ContextChangedEvent) tea.Cmd {
	if e.HandleContextChangedDefault(ev) {
		return e.HandlePlanInvalidated(sdk.PlanInvalidatedEvent{})
	}
	return nil
}

func (e *Plugin) HandlePlanCompleted(ev sdk.PlanCompletedEvent) tea.Cmd {
	e.resetDerived()
	e.summary = ev.Summary
	return nil
}

func (e *Plugin) requestAssessment() tea.Cmd {
	e.status = sdk.StatusLoading
	current := e.GetCtx()
	e.reportContext = reportContext{Project: e.input.Project, Chdir: e.input.Chdir, Workspace: current.Workspace.String()}
	if e.reportContext.Chdir == "" {
		e.reportContext.Chdir = current.Chdir.String()
	}
	if e.reportContext.Chdir == "" {
		e.reportContext.Chdir = "."
	}
	var cmd tea.Cmd
	e.pending, cmd = e.requestRisk(e.summary)
	return cmd
}

// HandleRiskAssessed implements sdk.RiskAssessedHandler: only the result of the
// pending request counts.
func (e *Plugin) HandleRiskAssessed(ev sdk.RiskAssessedEvent) tea.Cmd {
	if !e.Busy() || ev.Request != e.pending {
		return nil
	}
	e.acceptAssessment(ev)
	return nil
}

// setAssessment keeps rows indexing the assessment's rows.
func (e *Plugin) setAssessment(assessment *sdk.PolicyAssessment) {
	e.assessment, e.rows = assessment, nil
	if assessment != nil {
		e.rows = assessment.Rows()
	}
}

func (e *Plugin) acceptAssessment(ev sdk.RiskAssessedEvent) {
	if ev.Err != nil {
		e.status = sdk.StatusError
		e.planError = ev.Err.Error()
		return
	}
	e.summary = ev.Summary
	assessment := ev.Assessment
	e.setAssessment(&assessment)

	// Render a copy so selected-policy classifications never mutate Plan's summary.
	summary := sdk.ApplyAssessment(ev.Summary, e.assessment)
	if summary != nil {
		changes := summary.Changes[:0]
		for _, change := range summary.Changes {
			if row, ok := e.rows[change.Resource.InstanceKey()]; ok && (row.Level != nil || row.Status == sdk.ResourceUnavailable) {
				changes = append(changes, change)
			}
		}
		summary.Changes = changes
	}
	e.Analyze(summary)
	if !assessment.Usable() {
		e.status = sdk.StatusError
		e.riskDiagnostic = string(assessment.Status)
		if assessment.ErrorCode != "" {
			e.riskDiagnostic = sdk.DiagnosticError{Code: assessment.ErrorCode, Hint: ev.Hint}.Error()
		}
	}
}
