package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Dir != "." {
		t.Errorf("DefaultConfig().Dir = %q, want %q", cfg.Dir, ".")
	}
}

func TestTerraformBinary_ReturnsConfiguredValue(t *testing.T) {
	cfg := Config{Terraform: TerraformConfig{Bin: "tofu"}}
	if cfg.TerraformBinary() != "tofu" {
		t.Errorf("TerraformBinary() = %q, want %q", cfg.TerraformBinary(), "tofu")
	}
}

func TestTerraformBinary_ReturnsDefaultWhenNotConfigured(t *testing.T) {
	cfg := Config{}
	if cfg.TerraformBinary() != "terraform" {
		t.Errorf("TerraformBinary() = %q, want %q", cfg.TerraformBinary(), "terraform")
	}
}

func TestPluginConfig_IsEnabled(t *testing.T) {
	tests := []struct {
		name     string
		config   PluginConfig
		expected bool
	}{
		{
			name:     "nil Enabled pointer defaults to true",
			config:   PluginConfig{Enabled: nil},
			expected: true,
		},
		{
			name:     "explicit true",
			config:   PluginConfig{Enabled: boolPtr(true)},
			expected: true,
		},
		{
			name:     "explicit false",
			config:   PluginConfig{Enabled: boolPtr(false)},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.IsEnabled()
			if result != tt.expected {
				t.Errorf("IsEnabled() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func boolPtr(b bool) *bool { return &b }

func TestHasTerraformFiles(t *testing.T) {
	tests := []struct {
		name     string
		files    []string
		expected bool
	}{
		{"directory with .tf file", []string{"main.tf"}, true},
		{"directory with .tofu file", []string{"main.tofu"}, true},
		{"directory without terraform files", []string{"readme.md", "script.sh"}, false},
		{"empty directory", []string{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for _, f := range tt.files {
				err := os.WriteFile(filepath.Join(dir, f), []byte(""), 0644)
				if err != nil {
					t.Fatalf("failed to write file: %v", err)
				}
			}
			result := HasTerraformFiles(dir)
			if result != tt.expected {
				t.Errorf("HasTerraformFiles() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestHasTerraformFiles_NonexistentDir(t *testing.T) {
	result := HasTerraformFiles("/nonexistent/path/that/does/not/exist")
	if result {
		t.Error("HasTerraformFiles() on nonexistent dir should return false")
	}
}

func TestWorkingDir_NoBaseDir(t *testing.T) {
	cfg := Config{Dir: "/project/root"}
	if cfg.WorkingDir() != "/project/root" {
		t.Errorf("WorkingDir() = %q, want %q", cfg.WorkingDir(), "/project/root")
	}
}

func TestWorkingDir_RelativeBaseDir(t *testing.T) {
	cfg := Config{Dir: "/project/root", BaseDir: "infra/prod"}
	if cfg.WorkingDir() != "/project/root/infra/prod" {
		t.Errorf("WorkingDir() = %q, want %q", cfg.WorkingDir(), "/project/root/infra/prod")
	}
}

func TestWorkingDir_AbsoluteBaseDir(t *testing.T) {
	cfg := Config{Dir: "/project/root", BaseDir: "/other/path"}
	if cfg.WorkingDir() != "/other/path" {
		t.Errorf("WorkingDir() = %q, want %q", cfg.WorkingDir(), "/other/path")
	}
}
