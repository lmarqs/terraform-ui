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
	sdk.PluginBase
	status   sdk.Status
	info     *sdk.VersionInfo
	errMsg   string
	version  string
	cancelFn context.CancelFunc
}

// New creates a new version plugin.
func New(svc sdk.Service) sdk.Plugin {
	p := &Plugin{PluginBase: sdk.NewPluginBase("version", "Version", "Show version information")}
	p.Svc = svc
	return p
}

func (p *Plugin) Ready() bool { return p.status == sdk.StatusDone }

// Configure applies plugin-specific options from config.
func (p *Plugin) Configure(cfg map[string]interface{}) error {
	if v, ok := cfg["tfui_version"].(string); ok {
		p.version = v
	}
	return nil
}

// Init wires the plugin to its shared dependencies.
func (p *Plugin) Init(deps *sdk.PluginDeps) tea.Cmd {
	p.InitBase(deps)
	return nil
}

// Cancel aborts any in-flight terraform operation.
func (p *Plugin) Cancel() {
	if p.cancelFn != nil {
		p.cancelFn()
		p.cancelFn = nil
	}
}

// Activate triggers version loading when the user enters the plugin.
func (p *Plugin) Activate() tea.Cmd {
	p.Cancel()
	ctx, cancel := context.WithCancel(context.Background())
	p.cancelFn = cancel
	p.status = sdk.StatusLoading
	svc := p.Svc
	return func() tea.Msg {
		info, err := svc.Version(ctx)
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

// Output produces stdout content for standalone/CI mode.
func (p *Plugin) Output(jsonOutput bool) ([]byte, error) {
	platform := runtime.GOOS + "_" + runtime.GOARCH
	ver := p.version
	if ver == "" {
		ver = "unknown"
	}

	if jsonOutput {
		out := struct {
			TfuiVersion       string            `json:"tfui_version"`
			Platform          string            `json:"platform"`
			TerraformVersion  string            `json:"terraform_version,omitempty"`
			TerraformPlatform string            `json:"terraform_platform,omitempty"`
			Providers         map[string]string `json:"provider_selections,omitempty"`
		}{
			TfuiVersion: ver,
			Platform:    platform,
		}
		if p.info != nil && p.info.TerraformVersion != "" {
			out.TerraformVersion = p.info.TerraformVersion
			out.TerraformPlatform = platform
			out.Providers = p.info.Providers
		}
		return sdk.MarshalJSON(out), nil
	}

	var b strings.Builder
	fmt.Fprintf(&b, "tfui v%s\non %s\n", ver, platform)
	if p.info != nil && p.info.TerraformVersion != "" {
		fmt.Fprintf(&b, "\nterraform v%s\non %s\n", p.info.TerraformVersion, platform)
		if len(p.info.Providers) > 0 {
			keys := make([]string, 0, len(p.info.Providers))
			for k := range p.info.Providers {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				fmt.Fprintf(&b, "+ provider %s v%s\n", k, p.info.Providers[k])
			}
		}
	}
	return []byte(b.String()), nil
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
