package config

import "path/filepath"

func Resolve(root *RootConfig, child *ChildConfig, workspace string) *ResolvedConfig {
	r := &ResolvedConfig{}

	if root == nil {
		return r
	}

	r.parallelism = root.Defaults.Parallelism
	r.lock = root.Defaults.Lock
	r.varFiles = append(r.varFiles, root.Defaults.VarFiles...)
	r.vars = copyVars(root.Defaults.Vars)
	r.plugins = copyPlugins(root.Defaults.Plugins)

	if child == nil {
		return r
	}

	r.varFiles = append(r.varFiles, child.VarFiles...)
	mergeVars(r, child.Vars)
	mergePlugins(r, child.Plugins)

	if workspace == "" {
		return r
	}

	ws := matchWorkspace(child.Workspaces, workspace)
	if ws == nil {
		return r
	}

	r.varFiles = append(r.varFiles, ws.VarFiles...)
	mergeVars(r, ws.Vars)
	mergePlugins(r, ws.Plugins)
	if ws.LockTimeout != "" {
		r.lockTimeout = ws.LockTimeout
	}

	return r
}

func matchWorkspace(workspaces []WorkspaceConfig, name string) *WorkspaceConfig {
	var globMatch *WorkspaceConfig
	for i := range workspaces {
		ws := &workspaces[i]
		if ws.Name == name {
			return ws
		}
		if globMatch == nil {
			if matched, _ := filepath.Match(ws.Name, name); matched {
				globMatch = ws
			}
		}
	}
	return globMatch
}

func copyVars(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}
	out := make(map[string]string, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

func mergeVars(r *ResolvedConfig, vars map[string]string) {
	if len(vars) == 0 {
		return
	}
	if r.vars == nil {
		r.vars = make(map[string]string)
	}
	for k, v := range vars {
		r.vars[k] = v
	}
}

func copyPlugins(src map[string]PluginSettings) map[string]PluginSettings {
	if len(src) == 0 {
		return nil
	}
	out := make(map[string]PluginSettings, len(src))
	for k, v := range src {
		out[k] = PluginSettings{
			Enabled: v.Enabled,
			Options: copyOptions(v.Options),
		}
	}
	return out
}

func mergePlugins(r *ResolvedConfig, plugins map[string]PluginSettings) {
	if len(plugins) == 0 {
		return
	}
	if r.plugins == nil {
		r.plugins = make(map[string]PluginSettings)
	}
	for name, ps := range plugins {
		existing, ok := r.plugins[name]
		if !ok {
			r.plugins[name] = PluginSettings{
				Enabled: ps.Enabled,
				Options: copyOptions(ps.Options),
			}
			continue
		}
		existing.Enabled = ps.Enabled
		if existing.Options == nil {
			existing.Options = make(map[string]interface{})
		}
		for k, v := range ps.Options {
			existing.Options[k] = v
		}
		r.plugins[name] = existing
	}
}

func copyOptions(src map[string]interface{}) map[string]interface{} {
	if len(src) == 0 {
		return nil
	}
	out := make(map[string]interface{}, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}
