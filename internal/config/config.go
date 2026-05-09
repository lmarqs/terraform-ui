package config

import (
	"os"
	"os/exec"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds all configuration for the tfui application.
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

	// Projects defines monorepo project discovery (similar to pnpm-workspace.yaml).
	Projects ProjectsConfig `yaml:"projects"`

	// Plugins is a map of plugin ID → plugin config.
	// Plugins not listed are enabled with default settings.
	Plugins map[string]PluginConfig `yaml:"plugins"`
}

// PluginConfig holds per-plugin configuration.
type PluginConfig struct {
	Enabled *bool                  `yaml:"enabled"`
	Options map[string]interface{} `yaml:",inline"`
}

// IsEnabled returns whether the plugin is enabled (defaults to true).
func (c PluginConfig) IsEnabled() bool {
	if c.Enabled == nil {
		return true
	}
	return *c.Enabled
}

// ProjectsConfig defines how to discover terraform projects in a monorepo.
type ProjectsConfig struct {
	// Paths is a list of glob patterns for module directories.
	// Example: ["infra/*", "modules/**", "envs/production"]
	Paths []string `yaml:"paths"`
}

// ConfigFileName is the name of the config file to look for.
const ConfigFileName = "tfui.yaml"

// DefaultConfig returns the default configuration.
func DefaultConfig() Config {
	return Config{
		Dir:  ".",
		Mode: "progress",
	}
}

// Load reads the config file from the given directory or its parents.
// It walks up the directory tree looking for tfui.yaml (like pnpm-workspace.yaml).
func Load(dir string) (Config, error) {
	cfg := DefaultConfig()
	cfg.Dir = dir

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return cfg, nil
	}

	configPath := findConfigFile(absDir)
	if configPath == "" {
		return cfg, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return cfg, nil
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, err
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

// DiscoverProjects returns all terraform project directories matching the configured patterns.
func (c Config) DiscoverProjects() ([]string, error) {
	if len(c.Projects.Paths) == 0 {
		return []string{c.Dir}, nil
	}

	absDir, err := filepath.Abs(c.Dir)
	if err != nil {
		return nil, err
	}

	var modules []string
	seen := make(map[string]bool)

	for _, pattern := range c.Projects.Paths {
		fullPattern := filepath.Join(absDir, pattern)
		matches, err := filepath.Glob(fullPattern)
		if err != nil {
			continue
		}

		for _, match := range matches {
			if seen[match] {
				continue
			}
			if hasTerraformFiles(match) {
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

// DetectBinary returns the terraform binary to use.
// If the given binary is non-empty, it is returned as-is.
// Otherwise, it checks if "tofu" is on PATH (preferring OpenTofu),
// and falls back to "terraform".
func DetectBinary(configured string) string {
	if configured != "" {
		return configured
	}
	if _, err := exec.LookPath("tofu"); err == nil {
		return "tofu"
	}
	return "terraform"
}

func hasTerraformFiles(dir string) bool {
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
