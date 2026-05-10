package config

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"gopkg.in/yaml.v3"
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

// TerraformBinary returns the resolved terraform binary path for backwards compatibility.
func (c Config) TerraformBinary() string {
	return DetectBinary(c.Terraform.Bin)
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
type ScopeConfig struct {
	// Paths is a list of glob patterns for module directories.
	// Example: ["infra/*", "modules/**", "envs/production"]
	Paths []string `yaml:"paths"`
}

// ConfigFileName is the expected filename for tfui configuration, searched
// upward from the working directory.
const ConfigFileName = "tfui.yaml"

// DefaultConfig returns a Config with sensible defaults: current directory as
// working dir and "progress" as the UI mode.
func DefaultConfig() Config {
	return Config{
		Dir:  ".",
		Mode: "progress",
	}
}

// Load reads the tfui.yaml configuration file from the given directory or its
// ancestor directories. It walks up the directory tree until it finds a config
// file or reaches the filesystem root, similar to how pnpm-workspace.yaml is resolved.
func Load(dir string) (Config, error) {
	cfg := DefaultConfig()
	cfg.Dir = dir

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return cfg, fmt.Errorf("resolving directory path: %w", err)
	}

	configPath := findConfigFile(absDir)
	if configPath == "" {
		return cfg, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return cfg, fmt.Errorf("reading config file %s: %w", configPath, err)
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parsing config file %s: %w", configPath, err)
	}

	cfg.Dir = dir
	return cfg, nil
}

func findConfigFile(dir string) string {
	for {
		path := filepath.Join(dir, ConfigFileName)
		if _, err := os.Stat(path); err == nil {
			return path
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

// DiscoverScopes returns all terraform project directories matching the glob
// patterns configured in Scope.Paths. If no patterns are configured, it
// auto-discovers terraform subdirectories (one level deep). If only the root
// directory contains terraform files, it returns just the root directory.
func (c Config) DiscoverScopes() ([]string, error) {
	absDir, err := filepath.Abs(c.Dir)
	if err != nil {
		return nil, err
	}

	if len(c.Scope.Paths) == 0 {
		return c.autoDiscoverScopes(absDir)
	}

	var modules []string
	seen := make(map[string]bool)

	for _, pattern := range c.Scope.Paths {
		fullPattern := filepath.Join(absDir, pattern)
		matches, err := filepath.Glob(fullPattern)
		if err != nil {
			continue
		}

		for _, match := range matches {
			if seen[match] {
				continue
			}
			if HasTerraformFiles(match) {
				seen[match] = true
				rel, err := filepath.Rel(absDir, match)
				if err != nil {
					rel = match
				}
				modules = append(modules, rel)
			}
		}
	}

	return modules, nil
}

// autoDiscoverScopes walks the directory tree to find terraform projects.
// Skips hidden directories and stops descending once a dir has .tf files.
// If only the root directory has terraform files, returns just the root.
func (c Config) autoDiscoverScopes(absDir string) ([]string, error) {
	var projects []string

	entries, err := os.ReadDir(absDir)
	if err != nil {
		return []string{c.Dir}, nil
	}

	for _, entry := range entries {
		if !entry.IsDir() || entry.Name()[0] == '.' {
			continue
		}
		subPath := filepath.Join(absDir, entry.Name())
		found := discoverTerraformDirs(subPath, absDir)
		projects = append(projects, found...)
	}

	if len(projects) <= 1 {
		return []string{c.Dir}, nil
	}

	return projects, nil
}

func discoverTerraformDirs(dir, root string) []string {
	if HasTerraformFiles(dir) {
		rel, err := filepath.Rel(root, dir)
		if err != nil {
			rel = dir
		}
		return []string{rel}
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var results []string
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name()[0] == '.' {
			continue
		}
		results = append(results, discoverTerraformDirs(filepath.Join(dir, entry.Name()), root)...)
	}
	return results
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
