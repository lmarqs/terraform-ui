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
	if cfg.Mode != "progress" {
		t.Errorf("DefaultConfig().Mode = %q, want %q", cfg.Mode, "progress")
	}
	if cfg.TerraformBinary != "" {
		t.Errorf("DefaultConfig().TerraformBinary = %q, want empty", cfg.TerraformBinary)
	}
	if cfg.Workspace != "" {
		t.Errorf("DefaultConfig().Workspace = %q, want empty", cfg.Workspace)
	}
	if len(cfg.Targets) != 0 {
		t.Errorf("DefaultConfig().Targets = %v, want empty", cfg.Targets)
	}
	if len(cfg.Projects.Paths) != 0 {
		t.Errorf("DefaultConfig().Projects.Paths = %v, want empty", cfg.Projects.Paths)
	}
	if cfg.Plugins != nil {
		t.Errorf("DefaultConfig().Plugins = %v, want nil", cfg.Plugins)
	}
}

func TestLoad_NoConfigFile(t *testing.T) {
	dir := t.TempDir()

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if cfg.Dir != dir {
		t.Errorf("Load().Dir = %q, want %q", cfg.Dir, dir)
	}
	if cfg.Mode != "progress" {
		t.Errorf("Load().Mode = %q, want %q", cfg.Mode, "progress")
	}
}

func TestLoad_ValidConfig(t *testing.T) {
	dir := t.TempDir()

	configContent := `
terraform_binary: /usr/local/bin/tofu
projects:
  paths:
    - "infra/*"
    - "modules/**"
plugins:
  plan:
    enabled: true
  state:
    enabled: false
    refresh_interval: 30
`
	err := os.WriteFile(filepath.Join(dir, ConfigFileName), []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if cfg.Dir != dir {
		t.Errorf("Load().Dir = %q, want %q", cfg.Dir, dir)
	}
	if cfg.TerraformBinary != "/usr/local/bin/tofu" {
		t.Errorf("Load().TerraformBinary = %q, want %q", cfg.TerraformBinary, "/usr/local/bin/tofu")
	}
	if len(cfg.Projects.Paths) != 2 {
		t.Fatalf("Load().Projects.Paths length = %d, want 2", len(cfg.Projects.Paths))
	}
	if cfg.Projects.Paths[0] != "infra/*" {
		t.Errorf("Load().Projects.Paths[0] = %q, want %q", cfg.Projects.Paths[0], "infra/*")
	}
	if cfg.Projects.Paths[1] != "modules/**" {
		t.Errorf("Load().Projects.Paths[1] = %q, want %q", cfg.Projects.Paths[1], "modules/**")
	}
	if len(cfg.Plugins) != 2 {
		t.Fatalf("Load().Plugins length = %d, want 2", len(cfg.Plugins))
	}
	if !cfg.Plugins["plan"].IsEnabled() {
		t.Error("Load().Plugins[plan] should be enabled")
	}
	if cfg.Plugins["state"].IsEnabled() {
		t.Error("Load().Plugins[state] should be disabled")
	}
}

func TestLoad_WalksUpDirectoryTree(t *testing.T) {
	root := t.TempDir()
	subDir := filepath.Join(root, "a", "b", "c")
	err := os.MkdirAll(subDir, 0755)
	if err != nil {
		t.Fatalf("failed to create subdirectory: %v", err)
	}

	configContent := `
terraform_binary: /usr/bin/terraform
`
	err = os.WriteFile(filepath.Join(root, ConfigFileName), []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := Load(subDir)
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if cfg.TerraformBinary != "/usr/bin/terraform" {
		t.Errorf("Load().TerraformBinary = %q, want %q", cfg.TerraformBinary, "/usr/bin/terraform")
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	dir := t.TempDir()

	err := os.WriteFile(filepath.Join(dir, ConfigFileName), []byte("{{invalid yaml"), 0644)
	if err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	_, err = Load(dir)
	if err == nil {
		t.Error("Load() with invalid YAML should return error")
	}
}

func TestDiscoverProjects_NoPatterns(t *testing.T) {
	cfg := Config{Dir: "/some/dir"}

	projects, err := cfg.DiscoverProjects()
	if err != nil {
		t.Fatalf("DiscoverProjects() returned error: %v", err)
	}

	if len(projects) != 1 {
		t.Fatalf("DiscoverProjects() length = %d, want 1", len(projects))
	}
	if projects[0] != "/some/dir" {
		t.Errorf("DiscoverProjects()[0] = %q, want %q", projects[0], "/some/dir")
	}
}

func TestDiscoverProjects_WithPatterns(t *testing.T) {
	root := t.TempDir()

	// Create directories with .tf files
	infraDir := filepath.Join(root, "infra", "vpc")
	err := os.MkdirAll(infraDir, 0755)
	if err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	err = os.WriteFile(filepath.Join(infraDir, "main.tf"), []byte(""), 0644)
	if err != nil {
		t.Fatalf("failed to write .tf file: %v", err)
	}

	infraDir2 := filepath.Join(root, "infra", "ecs")
	err = os.MkdirAll(infraDir2, 0755)
	if err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	err = os.WriteFile(filepath.Join(infraDir2, "service.tf"), []byte(""), 0644)
	if err != nil {
		t.Fatalf("failed to write .tf file: %v", err)
	}

	// Create a directory without .tf files (should not match)
	emptyDir := filepath.Join(root, "infra", "docs")
	err = os.MkdirAll(emptyDir, 0755)
	if err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	err = os.WriteFile(filepath.Join(emptyDir, "readme.md"), []byte(""), 0644)
	if err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	cfg := Config{
		Dir: root,
		Projects: ProjectsConfig{
			Paths: []string{"infra/*"},
		},
	}

	projects, err := cfg.DiscoverProjects()
	if err != nil {
		t.Fatalf("DiscoverProjects() returned error: %v", err)
	}

	if len(projects) != 2 {
		t.Fatalf("DiscoverProjects() length = %d, want 2 (got %v)", len(projects), projects)
	}

	// Check that returned paths are relative
	for _, p := range projects {
		if filepath.IsAbs(p) {
			t.Errorf("DiscoverProjects() returned absolute path: %q", p)
		}
	}
}

func TestDiscoverProjects_WithTofuFiles(t *testing.T) {
	root := t.TempDir()

	dir := filepath.Join(root, "modules", "network")
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	err = os.WriteFile(filepath.Join(dir, "main.tofu"), []byte(""), 0644)
	if err != nil {
		t.Fatalf("failed to write .tofu file: %v", err)
	}

	cfg := Config{
		Dir: root,
		Projects: ProjectsConfig{
			Paths: []string{"modules/*"},
		},
	}

	projects, err := cfg.DiscoverProjects()
	if err != nil {
		t.Fatalf("DiscoverProjects() returned error: %v", err)
	}

	if len(projects) != 1 {
		t.Fatalf("DiscoverProjects() length = %d, want 1", len(projects))
	}
}

func TestDiscoverProjects_DeduplicatesResults(t *testing.T) {
	root := t.TempDir()

	dir := filepath.Join(root, "infra")
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	err = os.WriteFile(filepath.Join(dir, "main.tf"), []byte(""), 0644)
	if err != nil {
		t.Fatalf("failed to write .tf file: %v", err)
	}

	cfg := Config{
		Dir: root,
		Projects: ProjectsConfig{
			// Both patterns match the same directory
			Paths: []string{"infra", "infra"},
		},
	}

	projects, err := cfg.DiscoverProjects()
	if err != nil {
		t.Fatalf("DiscoverProjects() returned error: %v", err)
	}

	if len(projects) != 1 {
		t.Errorf("DiscoverProjects() should deduplicate, got %d entries: %v", len(projects), projects)
	}
}

func TestDiscoverProjects_InvalidGlobPattern(t *testing.T) {
	root := t.TempDir()

	cfg := Config{
		Dir: root,
		Projects: ProjectsConfig{
			Paths: []string{"[invalid"},
		},
	}

	projects, err := cfg.DiscoverProjects()
	if err != nil {
		t.Fatalf("DiscoverProjects() returned error: %v", err)
	}

	// Invalid glob patterns are skipped, resulting in empty list
	if len(projects) != 0 {
		t.Errorf("DiscoverProjects() with invalid glob = %v, want empty", projects)
	}
}

func TestDetectBinary_ConfiguredValue(t *testing.T) {
	result := DetectBinary("terraform")
	if result != "terraform" {
		t.Errorf("DetectBinary(%q) = %q, want %q", "terraform", result, "terraform")
	}

	result = DetectBinary("/usr/local/bin/tofu")
	if result != "/usr/local/bin/tofu" {
		t.Errorf("DetectBinary(%q) = %q, want %q", "/usr/local/bin/tofu", result, "/usr/local/bin/tofu")
	}
}

func TestDetectBinary_EmptyFallsBack(t *testing.T) {
	result := DetectBinary("")
	// Should return either "tofu" or "terraform" depending on PATH
	if result != "tofu" && result != "terraform" {
		t.Errorf("DetectBinary(\"\") = %q, want \"tofu\" or \"terraform\"", result)
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

func TestFindConfigFile_WalksUpAndStopsAtRoot(t *testing.T) {
	root := t.TempDir()

	// No config file anywhere
	subDir := filepath.Join(root, "a", "b", "c")
	err := os.MkdirAll(subDir, 0755)
	if err != nil {
		t.Fatalf("failed to create subdirectory: %v", err)
	}

	result := findConfigFile(subDir)
	if result != "" {
		t.Errorf("findConfigFile() with no config = %q, want empty", result)
	}
}

func TestFindConfigFile_FindsInCurrentDir(t *testing.T) {
	dir := t.TempDir()

	configPath := filepath.Join(dir, ConfigFileName)
	err := os.WriteFile(configPath, []byte(""), 0644)
	if err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	result := findConfigFile(dir)
	if result != configPath {
		t.Errorf("findConfigFile() = %q, want %q", result, configPath)
	}
}

func TestFindConfigFile_FindsInParentDir(t *testing.T) {
	root := t.TempDir()
	subDir := filepath.Join(root, "child")
	err := os.MkdirAll(subDir, 0755)
	if err != nil {
		t.Fatalf("failed to create subdirectory: %v", err)
	}

	configPath := filepath.Join(root, ConfigFileName)
	err = os.WriteFile(configPath, []byte(""), 0644)
	if err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	result := findConfigFile(subDir)
	if result != configPath {
		t.Errorf("findConfigFile() = %q, want %q", result, configPath)
	}
}

func TestHasTerraformFiles(t *testing.T) {
	tests := []struct {
		name     string
		files    []string
		expected bool
	}{
		{
			name:     "directory with .tf file",
			files:    []string{"main.tf"},
			expected: true,
		},
		{
			name:     "directory with .tofu file",
			files:    []string{"main.tofu"},
			expected: true,
		},
		{
			name:     "directory without terraform files",
			files:    []string{"readme.md", "script.sh"},
			expected: false,
		},
		{
			name:     "empty directory",
			files:    []string{},
			expected: false,
		},
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

			result := hasTerraformFiles(dir)
			if result != tt.expected {
				t.Errorf("hasTerraformFiles() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestHasTerraformFiles_NonexistentDir(t *testing.T) {
	result := hasTerraformFiles("/nonexistent/path/that/does/not/exist")
	if result {
		t.Error("hasTerraformFiles() on nonexistent dir should return false")
	}
}

func boolPtr(v bool) *bool {
	return &v
}
