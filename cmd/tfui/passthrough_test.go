package main

import "testing"

func TestPassthroughWarning_WhenArgsAndMode_ShouldWarnOnlyForLiveRuns(t *testing.T) {
	tests := []struct {
		name      string
		extraArgs []string
		recording bool
		want      string
	}{
		{name: "no args on a live run", extraArgs: nil, recording: false, want: ""},
		{name: "no args on a recording run", extraArgs: nil, recording: true, want: ""},
		{
			name:      "args on a recording run",
			extraArgs: []string{"-compact-warnings", "-no-color"},
			recording: true,
			want:      "",
		},
		{
			name:      "args on a live run",
			extraArgs: []string{"-compact-warnings", "-no-color"},
			recording: false,
			want:      "warning: ignoring arguments after --: -compact-warnings -no-color; only a -macro or -dry-run run records them\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := passthroughWarning(tt.extraArgs, tt.recording)
			if got != tt.want {
				t.Errorf("passthroughWarning(%q, %v) = %q, want %q", tt.extraArgs, tt.recording, got, tt.want)
			}
		})
	}
}
