package risk

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

// outcomeKind is where the latest Risk run stands: idle, loading, or how it ended.
type outcomeKind int

const (
	outcomeIdle outcomeKind = iota
	outcomeLoading
	outcomeInvalidThreshold
	outcomePlanFailed
	outcomeUnusable
	outcomeDone
)

// outcome is the latest Risk run's state, decided by activation and
// acceptAssessment, with the line Stderr reports for it.
type outcome struct {
	kind       outcomeKind
	diagnostic string
	threshold  string // a done run's threshold result
}

// unavailable is the diagnostic line for a run that produced no usable assessment.
func unavailable(detail string) string { return "risk unavailable: " + detail }

// Busy reports that the host runs terraform plan for the pending request, which
// it does only when Risk retains no reviewed plan. Ready covers headless waiting.
func (e *Plugin) Busy() bool { return e.outcome.kind == outcomeLoading && e.summary == nil }

// Assessment returns the current assessment, or nil after invalidation.
func (e *Plugin) Assessment() *sdk.PolicyAssessment { return e.assessment }

func (e *Plugin) resetDerived() {
	e.stack.Clear()
	e.offset = 0
	e.panel.ResetScroll()
	e.outcome = outcome{}
	e.assessment = nil
	e.groups, e.items = nil, nil
	e.selected = 0
}

// Cancel abandons the pending request; its late result is then ignored.
func (e *Plugin) Cancel() {
	if e.outcome.kind == outcomeLoading {
		e.outcome = outcome{kind: outcomeUnusable, diagnostic: unavailable(string(sdk.ErrorCancelled))}
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
	e.outcome = outcome{kind: outcomeLoading}
	e.requested = e.GetCtx()
	var cmd tea.Cmd
	e.pending, cmd = e.requestRisk(e.summary)
	return cmd
}

// HandleRiskAssessed implements sdk.RiskAssessedHandler: only the result of the
// pending request counts.
func (e *Plugin) HandleRiskAssessed(ev sdk.RiskAssessedEvent) tea.Cmd {
	if e.outcome.kind != outcomeLoading || ev.Request != e.pending {
		return nil
	}
	e.acceptAssessment(ev)
	return nil
}

func (e *Plugin) acceptAssessment(ev sdk.RiskAssessedEvent) {
	if ev.Err != nil {
		e.outcome = outcome{kind: outcomePlanFailed, diagnostic: ev.Err.Error()}
		return
	}
	e.summary = ev.Summary
	assessment := ev.Assessment
	e.assessment = &assessment
	e.analyze(ev.Summary, assessment.Rows())
	if !assessment.Usable() {
		e.outcome = outcome{kind: outcomeUnusable, diagnostic: unavailable(failureDiagnostic(assessment, ev.Hint))}
		return
	}
	e.outcome = outcome{kind: outcomeDone, threshold: thresholdResult(assessment, e.threshold)}
	if e.outcome.threshold == "met" {
		e.outcome.diagnostic = fmt.Sprintf("risk threshold met: %s >= %s", assessment.OverallLevel.String(), thresholdName(e.threshold))
	}
}
