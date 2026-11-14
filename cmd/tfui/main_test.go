package main

import (
	"runtime/debug"
	"testing"
)

func TestResolveVersion_WhenSourcesVary_ShouldReturnVersionWithoutLeadingV(t *testing.T) {
	buildInfo := func(v string) *debug.BuildInfo {
		return &debug.BuildInfo{Main: debug.Module{Version: v}}
	}
	tests := []struct {
		name          string
		ldflagVersion string
		info          *debug.BuildInfo
		ok            bool
		want          string
	}{
		{name: "ldflag wins over build info", ldflagVersion: "1.28.0", info: buildInfo("v1.27.4"), ok: true, want: "1.28.0"},
		{name: "ldflag returned unchanged", ldflagVersion: "v1.28.0", info: nil, ok: false, want: "v1.28.0"},
		{name: "go install module version strips leading v", info: buildInfo("v1.27.4"), ok: true, want: "1.27.4"},
		{name: "pseudo-version strips leading v", info: buildInfo("v1.27.4-0.20261101120000-abcdef123456"), ok: true, want: "1.27.4-0.20261101120000-abcdef123456"},
		{name: "only one leading v stripped", info: buildInfo("vv1.0.0"), ok: true, want: "v1.0.0"},
		{name: "version without v kept", info: buildInfo("1.27.4"), ok: true, want: "1.27.4"},
		{name: "devel build falls back to snapshot", info: buildInfo("(devel)"), ok: true, want: "0.0.0-SNAPSHOT"},
		{name: "empty module version falls back to snapshot", info: buildInfo(""), ok: true, want: "0.0.0-SNAPSHOT"},
		{name: "no build info falls back to snapshot", info: nil, ok: false, want: "0.0.0-SNAPSHOT"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveVersion(tt.ldflagVersion, tt.info, tt.ok); got != tt.want {
				t.Errorf("resolveVersion(%q, %v, %v) = %q, want %q", tt.ldflagVersion, tt.info, tt.ok, got, tt.want)
			}
		})
	}
}
