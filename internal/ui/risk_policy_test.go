package ui

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	sdkui "github.com/lmarqs/terraform-ui/pkg/sdk/ui"
)

func drainPolicyAppCommand(app *App, cmd tea.Cmd) {
	if cmd == nil {
		return
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, sub := range batch {
			drainPolicyAppCommand(app, sub)
		}
		return
	}
	if _, ok := msg.(sdkui.TimerTickMsg); ok {
		return // a tick advances the clock, not the assessment, and would chain forever while one is pending
	}
	model, next := app.Update(msg)
	*app = model.(App)
	drainPolicyAppCommand(app, next)
}

func writePolicyFile(t *testing.T, root, name, source string) {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
}
