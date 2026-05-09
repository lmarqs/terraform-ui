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
	// Dir is the working directory for terraform operations.
	Dir string `yaml:"-"`

	// Workspace is the terraform workspace name.
	Workspace string `yaml:"-"`

	// TerraformBinary is the path to the terraform binary.
	TerraformBinary string `yaml:"terraform_binary"`

	// Mode is the UI mode for non-interactive commands: silent, spinner, progress, agent.
	Mode string `yaml:"-"`

	// Targets is a list of resource targets for plan/apply.
	Targets []string `yaml:"-"`

	// Context defines monorepo project discovery (similar to pnpm-workspace.yaml).
	Context ContextConfig `yaml:"context"`

	// Plugins is a map of plugin ID → plugin config.
	// Plugins not listed are enabled with default settings.
	Plugins map[string]PluginConfig `yaml:"plugins"`
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

// ContextConfig defines how to discover terraform projects in a monorepo
// by specifying glob patterns that match directories containing .tf files.
type ContextConfig struct {
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

// DiscoverContext returns all terraform project directories matching the glob
// patterns configured in Context.Paths. If no patterns are configured, it
// auto-discovers terraform subdirectories (one level deep). If only the root
// directory contains terraform files, it returns just the root directory.
func (c Config) DiscoverContext() ([]string, error) {
	absDir, err := filepath.Abs(c.Dir)
	if err != nil {
		return nil, err
	}

	if len(c.Context.Paths) == 0 {
		return c.autoDiscoverContext(absDir)
	}

	var modules []string
	seen := make(map[string]bool)

	for _, pattern := range c.Context.Paths {
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

// autoDiscoverContext walks the directory tree to find terraform projects.
// Skips hidden directories and stops descending once a dir has .tf files.
// If only the root directory has terraform files, returns just the root.
func (c Config) autoDiscoverContext(absDir string) ([]string, error) {
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
