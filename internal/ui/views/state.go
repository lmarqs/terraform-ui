package views

import (
	"fmt"
	"strings"

	"github.com/lmarqs/terraform-ui/internal/terraform"
	"github.com/lmarqs/terraform-ui/internal/ui/styles"
)

type StateStatus int

const (
	StateStatusIdle StateStatus = iota
	StateStatusLoading
	StateStatusDone
	StateStatusError
)

type StateView struct {
	status    StateStatus
	resources []terraform.Resource
	filtered  []terraform.Resource
	filter    string
	errMsg    string
	selected  int
}

func NewStateView() StateView { return StateView{} }

func (v StateView) SetLoading() StateView {
	v.status = StateStatusLoading
	v.resources = nil
	v.filtered = nil
	v.errMsg = ""
	v.selected = 0
	v.filter = ""
	return v
}

func (v StateView) SetResources(resources []terraform.Resource) StateView {
	v.status = StateStatusDone
	v.resources = resources
	v.filtered = resources
	v.errMsg = ""
	v.selected = 0
	v.filter = ""
	return v
}

func (v StateView) SetError(err string) StateView {
	v.status = StateStatusError
	v.errMsg = err
	v.resources = nil
	v.filtered = nil
	return v
}

func (v StateView) SetFilter(filter string) StateView {
	v.filter = filter
	v.selected = 0
	if filter == "" {
		v.filtered = v.resources
		return v
	}
	lower := strings.ToLower(filter)
	var result []terraform.Resource
	for _, r := range v.resources {
		if strings.Contains(strings.ToLower(r.Address), lower) {
			result = append(result, r)
		}
	}
	v.filtered = result
	return v
}

func (v StateView) AppendFilter(ch string) StateView {
	return v.SetFilter(v.filter + ch)
}

func (v StateView) BackspaceFilter() StateView {
	if len(v.filter) > 0 {
		return v.SetFilter(v.filter[:len(v.filter)-1])
	}
	return v
}

func (v StateView) Filter() string { return v.filter }

func (v StateView) MoveUp() StateView {
	if v.selected > 0 {
		v.selected--
	}
	return v
}

func (v StateView) MoveDown() StateView {
	if v.selected < len(v.filtered)-1 {
		v.selected++
	}
	return v
}

func (v StateView) Selected() int { return v.selected }

func (v StateView) SelectedResource() terraform.Resource {
	if v.selected < len(v.filtered) {
		return v.filtered[v.selected]
	}
	return terraform.Resource{}
}

func (v StateView) Render(width, height int) string {
	switch v.status {
	case StateStatusIdle:
		title := styles.StyleTitle.Render("State Browser")
		placeholder := styles.StyleFaintItalic.Render("Loading state...")
		return styles.StylePadded.Render(title + "\n\n" + placeholder)

	case StateStatusLoading:
		title := styles.StyleTitle.Render("State Browser")
		loading := styles.StyleFaintItalic.Render("Loading terraform state...")
		return styles.StylePadded.Render(title + "\n\n" + loading)

	case StateStatusError:
		title := styles.StyleTitle.Render("State Browser")
		errText := styles.StyleError.Render("Error: " + v.errMsg)
		hint := styles.StyleFaintItalic.Render("Press Esc to go back, r to retry")
		return styles.StylePadded.Render(title + "\n\n" + errText + "\n\n" + hint)

	case StateStatusDone:
		return v.renderResources(width, height)

	default:
		return ""
	}
}

func (v StateView) renderResources(width, height int) string {
	title := styles.StyleTitle.Render("State Browser")

	filterLine := ""
	if v.filter != "" {
		filterLine = styles.StyleKey.Render("filter: ") + v.filter + "\n\n"
	}

	if len(v.filtered) == 0 {
		noResources := styles.StyleFaintItalic.Render("No resources found.")
		return styles.StylePadded.Render(title + "\n\n" + filterLine + noResources)
	}

	var b strings.Builder

	// Calculate visible area
	maxVisible := height - 6
	if maxVisible < 3 {
		maxVisible = 3
	}

	startIdx := 0
	if v.selected >= maxVisible {
		startIdx = v.selected - maxVisible + 1
	}
	endIdx := startIdx + maxVisible
	if endIdx > len(v.filtered) {
		endIdx = len(v.filtered)
	}

	for i := startIdx; i < endIdx; i++ {
		r := v.filtered[i]
		row := v.renderResourceRow(r)
		if i == v.selected {
			row = styles.StyleSelected.Width(width - 6).Render(row)
		}
		b.WriteString(row)
		b.WriteByte('\n')
	}

	count := styles.StyleFaint.Render(fmt.Sprintf("%d resources", len(v.filtered)))
	if len(v.filtered) != len(v.resources) {
		count = styles.StyleFaint.Render(fmt.Sprintf("%d/%d resources", len(v.filtered), len(v.resources)))
	}

	hint := styles.StyleFaintItalic.Render("j/k navigate  / filter  Esc back")

	content := title + "\n\n" + filterLine + b.String() + "\n" + count + "\n" + hint
	return styles.StylePadded.Render(content)
}

func (v StateView) renderResourceRow(r terraform.Resource) string {
	address := r.Address
	typeInfo := styles.StyleFaint.Render(r.Type)

	row := fmt.Sprintf(" %s  %s", address, typeInfo)
	if r.Module != "" {
		module := styles.StyleKey.Render(fmt.Sprintf("[%s]", r.Module))
		row += " " + module
	}
	return row
}
