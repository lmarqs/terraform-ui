package risk

import (
	"context"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

type assessmentResultMsg struct {
	generation uint64
	summary    *sdk.PlanSummary
	assessment *sdk.PolicyAssessment
	err        error
}

func (e *Plugin) Busy() bool { return e.status == sdk.StatusLoading }

// Assessment returns the current assessment, or nil after invalidation.
func (e *Plugin) Assessment() *sdk.PolicyAssessment { return e.assessment }

func (e *Plugin) resetDerived() {
	e.Cancel()
	e.stack.Clear()
	e.offset = 0
	e.panel.ResetScroll()
	e.status = sdk.StatusIdle
	e.assessment = nil
	e.groups = nil
	e.overall = sdk.RiskNone
	e.selected, e.total = 0, 0
	e.riskDiagnostic, e.planError = "", ""
}

// Cancel invalidates publication even if the underlying operation ignores cancellation.
func (e *Plugin) Cancel() {
	if e.cancel != nil {
		e.cancel()
		e.cancel = nil
	}
	e.generation++
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

func (e *Plugin) startAssessment(summary *sdk.PlanSummary) tea.Cmd {
	ctx, cancel := context.WithCancel(context.Background())
	e.cancel = cancel
	e.status = sdk.StatusLoading
	generation := e.generation
	current := e.GetCtx()
	e.reportContext = reportContext{Project: e.input.Project, Chdir: e.input.Chdir, Workspace: current.Workspace.String()}
	if e.reportContext.Chdir == "" {
		e.reportContext.Chdir = current.Chdir.String()
	}
	if e.reportContext.Chdir == "" {
		e.reportContext.Chdir = "."
	}
	svc := current.Service
	opts := current.PlanOptions()
	assess := e.assess
	return func() tea.Msg {
		result := assessmentResultMsg{generation: generation, summary: summary}
		if result.summary == nil {
			dir, err := os.MkdirTemp("", "tfui-risk-*")
			if err != nil {
				result.err = err
				return result
			}
			defer func() { _ = os.RemoveAll(dir) }()
			opts.PlanFile = filepath.Join(dir, "plan.tfplan")
			result.summary, result.err = svc.Plan(ctx, opts)
		}
		if ctx.Err() != nil {
			result.err = ctx.Err()
		}
		if result.err == nil && assess != nil {
			assessment := assess(ctx, current, result.summary)
			result.assessment = &assessment
		}
		return result
	}
}

func (e *Plugin) acceptAssessment(msg assessmentResultMsg) {
	if msg.generation != e.generation {
		return
	}
	if e.cancel != nil {
		e.cancel()
		e.cancel = nil
	}
	if msg.err != nil {
		e.status = sdk.StatusError
		e.planError = msg.err.Error()
		return
	}
	e.summary = msg.summary
	if msg.assessment == nil {
		e.Analyze(msg.summary)
		return
	}
	e.assessment = msg.assessment

	// Render a copy so selected-policy classifications never mutate Plan's summary.
	summary := sdk.ApplyAssessment(msg.summary, msg.assessment)
	changes := summary.Changes[:0]
	for _, change := range summary.Changes {
		if row, ok := msg.assessment.Resource(change.Resource.InstanceKey()); ok && (row.Level != nil || row.Status == sdk.ResourceUnavailable) {
			changes = append(changes, change)
		}
	}
	summary.Changes = changes
	e.Analyze(summary)
	if !msg.assessment.Usable() {
		e.status = sdk.StatusError
		e.riskDiagnostic = string(msg.assessment.ErrorCode)
		if e.riskDiagnostic == "" {
			e.riskDiagnostic = string(msg.assessment.Status)
		}
	}
}
