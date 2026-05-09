package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

// AIPanel renders a floating overlay panel with AI-generated content.
type AIPanel struct {
	visible bool
	title   string
	content string
	loading bool
	width   int
	height  int
	scrollY int
}

// NewAIPanel creates a new hidden AI panel.
func NewAIPanel() AIPanel {
	return AIPanel{}
}

// Show makes the panel visible with initial loading state.
func (p AIPanel) Show(title string) AIPanel {
	p.visible = true
	p.title = title
	p.content = ""
	p.loading = true
	p.scrollY = 0
	return p
}

// Hide dismisses the panel.
func (p AIPanel) Hide() AIPanel {
	p.visible = false
	p.content = ""
	p.loading = false
	return p
}

// AppendContent adds streaming content to the panel.
func (p AIPanel) AppendContent(chunk string) AIPanel {
	p.content += chunk
	return p
}

// SetDone marks streaming as complete.
func (p AIPanel) SetDone() AIPanel {
	p.loading = false
	return p
}

// SetError sets an error message.
func (p AIPanel) SetError(err string) AIPanel {
	p.loading = false
	p.content = "Error: " + err
	return p
}

// IsVisible reports whether the panel is showing.
func (p AIPanel) IsVisible() bool {
	return p.visible
}

// SetSize updates the panel dimensions.
func (p AIPanel) SetSize(width, height int) AIPanel {
	p.width = width
	p.height = height
	return p
}

// ScrollUp scrolls the panel content up.
func (p AIPanel) ScrollUp() AIPanel {
	if p.scrollY > 0 {
		p.scrollY--
	}
	return p
}

// ScrollDown scrolls the panel content down.
func (p AIPanel) ScrollDown() AIPanel {
	p.scrollY++
	return p
}

// Render returns the panel as a styled overlay string.
func (p AIPanel) Render(width, height int) string {
	if !p.visible {
		return ""
	}

	panelWidth := width - 8
	if panelWidth < 40 {
		panelWidth = 40
	}
	if panelWidth > 80 {
		panelWidth = 80
	}

	panelHeight := height - 6
	if panelHeight < 5 {
		panelHeight = 5
	}

	// Title
	title := sdk.StyleTitle.Render("AI: " + p.title)

	// Content
	var body string
	if p.loading && p.content == "" {
		body = sdk.StyleFaintItalic.Render("Thinking...")
	} else {
		lines := strings.Split(p.content, "\n")
		// Wrap lines to panel width
		var wrapped []string
		for _, line := range lines {
			if len(line) <= panelWidth-4 {
				wrapped = append(wrapped, line)
			} else {
				for len(line) > panelWidth-4 {
					wrapped = append(wrapped, line[:panelWidth-4])
					line = line[panelWidth-4:]
				}
				if len(line) > 0 {
					wrapped = append(wrapped, line)
				}
			}
		}

		// Apply scroll
		maxScroll := len(wrapped) - panelHeight + 3
		if maxScroll < 0 {
			maxScroll = 0
		}
		if p.scrollY > maxScroll {
			p.scrollY = maxScroll
		}
		endIdx := p.scrollY + panelHeight - 3
		if endIdx > len(wrapped) {
			endIdx = len(wrapped)
		}
		if p.scrollY < len(wrapped) {
			body = strings.Join(wrapped[p.scrollY:endIdx], "\n")
		}

		if p.loading {
			body += "\n" + sdk.StyleFaintItalic.Render("▍")
		}
	}

	// Footer hint
	hint := sdk.StyleFaintItalic.Render("↑↓ scroll  Esc close")

	content := title + "\n\n" + body + "\n\n" + hint

	// Panel box style
	panelStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(sdk.ColorPrimary).
		Padding(1, 2).
		Width(panelWidth).
		MaxHeight(panelHeight)

	return panelStyle.Render(content)
}
