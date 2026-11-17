package plan

import (
	"cmp"
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
		var first sdk.InstanceKey
		for _, row := range assessment.Resources {
			key := sdk.InstanceKey{Address: row.Address, Deposed: row.Deposed}
			if row.ErrorCode != "" && (first.Address == "" || cmp.Or(cmp.Compare(key.Address, first.Address), cmp.Compare(key.Deposed, first.Deposed)) < 0) {
				first, e.riskDiagnostic = key, string(row.ErrorCode)
			}
		}
	}
	if e.rawSummary == nil {
		return
	}
	// Selected classifications are a display projection; the service cache and
	// PlanCompletedEvent retain the original evidence and artifact ownership.
	e.summary = sdk.ApplyAssessment(e.rawSummary, assessment)
	e.fuzzy.SetItems(e.summary.Changes)
	e.SetFilter(e.filter)
	if e.detailAddr != "" {
		for _, change := range e.summary.Changes {
			if change.Resource.InstanceKey() == e.detailKey() {
				e.detail = e.buildInspectContent(&change)
				break
			}
		}
	}
}

func (e *Plugin) detailKey() sdk.InstanceKey {
	return sdk.InstanceKey{Address: e.detailAddr, Deposed: e.detailDeposed}
}

func (e *Plugin) changeDiagnostic(key sdk.InstanceKey) string {
	if e.assessment == nil {
		return e.riskDiagnostic
	}
	if e.assessment.ErrorCode != "" {
		return string(e.assessment.ErrorCode)
	}
	if row, ok := e.assessment.Resource(key); ok {
		if row.ErrorCode != "" {
			return string(row.ErrorCode)
		}
		if row.Status == sdk.ResourceClassified || row.Status == sdk.ResourceNotAssessed {
			return ""
		}
	}
	return string(sdk.ErrorInputIncomplete)
}

func (e *Plugin) changeRiskBadge(change sdk.PlanChange) string {
	if e.assessment == nil {
		return e.riskBadge(change.Risk)
	}
	if diagnostic := e.changeDiagnostic(change.Resource.InstanceKey()); diagnostic != "" {
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
