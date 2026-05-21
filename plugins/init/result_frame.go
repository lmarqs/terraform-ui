package init

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/frames"
	"github.com/lmarqs/terraform-ui/pkg/sdk/ui"
)

// resultFrame wraps a StreamFrame to display init execution output.
// On success it emits DeactivateMsg (auto-return home) unless the user is
// viewing the log (in which case Esc from the stream frame deactivates).
// On error it stays visible so the user can review the failure.
type resultFrame struct {
	status sdk.Status
	timer  *ui.Timer
	errMsg string
	stream *frames.StreamFrame
}

func newResultFrame(timer *ui.Timer, stream *frames.StreamFrame) *resultFrame {
	return &resultFrame{
		status: sdk.StatusLoading,
		timer:  timer,
		stream: stream,
	}
}

func (f *resultFrame) ID() string { return "result" }

func (f *resultFrame) Update(msg tea.Msg) (sdk.Frame, tea.Cmd) {
	switch msg := msg.(type) {
	case ui.TimerTickMsg:
		return f, f.timer.Tick()

	case frames.StreamLineMsg, frames.StreamDoneMsg:
		_, cmd := f.stream.Update(msg)
		return f, cmd

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
		if f.status == sdk.StatusError {
			switch msg.String() {
			case "esc", "enter":
				return nil, nil
			}
		}
		if f.status == sdk.StatusDone {
			_, cmd := f.stream.Update(msg)
			return f, cmd
		}
	}
	return f, nil
}

func (f *resultFrame) View(width, height int) string {
	switch f.status {
	case sdk.StatusLoading:
		v := f.stream.View(width, height)
		if v == "" {
			return sdk.StyleFaintItalic.Render("Running terraform init... " + f.timer.FormatElapsed())
		}
		return v
	case sdk.StatusError:
		return sdk.StyleError.Render("Init failed: " + f.errMsg)
	default:
		return sdk.StyleSuccess.Render("Terraform initialized successfully.")
	}
}

func (f *resultFrame) Hints() []sdk.KeyHint {
	switch f.status {
	case sdk.StatusLoading:
		return f.stream.Hints()
	case sdk.StatusError:
		return []sdk.KeyHint{
			{Key: "Enter", Description: "back"},
			sdk.HintBack,
		}
	default:
		return nil
	}
}
