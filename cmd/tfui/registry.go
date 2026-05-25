package main

import (
	"github.com/lmarqs/terraform-ui/internal/config"
	"github.com/lmarqs/terraform-ui/internal/plugin"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	tfuiapply "github.com/lmarqs/terraform-ui/plugins/apply"
	tfuiblastradius "github.com/lmarqs/terraform-ui/plugins/blastradius"
	tfuichdir "github.com/lmarqs/terraform-ui/plugins/chdir"
	tfuiconsole "github.com/lmarqs/terraform-ui/plugins/console"
	tfuicontext "github.com/lmarqs/terraform-ui/plugins/context"
	tfuiforceunlock "github.com/lmarqs/terraform-ui/plugins/forceunlock"
	tfuiimport "github.com/lmarqs/terraform-ui/plugins/import"
	tfuiinit "github.com/lmarqs/terraform-ui/plugins/init"
	tfuioutput "github.com/lmarqs/terraform-ui/plugins/output"
	tfuiphantom "github.com/lmarqs/terraform-ui/plugins/phantom"
	tfuiplan "github.com/lmarqs/terraform-ui/plugins/plan"
	tfuirisk "github.com/lmarqs/terraform-ui/plugins/risk"
	tfuistate "github.com/lmarqs/terraform-ui/plugins/state"
	tfuitaint "github.com/lmarqs/terraform-ui/plugins/taint"
	tfuiuntaint "github.com/lmarqs/terraform-ui/plugins/untaint"
	tfuivalidate "github.com/lmarqs/terraform-ui/plugins/validate"
	tfuiversion "github.com/lmarqs/terraform-ui/plugins/version"
	tfuiworkspace "github.com/lmarqs/terraform-ui/plugins/workspace"
)

func buildRegistry(svc sdk.Service, cfg config.Config, rootCfg *config.RootConfig) *plugin.Registry {
	registry := plugin.NewRegistry()
	registry.RegisterFactory("context", tfuicontext.New, plugin.PluginMeta{Keybinding: "C", MenuVisible: true})
	registry.RegisterFactory("chdir", tfuichdir.New, plugin.PluginMeta{Keybinding: "", MenuVisible: false, Nav: plugin.NavPush})
	registry.RegisterFactory("state", tfuistate.New, plugin.PluginMeta{Keybinding: "s", MenuVisible: true})
	registry.RegisterFactory("plan", tfuiplan.New, plugin.PluginMeta{Keybinding: "p", MenuVisible: true})
	registry.RegisterFactory("apply", tfuiapply.New, plugin.PluginMeta{Keybinding: "a", MenuVisible: false})
	registry.RegisterFactory("workspace", tfuiworkspace.New, plugin.PluginMeta{Keybinding: "w", MenuVisible: true, Nav: plugin.NavPush})
	registry.RegisterFactory("console", tfuiconsole.New, plugin.PluginMeta{Keybinding: "~", MenuVisible: true})
	registry.RegisterFactory("output", tfuioutput.New, plugin.PluginMeta{Keybinding: "o", MenuVisible: true})
	registry.RegisterFactory("validate", tfuivalidate.New, plugin.PluginMeta{Keybinding: "v", MenuVisible: true})
	registry.RegisterFactory("init", tfuiinit.New, plugin.PluginMeta{Keybinding: "i", MenuVisible: true})
	registry.RegisterFactory("risk", tfuirisk.New, plugin.PluginMeta{Keybinding: "R", MenuVisible: true})
	registry.RegisterFactory("phantom", tfuiphantom.New, plugin.PluginMeta{Keybinding: "P", MenuVisible: true})
	registry.RegisterFactory("blastradius", tfuiblastradius.New, plugin.PluginMeta{Keybinding: "B", MenuVisible: true})
	registry.RegisterFactory("taint", tfuitaint.New, plugin.PluginMeta{MenuVisible: false, Nav: plugin.NavPush})
	registry.RegisterFactory("untaint", tfuiuntaint.New, plugin.PluginMeta{MenuVisible: false, Nav: plugin.NavPush})
	registry.RegisterFactory("import", tfuiimport.New, plugin.PluginMeta{MenuVisible: false, Nav: plugin.NavPush})
	registry.RegisterFactory("forceunlock", tfuiforceunlock.New, plugin.PluginMeta{MenuVisible: false, Nav: plugin.NavPush})
	registry.RegisterFactory("version", tfuiversion.New, plugin.PluginMeta{MenuVisible: false, Nav: plugin.NavPush})

	registry.Build(svc, cfg.Plugins)

	var memberPaths []string
	if rootCfg != nil && len(rootCfg.Members) > 0 {
		memberPaths = make([]string, len(rootCfg.Members))
		for i, m := range rootCfg.Members {
			memberPaths[i] = m.Path
		}
	}

	if ctxPlugin, ok := registry.ByID("context"); ok {
		if cp, ok := ctxPlugin.(*tfuicontext.Plugin); ok {
			cp.SetProjectDir(cfg.Dir)
			if len(memberPaths) > 0 {
				cp.SetMembers(memberPaths)
			}
		}
	}
	if chdirPlugin, ok := registry.ByID("chdir"); ok {
		if cp, ok := chdirPlugin.(*tfuichdir.Plugin); ok {
			if len(memberPaths) > 0 {
				cp.SetMembers(memberPaths)
			}
		}
	}
	if versionPlugin, ok := registry.ByID("version"); ok {
		_ = versionPlugin.Configure(map[string]interface{}{"tfui_version": version})
	}

	return registry
}
