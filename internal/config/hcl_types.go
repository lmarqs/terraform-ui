package config

type RootTerraformConfig struct {
	Bin string
}

type MemberConfig struct {
	Path string
}

type CacheConfig struct {
	StalenessThreshold string
}

type AIConfig struct {
	Enabled  bool
	Provider string
	Model    string
	Region   string
}

type PluginSettings struct {
	Enabled bool
	Options map[string]interface{}
}

type DefaultsConfig struct {
	Parallelism int
	Lock        *bool
	VarFiles    []string
	Vars        map[string]string
	Plugins     map[string]PluginSettings
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
	Cache     CacheConfig
	AI        AIConfig
	Defaults  DefaultsConfig
}

type WorkspaceConfig struct {
	Name        string
	VarFiles    []string
	Vars        map[string]string
	Plugins     map[string]PluginSettings
	LockTimeout string
}

type ChildConfig struct {
	VarFiles   []string
	Vars       map[string]string
	Plugins    map[string]PluginSettings
	Workspaces []WorkspaceConfig
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
}

func (r *ResolvedConfig) VarFiles() []string      { return r.varFiles }
func (r *ResolvedConfig) Vars() map[string]string { return r.vars }
func (r *ResolvedConfig) Parallelism() int        { return r.parallelism }
func (r *ResolvedConfig) Lock() *bool             { return r.lock }
func (r *ResolvedConfig) LockTimeout() string     { return r.lockTimeout }

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
