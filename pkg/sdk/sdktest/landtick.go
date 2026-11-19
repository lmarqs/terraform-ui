package sdktest

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk/ui"
)

// LandTick runs cmd as the runtime would, descending into batches, and fails
// the test unless a timer tick comes back. A ui.Timer ignores a TimerTickMsg
// while its own tick Cmd is outstanding, so a test that sends the message must
// first let that Cmd finish. It waits the tick's real second and runs every
// other Cmd in the batch.
func LandTick(t testing.TB, cmd tea.Cmd) {
	t.Helper()
	if !landTick(cmd) {
		t.Fatal("cmd produced no ui.TimerTickMsg")
	}
}

func landTick(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	switch msg := cmd().(type) {
	case ui.TimerTickMsg:
		return true
	case tea.BatchMsg:
		landed := false
		for _, child := range msg {
			landed = landTick(child) || landed
		}
		return landed
	}
	return false
}
