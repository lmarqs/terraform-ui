package main

import "testing"

func TestPassthroughWarning_WhenArgsAndMode_ShouldWarnOnlyForLiveRuns(t *testing.T) {
	tests := []struct {
		name        string
		extraArgs   []string
		macroActive bool
		want        string
	}{
		{name: "no args on a live run", extraArgs: nil, macroActive: false, want: ""},
		{name: "no args on a macro run", extraArgs: nil, macroActive: true, want: ""},
		{
			name:        "args on a macro run",
			extraArgs:   []string{"-compact-warnings", "-no-color"},
			macroActive: true,
			want:        "",
		},
		{
			name:        "args on a live run",
			extraArgs:   []string{"-compact-warnings", "-no-color"},
			macroActive: false,
			want:        "warning: ignoring arguments after --: -compact-warnings -no-color; only a -macro run records them\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := passthroughWarning(tt.extraArgs, tt.macroActive)
			if got != tt.want {
				t.Errorf("passthroughWarning(%q, %v) = %q, want %q", tt.extraArgs, tt.macroActive, got, tt.want)
			}
		})
	}
}
