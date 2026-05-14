package sdk

import "testing"

func TestGetFlag_WhenFlagExists_ShouldReturnValue(t *testing.T) {
	args := ActionArgs{
		Flags: map[string]string{"target": "aws_instance.web", "lock": "true"},
	}

	got := args.GetFlag("target", "")
	if got != "aws_instance.web" {
		t.Errorf("GetFlag(target) = %q, want %q", got, "aws_instance.web")
	}
}

func TestGetFlag_WhenFlagMissing_ShouldReturnDefault(t *testing.T) {
	args := ActionArgs{
		Flags: map[string]string{"target": "aws_instance.web"},
	}

	got := args.GetFlag("missing", "fallback")
	if got != "fallback" {
		t.Errorf("GetFlag(missing) = %q, want %q", got, "fallback")
	}
}

func TestGetFlag_WhenFlagsNil_ShouldReturnDefault(t *testing.T) {
	args := ActionArgs{}

	got := args.GetFlag("anything", "default")
	if got != "default" {
		t.Errorf("GetFlag(anything) = %q, want %q", got, "default")
	}
}

func TestGetArg_WhenIndexValid_ShouldReturnArg(t *testing.T) {
	args := ActionArgs{
		Positional: []string{"first", "second", "third"},
	}

	tests := []struct {
		name  string
		index int
		want  string
	}{
		{"ShouldReturnFirst", 0, "first"},
		{"ShouldReturnSecond", 1, "second"},
		{"ShouldReturnThird", 2, "third"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := args.GetArg(tt.index)
			if got != tt.want {
				t.Errorf("GetArg(%d) = %q, want %q", tt.index, got, tt.want)
			}
		})
	}
}

func TestGetArg_WhenIndexOutOfBounds_ShouldReturnEmpty(t *testing.T) {
	args := ActionArgs{
		Positional: []string{"only"},
	}

	got := args.GetArg(5)
	if got != "" {
		t.Errorf("GetArg(5) = %q, want empty string", got)
	}
}

func TestGetArg_WhenPositionalNil_ShouldReturnEmpty(t *testing.T) {
	args := ActionArgs{}

	got := args.GetArg(0)
	if got != "" {
		t.Errorf("GetArg(0) = %q, want empty string", got)
	}
}

func TestHasFlag_WhenFlagExists_ShouldReturnTrue(t *testing.T) {
	args := ActionArgs{
		Flags: map[string]string{"verbose": "true"},
	}

	if !args.HasFlag("verbose") {
		t.Error("HasFlag(verbose) = false, want true")
	}
}

func TestHasFlag_WhenFlagMissing_ShouldReturnFalse(t *testing.T) {
	args := ActionArgs{
		Flags: map[string]string{"verbose": "true"},
	}

	if args.HasFlag("missing") {
		t.Error("HasFlag(missing) = true, want false")
	}
}

func TestHasFlag_WhenFlagsNil_ShouldReturnFalse(t *testing.T) {
	args := ActionArgs{}

	if args.HasFlag("anything") {
		t.Error("HasFlag(anything) = true, want false")
	}
}
