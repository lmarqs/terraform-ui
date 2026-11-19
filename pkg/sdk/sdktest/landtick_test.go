package sdktest

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk/ui"
)

type recordingTB struct {
	testing.TB
	failed bool
}

func (r *recordingTB) Helper()        {}
func (r *recordingTB) Fatal(_ ...any) { r.failed = true }

func TestLandTick(t *testing.T) {
	tick := func() tea.Msg { return ui.TimerTickMsg{} }
	other := func() tea.Msg { return struct{}{} }

	tests := []struct {
		name       string
		cmd        tea.Cmd
		wantFailed bool
	}{
		{"tick cmd", tick, false},
		{"tick inside a batch with other work", func() tea.Msg { return tea.BatchMsg{other, nil, tick} }, false},
		{"tick inside a nested batch", func() tea.Msg { return tea.BatchMsg{func() tea.Msg { return tea.BatchMsg{tick} }} }, false},
		{"nil cmd", nil, true},
		{"no tick in the batch", func() tea.Msg { return tea.BatchMsg{other} }, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := &recordingTB{TB: t}
			LandTick(rec, tt.cmd)
			if rec.failed != tt.wantFailed {
				t.Errorf("failed = %v, want %v", rec.failed, tt.wantFailed)
			}
		})
	}
}
