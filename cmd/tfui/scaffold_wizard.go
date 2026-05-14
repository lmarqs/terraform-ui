package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/internal/scaffold"
)

type scaffoldState int

const (
	scaffoldBinary scaffoldState = iota
	scaffoldReview
	scaffoldConfirm
	scaffoldDone
	scaffoldAborted
)

type scaffoldResult struct {
	Content string
	Aborted bool
}

var availableBinaries = []string{"terraform", "tofu", "terragrunt"}

type scaffoldWizard struct {
	binaries       []string
	binarySelected int
	binary         string
	members        []scaffold.DetectedMember
	selected       int
	state          scaffoldState
	preview        string
}

func newScaffoldWizard(dir string) *scaffoldWizard {
	detected := scaffold.DetectBinary()
	binaryIdx := 0
	for i, b := range availableBinaries {
		if b == detected {
			binaryIdx = i
			break
		}
	}
	return &scaffoldWizard{
		binaries:       availableBinaries,
		binarySelected: binaryIdx,
		binary:         detected,
		members:        scaffold.DetectMembers(dir),
		state:          scaffoldBinary,
	}
}

func (w *scaffoldWizard) Init() tea.Cmd {
	return nil
}

func (w *scaffoldWizard) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch w.state {
		case scaffoldBinary:
			return w.updateBinary(msg)
		case scaffoldReview:
			return w.updateReview(msg)
		case scaffoldConfirm:
			return w.updateConfirm(msg)
		}
	}
	return w, nil
}

func (w *scaffoldWizard) updateBinary(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if w.binarySelected < len(w.binaries)-1 {
			w.binarySelected++
		}
	case "k", "up":
		if w.binarySelected > 0 {
			w.binarySelected--
		}
	case "enter":
		w.binary = w.binaries[w.binarySelected]
		w.state = scaffoldReview
	case "esc", "q", "ctrl+c":
		w.state = scaffoldAborted
		return w, tea.Quit
	}
	return w, nil
}

func (w *scaffoldWizard) updateReview(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if w.selected < len(w.members)-1 {
			w.selected++
		}
	case "k", "up":
		if w.selected > 0 {
			w.selected--
		}
	case " ":
		if w.selected < len(w.members) {
			w.members[w.selected].Enabled = !w.members[w.selected].Enabled
		}
	case "enter":
		w.preview = w.generateHCL()
		w.state = scaffoldConfirm
	case "esc", "q", "ctrl+c":
		w.state = scaffoldAborted
		return w, tea.Quit
	}
	return w, nil
}

func (w *scaffoldWizard) updateConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "enter":
		w.state = scaffoldDone
		return w, tea.Quit
	case "n", "esc":
		w.state = scaffoldReview
	case "q", "ctrl+c":
		w.state = scaffoldAborted
		return w, tea.Quit
	}
	return w, nil
}

func (w *scaffoldWizard) View() string {
	switch w.state {
	case scaffoldBinary:
		return w.viewBinary()
	case scaffoldReview:
		return w.viewReview()
	case scaffoldConfirm:
		return w.viewConfirm()
	default:
		return ""
	}
}

func (w *scaffoldWizard) viewBinary() string {
	var b strings.Builder

	b.WriteString("Select terraform binary:\n\n")

	for i, bin := range w.binaries {
		cursor := "  "
		if i == w.binarySelected {
			cursor = "> "
		}
		fmt.Fprintf(&b, "%s%s\n", cursor, bin)
	}

	b.WriteString("\nEnter select · Esc abort")
	return b.String()
}

func (w *scaffoldWizard) viewReview() string {
	var b strings.Builder

	fmt.Fprintf(&b, "Binary: %s\n\n", w.binary)

	if len(w.members) == 0 {
		b.WriteString("No terraform directories detected.\n")
		b.WriteString("\nPress Enter to generate minimal config, or Esc to abort.")
		return b.String()
	}

	b.WriteString("Detected members (Space to toggle, Enter to confirm):\n\n")

	for i, m := range w.members {
		checkbox := "[ ]"
		if m.Enabled {
			checkbox = "[x]"
		}

		cursor := "  "
		if i == w.selected {
			cursor = "> "
		}

		fmt.Fprintf(&b, "%s%s %s\n", cursor, checkbox, m.Path)
	}

	b.WriteString("\nEnter confirm · Space toggle · Esc abort")
	return b.String()
}

func (w *scaffoldWizard) viewConfirm() string {
	var b strings.Builder

	b.WriteString("Preview (tfui.hcl):\n\n")
	b.WriteString(w.preview)
	b.WriteString("\nWrite this file? (y/n)")

	return b.String()
}

func (w *scaffoldWizard) generateHCL() string {
	var enabled []string
	for _, m := range w.members {
		if m.Enabled {
			enabled = append(enabled, m.Path)
		}
	}
	return scaffold.BuildHCL(w.binary, enabled)
}

func (w *scaffoldWizard) result() scaffoldResult {
	if w.state == scaffoldAborted {
		return scaffoldResult{Aborted: true}
	}
	return scaffoldResult{Content: w.preview}
}
