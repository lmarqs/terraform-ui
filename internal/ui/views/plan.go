package views

import "github.com/lmarqs/terraform-ui/internal/ui/styles"

type PlanView struct{}

func NewPlanView() PlanView { return PlanView{} }

func (v PlanView) Render(width, height int) string {
	title := styles.StyleTitle.Render("Plan Review")
	placeholder := styles.StyleFaintItalic.Render("Press Enter to run terraform plan...")
	return styles.StylePadded.Render(title + "\n\n" + placeholder)
}
