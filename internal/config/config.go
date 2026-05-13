package config

import (
	"os"
	"os/exec"
	"path/filepath"
)

// Config holds runtime configuration for the tfui application.
// Populated from HCL config (tfui.hcl), CLI flags, and resolved config.
type Config struct {
	Dir         string
	BaseDir     string
	Workspace   string
	ActiveScope string
	ReadOnly    bool
	Terraform   TerraformConfig
	Logger      LoggerConfig
	Mode        string
	Targets     []string
	Plugins     map[string]PluginConfig
	Overrides   map[string]string
	ExtraArgs   []string
	VarFiles    []string
	Vars        map[string]string
}

type LoggerConfig struct {
	Dir string
}

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

func parseOverride(s string) (key, value string, ok bool) {
	for i, ch := range s {
		if ch == '=' || ch == ',' {
			return s[:i], s[i+1:], true
		}
	}
	return "", "", false
}

// WorkingDir returns the resolved terraform working directory.
func (c Config) WorkingDir() string {
	if c.BaseDir == "" {
		return c.Dir
	}
	if filepath.IsAbs(c.BaseDir) {
		return c.BaseDir
	}
	return filepath.Join(c.Dir, c.BaseDir)
}

type TerraformConfig struct {
	Bin string
}

// TerraformBinary returns the configured terraform binary path.
// Returns empty string if not configured (let terraform-exec handle it).
func (c Config) TerraformBinary() string {
	return c.Terraform.Bin
}

// PluginConfig holds per-plugin configuration.
type PluginConfig struct {
	Enabled *bool
	Options map[string]interface{}
}

// IsEnabled reports whether the plugin is enabled.
// Nil Enabled defaults to true.
func (c PluginConfig) IsEnabled() bool {
	if c.Enabled == nil {
		return true
	}
	return *c.Enabled
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Dir:  ".",
		Mode: "progress",
	}
}

// DetectBinary returns the terraform binary to use.
// Only used by the init wizard for auto-detection.
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
