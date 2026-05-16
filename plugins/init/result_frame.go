package init

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/ui"
)

// resultFrame displays init execution progress and results.
// On success: emits DeactivateMsg (auto-return home).
// On error: Enter pops back to form (pre-filled for retry).
type resultFrame struct {
	status sdk.Status
	timer  *ui.Timer
	errMsg string
}

func newResultFrame(timer *ui.Timer) *resultFrame {
	return &resultFrame{
		status: sdk.StatusLoading,
		timer:  timer,
	}
}

func (f *resultFrame) ID() string { return "result" }

func (f *resultFrame) Update(msg tea.Msg) (sdk.Frame, tea.Cmd) {
	switch msg := msg.(type) {
	case ui.TimerTickMsg:
		return f, f.timer.Tick()

	case InitResultMsg:
		f.timer.Stop()
		if msg.Err != nil {
			f.status = sdk.StatusError
			f.errMsg = msg.Err.Error()
			return f, nil
		}
		f.status = sdk.StatusDone
		return f, tea.Batch(
			func() tea.Msg { return sdk.PlanInvalidatedEvent{} },
			func() tea.Msg { return sdk.DeactivateMsg{} },
		)

	case tea.KeyMsg:
		if f.status == sdk.StatusError && msg.String() == "enter" {
			return nil, nil
		}
	}
	return f, nil
}

func (f *resultFrame) View(width, height int) string {
	switch f.status {
	case sdk.StatusLoading:
		return sdk.StyleFaintItalic.Render("Running terraform init... " + f.timer.FormatElapsed())
	case sdk.StatusError:
		return sdk.StyleError.Render("Init failed: " + f.errMsg)
	default:
		return sdk.StyleSuccess.Render("Terraform initialized successfully.")
	}
}

func (f *resultFrame) Hints() []sdk.KeyHint {
	switch f.status {
	case sdk.StatusLoading:
		return (sdk.HintSetBack).Hints()
	case sdk.StatusError:
		return []sdk.KeyHint{
			{Key: "Enter", Description: "back"},
			sdk.HintBack,
		}
	default:
		return nil
	}
}
