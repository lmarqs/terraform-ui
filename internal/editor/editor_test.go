package editor

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDetectEditor(t *testing.T) {
	tests := []struct {
		name     string
		visual   string
		editor   string
		expected string
	}{
		{
			name:     "VISUAL set takes priority",
			visual:   "code",
			editor:   "vim",
			expected: "code",
		},
		{
			name:     "EDITOR used when VISUAL is empty",
			visual:   "",
			editor:   "nano",
			expected: "nano",
		},
		{
			name:     "defaults to vi when neither set",
			visual:   "",
			editor:   "",
			expected: "vi",
		},
		{
			name:     "VISUAL with full path",
			visual:   "/usr/local/bin/nvim",
			editor:   "",
			expected: "/usr/local/bin/nvim",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("VISUAL", tt.visual)
			t.Setenv("EDITOR", tt.editor)

			result := DetectEditor()
			if result != tt.expected {
				t.Errorf("DetectEditor() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestBuildArgs(t *testing.T) {
	tests := []struct {
		name     string
		editor   string
		loc      SourceLocation
		expected []string
	}{
		{
			name:     "vim produces +LINE FILE",
			editor:   "vim",
			loc:      SourceLocation{File: "/tmp/main.tf", Line: 42},
			expected: []string{"+42", "/tmp/main.tf"},
		},
		{
			name:     "nvim produces +LINE FILE",
			editor:   "nvim",
			loc:      SourceLocation{File: "/tmp/main.tf", Line: 10},
			expected: []string{"+10", "/tmp/main.tf"},
		},
		{
			name:     "vim with full path",
			editor:   "/usr/bin/vim",
			loc:      SourceLocation{File: "/tmp/main.tf", Line: 5},
			expected: []string{"+5", "/tmp/main.tf"},
		},
		{
			name:     "code produces --goto FILE:LINE --wait",
			editor:   "code",
			loc:      SourceLocation{File: "/tmp/main.tf", Line: 15},
			expected: []string{"--goto", "/tmp/main.tf:15", "--wait"},
		},
		{
			name:     "code with col produces --goto FILE:LINE:COL --wait",
			editor:   "code",
			loc:      SourceLocation{File: "/tmp/main.tf", Line: 15, Col: 3},
			expected: []string{"--goto", "/tmp/main.tf:15:3", "--wait"},
		},
		{
			name:     "vscode in path detected as code",
			editor:   "/usr/local/bin/vscode",
			loc:      SourceLocation{File: "/tmp/main.tf", Line: 7},
			expected: []string{"--goto", "/tmp/main.tf:7", "--wait"},
		},
		{
			name:     "nano produces +LINE FILE",
			editor:   "nano",
			loc:      SourceLocation{File: "/tmp/main.tf", Line: 20},
			expected: []string{"+20", "/tmp/main.tf"},
		},
		{
			name:     "emacs produces +LINE FILE",
			editor:   "emacs",
			loc:      SourceLocation{File: "/tmp/main.tf", Line: 33},
			expected: []string{"+33", "/tmp/main.tf"},
		},
		{
			name:     "emacsclient produces +LINE FILE",
			editor:   "emacsclient",
			loc:      SourceLocation{File: "/tmp/main.tf", Line: 12},
			expected: []string{"+12", "/tmp/main.tf"},
		},
		{
			name:     "hx produces FILE:LINE",
			editor:   "hx",
			loc:      SourceLocation{File: "/tmp/main.tf", Line: 8},
			expected: []string{"/tmp/main.tf:8"},
		},
		{
			name:     "helix produces FILE:LINE",
			editor:   "helix",
			loc:      SourceLocation{File: "/tmp/main.tf", Line: 99},
			expected: []string{"/tmp/main.tf:99"},
		},
		{
			name:     "subl produces FILE:LINE",
			editor:   "subl",
			loc:      SourceLocation{File: "/tmp/main.tf", Line: 4},
			expected: []string{"/tmp/main.tf:4"},
		},
		{
			name:     "sublime_text produces FILE:LINE",
			editor:   "sublime_text",
			loc:      SourceLocation{File: "/tmp/main.tf", Line: 55},
			expected: []string{"/tmp/main.tf:55"},
		},
		{
			name:     "micro produces FILE +LINE",
			editor:   "micro",
			loc:      SourceLocation{File: "/tmp/main.tf", Line: 17},
			expected: []string{"/tmp/main.tf", "+17"},
		},
		{
			name:     "unknown editor defaults to +LINE FILE",
			editor:   "ed",
			loc:      SourceLocation{File: "/tmp/main.tf", Line: 3},
			expected: []string{"+3", "/tmp/main.tf"},
		},
		{
			name:     "line 0 produces FILE only for vim",
			editor:   "vim",
			loc:      SourceLocation{File: "/tmp/main.tf", Line: 0},
			expected: []string{"/tmp/main.tf"},
		},
		{
			name:     "line 0 produces FILE only for code",
			editor:   "code",
			loc:      SourceLocation{File: "/tmp/main.tf", Line: 0},
			expected: []string{"/tmp/main.tf"},
		},
		{
			name:     "line 0 produces FILE only for unknown",
			editor:   "whatever",
			loc:      SourceLocation{File: "/tmp/main.tf", Line: 0},
			expected: []string{"/tmp/main.tf"},
		},
		{
			name:     "negative line treated as no line",
			editor:   "vim",
			loc:      SourceLocation{File: "/tmp/main.tf", Line: -1},
			expected: []string{"/tmp/main.tf"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildArgs(tt.editor, tt.loc, nil)
			if len(result) != len(tt.expected) {
				t.Fatalf("buildArgs(%q, %+v) returned %d args %v, want %d args %v",
					tt.editor, tt.loc, len(result), result, len(tt.expected), tt.expected)
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("buildArgs(%q, %+v)[%d] = %q, want %q",
						tt.editor, tt.loc, i, result[i], tt.expected[i])
				}
			}
		})
	}
}

func TestSourceLocationFields(t *testing.T) {
	loc := SourceLocation{
		File: "/path/to/main.tf",
		Line: 42,
		Col:  7,
	}

	if loc.File != "/path/to/main.tf" {
		t.Errorf("File = %q, want %q", loc.File, "/path/to/main.tf")
	}
	if loc.Line != 42 {
		t.Errorf("Line = %d, want %d", loc.Line, 42)
	}
	if loc.Col != 7 {
		t.Errorf("Col = %d, want %d", loc.Col, 7)
	}
}

func TestEditorClosedMsgFields(t *testing.T) {
	msg := EditorClosedMsg{
		File:     "/path/to/main.tf",
		Modified: true,
		Err:      nil,
	}

	if msg.File != "/path/to/main.tf" {
		t.Errorf("File = %q, want %q", msg.File, "/path/to/main.tf")
	}
	if !msg.Modified {
		t.Error("Modified = false, want true")
	}
	if msg.Err != nil {
		t.Errorf("Err = %v, want nil", msg.Err)
	}

	errMsg := EditorClosedMsg{
		File:     "/other/file.tf",
		Modified: false,
		Err:      errors.New("test error"),
	}
	if errMsg.Err == nil {
		t.Error("Err = nil, want non-nil")
	}
}

func TestOpenMultiple_EmptyReturnsNil(t *testing.T) {
	cmd := OpenMultiple(nil)
	if cmd != nil {
		t.Error("OpenMultiple(nil) should return nil")
	}
	cmd = OpenMultiple([]SourceLocation{})
	if cmd != nil {
		t.Error("OpenMultiple([]) should return nil")
	}
}

func TestOpenMultiple_SingleDelegatesToOpen(t *testing.T) {
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "vim")
	locs := []SourceLocation{{File: "/tmp/test.tf", Line: 5}}
	cmd := OpenMultiple(locs)
	if cmd == nil {
		t.Error("OpenMultiple with 1 loc should return non-nil cmd")
	}
}

func TestOpenMultiple_CodeMultipleFiles(t *testing.T) {
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "code --wait")
	locs := []SourceLocation{
		{File: "/tmp/a.tf", Line: 10},
		{File: "/tmp/b.tf", Line: 20},
		{File: "/tmp/c.tf", Line: 0},
	}
	cmd := OpenMultiple(locs)
	if cmd == nil {
		t.Error("OpenMultiple with code editor should return non-nil cmd")
	}
}

func TestOpenMultiple_CodeWithoutWaitFlag(t *testing.T) {
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "code")
	locs := []SourceLocation{
		{File: "/tmp/a.tf", Line: 10},
		{File: "/tmp/b.tf", Line: 20},
	}
	cmd := OpenMultiple(locs)
	if cmd == nil {
		t.Error("OpenMultiple with code (no --wait) should return non-nil cmd")
	}
}

func TestOpenMultiple_NonCodeFallsBackToFirst(t *testing.T) {
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "vim")
	locs := []SourceLocation{
		{File: "/tmp/a.tf", Line: 10},
		{File: "/tmp/b.tf", Line: 20},
	}
	cmd := OpenMultiple(locs)
	if cmd == nil {
		t.Error("OpenMultiple with vim should return non-nil cmd (first file)")
	}
}

func TestSplitCommand(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantBin  string
		wantArgs []string
	}{
		{"simple binary", "vim", "vim", nil},
		{"binary with path", "/usr/bin/vim", "/usr/bin/vim", nil},
		{"binary with args", "code --wait", "code", []string{"--wait"}},
		{"binary with multiple args", "emacsclient -n -c", "emacsclient", []string{"-n", "-c"}},
		{"empty string", "", "vi", nil},
		{"extra whitespace", "  code   --wait  ", "code", []string{"--wait"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bin, args := splitCommand(tt.input)
			if bin != tt.wantBin {
				t.Errorf("splitCommand(%q) bin = %q, want %q", tt.input, bin, tt.wantBin)
			}
			if len(args) != len(tt.wantArgs) {
				t.Fatalf("splitCommand(%q) args = %v, want %v", tt.input, args, tt.wantArgs)
			}
			for i := range args {
				if args[i] != tt.wantArgs[i] {
					t.Errorf("splitCommand(%q) args[%d] = %q, want %q", tt.input, i, args[i], tt.wantArgs[i])
				}
			}
		})
	}
}

func TestBuildArgs_CodeWithExistingWait(t *testing.T) {
	loc := SourceLocation{File: "/tmp/main.tf", Line: 10}

	t.Run("ShouldNotDuplicateWaitFlag", func(t *testing.T) {
		result := buildArgs("code", loc, []string{"--wait"})
		waitCount := 0
		for _, a := range result {
			if a == "--wait" {
				waitCount++
			}
		}
		if waitCount != 0 {
			t.Errorf("expected no --wait in locArgs (already in editorArgs), got %v", result)
		}
	})

	t.Run("ShouldAddWaitWhenNotPresent", func(t *testing.T) {
		result := buildArgs("code", loc, nil)
		hasWait := false
		for _, a := range result {
			if a == "--wait" {
				hasWait = true
			}
		}
		if !hasWait {
			t.Errorf("expected --wait in result, got %v", result)
		}
	})
}

func TestOpen_WhenFileExists_ShouldReturnNonNilCmd(t *testing.T) {
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "vim")

	dir := t.TempDir()
	file := filepath.Join(dir, "main.tf")
	if err := os.WriteFile(file, []byte("resource {}"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := Open(SourceLocation{File: file, Line: 10})
	if cmd == nil {
		t.Error("Open() should return non-nil cmd for valid file")
	}
}

func TestOpen_WhenFileDoesNotExist_ShouldReturnNonNilCmd(t *testing.T) {
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "vim")

	cmd := Open(SourceLocation{File: "/nonexistent/file.tf", Line: 5})
	if cmd == nil {
		t.Error("Open() should return non-nil cmd even for nonexistent file")
	}
}

func TestOpenFile_ShouldReturnNonNilCmd(t *testing.T) {
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "vim")

	dir := t.TempDir()
	file := filepath.Join(dir, "main.tf")
	if err := os.WriteFile(file, []byte("resource {}"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := OpenFile(file)
	if cmd == nil {
		t.Error("OpenFile() should return non-nil cmd")
	}
}

func TestOpenFile_WhenFileDoesNotExist_ShouldReturnNonNilCmd(t *testing.T) {
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "vim")

	cmd := OpenFile("/nonexistent/path/file.tf")
	if cmd == nil {
		t.Error("OpenFile() should return non-nil cmd even for nonexistent file")
	}
}

func TestMakeEditorCallback_WhenFileModified_ShouldReportModified(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "test.tf")
	if err := os.WriteFile(file, []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}

	mtimeBefore := time.Now().Add(-1 * time.Second)

	cb := makeEditorCallback(file, mtimeBefore)
	msg := cb(nil)

	closedMsg, ok := msg.(EditorClosedMsg)
	if !ok {
		t.Fatalf("expected EditorClosedMsg, got %T", msg)
	}
	if closedMsg.File != file {
		t.Errorf("File = %q, want %q", closedMsg.File, file)
	}
	if !closedMsg.Modified {
		t.Error("Modified = false, want true (file mtime is after mtimeBefore)")
	}
	if closedMsg.Err != nil {
		t.Errorf("Err = %v, want nil", closedMsg.Err)
	}
}

func TestMakeEditorCallback_WhenFileNotModified_ShouldReportUnmodified(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "test.tf")
	if err := os.WriteFile(file, []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(file)
	if err != nil {
		t.Fatal(err)
	}
	mtimeBefore := info.ModTime()

	cb := makeEditorCallback(file, mtimeBefore)
	msg := cb(nil)

	closedMsg, ok := msg.(EditorClosedMsg)
	if !ok {
		t.Fatalf("expected EditorClosedMsg, got %T", msg)
	}
	if closedMsg.Modified {
		t.Error("Modified = true, want false (file was not modified)")
	}
}

func TestMakeEditorCallback_WhenFileDoesNotExist_ShouldReportUnmodified(t *testing.T) {
	cb := makeEditorCallback("/nonexistent/path/file.tf", time.Now())
	msg := cb(nil)

	closedMsg, ok := msg.(EditorClosedMsg)
	if !ok {
		t.Fatalf("expected EditorClosedMsg, got %T", msg)
	}
	if closedMsg.Modified {
		t.Error("Modified = true, want false (file does not exist)")
	}
	if closedMsg.Err != nil {
		t.Errorf("Err = %v, want nil", closedMsg.Err)
	}
}

func TestMakeEditorCallback_WhenEditorReturnsError_ShouldPropagateError(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "test.tf")
	if err := os.WriteFile(file, []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}

	editorErr := errors.New("editor crashed")
	cb := makeEditorCallback(file, time.Now().Add(-1*time.Second))
	msg := cb(editorErr)

	closedMsg, ok := msg.(EditorClosedMsg)
	if !ok {
		t.Fatalf("expected EditorClosedMsg, got %T", msg)
	}
	if closedMsg.Err != editorErr {
		t.Errorf("Err = %v, want %v", closedMsg.Err, editorErr)
	}
}

func TestMakeMultiFileCallback_WhenNoError_ShouldReportModifiedTrue(t *testing.T) {
	cb := makeMultiFileCallback("/tmp/primary.tf")
	msg := cb(nil)

	closedMsg, ok := msg.(EditorClosedMsg)
	if !ok {
		t.Fatalf("expected EditorClosedMsg, got %T", msg)
	}
	if closedMsg.File != "/tmp/primary.tf" {
		t.Errorf("File = %q, want %q", closedMsg.File, "/tmp/primary.tf")
	}
	if !closedMsg.Modified {
		t.Error("Modified = false, want true")
	}
	if closedMsg.Err != nil {
		t.Errorf("Err = %v, want nil", closedMsg.Err)
	}
}

func TestMakeMultiFileCallback_WhenError_ShouldPropagateError(t *testing.T) {
	editorErr := errors.New("code exited with error")
	cb := makeMultiFileCallback("/tmp/primary.tf")
	msg := cb(editorErr)

	closedMsg, ok := msg.(EditorClosedMsg)
	if !ok {
		t.Fatalf("expected EditorClosedMsg, got %T", msg)
	}
	if closedMsg.Err != editorErr {
		t.Errorf("Err = %v, want %v", closedMsg.Err, editorErr)
	}
	if !closedMsg.Modified {
		t.Error("Modified = false, want true (always true for multi-file)")
	}
}

func TestHasFlag(t *testing.T) {
	tests := []struct {
		name string
		args []string
		flag string
		want bool
	}{
		{"found in args", []string{"--wait", "--goto"}, "--wait", true},
		{"not found in args", []string{"--goto"}, "--wait", false},
		{"nil args", nil, "--wait", false},
		{"empty args", []string{}, "--wait", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasFlag(tt.args, tt.flag)
			if got != tt.want {
				t.Errorf("hasFlag(%v, %q) = %v, want %v", tt.args, tt.flag, got, tt.want)
			}
		})
	}
}
