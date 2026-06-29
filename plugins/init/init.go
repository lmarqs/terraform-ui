package init

import (
	"context"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	sdkframes "github.com/lmarqs/terraform-ui/pkg/sdk/frames"
	"github.com/lmarqs/terraform-ui/pkg/sdk/ui"
)

// initSubmitMsg triggers the init execution after form submission.
type initSubmitMsg struct{}

// InitResultMsg is sent when the init operation completes.
type InitResultMsg struct {
	Err      error
	Duration time.Duration
}

// Plugin implements the terraform init feature.
type Plugin struct {
	sdk.PluginBase
	timer ui.Timer
	stack *sdk.Stack
	lw    *sdkframes.LineWriter
	ch    <-chan string

	// Form state (preserved across runs for re-fill). Each field maps to a
	// terraform init flag that terraform-exec accepts.
	upgrade        bool
	reconfigure    bool
	backend        bool
	get            bool
	lock           bool
	forceCopy      bool
	lockTimeout    string
	fromModule     string
	pluginDir      []string
	backendConfigs []string
	cancelFn       context.CancelFunc
}

// New creates a new init plugin.
func New(svc sdk.Service) sdk.Plugin {
	p := &Plugin{
		PluginBase: sdk.NewPluginBase("init", "Init", "Initialize terraform working directory"),
		stack:      sdk.NewStack(),
		backend:    true,
		get:        true,
		lock:       true,
	}
	p.Svc = svc
	return p
}

func (p *Plugin) Ready() bool       { return true }
func (p *Plugin) Stack() *sdk.Stack { return p.stack }

func (p *Plugin) Configure(_ map[string]interface{}) error { return nil }

func (p *Plugin) Busy() bool {
	if top := p.stack.Peek(); top != nil {
		if rf, ok := top.(*resultFrame); ok {
			return rf.status == sdk.StatusLoading
		}
	}
	return false
}

func (p *Plugin) Init(deps *sdk.PluginDeps) tea.Cmd {
	p.InitBase(deps)
	return nil
}

// HandleContextChanged implements sdk.ContextChangedHandler.
func (p *Plugin) HandleContextChanged(ev sdk.ContextChangedEvent) tea.Cmd {
	if !p.HandleContextChangedDefault(ev) {
		return nil
	}
	p.stack.Reset()
	return nil
}

// Activate stores the typed input and returns the initial command.
func (p *Plugin) Activate(input Input) tea.Cmd {
	p.resetState()
	p.upgrade = input.Upgrade
	p.reconfigure = input.Reconfigure
	if input.Backend != nil {
		p.backend = *input.Backend
	}
	p.backendConfigs = input.BackendConfig
	p.stack.Reset()
	p.stack.Push(p.buildForm())
	if input.Upgrade || input.Reconfigure || input.Backend != nil || len(input.BackendConfig) > 0 {
		return func() tea.Msg { return initSubmitMsg{} }
	}
	return nil
}

func (p *Plugin) resetState() {
	p.upgrade = false
	p.reconfigure = false
	p.backend = true
	p.get = true
	p.lock = true
	p.forceCopy = false
	p.lockTimeout = ""
	p.fromModule = ""
	p.pluginDir = nil
	p.backendConfigs = nil
}

func (p *Plugin) Update(msg tea.Msg) (sdk.Plugin, tea.Cmd) {
	switch msg.(type) {
	case initSubmitMsg:
		p.stack.Reset()
		p.stack.Push(p.buildForm())
		lw, ch := sdkframes.NewLineWriter()
		p.lw = lw
		p.ch = ch
		sf := sdkframes.NewStreamFrame("terraform init", ch, p.Cancel)
		rf := newResultFrame(&p.timer, sf)
		p.stack.Push(rf)
		return p, p.submit(lw)

	case sdkframes.StreamLineMsg:
		if top := p.stack.Peek(); top != nil {
			_, cmd := top.Update(msg)
			return p, cmd
		}
		if p.ch != nil {
			return p, sdkframes.WaitForLine(p.ch)
		}

	case sdkframes.StreamDoneMsg:
		if top := p.stack.Peek(); top != nil {
			_, cmd := top.Update(msg)
			return p, cmd
		}

	case InitResultMsg, ui.TimerTickMsg:
		if top := p.stack.Peek(); top != nil {
			_, cmd := top.Update(msg)
			return p, cmd
		}
	}
	return p, nil
}

func (p *Plugin) View(width, height int) string {
	return p.stack.View(width, height)
}

func (p *Plugin) buildForm() *sdkframes.FormFrame {
	return sdkframes.NewFormFrame(sdkframes.FormOpts{
		Fields: []sdkframes.FormField{
			toggleField("upgrade", &p.upgrade),
			toggleField("reconfigure", &p.reconfigure),
			toggleField("backend", &p.backend),
			toggleField("get", &p.get),
			toggleField("lock", &p.lock),
			toggleField("force-copy", &p.forceCopy),
			textField("lock-timeout", &p.lockTimeout),
			textField("from-module", &p.fromModule),
			listField("plugin-dir", &p.pluginDir),
			listField("backend-config", &p.backendConfigs),
			{
				Label:      "",
				Value:      func() string { return "Run terraform init" },
				Selectable: true,
				IsAction:   true,
				OnSelect:   p.submitFromForm,
			},
		},
	})
}

// toggleField builds a checkbox form field bound to a boolean. Space (or Enter)
// flips it.
func toggleField(label string, v *bool) sdkframes.FormField {
	return sdkframes.FormField{
		Label:      label,
		Value:      func() string { return checkbox(*v) },
		Selectable: true,
		Toggle:     true,
		OnSelect:   func() tea.Cmd { *v = !*v; return nil },
	}
}

// textField builds a form field that opens a text prompt to edit a string.
func textField(label string, v *string) sdkframes.FormField {
	return sdkframes.FormField{
		Label:      label,
		Value:      func() string { return display(*v) },
		Selectable: true,
		OnSelect:   func() tea.Cmd { return editText(label, *v, func(s string) { *v = s }) },
	}
}

// listField builds a form field that edits a space-separated string slice.
func listField(label string, v *[]string) sdkframes.FormField {
	return sdkframes.FormField{
		Label:      label,
		Value:      func() string { return display(strings.Join(*v, " ")) },
		Selectable: true,
		OnSelect: func() tea.Cmd {
			return editText(label, strings.Join(*v, " "), func(s string) {
				if fields := strings.Fields(s); len(fields) > 0 {
					*v = fields
				} else {
					*v = nil
				}
			})
		},
	}
}

// editText returns a command that opens a text input prompt and stores the
// result via set.
func editText(label, current string, set func(string)) tea.Cmd {
	return func() tea.Msg {
		return sdk.RequestInputMsg{
			Request: sdk.InputText(label+":", current, func(value string) tea.Cmd {
				set(value)
				return nil
			}),
		}
	}
}

func (p *Plugin) submitFromForm() tea.Cmd {
	return func() tea.Msg { return initSubmitMsg{} }
}

// Cancel aborts any in-flight terraform operation.
func (p *Plugin) Cancel() {
	if p.cancelFn != nil {
		p.cancelFn()
		p.cancelFn = nil
	}
}

func (p *Plugin) submit(lw *sdkframes.LineWriter) tea.Cmd {
	p.Cancel()
	ctx, cancel := context.WithCancel(context.Background())
	p.cancelFn = cancel
	svc := p.Svc
	get, lock := p.get, p.lock
	opts := sdk.InitOptions{
		Upgrade:       p.upgrade,
		Reconfigure:   p.reconfigure,
		BackendConfig: p.backendConfigs,
		ForceCopy:     p.forceCopy,
		Get:           &get,
		Lock:          &lock,
		LockTimeout:   p.lockTimeout,
		FromModule:    p.fromModule,
		PluginDir:     p.pluginDir,
		Writer:        lw,
	}
	if !p.backend {
		opts.Backend = sdk.BackendDisabled
	}

	ch := p.ch
	start := time.Now()
	return tea.Batch(
		func() tea.Msg {
			err := svc.Init(ctx, opts)
			lw.Close()
			return InitResultMsg{Err: err, Duration: time.Since(start)}
		},
		p.timer.Start(),
		sdkframes.WaitForLine(ch),
	)
}

// display renders a form value, falling back to "(none)" when empty.
func display(v string) string {
	if v == "" {
		return "(none)"
	}
	return v
}

func checkbox(v bool) string {
	if v {
		return "[x]"
	}
	return "[ ]"
}

// Stdout produces stdout content for standalone/CI mode.
func (p *Plugin) Stdout() ([]byte, error) {
	return []byte("Initialized successfully.\n"), nil
}
