package sdk

import tea "github.com/charmbracelet/bubbletea"

// InputRequestMode defines the type of user input being requested.
type InputRequestMode int

const (
	InputRequestText   InputRequestMode = iota // free text input
	InputRequestBool                           // y/n confirmation
	InputRequestSelect                         // pick from options
	InputRequestFilter                         // fzf-style live filter
)

// InputRequest is sent by a plugin to request user input via the app's input system.
type InputRequest struct {
	PluginID string
	Mode     InputRequestMode
	Prompt   string   // "Apply 3 resources? (y/n)", "Target address:", etc.
	Options  []string // for select mode
	Default  string   // pre-filled value
	Callback func(answer string) tea.Cmd
}

// InputResponseMsg is sent back to the plugin with the user's answer.
type InputResponseMsg struct {
	PluginID string
	Answer   string
	Canceled bool
}

// InputConfirm creates a bool confirmation InputRequest.
// Confirms on y/Y/enter; cancels on n/N/esc (Enter/Esc parity).
func InputConfirm(prompt string, onYes func() tea.Cmd) InputRequest {
	return InputRequest{
		Mode:   InputRequestBool,
		Prompt: prompt + " (y/n)",
		Callback: func(answer string) tea.Cmd {
			if answer == "y" || answer == "yes" {
				return onYes()
			}
			return nil
		},
	}
}

// InputText creates a free text InputRequest.
func InputText(prompt, defaultValue string, onSubmit func(string) tea.Cmd) InputRequest {
	return InputRequest{
		Mode:    InputRequestText,
		Prompt:  prompt,
		Default: defaultValue,
		Callback: func(answer string) tea.Cmd {
			return onSubmit(answer)
		},
	}
}

// InputSelect creates a selection InputRequest.
func InputSelect(prompt string, options []string, onSelect func(string) tea.Cmd) InputRequest {
	return InputRequest{
		Mode:    InputRequestSelect,
		Prompt:  prompt,
		Options: options,
		Callback: func(answer string) tea.Cmd {
			return onSelect(answer)
		},
	}
}

// RequestInputMsg wraps an InputRequest as a tea.Msg for dispatch.
type RequestInputMsg struct {
	Request InputRequest
}
