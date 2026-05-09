package views

import "github.com/lmarqs/terraform-ui/internal/ui/styles"

type StateView struct{}

func NewStateView() StateView { return StateView{} }

func (v StateView) Render(width, height int) string {
	title := styles.StyleTitle.Render("State Browser")
	placeholder := styles.StyleFaintItalic.Render("No resources loaded. Press 'r' to refresh state.")
	return styles.StylePadded.Render(title + "\n\n" + placeholder)
}
