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

func TestLogDir_WhenAbsoluteLoggerDir_ShouldReturnItDirectly(t *testing.T) {
	cfg := Config{Dir: "/project", Logger: LoggerConfig{Dir: "/var/log/tfui"}}
	got := cfg.LogDir()
	if got != "/var/log/tfui" {
		t.Errorf("LogDir() = %q, want %q", got, "/var/log/tfui")
	}
}

func TestLogDir_WhenRelativeLoggerDir_ShouldJoinWithConfigDir(t *testing.T) {
	cfg := Config{Dir: "/project", Logger: LoggerConfig{Dir: "logs"}}
	got := cfg.LogDir()
	want := filepath.Join("/project", "logs")
	if got != want {
		t.Errorf("LogDir() = %q, want %q", got, want)
	}
}

func TestLogDir_WhenNoLoggerDir_ShouldUseHomeDefault(t *testing.T) {
	cfg := Config{Dir: "/project"}
	got := cfg.LogDir()
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("cannot get home dir: %v", err)
	}
	want := filepath.Join(home, ".tfui", "logs")
	if got != want {
		t.Errorf("LogDir() = %q, want %q", got, want)
	}
}

func TestApplyOverrides_WhenValidOverrides_ShouldApplyAll(t *testing.T) {
	tests := []struct {
		name      string
		overrides []string
		checkFn   func(t *testing.T, cfg *Config)
	}{
		{
			name:      "ShouldSetBaseDir",
			overrides: []string{"basedir=infra/prod"},
			checkFn: func(t *testing.T, cfg *Config) {
				if cfg.BaseDir != "infra/prod" {
					t.Errorf("BaseDir = %q, want %q", cfg.BaseDir, "infra/prod")
				}
			},
		},
		{
			name:      "ShouldSetTerraformBin",
			overrides: []string{"terraform.bin=tofu"},
			checkFn: func(t *testing.T, cfg *Config) {
				if cfg.Terraform.Bin != "tofu" {
					t.Errorf("Terraform.Bin = %q, want %q", cfg.Terraform.Bin, "tofu")
				}
			},
		},
		{
			name:      "ShouldSetLoggerDir",
			overrides: []string{"logger.dir=/tmp/logs"},
			checkFn: func(t *testing.T, cfg *Config) {
				if cfg.Logger.Dir != "/tmp/logs" {
					t.Errorf("Logger.Dir = %q, want %q", cfg.Logger.Dir, "/tmp/logs")
				}
			},
		},
		{
			name:      "ShouldStoreUnknownKeyInOverridesMap",
			overrides: []string{"custom.key=custom-value"},
			checkFn: func(t *testing.T, cfg *Config) {
				if cfg.Overrides["custom.key"] != "custom-value" {
					t.Errorf("Overrides[custom.key] = %q, want %q", cfg.Overrides["custom.key"], "custom-value")
				}
			},
		},
		{
			name:      "ShouldHandleMultipleOverrides",
			overrides: []string{"basedir=a", "terraform.bin=b"},
			checkFn: func(t *testing.T, cfg *Config) {
				if cfg.BaseDir != "a" {
					t.Errorf("BaseDir = %q, want %q", cfg.BaseDir, "a")
				}
				if cfg.Terraform.Bin != "b" {
					t.Errorf("Terraform.Bin = %q, want %q", cfg.Terraform.Bin, "b")
				}
			},
		},
		{
			name:      "ShouldSkipInvalidOverridesWithoutEquals",
			overrides: []string{"noequals"},
			checkFn: func(t *testing.T, cfg *Config) {
				if len(cfg.Overrides) != 0 {
					t.Errorf("Overrides should be empty for invalid override, got %v", cfg.Overrides)
				}
			},
		},
		{
			name:      "ShouldAcceptCommaAsSeparator",
			overrides: []string{"key,value"},
			checkFn: func(t *testing.T, cfg *Config) {
				if cfg.Overrides["key"] != "value" {
					t.Errorf("Overrides[key] = %q, want %q", cfg.Overrides["key"], "value")
				}
			},
		},
		{
			name:      "ShouldHandleEmptyValue",
			overrides: []string{"basedir="},
			checkFn: func(t *testing.T, cfg *Config) {
				if cfg.BaseDir != "" {
					t.Errorf("BaseDir = %q, want empty", cfg.BaseDir)
				}
				if cfg.Overrides["basedir"] != "" {
					t.Errorf("Overrides[basedir] = %q, want empty", cfg.Overrides["basedir"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{}
			cfg.ApplyOverrides(tt.overrides)
			tt.checkFn(t, cfg)
		})
	}
}

func TestApplyOverrides_WhenOverridesMapAlreadyExists_ShouldAppend(t *testing.T) {
	cfg := &Config{
		Overrides: map[string]string{"existing": "value"},
	}
	cfg.ApplyOverrides([]string{"new=entry"})

	if cfg.Overrides["existing"] != "value" {
		t.Errorf("existing override lost")
	}
	if cfg.Overrides["new"] != "entry" {
		t.Errorf("new override not added")
	}
}

func TestLogDir_WhenUserHomeDirFails_ShouldReturnEmpty(t *testing.T) {
	t.Setenv("HOME", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("PLAN9", "")

	cfg := Config{Dir: "/project"}
	got := cfg.LogDir()

	home, err := os.UserHomeDir()
	if err != nil {
		if got != "" {
			t.Errorf("LogDir() = %q, want empty when HOME is unset", got)
		}
	} else {
		want := filepath.Join(home, ".tfui", "logs")
		if got != want {
			t.Errorf("LogDir() = %q, want %q", got, want)
		}
	}
}

func TestParseOverride_WhenTableDriven_ShouldHandleAllCases(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantKey   string
		wantValue string
		wantOk    bool
	}{
		{"ShouldParseSimpleKeyValue", "key=value", "key", "value", true},
		{"ShouldParseWithEmptyValue", "key=", "key", "", true},
		{"ShouldParseWithEqualsInValue", "key=val=ue", "key", "val=ue", true},
		{"ShouldReturnFalseForNoSeparator", "noseparator", "", "", false},
		{"ShouldReturnFalseForEmptyString", "", "", "", false},
		{"ShouldParseCommaAsSeparator", "key,value", "key", "value", true},
		{"ShouldPreferFirstSeparator", "a=b,c", "a", "b,c", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, value, ok := parseOverride(tt.input)
			if ok != tt.wantOk {
				t.Errorf("parseOverride(%q) ok = %v, want %v", tt.input, ok, tt.wantOk)
			}
			if key != tt.wantKey {
				t.Errorf("parseOverride(%q) key = %q, want %q", tt.input, key, tt.wantKey)
			}
			if value != tt.wantValue {
				t.Errorf("parseOverride(%q) value = %q, want %q", tt.input, value, tt.wantValue)
			}
		})
	}
}
