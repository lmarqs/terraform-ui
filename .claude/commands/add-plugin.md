---
allowed-tools: Read, Write, Edit, Bash(go build:*), Bash(go vet:*)
description: Add a new plugin to the application
---

## Add a new plugin

Create a new plugin that implements the `Plugin` interface from `internal/plugin/plugin.go`.

### Steps

1. **Create the plugin directory and file**

   Create `plugins/<name>/<name>.go` with the plugin implementation.

2. **Implement the Plugin interface**

   The plugin must satisfy this interface (defined in `internal/plugin/plugin.go`):

   ```go
   type Plugin interface {
       ID() string           // unique identifier, used as key in tfui.hcl plugins map
       Name() string         // human-readable display name
       Description() string  // one-line description for help/status bar
       KeyBinding() string   // single key to activate from home (e.g., "p", "R", "b")

       Init(ctx *Context) tea.Cmd           // initialize with shared context
       Update(msg tea.Msg) (Plugin, tea.Cmd) // handle messages (bubbletea pattern)
       View(width, height int) string       // render the plugin view

       Configure(cfg map[string]interface{}) error  // apply plugin-specific config from tfui.hcl
       Ready() bool                                  // whether data is loaded and view is ready
   }
   ```

   Reference implementation: `plugins/risk/risk.go`

3. **Follow the standard plugin structure**

   ```go
   package <name>

   import (
       tea "github.com/charmbracelet/bubbletea"
       "github.com/lmarqs/terraform-ui/internal/plugin"
       "github.com/lmarqs/terraform-ui/internal/terraform"
       "github.com/lmarqs/terraform-ui/internal/ui/styles"
   )

   type Status int

   const (
       StatusIdle Status = iota
       StatusLoading
       StatusDone
       StatusError
   )

   type Plugin struct {
       svc    terraform.Service
       status Status
       // plugin-specific fields...
   }

   func New(svc terraform.Service) plugin.Plugin {
       return &Plugin{svc: svc}
   }

   func (p *Plugin) ID() string          { return "<id>" }
   func (p *Plugin) Name() string        { return "<Display Name>" }
   func (p *Plugin) Description() string { return "<one-line description>" }
   func (p *Plugin) KeyBinding() string  { return "<key>" }
   func (p *Plugin) Ready() bool         { return p.status == StatusDone }

   func (p *Plugin) Configure(cfg map[string]interface{}) error {
       return nil
   }

   func (p *Plugin) Init(ctx *plugin.Context) tea.Cmd {
       p.svc = ctx.Service
       return nil
   }

   func (p *Plugin) Update(msg tea.Msg) (plugin.Plugin, tea.Cmd) {
       // handle tea.KeyMsg and custom messages
       return p, nil
   }

   func (p *Plugin) View(width, height int) string {
       // use styles.StyleTitle, styles.StylePadded, etc.
       return ""
   }
   ```

4. **Register the plugin factory in `cmd/tfui/main.go`**

   Add the import and register the factory with the plugin registry:

   ```go
   import "<name>plugin" "github.com/lmarqs/terraform-ui/plugins/<name>"

   // In the plugin registration section:
   registry.RegisterFactory("<name>", func(svc terraform.Service) plugin.Plugin {
       return <name>plugin.New()
   })
   ```

5. **Add documentation at `docs/plugins/<name>.md`**

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

   Include sections: Overview, Usage (keybindings table), Configuration (tfui.hcl example), Screenshots/Output, Related.

6. **Update the plugin index at `docs/plugins/index.md`**

   Add a row to the plugins table and the appropriate category section.

7. **Verify the build**

   Run `go build ./...` and `go vet ./...` to confirm everything compiles.

### Key patterns

- Plugin types are named `Plugin` by convention
- Constructor is `New(svc terraform.Service)` returning `plugin.Plugin`
- Use `styles.*` from `internal/ui/styles` for all formatting (never inline lipgloss)
- Use `strings.Builder` for render loops
- Async operations return `tea.Cmd` functions that produce custom message types
- Key bindings: lowercase for primary operations (p, a, s, w, m, b), uppercase for analysis (R, P)
- Categories: `operations` (modifies infra), `analysis` (read-only insight), `navigation` (state/workspace browsing)
