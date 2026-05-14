package version

import (
	"context"
	"fmt"
	"runtime"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

// VersionResultMsg is sent when the version fetch completes.
type VersionResultMsg struct {
	Info *sdk.VersionInfo
	Err  error
}

// Plugin implements the version info viewer.
type Plugin struct {
	svc     sdk.Service
	status  sdk.Status
	info    *sdk.VersionInfo
	errMsg  string
	version string
}

// New creates a new version plugin.
func New(svc sdk.Service) sdk.Plugin {
	return &Plugin{svc: svc}
}

func (p *Plugin) ID() string          { return "version" }
func (p *Plugin) Name() string        { return "Version" }
func (p *Plugin) Description() string { return "Show version information" }
func (p *Plugin) Ready() bool         { return p.status == sdk.StatusDone }

// Configure applies plugin-specific options from config.
func (p *Plugin) Configure(cfg map[string]interface{}) error {
	if v, ok := cfg["tfui_version"].(string); ok {
		p.version = v
	}
	return nil
}

// Init initializes the plugin with shared context.
func (p *Plugin) Init(ctx *sdk.Context) tea.Cmd {
	p.svc = ctx.Service
	return nil
}

// Activate triggers version loading when the user enters the plugin.
func (p *Plugin) Activate() tea.Cmd {
	p.status = sdk.StatusLoading
	svc := p.svc
	return func() tea.Msg {
		info, err := svc.Version(context.Background())
		return VersionResultMsg{Info: info, Err: err}
	}
}

// Hints returns key hints for the version view.
func (p *Plugin) Hints() []sdk.KeyHint {
	return []sdk.KeyHint{sdk.HintBack}
}

// Update processes messages.
func (p *Plugin) Update(msg tea.Msg) (sdk.Plugin, tea.Cmd) {
	switch msg := msg.(type) {
	case VersionResultMsg:
		if msg.Err != nil {
			p.status = sdk.StatusError
			p.errMsg = msg.Err.Error()
		} else {
			p.status = sdk.StatusDone
			p.info = msg.Info
		}
		return p, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			return p, func() tea.Msg { return sdk.DeactivateMsg{} }
		}
	}
	return p, nil
}

// View renders the version information.
func (p *Plugin) View(width, height int) string {
	switch p.status {
	case sdk.StatusIdle, sdk.StatusLoading:
		return sdk.StyleFaintItalic.Render("Loading version info...")
	case sdk.StatusError:
		return p.renderWithTfuiVersion() + "\n\n" + sdk.StyleError.Render("terraform: "+p.errMsg)
	default:
		return p.renderFull()
	}
}

func (p *Plugin) renderWithTfuiVersion() string {
	platform := runtime.GOOS + "_" + runtime.GOARCH
	ver := p.version
	if ver == "" {
		ver = "unknown"
	}
	return fmt.Sprintf("tfui v%s\non %s", ver, platform)
}

func (p *Plugin) renderFull() string {
	var b strings.Builder
	b.WriteString(p.renderWithTfuiVersion())

	if p.info != nil && p.info.TerraformVersion != "" {
		platform := runtime.GOOS + "_" + runtime.GOARCH
		fmt.Fprintf(&b, "\n\nterraform v%s\non %s", p.info.TerraformVersion, platform)

		if len(p.info.Providers) > 0 {
			keys := make([]string, 0, len(p.info.Providers))
			for k := range p.info.Providers {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				fmt.Fprintf(&b, "\n+ provider %s v%s", k, p.info.Providers[k])
			}
		}
	}

	return b.String()
}
