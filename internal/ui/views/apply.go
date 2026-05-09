package views

import "github.com/lmarqs/terraform-ui/internal/ui/styles"

type ApplyStatus int

const (
	ApplyStatusIdle ApplyStatus = iota
	ApplyStatusRunning
	ApplyStatusSuccess
	ApplyStatusError
)

type ApplyView struct {
	status ApplyStatus
	errMsg string
}

func NewApplyView() ApplyView { return ApplyView{} }

func (v ApplyView) SetRunning() ApplyView {
	v.status = ApplyStatusRunning
	v.errMsg = ""
	return v
}

func (v ApplyView) SetSuccess() ApplyView {
	v.status = ApplyStatusSuccess
	v.errMsg = ""
	return v
}

func (v ApplyView) SetError(err string) ApplyView {
	v.status = ApplyStatusError
	v.errMsg = err
	return v
}

func (v ApplyView) Status() ApplyStatus { return v.status }

func (v ApplyView) Render(width, height int) string {
	title := styles.StyleTitle.Render("Apply")

	switch v.status {
	case ApplyStatusIdle:
		placeholder := styles.StyleFaintItalic.Render("Run plan first, then apply changes here.")
		return styles.StylePadded.Render(title + "\n\n" + placeholder)

	case ApplyStatusRunning:
		running := styles.StyleFaintItalic.Render("Applying changes...")
		return styles.StylePadded.Render(title + "\n\n" + running)

	case ApplyStatusSuccess:
		success := styles.StyleSuccess.Render("Apply complete! Resources are up-to-date.")
		hint := styles.StyleFaintItalic.Render("Press Esc to go back")
		return styles.StylePadded.Render(title + "\n\n" + success + "\n\n" + hint)

	case ApplyStatusError:
		errText := styles.StyleError.Render("Apply failed: " + v.errMsg)
		hint := styles.StyleFaintItalic.Render("Press Esc to go back")
		return styles.StylePadded.Render(title + "\n\n" + errText + "\n\n" + hint)

	default:
		return ""
	}
}
