package macro

import (
	"testing"
)

func TestParseTapeValid(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Command
	}{
		{
			"single key",
			"key p",
			[]Command{{Type: CmdKey, Args: []string{"p"}, Line: 1}},
		},
		{
			"multiple lines",
			"key p\nwait ready\nassert view create",
			[]Command{
				{Type: CmdKey, Args: []string{"p"}, Line: 1},
				{Type: CmdWaitReady, Line: 2},
				{Type: CmdAssertView, Args: []string{"create"}, Line: 3},
			},
		},
		{
			"inline semicolons",
			"key p; wait ready; assert view create",
			[]Command{
				{Type: CmdKey, Args: []string{"p"}, Line: 1},
				{Type: CmdWaitReady, Line: 2},
				{Type: CmdAssertView, Args: []string{"create"}, Line: 3},
			},
		},
		{
			"wait view with spaces in substring",
			"wait view to add",
			[]Command{{Type: CmdWaitView, Args: []string{"to add"}, Line: 1}},
		},
		{
			"assert view with spaces",
			"assert view 3 resources to create",
			[]Command{{Type: CmdAssertView, Args: []string{"3 resources to create"}, Line: 1}},
		},
		{
			"screenshot",
			"screenshot /tmp/output.txt",
			[]Command{{Type: CmdScreenshot, Args: []string{"/tmp/output.txt"}, Line: 1}},
		},
		{
			"resize",
			"resize 120 40",
			[]Command{{Type: CmdResize, Args: []string{"120", "40"}, Line: 1}},
		},
		{
			"sleep",
			"sleep 500ms",
			[]Command{{Type: CmdSleep, Args: []string{"500ms"}, Line: 1}},
		},
		{
			"sleep seconds",
			"sleep 2s",
			[]Command{{Type: CmdSleep, Args: []string{"2s"}, Line: 1}},
		},
		{
			"comments and empty lines",
			"# this is a comment\n\nkey p\n# another comment\nassert view ok\n\n",
			[]Command{
				{Type: CmdKey, Args: []string{"p"}, Line: 3},
				{Type: CmdAssertView, Args: []string{"ok"}, Line: 5},
			},
		},
		{
			"special keys",
			"key enter\nkey esc\nkey space\nkey ctrl+c",
			[]Command{
				{Type: CmdKey, Args: []string{"enter"}, Line: 1},
				{Type: CmdKey, Args: []string{"esc"}, Line: 2},
				{Type: CmdKey, Args: []string{"space"}, Line: 3},
				{Type: CmdKey, Args: []string{"ctrl+c"}, Line: 4},
			},
		},
		{
			"whitespace trimmed",
			"  key p  \n  wait ready  ",
			[]Command{
				{Type: CmdKey, Args: []string{"p"}, Line: 1},
				{Type: CmdWaitReady, Line: 2},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseTape([]byte(tt.input))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tt.expected) {
				t.Fatalf("len = %d, want %d", len(got), len(tt.expected))
			}
			for i, cmd := range got {
				exp := tt.expected[i]
				if cmd.Type != exp.Type {
					t.Errorf("[%d] type = %d, want %d", i, cmd.Type, exp.Type)
				}
				if cmd.Line != exp.Line {
					t.Errorf("[%d] line = %d, want %d", i, cmd.Line, exp.Line)
				}
				if len(cmd.Args) != len(exp.Args) {
					t.Errorf("[%d] args = %v, want %v", i, cmd.Args, exp.Args)
					continue
				}
				for j, arg := range cmd.Args {
					if arg != exp.Args[j] {
						t.Errorf("[%d] args[%d] = %q, want %q", i, j, arg, exp.Args[j])
					}
				}
			}
		})
	}
}

func TestParseTapeErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"unknown command", "fly away"},
		{"key no arg", "key"},
		{"key too many args", "key a b"},
		{"wait no arg", "wait"},
		{"wait unknown target", "wait forever"},
		{"wait ready with args", "wait ready now"},
		{"wait view no substring", "wait view"},
		{"assert no args", "assert"},
		{"assert unknown target with insufficient args", "assert something"},
		{"assert unknown target", "assert foo bar"},
		{"assert view no substring", "assert view"},
		{"screenshot no arg", "screenshot"},
		{"screenshot too many args", "screenshot a b"},
		{"resize no args", "resize"},
		{"resize one arg", "resize 80"},
		{"resize three args", "resize 80 24 1"},
		{"resize non-integer width", "resize abc 24"},
		{"resize non-integer height", "resize 80 abc"},
		{"sleep no arg", "sleep"},
		{"sleep invalid duration", "sleep forever"},
		{"sleep too many args", "sleep 1s 2s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseTape([]byte(tt.input))
			if err == nil {
				t.Error("expected error")
			}
		})
	}
}

func TestParseTapeEmptyInput(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty string", ""},
		{"only whitespace", "   \n  \n  "},
		{"only comments", "# comment\n# another"},
		{"only newlines", "\n\n\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseTape([]byte(tt.input))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != 0 {
				t.Errorf("len = %d, want 0", len(got))
			}
		})
	}
}

func TestParseTapeLineNumbers(t *testing.T) {
	input := `# header comment

key p
# middle comment
wait ready

assert view hello`

	cmds, err := ParseTape([]byte(input))
	if err != nil {
		t.Fatal(err)
	}

	if cmds[0].Line != 3 {
		t.Errorf("first command line = %d, want 3", cmds[0].Line)
	}
	if cmds[1].Line != 5 {
		t.Errorf("second command line = %d, want 5", cmds[1].Line)
	}
	if cmds[2].Line != 7 {
		t.Errorf("third command line = %d, want 7", cmds[2].Line)
	}
}

func TestParseTapeErrorLineNumbers(t *testing.T) {
	input := "key p\nwait ready\nbad command"
	_, err := ParseTape([]byte(input))
	if err == nil {
		t.Fatal("expected error")
	}
	// Error should reference line 3
	if !containsStr(err.Error(), "line 3") {
		t.Errorf("error should reference line 3, got: %v", err)
	}
}

func TestParseLineEmptyCommand(t *testing.T) {
	_, err := parseLine("", 1)
	if err == nil {
		t.Error("expected error for empty line")
	}
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
