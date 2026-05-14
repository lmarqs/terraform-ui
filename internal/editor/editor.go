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
	bin, editorArgs := splitCommand(editor)
	locArgs := buildArgs(bin, loc, editorArgs)
	args := append(editorArgs, locArgs...)

	// Record mtime before opening
	var mtimeBefore time.Time
	if info, err := os.Stat(loc.File); err == nil {
		mtimeBefore = info.ModTime()
	}

	c := exec.Command(bin, args...)
	return tea.ExecProcess(c, makeEditorCallback(loc.File, mtimeBefore))
}

// makeEditorCallback returns a callback function that checks whether the file
// was modified during editing (by comparing mtime).
func makeEditorCallback(file string, mtimeBefore time.Time) func(error) tea.Msg {
	return func(err error) tea.Msg {
		modified := false
		if info, statErr := os.Stat(file); statErr == nil {
			modified = info.ModTime().After(mtimeBefore)
		}
		return EditorClosedMsg{
			File:     file,
			Modified: modified,
			Err:      err,
		}
	}
}

// makeMultiFileCallback returns a callback for multi-file editor invocations.
// It always reports modified=true since tracking multiple files is impractical.
func makeMultiFileCallback(primaryFile string) func(error) tea.Msg {
	return func(err error) tea.Msg {
		return EditorClosedMsg{File: primaryFile, Modified: true, Err: err}
	}
}

// splitCommand splits an editor string like "code --wait" into binary and args.
func splitCommand(editor string) (string, []string) {
	parts := strings.Fields(editor)
	if len(parts) == 0 {
		return "vi", nil
	}
	return parts[0], parts[1:]
}

// OpenMultiple opens multiple source locations in a single editor invocation.
// For editors that support multiple --goto args (VS Code), all files open at once.
// For others, only the first location is opened.
func OpenMultiple(locs []SourceLocation) tea.Cmd {
	if len(locs) == 0 {
		return nil
	}
	if len(locs) == 1 {
		return Open(locs[0])
	}

	ed := detectEditor()
	bin, editorArgs := splitCommand(ed)
	baseLower := strings.ToLower(filepath.Base(bin))

	// VS Code supports multiple --goto args
	if strings.Contains(baseLower, "code") {
		var args []string
		args = append(args, editorArgs...)
		for _, loc := range locs {
			if loc.Line > 0 {
				args = append(args, "--goto", fmt.Sprintf("%s:%d", loc.File, loc.Line))
			} else {
				args = append(args, loc.File)
			}
		}
		if !hasFlag(editorArgs, "--wait") {
			args = append(args, "--wait")
		}

		c := exec.Command(bin, args...)
		return tea.ExecProcess(c, makeMultiFileCallback(locs[0].File))
	}

	// Fallback: open first file only
	return Open(locs[0])
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
func buildArgs(bin string, loc SourceLocation, existingArgs []string) []string {
	if loc.Line <= 0 {
		return []string{loc.File}
	}

	base := filepath.Base(bin)
	baseLower := strings.ToLower(base)

	switch {
	case strings.Contains(baseLower, "nvim") || strings.Contains(baseLower, "vim"):
		return []string{fmt.Sprintf("+%d", loc.Line), loc.File}
	case strings.Contains(baseLower, "code"):
		args := []string{"--goto", fmt.Sprintf("%s:%d", loc.File, loc.Line)}
		if loc.Col > 0 {
			args = []string{"--goto", fmt.Sprintf("%s:%d:%d", loc.File, loc.Line, loc.Col)}
		}
		if !hasFlag(existingArgs, "--wait") {
			args = append(args, "--wait")
		}
		return args
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
		return []string{fmt.Sprintf("+%d", loc.Line), loc.File}
	}
}

func hasFlag(args []string, flag string) bool {
	for _, a := range args {
		if a == flag {
			return true
		}
	}
	return false
}
