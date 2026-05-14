---
allowed-tools: Read, Write, Edit, Bash(go build:*), Bash(go vet:*)
description: Add a new plugin to the application
---

## Add a new plugin

Create a new plugin that implements the `Plugin` interface from `pkg/sdk/plugin.go`.

### Steps

1. **Create the plugin directory and file**

   Create `plugins/<name>/<name>.go` with the plugin implementation.

2. **Implement the Plugin interface**

   The plugin must satisfy this interface (defined in `pkg/sdk/plugin.go`):

   ```go
   type Plugin interface {
       ID() string
       Name() string
       Description() string
       Init(ctx *Context) tea.Cmd
       Update(msg tea.Msg) (Plugin, tea.Cmd)
       View(width, height int) string
       Configure(cfg map[string]interface{}) error
       Ready() bool
   }
   ```

   Optional interfaces: `Activatable`, `Countable`, `Hintable`, `Pinnable`, `Stackable`.

   Reference implementation: `plugins/state/state.go`

3. **Follow the standard plugin structure**

   ```go
   package <name>

   import (
       tea "github.com/charmbracelet/bubbletea"
       "github.com/lmarqs/terraform-ui/pkg/sdk"
   )

   type Plugin struct {
       svc     sdk.Service
       log     *slog.Logger
       pins    *sdk.PinService
       options *sdk.ResolvedOptions
       status  sdk.Status
   }

   func New(svc sdk.Service) sdk.Plugin {
       return &Plugin{svc: svc}
   }

   func (p *Plugin) ID() string          { return "<id>" }
   func (p *Plugin) Name() string        { return "<Display Name>" }
   func (p *Plugin) Description() string { return "<one-line description>" }
   func (p *Plugin) Ready() bool         { return p.status == sdk.StatusDone }

   func (p *Plugin) Configure(cfg map[string]interface{}) error {
       return nil
   }

   func (p *Plugin) Init(ctx *sdk.Context) tea.Cmd {
       p.svc = ctx.Service
       p.log = ctx.Logger
       p.pins = ctx.Pins
       p.options = ctx.Options
       return nil
   }

   func (p *Plugin) Update(msg tea.Msg) (sdk.Plugin, tea.Cmd) {
       return p, nil
   }

   func (p *Plugin) View(width, height int) string {
       return ""
   }
   ```

4. **Register the plugin factory in `cmd/tfui/main.go`**

   ```go
   import tfui<name> "github.com/lmarqs/terraform-ui/plugins/<name>"

   // In the plugin registration section:
   registry.RegisterFactory("<name>", tfui<name>.New, plugin.PluginMeta{
       Keybinding: "<key>", MenuVisible: true,
   })
   ```

5. **Add the import alias rule to `.golangci.yaml`**

   Under `linters.settings.importas.alias`, add:

   ```yaml
   - pkg: github.com/lmarqs/terraform-ui/plugins/<name>
     alias: tfui<name>
   ```

6. **Add documentation at `docs/plugins/<name>.md`**

   Use the standard frontmatter format:

   ```yaml
   ---
   layout: plugin
   title: <Display Name>
   id: <plugin-id>
   key: <keyboard shortcut>
   description: <one-line description>
   category: <operations|analysis|navigation>
   default_enabled: true
   ---
   ```

   Include sections: Overview, Usage (keybindings table), Configuration (tfui.hcl example), Related.

7. **Update the plugin index at `docs/plugins/index.md`**

   Add a row to the plugins table and the appropriate category section.

8. **Verify the build**

   Run `go build ./...` and `go vet ./...` to confirm everything compiles.

### Key patterns

- Plugin types are named `Plugin` by convention
- Constructor is `New(svc sdk.Service)` returning `sdk.Plugin`
- Use `sdk.Style*` from `pkg/sdk/styles.go` for formatting
- Use `strings.Builder` for render loops
- Async operations return `tea.Cmd` that produce custom message types
- Keybindings: lowercase for terraform ops, uppercase for analysis features
- Plugins import ONLY `pkg/sdk` — never `internal/`
