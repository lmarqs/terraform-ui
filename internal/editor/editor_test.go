package editor

import (
	"testing"
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
		// vim/nvim
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
		// code/vscode
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
		// nano
		{
			name:     "nano produces +LINE FILE",
			editor:   "nano",
			loc:      SourceLocation{File: "/tmp/main.tf", Line: 20},
			expected: []string{"+20", "/tmp/main.tf"},
		},
		// emacs
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
		// helix/hx
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
		// sublime
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
		// micro
		{
			name:     "micro produces FILE +LINE",
			editor:   "micro",
			loc:      SourceLocation{File: "/tmp/main.tf", Line: 17},
			expected: []string{"/tmp/main.tf", "+17"},
		},
		// unknown editor
		{
			name:     "unknown editor defaults to +LINE FILE",
			editor:   "ed",
			loc:      SourceLocation{File: "/tmp/main.tf", Line: 3},
			expected: []string{"+3", "/tmp/main.tf"},
		},
		// Line = 0 (no line)
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
			result := buildArgs(tt.editor, tt.loc)
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

	// Test with error
	errMsg := EditorClosedMsg{
		File:     "/other/file.tf",
		Modified: false,
		Err:      &testError{},
	}
	if errMsg.Err == nil {
		t.Error("Err = nil, want non-nil")
	}
}

type testError struct{}

func (e *testError) Error() string { return "test error" }
