package plan

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

type planAssessedMsg struct {
	generation uint64
	assessment *sdk.PolicyAssessment
}

func (e *Plugin) reassess() tea.Cmd {
	e.cancelInflight()
	e.setAssessment(&sdk.PolicyAssessment{Status: sdk.AssessmentUnavailable, ErrorCode: sdk.ErrorInputIncomplete})
	e.status = sdk.StatusLoading
	ctx, cancel := context.WithCancel(context.Background())
	e.cancelFn = cancel
	current, summary, assess, generation := e.GetCtx(), e.rawSummary, e.assess, e.generation
	return func() tea.Msg {
		assessment := assess(ctx, current, summary)
		return planAssessedMsg{generation: generation, assessment: &assessment}
	}
}

func (e *Plugin) setAssessment(assessment *sdk.PolicyAssessment) {
	e.assessment = assessment
	if assessment == nil {
		return
	}
	e.riskDiagnostic = string(assessment.ErrorCode)
	if !assessment.Usable() && e.riskDiagnostic == "" {
		e.riskDiagnostic = string(sdk.ErrorInputIncomplete)
		first := ""
		for _, row := range assessment.Resources {
			if row.ErrorCode != "" && (first == "" || row.Address < first) {
				first, e.riskDiagnostic = row.Address, string(row.ErrorCode)
			}
		}
	}
	if e.rawSummary == nil {
		return
	}
	// Selected classifications are a display projection; the service cache and
	// PlanCompletedEvent retain the original evidence and artifact ownership.
	summary := *e.rawSummary
	summary.Changes = append([]sdk.PlanChange(nil), summary.Changes...)
	for i := range summary.Changes {
		change := &summary.Changes[i]
		change.Risk = sdk.RiskNone
		for _, row := range assessment.Resources {
			if row.Address == change.Resource.Address && row.Status == sdk.ResourceClassified && row.Level != nil {
				change.Risk = *row.Level
			}
		}
	}
	e.summary = &summary
	e.fuzzy.SetItems(summary.Changes)
	e.SetFilter(e.filter)
	if e.detailAddr != "" {
		for _, change := range summary.Changes {
			if change.Resource.Address == e.detailAddr {
				e.detail = e.buildInspectContent(&change)
				break
			}
		}
	}
}

func (e *Plugin) changeDiagnostic(address string) string {
	if e.assessment == nil {
		return e.riskDiagnostic
	}
	if e.assessment.ErrorCode != "" {
		return string(e.assessment.ErrorCode)
	}
	for _, row := range e.assessment.Resources {
		if row.Address == address {
			if row.ErrorCode != "" {
				return string(row.ErrorCode)
			}
			if row.Status == sdk.ResourceClassified || row.Status == sdk.ResourceNotAssessed {
				return ""
			}
		}
	}
	return string(sdk.ErrorInputIncomplete)
}

func (e *Plugin) changeRiskBadge(change sdk.PlanChange) string {
	if e.assessment == nil {
		return e.riskBadge(change.Risk)
	}
	if diagnostic := e.changeDiagnostic(change.Resource.Address); diagnostic != "" {
		return sdk.StyleError.Render("[risk unavailable: " + diagnostic + "]")
	}
	return sdk.RiskBadge(change.Risk)
}

func (e *Plugin) invalidateAssessment() {
	if e.assess == nil {
		return
	}
	e.cancelInflight()
	e.setAssessment(&sdk.PolicyAssessment{Status: sdk.AssessmentUnavailable, ErrorCode: sdk.ErrorInputIncomplete})
	e.status = sdk.StatusDone
}
