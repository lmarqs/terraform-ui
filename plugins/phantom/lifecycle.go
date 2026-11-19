package phantom

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

// Busy reports that the host runs terraform plan for the pending request, which
// holds the directory lock until it ends.
func (e *Plugin) Busy() bool { return e.status == sdk.StatusLoading }

// Cancel abandons the pending request; its late result is then ignored.
func (e *Plugin) Cancel() {
	if e.status == sdk.StatusLoading {
		e.timer.Stop()
		e.status = sdk.StatusIdle
	}
}

func (e *Plugin) Shutdown() { e.Cancel() }

// Activate asks the host to plan when no plan was reviewed in this session. A
// reviewed plan, or a request already running, needs nothing.
func (e *Plugin) Activate() tea.Cmd {
	if e.status == sdk.StatusLoading || e.status == sdk.StatusDone {
		return nil
	}
	e.status = sdk.StatusLoading
	var cmd tea.Cmd
	e.pending, cmd = e.requestPlan(nil)
	return tea.Batch(cmd, e.timer.Start())
}

func (e *Plugin) HandlePlanCompleted(ev sdk.PlanCompletedEvent) tea.Cmd {
	e.timer.Stop()
	e.Analyze(ev.Summary)
	return nil
}

func (e *Plugin) HandlePlanInvalidated(_ sdk.PlanInvalidatedEvent) tea.Cmd {
	e.timer.Stop()
	e.status, e.phantoms, e.total, e.real, e.selected = sdk.StatusIdle, nil, 0, 0, 0
	return nil
}

func (e *Plugin) HandleContextChanged(ev sdk.ContextChangedEvent) tea.Cmd {
	if e.HandleContextChangedDefault(ev) {
		return e.HandlePlanInvalidated(sdk.PlanInvalidatedEvent{})
	}
	return nil
}

// HandleRiskAssessed implements sdk.RiskAssessedHandler: only the result of the
// pending request counts. Phantom reads the plan the host acquired and ignores
// the assessment.
func (e *Plugin) HandleRiskAssessed(ev sdk.RiskAssessedEvent) tea.Cmd {
	if e.status != sdk.StatusLoading || ev.Request != e.pending {
		return nil
	}
	e.timer.Stop()
	if ev.Summary == nil {
		e.status, e.failure = sdk.StatusError, planFailure(ev)
		return nil
	}
	e.Analyze(ev.Summary)
	return nil
}

// planFailure says why the host returned no plan: terraform's own error, or the
// code of the policy failure that stopped the host before it planned.
func planFailure(ev sdk.RiskAssessedEvent) string {
	const prefix = "plan unavailable"
	switch {
	case ev.Err != nil:
		return prefix + ": " + ev.Err.Error()
	case ev.Assessment.ErrorCode != "":
		return prefix + ": " + sdk.DiagnosticError{Code: ev.Assessment.ErrorCode, Hint: ev.Hint}.Error()
	}
	return prefix
}
