package sdk

import "testing"

func TestWorkspace_String_WhenCalled_ShouldReturnUnderlyingString(t *testing.T) {
	tests := []struct {
		name      string
		workspace Workspace
		want      string
	}{
		{"ShouldReturnDefault", WorkspaceDefault, "default"},
		{"ShouldReturnCustomName", NewWorkspace("production"), "production"},
		{"ShouldReturnEmptyForZeroValue", Workspace(""), ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.workspace.String()
			if got != tt.want {
				t.Errorf("Workspace(%q).String() = %q, want %q", string(tt.workspace), got, tt.want)
			}
		})
	}
}

func TestWorkspace_IsDefault_WhenCalled_ShouldIdentifyDefaultWorkspace(t *testing.T) {
	tests := []struct {
		name      string
		workspace Workspace
		want      bool
	}{
		{"ShouldReturnTrueForDefault", WorkspaceDefault, true},
		{"ShouldReturnFalseForCustom", NewWorkspace("staging"), false},
		{"ShouldReturnFalseForZeroValue", Workspace(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.workspace.IsDefault()
			if got != tt.want {
				t.Errorf("Workspace(%q).IsDefault() = %v, want %v", string(tt.workspace), got, tt.want)
			}
		})
	}
}

func TestWorkspace_IsZero_WhenCalled_ShouldIdentifyEmptyWorkspace(t *testing.T) {
	tests := []struct {
		name      string
		workspace Workspace
		want      bool
	}{
		{"ShouldReturnTrueForEmpty", Workspace(""), true},
		{"ShouldReturnFalseForDefault", WorkspaceDefault, false},
		{"ShouldReturnFalseForCustom", NewWorkspace("dev"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.workspace.IsZero()
			if got != tt.want {
				t.Errorf("Workspace(%q).IsZero() = %v, want %v", string(tt.workspace), got, tt.want)
			}
		})
	}
}

func TestNewWorkspace_WhenCalled_ShouldConstructCorrectly(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  Workspace
	}{
		{"ShouldCreateFromName", "production", Workspace("production")},
		{"ShouldCreateDefault", "default", WorkspaceDefault},
		{"ShouldCreateEmpty", "", Workspace("")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewWorkspace(tt.input)
			if got != tt.want {
				t.Errorf("NewWorkspace(%q) = %q, want %q", tt.input, string(got), string(tt.want))
			}
		})
	}
}
