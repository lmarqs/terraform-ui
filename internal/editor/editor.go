package editor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// SourceLocation identifies a position in a terraform source file.
type SourceLocation struct {
	File string // absolute path to .tf file
	Line int    // line number (1-based)
	Col  int    // column (1-based, 0 means unknown)
}

// EditorClosedMsg is sent when the editor process exits.
type EditorClosedMsg struct {
	File     string
	Modified bool // true if file mtime changed during editing
	Err      error
}

// Open suspends the TUI and opens the user's editor at the given location.
// Uses tea.ExecProcess for proper terminal handoff.
func Open(loc SourceLocation) tea.Cmd {
	editor := detectEditor()
	args := buildArgs(editor, loc)

	// Record mtime before opening
	var mtimeBefore time.Time
	if info, err := os.Stat(loc.File); err == nil {
		mtimeBefore = info.ModTime()
	}

	c := exec.Command(editor, args...)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		modified := false
		if info, statErr := os.Stat(loc.File); statErr == nil {
			modified = info.ModTime().After(mtimeBefore)
		}
		return EditorClosedMsg{
			File:     loc.File,
			Modified: modified,
			Err:      err,
		}
	})
}

// OpenFile opens a file without jumping to a specific line.
func OpenFile(file string) tea.Cmd {
	return Open(SourceLocation{File: file, Line: 0})
}

// detectEditor returns the user's preferred editor.
func detectEditor() string {
	if e := os.Getenv("VISUAL"); e != "" {
		return e
	}
	if e := os.Getenv("EDITOR"); e != "" {
		return e
	}
	return "vi"
}

// DetectEditor returns the editor that would be used (for display purposes).
func DetectEditor() string {
	return detectEditor()
}

// buildArgs constructs editor-specific command line arguments for line jumping.
func buildArgs(editor string, loc SourceLocation) []string {
	if loc.Line <= 0 {
		return []string{loc.File}
	}

	base := filepath.Base(editor)
	// Handle editors that might have path prefixes or suffixes
	baseLower := strings.ToLower(base)

	switch {
	case strings.Contains(baseLower, "nvim") || strings.Contains(baseLower, "vim"):
		return []string{fmt.Sprintf("+%d", loc.Line), loc.File}
	case strings.Contains(baseLower, "code"):
		if loc.Col > 0 {
			return []string{"--goto", fmt.Sprintf("%s:%d:%d", loc.File, loc.Line, loc.Col), "--wait"}
		}
		return []string{"--goto", fmt.Sprintf("%s:%d", loc.File, loc.Line), "--wait"}
	case strings.Contains(baseLower, "nano"):
		return []string{fmt.Sprintf("+%d", loc.Line), loc.File}
	case strings.Contains(baseLower, "emacs"), strings.Contains(baseLower, "emacsclient"):
		return []string{fmt.Sprintf("+%d", loc.Line), loc.File}
	case strings.Contains(baseLower, "subl"), strings.Contains(baseLower, "sublime"):
		return []string{fmt.Sprintf("%s:%d", loc.File, loc.Line)}
	case strings.Contains(baseLower, "micro"):
		return []string{loc.File, fmt.Sprintf("+%d", loc.Line)}
	case strings.Contains(baseLower, "hx"), strings.Contains(baseLower, "helix"):
		return []string{fmt.Sprintf("%s:%d", loc.File, loc.Line)}
	default:
		// Fallback: try +line syntax (works for many editors)
		return []string{fmt.Sprintf("+%d", loc.Line), loc.File}
	}
}
