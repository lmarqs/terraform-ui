package views

import "github.com/lmarqs/terraform-ui/internal/ui/styles"

type ApplyView struct{}

func NewApplyView() ApplyView { return ApplyView{} }

func (v ApplyView) Render(width, height int) string {
	title := styles.StyleTitle.Render("Apply")
	placeholder := styles.StyleFaintItalic.Render("Run plan first, then apply changes here.")
	return styles.StylePadded.Render(title + "\n\n" + placeholder)
}
