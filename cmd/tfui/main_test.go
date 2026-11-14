package main

import (
	"errors"
	"fmt"
	"runtime/debug"
	"testing"

	"github.com/lmarqs/terraform-ui/internal/macro"
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

func TestExitStatus_WhenErrorVaries_ShouldMapToExitCodeAndStderr(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantCode   int
		wantStderr string
	}{
		{name: "nil exits zero silently", err: nil, wantCode: 0, wantStderr: ""},
		{name: "exitError carries its code without a message", err: &exitError{code: 2}, wantCode: 2, wantStderr: ""},
		{name: "wrapped exitError still carries its code", err: fmt.Errorf("run: %w", &exitError{code: 2}), wantCode: 2, wantStderr: ""},
		{name: "macro run error uses its code and prints both lines", err: &macro.RunError{Code: 3, Message: "boom"}, wantCode: 3, wantStderr: "Error: boom\nmacro: boom\n"},
		{name: "other error exits one with Error prefix", err: errors.New("bad"), wantCode: 1, wantStderr: "Error: bad\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, stderr := exitStatus(tt.err)
			if code != tt.wantCode {
				t.Errorf("exitStatus(%v) code = %d, want %d", tt.err, code, tt.wantCode)
			}
			if stderr != tt.wantStderr {
				t.Errorf("exitStatus(%v) stderr = %q, want %q", tt.err, stderr, tt.wantStderr)
			}
		})
	}
}
