package views

import (
	"fmt"
	"strings"

	"github.com/lmarqs/terraform-ui/internal/terraform"
	"github.com/lmarqs/terraform-ui/internal/ui/styles"
)

type PlanStatus int

const (
	PlanStatusIdle PlanStatus = iota
	PlanStatusLoading
	PlanStatusDone
	PlanStatusError
)

type PlanView struct {
	status   PlanStatus
	summary  *terraform.PlanSummary
	errMsg   string
	selected int
}

func NewPlanView() PlanView { return PlanView{} }

func (v PlanView) SetLoading() PlanView {
	v.status = PlanStatusLoading
	v.summary = nil
	v.errMsg = ""
	v.selected = 0
	return v
}

func (v PlanView) SetResult(summary *terraform.PlanSummary) PlanView {
	v.status = PlanStatusDone
	v.summary = summary
	v.errMsg = ""
	v.selected = 0
	return v
}

func (v PlanView) SetError(err string) PlanView {
	v.status = PlanStatusError
	v.errMsg = err
	v.summary = nil
	return v
}

func (v PlanView) MoveUp() PlanView {
	if v.selected > 0 {
		v.selected--
	}
	return v
}

func (v PlanView) MoveDown() PlanView {
	if v.summary != nil && v.selected < len(v.summary.Changes)-1 {
		v.selected++
	}
	return v
}

func (v PlanView) Selected() int { return v.selected }

func (v PlanView) Render(width, height int) string {
	switch v.status {
	case PlanStatusIdle:
		title := styles.StyleTitle.Render("Plan Review")
		placeholder := styles.StyleFaintItalic.Render("Press Enter to run terraform plan...")
		return styles.StylePadded.Render(title + "\n\n" + placeholder)

	case PlanStatusLoading:
		title := styles.StyleTitle.Render("Plan Review")
		loading := styles.StyleFaintItalic.Render("Running terraform plan...")
		return styles.StylePadded.Render(title + "\n\n" + loading)

	case PlanStatusError:
		title := styles.StyleTitle.Render("Plan Review")
		errText := styles.StyleError.Render("Error: " + v.errMsg)
		hint := styles.StyleFaintItalic.Render("Press Esc to go back, Enter to retry")
		return styles.StylePadded.Render(title + "\n\n" + errText + "\n\n" + hint)

	case PlanStatusDone:
		return v.renderResults(width, height)

	default:
		return ""
	}
}

func (v PlanView) renderResults(width, height int) string {
	title := styles.StyleTitle.Render("Plan Review")

	if v.summary == nil || len(v.summary.Changes) == 0 {
		noChanges := styles.StyleSuccess.Render("No changes. Infrastructure is up-to-date.")
		return styles.StylePadded.Render(title + "\n\n" + noChanges)
	}

	var b strings.Builder

	// Calculate visible area (title + summary + hint take ~4 lines)
	maxVisible := height - 6
	if maxVisible < 3 {
		maxVisible = 3
	}

	// Determine scroll window
	startIdx := 0
	if v.selected >= maxVisible {
		startIdx = v.selected - maxVisible + 1
	}
	endIdx := startIdx + maxVisible
	if endIdx > len(v.summary.Changes) {
		endIdx = len(v.summary.Changes)
	}

	for i := startIdx; i < endIdx; i++ {
		change := v.summary.Changes[i]
		row := v.renderChangeRow(change, width)
		if i == v.selected {
			row = styles.StyleSelected.Width(width - 6).Render(row)
		}
		b.WriteString(row)
		b.WriteByte('\n')
	}

	summary := v.renderSummaryLine()
	hint := styles.StyleFaintItalic.Render("j/k navigate  a apply  Esc back")

	content := title + "\n\n" + b.String() + "\n" + summary + "\n" + hint
	return styles.StylePadded.Render(content)
}

func (v PlanView) renderChangeRow(change terraform.PlanChange, width int) string {
	symbol := actionSymbol(change.Action)
	address := change.Resource.Address
	risk := riskBadge(change.Risk)

	if change.IsPhantom {
		address = styles.StylePhantom.Render(address)
		symbol = styles.StylePhantom.Render(symbol)
	}

	row := fmt.Sprintf(" %s %s", symbol, address)
	if risk != "" {
		row += " " + risk
	}
	return row
}

func actionSymbol(action terraform.Action) string {
	switch action {
	case terraform.ActionCreate:
		return styles.StyleCreate.Render("+")
	case terraform.ActionUpdate:
		return styles.StyleUpdate.Render("~")
	case terraform.ActionDelete:
		return styles.StyleDelete.Render("-")
	case terraform.ActionDeleteThenCreate, terraform.ActionCreateThenDelete:
		return styles.StyleReplace.Render("-/+")
	case terraform.ActionRead:
		return styles.StyleFaint.Render("<=")
	default:
		return " "
	}
}

func riskBadge(risk terraform.RiskLevel) string {
	switch risk {
	case terraform.RiskLow:
		return styles.StyleRiskLow.Render("[low]")
	case terraform.RiskMedium:
		return styles.StyleRiskMedium.Render("[medium]")
	case terraform.RiskHigh:
		return styles.StyleRiskHigh.Render("[HIGH]")
	case terraform.RiskCritical:
		return styles.StyleRiskCritical.Render("[CRITICAL]")
	default:
		return ""
	}
}

func (v PlanView) renderSummaryLine() string {
	s := v.summary
	parts := []string{}
	if s.ToCreate > 0 {
		parts = append(parts, styles.StyleCreate.Render(fmt.Sprintf("%d to add", s.ToCreate)))
	}
	if s.ToUpdate > 0 {
		parts = append(parts, styles.StyleUpdate.Render(fmt.Sprintf("%d to change", s.ToUpdate)))
	}
	if s.ToDelete > 0 {
		parts = append(parts, styles.StyleDelete.Render(fmt.Sprintf("%d to destroy", s.ToDelete)))
	}
	if s.ToReplace > 0 {
		parts = append(parts, styles.StyleReplace.Render(fmt.Sprintf("%d to replace", s.ToReplace)))
	}

	if len(parts) == 0 {
		return styles.StyleFaint.Render("Plan: no changes")
	}
	return "Plan: " + strings.Join(parts, ", ")
}
