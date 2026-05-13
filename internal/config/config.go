package config

import (
	"os"
	"os/exec"
	"path/filepath"
)

// Config holds all configuration for the tfui application, loaded from
// tfui.yaml and/or CLI flags. It controls working directory, binary path,
// UI mode, plugin settings, and monorepo project discovery.
type Config struct {
	// Dir is the project root directory (where tfui.yaml lives).
	Dir string `yaml:"-"`

	// BaseDir is an optional override for the terraform working directory,
	// relative to Dir. Like TypeScript's compilerOptions.rootDir.
	// If empty, Dir is used as the terraform working directory.
	BaseDir string `yaml:"basedir"`

	// Workspace is the terraform workspace name.
	Workspace string `yaml:"-"`

	// ActiveScope is the pre-selected scope (relative to Dir).
	// Set via --scope flag for non-interactive scope selection.
	ActiveScope string `yaml:"-"`

	// ReadOnly indicates data was loaded from external source (--plan/--state).
	// Disables scope picker and mutating UI elements.
	ReadOnly bool `yaml:"-"`

	// Terraform holds terraform-specific configuration (binary path, etc.).
	Terraform TerraformConfig `yaml:"terraform"`

	// Logger holds logging configuration.
	Logger LoggerConfig `yaml:"logger"`

	// Mode is the UI mode for non-interactive commands: silent, spinner, progress, agent.
	Mode string `yaml:"-"`

	// Targets is a list of resource targets for plan/apply.
	Targets []string `yaml:"-"`

	// Scope defines monorepo project discovery (similar to pnpm-workspace.yaml).
	Scope ScopeConfig `yaml:"scope"`

	// Plugins is a map of plugin ID → plugin config.
	// Plugins not listed are enabled with default settings.
	Plugins map[string]PluginConfig `yaml:"plugins"`

	// Overrides holds key=value pairs from --config CLI flag.
	// Applied after yaml loading, overriding any matching values.
	Overrides map[string]string `yaml:"-"`

	// ExtraArgs holds raw arguments passed after -- separator.
	// These are passed through to terraform unmodified.
	ExtraArgs []string `yaml:"-"`
}

// LoggerConfig holds logging configuration.
type LoggerConfig struct {
	// Dir is the directory for log files. Defaults to ~/.tfui/logs.
	Dir string `yaml:"dir"`
}

// LogDir returns the resolved log directory.
func (c Config) LogDir() string {
	if c.Logger.Dir != "" {
		if filepath.IsAbs(c.Logger.Dir) {
			return c.Logger.Dir
		}
		return filepath.Join(c.Dir, c.Logger.Dir)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".tfui", "logs")
}

// ApplyOverrides applies --config key=value pairs to the config.
// Keys use dot-notation matching yaml structure (e.g., "logger.dir", "terraform.bin").
func (c *Config) ApplyOverrides(overrides []string) {
	if c.Overrides == nil {
		c.Overrides = make(map[string]string)
	}
	for _, ov := range overrides {
		key, value, found := parseOverride(ov)
		if !found {
			continue
		}
		c.Overrides[key] = value
		switch key {
		case "basedir":
			c.BaseDir = value
		case "terraform.bin":
			c.Terraform.Bin = value
		case "logger.dir":
			c.Logger.Dir = value
		}
	}
}

// parseOverride splits "key=value" or "key,value" into parts.
func parseOverride(s string) (key, value string, ok bool) {
	for i, ch := range s {
		if ch == '=' || ch == ',' {
			return s[:i], s[i+1:], true
		}
	}
	return "", "", false
}

// WorkingDir returns the resolved terraform working directory.
// If BaseDir is set, it's resolved relative to Dir. Otherwise Dir is used.
func (c Config) WorkingDir() string {
	if c.BaseDir == "" {
		return c.Dir
	}
	if filepath.IsAbs(c.BaseDir) {
		return c.BaseDir
	}
	return filepath.Join(c.Dir, c.BaseDir)
}

// TerraformConfig holds terraform-specific configuration.
type TerraformConfig struct {
	// Bin is the path to the terraform/tofu binary.
	// Auto-detects if empty: prefers tofu, falls back to terraform.
	Bin string `yaml:"bin"`
}

// TerraformBinary returns the configured terraform binary path.
// Returns empty string if not configured (let terraform-exec handle it).
func (c Config) TerraformBinary() string {
	return c.Terraform.Bin
}

// PluginConfig holds per-plugin configuration as declared in tfui.yaml.
// The Enabled field controls whether the plugin is active; Options holds
// arbitrary plugin-specific settings.
type PluginConfig struct {
	Enabled *bool                  `yaml:"enabled"`
	Options map[string]interface{} `yaml:",inline"`
}

// IsEnabled reports whether the plugin is enabled. If the Enabled field is nil
// (not explicitly set in config), the plugin defaults to enabled.
func (c PluginConfig) IsEnabled() bool {
	if c.Enabled == nil {
		return true
	}
	return *c.Enabled
}

// ScopeConfig defines how to discover terraform projects in a monorepo
// by specifying glob patterns that match directories containing .tf files.
// Deprecated: use chdir.members in tfui.hcl instead.
type ScopeConfig struct {
	Paths []string `yaml:"paths"`
}

// DefaultConfig returns a Config with sensible defaults: current directory as
// working dir and "progress" as the UI mode.
func DefaultConfig() Config {
	return Config{
		Dir:  ".",
		Mode: "progress",
	}
}


// DetectBinary returns the terraform binary to use. If configured is non-empty,
// it is returned as-is. Otherwise, it prefers "tofu" (OpenTofu) if available on
// PATH, falling back to "terraform".
func DetectBinary(configured string) string {
	if configured != "" {
		return configured
	}
	if _, err := exec.LookPath("tofu"); err == nil {
		return "tofu"
	}
	return "terraform"
}

// HasTerraformFiles checks if a directory contains .tf or .tofu files.
func HasTerraformFiles(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !e.IsDir() && (filepath.Ext(e.Name()) == ".tf" || filepath.Ext(e.Name()) == ".tofu") {
			return true
		}
	}
	return false
}
