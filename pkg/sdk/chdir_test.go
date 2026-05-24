package sdk

import "testing"

func TestChdir_String_WhenCalled_ShouldReturnUnderlyingString(t *testing.T) {
	tests := []struct {
		name  string
		chdir Chdir
		want  string
	}{
		{"ShouldReturnPath", Chdir("modules/vpc"), "modules/vpc"},
		{"ShouldReturnEmptyForZeroValue", Chdir(""), ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.chdir.String(); got != tt.want {
				t.Errorf("Chdir(%q).String() = %q, want %q", string(tt.chdir), got, tt.want)
			}
		})
	}
}

func TestChdir_IsZero_WhenCalled_ShouldIdentifyEmptyChdir(t *testing.T) {
	tests := []struct {
		name  string
		chdir Chdir
		want  bool
	}{
		{"ShouldReturnTrueForEmpty", Chdir(""), true},
		{"ShouldReturnFalseForPath", Chdir("modules/vpc"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.chdir.IsZero(); got != tt.want {
				t.Errorf("Chdir(%q).IsZero() = %v, want %v", string(tt.chdir), got, tt.want)
			}
		})
	}
}
