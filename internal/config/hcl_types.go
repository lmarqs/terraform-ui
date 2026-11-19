package config

type RootTerraformConfig struct {
	Bin string
}

type MemberConfig struct {
	Path string
}

type PluginSettings struct {
	Enabled bool
	Options map[string]interface{}
}

// PolicySelector is one declared risk_policy. Value is the raw declared string,
// relative to Origin, the directory of the declaring tfui.hcl; "" selects the
// built-in policy. Invalid marks a declaration that cannot select a policy, which
// the loaders record as data so Plan stays available.
type PolicySelector struct {
	Value   string
	Origin  string
	Invalid bool
}

type DefaultsConfig struct {
	Parallelism int
	Lock        *bool
	VarFiles    []string
	Vars        map[string]string
	Plugins     map[string]PluginSettings
	RiskPolicy  *PolicySelector
}

func (d *DefaultsConfig) PluginConfig(id string) PluginSettings {
	if d.Plugins == nil {
		return PluginSettings{}
	}
	ps, ok := d.Plugins[id]
	if !ok {
		return PluginSettings{}
	}
	return ps
}

type RootConfig struct {
	Terraform RootTerraformConfig
	Members   []MemberConfig
	Defaults  DefaultsConfig
}

type WorkspaceConfig struct {
	Name        string
	VarFiles    []string
	Vars        map[string]string
	Plugins     map[string]PluginSettings
	LockTimeout string
	RiskPolicy  *PolicySelector
}

type ChildConfig struct {
	VarFiles   []string
	Vars       map[string]string
	Plugins    map[string]PluginSettings
	Workspaces []WorkspaceConfig
	RiskPolicy *PolicySelector
}

func (c *ChildConfig) PluginConfig(id string) PluginSettings {
	if c.Plugins == nil {
		return PluginSettings{}
	}
	ps, ok := c.Plugins[id]
	if !ok {
		return PluginSettings{}
	}
	return ps
}

type ResolvedConfig struct {
	varFiles    []string
	vars        map[string]string
	parallelism int
	lock        *bool
	lockTimeout string
	plugins     map[string]PluginSettings
	riskPolicy  *PolicySelector
}

func (r *ResolvedConfig) VarFiles() []string      { return r.varFiles }
func (r *ResolvedConfig) Vars() map[string]string { return r.vars }
func (r *ResolvedConfig) Parallelism() int        { return r.parallelism }
func (r *ResolvedConfig) Lock() *bool             { return r.lock }
func (r *ResolvedConfig) LockTimeout() string     { return r.lockTimeout }

// RiskPolicy returns the selector of the last layer that declares one, or nil.
func (r *ResolvedConfig) RiskPolicy() *PolicySelector { return r.riskPolicy }

// PluginConfigs converts the resolved per-plugin settings into the shape the
// plugin registry consumes, so a `plugin` block in tfui.hcl reaches the plugin
// it names. Returns nil when no plugin block was declared anywhere.
func (r *ResolvedConfig) PluginConfigs() map[string]PluginConfig {
	if len(r.plugins) == 0 {
		return nil
	}
	out := make(map[string]PluginConfig, len(r.plugins))
	for id, ps := range r.plugins {
		enabled := ps.Enabled
		out[id] = PluginConfig{Enabled: &enabled, Options: ps.Options}
	}
	return out
}

func (r *ResolvedConfig) PluginConfig(id string) PluginSettings {
	if r.plugins == nil {
		return PluginSettings{}
	}
	ps, ok := r.plugins[id]
	if !ok {
		return PluginSettings{}
	}
	return ps
}
