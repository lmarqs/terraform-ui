package plan

import (
	"cmp"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

// requestAssessment hides the current classification until the host assesses
// the reviewed plan again; Busy covers the wait so headless output waits too.
func (e *Plugin) requestAssessment() tea.Cmd {
	if e.summary == nil {
		return nil
	}
	e.assessing = true
	e.setAssessment(nil)
	return e.requestRisk(e.summary)
}

// HandleRiskAssessed implements sdk.RiskAssessedHandler. The host does not
// supersede requests for an older plan, so only the reviewed plan's result counts;
// a nil-summary event answers another plugin's host-planned request.
func (e *Plugin) HandleRiskAssessed(ev sdk.RiskAssessedEvent) tea.Cmd {
	if ev.Summary == nil || ev.Summary != e.summary {
		return nil
	}
	e.assessing = false
	e.riskHint = ev.Hint
	e.setAssessment(&ev.Assessment)
	return nil
}

func (e *Plugin) setAssessment(assessment *sdk.PolicyAssessment) {
	e.assessment = assessment
	e.rows = nil
	if assessment != nil {
		e.rows = assessment.Rows()
		e.assessedRisk = e.highestRisk()
	}
	if e.detailAddr == "" || e.summary == nil {
		return
	}
	for _, change := range e.summary.Changes {
		if change.Resource.InstanceKey() == e.detailKey() {
			e.detail = e.buildInspectContent(&change)
			return
		}
	}
}

func (e *Plugin) detailKey() sdk.InstanceKey {
	return sdk.InstanceKey{Address: e.detailAddr, Deposed: e.detailDeposed}
}

// riskCell is the assessed level of change, or the code that leaves it
// unclassified. Without an assessment the change shows no risk at all.
func (e *Plugin) riskCell(change sdk.PlanChange) (sdk.RiskLevel, sdk.ErrorCode) {
	if e.assessment == nil {
		return sdk.RiskNone, ""
	}
	if e.assessment.ErrorCode != "" {
		return sdk.RiskNone, e.assessment.ErrorCode
	}
	row, ok := e.rows[change.Resource.InstanceKey()]
	if ok && row.ErrorCode != "" {
		return sdk.RiskNone, row.ErrorCode
	}
	if !ok || (row.Status != sdk.ResourceClassified && row.Status != sdk.ResourceNotAssessed) {
		return sdk.RiskNone, sdk.ErrorInputIncomplete
	}
	if row.Status == sdk.ResourceClassified && row.Level != nil {
		return *row.Level, ""
	}
	return sdk.RiskNone, ""
}

func (e *Plugin) riskBadge(change sdk.PlanChange) string {
	level, diagnostic := e.riskCell(change)
	if diagnostic != "" {
		return sdk.StyleError.Render("[risk unavailable: " + string(diagnostic) + "]")
	}
	return sdk.RiskBadge(level)
}

// overallRisk is the highest assessed level across the reviewed plan, computed
// once per assessment because View renders it on every frame.
func (e *Plugin) overallRisk() sdk.RiskLevel {
	if e.assessment == nil {
		return sdk.RiskNone
	}
	return e.assessedRisk
}

func (e *Plugin) highestRisk() sdk.RiskLevel {
	overall := sdk.RiskNone
	for _, change := range e.summary.Changes {
		if level, _ := e.riskCell(change); level > overall {
			overall = level
		}
	}
	return overall
}

// assessmentDiagnostic is the code that leaves the plan's overall risk
// unavailable: the assessment's own failure, else the first failed instance in
// key order.
func (e *Plugin) assessmentDiagnostic() sdk.ErrorCode {
	if e.assessment == nil {
		return ""
	}
	if e.assessment.ErrorCode != "" || e.assessment.Usable() {
		return e.assessment.ErrorCode
	}
	diagnostic := sdk.ErrorInputIncomplete
	var first sdk.InstanceKey
	for _, row := range e.assessment.Resources {
		key := sdk.InstanceKey{Address: row.Address, Deposed: row.Deposed}
		if row.ErrorCode != "" && (first.Address == "" || cmp.Or(cmp.Compare(key.Address, first.Address), cmp.Compare(key.Deposed, first.Deposed)) < 0) {
			first, diagnostic = key, row.ErrorCode
		}
	}
	return diagnostic
}
