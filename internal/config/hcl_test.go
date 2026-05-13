package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeHCL(t *testing.T, dir, content string) {
	t.Helper()
	err := os.WriteFile(filepath.Join(dir, HCLConfigFileName), []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}
}

// --- LoadRoot: valid full config ---

func TestLoadRoot_WhenValidFullConfig_ShouldParseAllBlocks(t *testing.T) {
	dir := t.TempDir()
	writeHCL(t, dir, `
terraform {
  bin = "/usr/local/bin/tofu"
}

chdir {
  members = [
    "modules/vpc",
    "modules/ecs",
  ]
}

cache {
  staleness_threshold = "5m"
}

ai {
  enabled  = true
  provider = "bedrock"
  model    = "claude-sonnet-4-6-20250514"
  region   = "us-east-1"
}

defaults {
  parallelism = 10
  lock        = true

  var_file "common/tags.tfvars" {}

  plugin "risk" {
    enabled = true
    level   = "high"
  }
}
`)

	cfg, err := LoadRoot(dir)
	if err != nil {
		t.Fatalf("LoadRoot() error: %v", err)
	}

	if cfg.Terraform.Bin != "/usr/local/bin/tofu" {
		t.Errorf("Terraform.Bin = %q, want %q", cfg.Terraform.Bin, "/usr/local/bin/tofu")
	}

	if len(cfg.Chdir.Members) != 2 {
		t.Fatalf("Chdir.Members length = %d, want 2", len(cfg.Chdir.Members))
	}
	if cfg.Chdir.Members[0] != "modules/vpc" {
		t.Errorf("Chdir.Members[0] = %q, want %q", cfg.Chdir.Members[0], "modules/vpc")
	}
	if cfg.Chdir.Members[1] != "modules/ecs" {
		t.Errorf("Chdir.Members[1] = %q, want %q", cfg.Chdir.Members[1], "modules/ecs")
	}

	if cfg.Cache.StalenessThreshold != "5m" {
		t.Errorf("Cache.StalenessThreshold = %q, want %q", cfg.Cache.StalenessThreshold, "5m")
	}

	if !cfg.AI.Enabled {
		t.Error("AI.Enabled = false, want true")
	}
	if cfg.AI.Provider != "bedrock" {
		t.Errorf("AI.Provider = %q, want %q", cfg.AI.Provider, "bedrock")
	}
	if cfg.AI.Model != "claude-sonnet-4-6-20250514" {
		t.Errorf("AI.Model = %q, want %q", cfg.AI.Model, "claude-sonnet-4-6-20250514")
	}
	if cfg.AI.Region != "us-east-1" {
		t.Errorf("AI.Region = %q, want %q", cfg.AI.Region, "us-east-1")
	}

	if cfg.Defaults.Parallelism != 10 {
		t.Errorf("Defaults.Parallelism = %d, want 10", cfg.Defaults.Parallelism)
	}
	if cfg.Defaults.Lock == nil || !*cfg.Defaults.Lock {
		t.Error("Defaults.Lock should be true")
	}
}

// --- LoadRoot: minimal config (terraform.bin only) ---

func TestLoadRoot_WhenMinimalConfig_ShouldSucceed(t *testing.T) {
	dir := t.TempDir()
	writeHCL(t, dir, `
terraform {
  bin = "terraform"
}
`)

	cfg, err := LoadRoot(dir)
	if err != nil {
		t.Fatalf("LoadRoot() error: %v", err)
	}

	if cfg.Terraform.Bin != "terraform" {
		t.Errorf("Terraform.Bin = %q, want %q", cfg.Terraform.Bin, "terraform")
	}
	if len(cfg.Chdir.Members) != 0 {
		t.Errorf("Chdir.Members should be empty, got %v", cfg.Chdir.Members)
	}
}

// --- LoadRoot: empty file is valid ---

func TestLoadRoot_WhenEmptyFile_ShouldReturnEmptyConfig(t *testing.T) {
	dir := t.TempDir()
	writeHCL(t, dir, ``)

	cfg, err := LoadRoot(dir)
	if err != nil {
		t.Fatalf("LoadRoot() error: %v", err)
	}

	if cfg.Terraform.Bin != "" {
		t.Errorf("Terraform.Bin = %q, want empty", cfg.Terraform.Bin)
	}
	if len(cfg.Chdir.Members) != 0 {
		t.Errorf("Chdir.Members should be empty, got %v", cfg.Chdir.Members)
	}
}

// --- LoadRoot: no terraform block is valid ---

func TestLoadRoot_WhenOnlyDefaults_ShouldSucceed(t *testing.T) {
	dir := t.TempDir()
	writeHCL(t, dir, `
defaults {
  parallelism = 5
}
`)

	cfg, err := LoadRoot(dir)
	if err != nil {
		t.Fatalf("LoadRoot() error: %v", err)
	}

	if cfg.Terraform.Bin != "" {
		t.Errorf("Terraform.Bin = %q, want empty (not required)", cfg.Terraform.Bin)
	}
	if cfg.Defaults.Parallelism != 5 {
		t.Errorf("Defaults.Parallelism = %d, want 5", cfg.Defaults.Parallelism)
	}
}

// --- LoadRoot: terraform.bin empty is valid (passthrough) ---

func TestLoadRoot_WhenNoTerraformBin_ShouldReturnEmptyBin(t *testing.T) {
	dir := t.TempDir()
	writeHCL(t, dir, `
terraform {}
`)

	cfg, err := LoadRoot(dir)
	if err != nil {
		t.Fatalf("LoadRoot() error: %v", err)
	}

	if cfg.Terraform.Bin != "" {
		t.Errorf("Terraform.Bin = %q, want empty", cfg.Terraform.Bin)
	}
}

// --- LoadRoot validation ---

func TestLoadRoot_WhenInvalidHCLSyntax_ShouldReturnError(t *testing.T) {
	dir := t.TempDir()
	writeHCL(t, dir, `
terraform {
  bin = "terraform"
  this is {{{{ not valid
}
`)

	_, err := LoadRoot(dir)
	if err == nil {
		t.Error("LoadRoot() should return error for invalid HCL syntax")
	}
}

func TestLoadRoot_WhenNoConfigFile_ShouldReturnSpecificError(t *testing.T) {
	dir := t.TempDir()

	_, err := LoadRoot(dir)
	if err == nil {
		t.Fatal("LoadRoot() should return error when no config file exists")
	}

	var notFound *ConfigNotFoundError
	if !errors.As(err, &notFound) {
		t.Errorf("error should be *ConfigNotFoundError, got %T: %v", err, err)
	}
}

func TestLoadRoot_WhenPermissionDenied_ShouldReturnError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping permission test when running as root")
	}

	dir := t.TempDir()
	configPath := filepath.Join(dir, HCLConfigFileName)
	err := os.WriteFile(configPath, []byte(`terraform { bin = "terraform" }`), 0000)
	if err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	_, err = LoadRoot(dir)
	if err == nil {
		t.Error("LoadRoot() should return error for unreadable config file")
	}
}

// --- LoadRoot: no walk-up (explicit only) ---

func TestLoadRoot_WhenFileInParentDir_ShouldNotWalkUp(t *testing.T) {
	root := t.TempDir()
	subDir := filepath.Join(root, "a", "b", "c")
	err := os.MkdirAll(subDir, 0755)
	if err != nil {
		t.Fatalf("failed to create subdirectory: %v", err)
	}

	writeHCL(t, root, `
terraform {
  bin = "/usr/bin/terraform"
}
`)

	_, err = LoadRoot(subDir)
	if err == nil {
		t.Fatal("LoadRoot() should NOT walk up directories")
	}

	var notFound *ConfigNotFoundError
	if !errors.As(err, &notFound) {
		t.Errorf("error should be *ConfigNotFoundError, got %T: %v", err, err)
	}
}

func TestLoadRoot_WhenFileInCurrentDir_ShouldLoadDirectly(t *testing.T) {
	dir := t.TempDir()
	writeHCL(t, dir, `
terraform {
  bin = "tofu"
}
`)

	cfg, err := LoadRoot(dir)
	if err != nil {
		t.Fatalf("LoadRoot() error: %v", err)
	}

	if cfg.Terraform.Bin != "tofu" {
		t.Errorf("Terraform.Bin = %q, want %q", cfg.Terraform.Bin, "tofu")
	}
}

// --- LoadRoot: chdir.members ---

func TestLoadRoot_WhenChdirMembersExplicitList_ShouldPreserveOrder(t *testing.T) {
	dir := t.TempDir()
	writeHCL(t, dir, `
terraform {
  bin = "terraform"
}

chdir {
  members = [
    "modules/ecs",
    "modules/vpc",
    "modules/rds",
  ]
}
`)

	cfg, err := LoadRoot(dir)
	if err != nil {
		t.Fatalf("LoadRoot() error: %v", err)
	}

	expected := []string{"modules/ecs", "modules/vpc", "modules/rds"}
	if len(cfg.Chdir.Members) != len(expected) {
		t.Fatalf("Chdir.Members length = %d, want %d", len(cfg.Chdir.Members), len(expected))
	}
	for i, want := range expected {
		if cfg.Chdir.Members[i] != want {
			t.Errorf("Chdir.Members[%d] = %q, want %q", i, cfg.Chdir.Members[i], want)
		}
	}
}

func TestLoadRoot_WhenChdirMembersEmpty_ShouldBeSingleModule(t *testing.T) {
	dir := t.TempDir()
	writeHCL(t, dir, `
terraform {
  bin = "terraform"
}

chdir {
  members = []
}
`)

	cfg, err := LoadRoot(dir)
	if err != nil {
		t.Fatalf("LoadRoot() error: %v", err)
	}

	if len(cfg.Chdir.Members) != 0 {
		t.Errorf("Chdir.Members = %v, want empty (single-module)", cfg.Chdir.Members)
	}
}

func TestLoadRoot_WhenNoChdirBlock_ShouldDefaultToEmpty(t *testing.T) {
	dir := t.TempDir()
	writeHCL(t, dir, `
terraform {
  bin = "terraform"
}
`)

	cfg, err := LoadRoot(dir)
	if err != nil {
		t.Fatalf("LoadRoot() error: %v", err)
	}

	if len(cfg.Chdir.Members) != 0 {
		t.Errorf("Chdir.Members = %v, want empty", cfg.Chdir.Members)
	}
}

// --- LoadRoot: defaults block ---

func TestLoadRoot_WhenDefaultsParallelism_ShouldParse(t *testing.T) {
	dir := t.TempDir()
	writeHCL(t, dir, `
terraform {
  bin = "terraform"
}

defaults {
  parallelism = 20
}
`)

	cfg, err := LoadRoot(dir)
	if err != nil {
		t.Fatalf("LoadRoot() error: %v", err)
	}

	if cfg.Defaults.Parallelism != 20 {
		t.Errorf("Defaults.Parallelism = %d, want 20", cfg.Defaults.Parallelism)
	}
}

func TestLoadRoot_WhenDefaultsLock_ShouldParseBool(t *testing.T) {
	tests := []struct {
		name    string
		hcl     string
		wantNil bool
		want    bool
	}{
		{
			name: "ShouldBeTrueWhenSet",
			hcl: `
terraform { bin = "terraform" }
defaults { lock = true }
`,
			wantNil: false,
			want:    true,
		},
		{
			name: "ShouldBeFalseWhenSet",
			hcl: `
terraform { bin = "terraform" }
defaults { lock = false }
`,
			wantNil: false,
			want:    false,
		},
		{
			name: "ShouldBeNilWhenOmitted",
			hcl: `
terraform { bin = "terraform" }
defaults {}
`,
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeHCL(t, dir, tt.hcl)

			cfg, err := LoadRoot(dir)
			if err != nil {
				t.Fatalf("LoadRoot() error: %v", err)
			}

			if tt.wantNil {
				if cfg.Defaults.Lock != nil {
					t.Errorf("Defaults.Lock = %v, want nil", *cfg.Defaults.Lock)
				}
			} else {
				if cfg.Defaults.Lock == nil {
					t.Fatal("Defaults.Lock = nil, want non-nil")
				}
				if *cfg.Defaults.Lock != tt.want {
					t.Errorf("Defaults.Lock = %v, want %v", *cfg.Defaults.Lock, tt.want)
				}
			}
		})
	}
}

func TestLoadRoot_WhenDefaultsVarFiles_ShouldPreserveOrder(t *testing.T) {
	dir := t.TempDir()
	writeHCL(t, dir, `
terraform {
  bin = "terraform"
}

defaults {
  var_file "common/tags.tfvars" {}
  var_file "common/network.tfvars" {}
  var_file "common/iam.tfvars" {}
}
`)

	cfg, err := LoadRoot(dir)
	if err != nil {
		t.Fatalf("LoadRoot() error: %v", err)
	}

	expected := []string{"common/tags.tfvars", "common/network.tfvars", "common/iam.tfvars"}
	if len(cfg.Defaults.VarFiles) != len(expected) {
		t.Fatalf("Defaults.VarFiles length = %d, want %d", len(cfg.Defaults.VarFiles), len(expected))
	}
	for i, want := range expected {
		if cfg.Defaults.VarFiles[i] != want {
			t.Errorf("Defaults.VarFiles[%d] = %q, want %q", i, cfg.Defaults.VarFiles[i], want)
		}
	}
}

func TestLoadRoot_WhenDefaultsPlugins_ShouldParseNamedBlocks(t *testing.T) {
	dir := t.TempDir()
	writeHCL(t, dir, `
terraform {
  bin = "terraform"
}

defaults {
  plugin "risk" {
    enabled = true
    level   = "high"
  }

  plugin "phantom" {
    enabled = false
  }
}
`)

	cfg, err := LoadRoot(dir)
	if err != nil {
		t.Fatalf("LoadRoot() error: %v", err)
	}

	risk := cfg.Defaults.PluginConfig("risk")
	if !risk.Enabled {
		t.Error("defaults plugin risk should be enabled")
	}
	if risk.Options["level"] != "high" {
		t.Errorf("defaults plugin risk level = %v, want %q", risk.Options["level"], "high")
	}

	phantom := cfg.Defaults.PluginConfig("phantom")
	if phantom.Enabled {
		t.Error("defaults plugin phantom should be disabled")
	}
}

// --- LoadChild: valid parsing ---

func TestLoadChild_WhenValidConfig_ShouldParseTopLevelAndWorkspaces(t *testing.T) {
	dir := t.TempDir()
	writeHCL(t, dir, `
plugin "risk" {
  level = "critical"
}

var_file "base.tfvars" {}

workspace "default" {
  var_file "dev.tfvars" {}
}

workspace "production" {
  var_file "prod.tfvars" {}
  var "lock_timeout" { value = "30s" }
}
`)

	cfg, err := LoadChild(dir)
	if err != nil {
		t.Fatalf("LoadChild() error: %v", err)
	}

	risk := cfg.PluginConfig("risk")
	if risk.Options["level"] != "critical" {
		t.Errorf("plugin risk level = %v, want %q", risk.Options["level"], "critical")
	}

	if len(cfg.VarFiles) != 1 {
		t.Fatalf("VarFiles length = %d, want 1", len(cfg.VarFiles))
	}
	if cfg.VarFiles[0] != "base.tfvars" {
		t.Errorf("VarFiles[0] = %q, want %q", cfg.VarFiles[0], "base.tfvars")
	}

	if len(cfg.Workspaces) != 2 {
		t.Fatalf("Workspaces length = %d, want 2", len(cfg.Workspaces))
	}
}

func TestLoadChild_WhenWorkspaceVarBlocks_ShouldParseVars(t *testing.T) {
	dir := t.TempDir()
	writeHCL(t, dir, `
workspace "production" {
  var "lock_timeout" { value = "30s" }
  var "region"       { value = "us-east-1" }
}
`)

	cfg, err := LoadChild(dir)
	if err != nil {
		t.Fatalf("LoadChild() error: %v", err)
	}

	ws := cfg.Workspaces[0]
	if ws.Name != "production" {
		t.Fatalf("workspace name = %q, want %q", ws.Name, "production")
	}
	if len(ws.Vars) != 2 {
		t.Fatalf("workspace vars length = %d, want 2", len(ws.Vars))
	}
	if ws.Vars["lock_timeout"] != "30s" {
		t.Errorf("var lock_timeout = %q, want %q", ws.Vars["lock_timeout"], "30s")
	}
	if ws.Vars["region"] != "us-east-1" {
		t.Errorf("var region = %q, want %q", ws.Vars["region"], "us-east-1")
	}
}

func TestLoadChild_WhenNoFile_ShouldReturnSpecificError(t *testing.T) {
	dir := t.TempDir()

	_, err := LoadChild(dir)
	if err == nil {
		t.Fatal("LoadChild() should return error when no config file exists")
	}

	var notFound *ConfigNotFoundError
	if !errors.As(err, &notFound) {
		t.Errorf("error should be *ConfigNotFoundError, got %T: %v", err, err)
	}
}

// --- LoadChild: locked field rejection ---

func TestLoadChild_WhenLockedFieldPresent_ShouldRejectWithError(t *testing.T) {
	tests := []struct {
		name    string
		hcl     string
		keyword string
	}{
		{
			name: "ShouldRejectTerraformBlock",
			hcl: `
terraform {
  bin = "terraform"
}
`,
			keyword: "terraform",
		},
		{
			name: "ShouldRejectChdirBlock",
			hcl: `
chdir {
  members = ["modules/vpc"]
}
`,
			keyword: "chdir",
		},
		{
			name: "ShouldRejectCacheBlock",
			hcl: `
cache {
  staleness_threshold = "5m"
}
`,
			keyword: "cache",
		},
		{
			name: "ShouldRejectAIBlock",
			hcl: `
ai {
  enabled = true
}
`,
			keyword: "ai",
		},
		{
			name: "ShouldRejectDefaultsBlock",
			hcl: `
defaults {
  parallelism = 10
}
`,
			keyword: "defaults",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeHCL(t, dir, tt.hcl)

			_, err := LoadChild(dir)
			if err == nil {
				t.Fatalf("LoadChild() should reject %q block in child config", tt.keyword)
			}
			if !strings.Contains(err.Error(), tt.keyword) {
				t.Errorf("error message should mention %q, got: %v", tt.keyword, err)
			}
		})
	}
}

// --- Resolve: root only ---

func TestResolve_WhenRootOnly_ShouldUseDefaults(t *testing.T) {
	root := &RootConfig{
		Terraform: RootTerraformConfig{Bin: "terraform"},
		Defaults: DefaultsConfig{
			Parallelism: 10,
			VarFiles:    []string{"common/tags.tfvars"},
		},
	}

	resolved := Resolve(root, nil, "default")

	if resolved.Parallelism() != 10 {
		t.Errorf("Parallelism() = %d, want 10", resolved.Parallelism())
	}

	varFiles := resolved.VarFiles()
	if len(varFiles) != 1 {
		t.Fatalf("VarFiles() length = %d, want 1", len(varFiles))
	}
	if varFiles[0] != "common/tags.tfvars" {
		t.Errorf("VarFiles()[0] = %q, want %q", varFiles[0], "common/tags.tfvars")
	}
}

// --- Resolve: root + child ---

func TestResolve_WhenRootAndChild_ShouldMerge(t *testing.T) {
	lockTrue := true
	root := &RootConfig{
		Terraform: RootTerraformConfig{Bin: "terraform"},
		Defaults: DefaultsConfig{
			Parallelism: 10,
			Lock:        &lockTrue,
			VarFiles:    []string{"common/tags.tfvars"},
		},
	}

	child := &ChildConfig{
		VarFiles: []string{"child/base.tfvars"},
	}

	resolved := Resolve(root, child, "default")

	varFiles := resolved.VarFiles()
	if len(varFiles) != 2 {
		t.Fatalf("VarFiles() length = %d, want 2", len(varFiles))
	}
	if varFiles[0] != "common/tags.tfvars" {
		t.Errorf("VarFiles()[0] = %q, want %q (root default)", varFiles[0], "common/tags.tfvars")
	}
	if varFiles[1] != "child/base.tfvars" {
		t.Errorf("VarFiles()[1] = %q, want %q (child)", varFiles[1], "child/base.tfvars")
	}

	if resolved.Parallelism() != 10 {
		t.Errorf("Parallelism() = %d, want 10 (inherited from root)", resolved.Parallelism())
	}
	if resolved.Lock() == nil || !*resolved.Lock() {
		t.Error("Lock() should be true (inherited from root)")
	}
}

// --- Resolve: root + child + workspace ---

func TestResolve_WhenWorkspaceMatches_ShouldApplyWorkspaceOverrides(t *testing.T) {
	root := &RootConfig{
		Terraform: RootTerraformConfig{Bin: "terraform"},
		Defaults: DefaultsConfig{
			VarFiles: []string{"common/tags.tfvars"},
		},
	}

	child := &ChildConfig{
		VarFiles: []string{"child/base.tfvars"},
		Workspaces: []WorkspaceConfig{
			{
				Name:     "production",
				VarFiles: []string{"child/prod.tfvars"},
				Vars:     map[string]string{"lock_timeout": "30s"},
			},
		},
	}

	resolved := Resolve(root, child, "production")

	varFiles := resolved.VarFiles()
	expected := []string{"common/tags.tfvars", "child/base.tfvars", "child/prod.tfvars"}
	if len(varFiles) != len(expected) {
		t.Fatalf("VarFiles() length = %d, want %d; got %v", len(varFiles), len(expected), varFiles)
	}
	for i, want := range expected {
		if varFiles[i] != want {
			t.Errorf("VarFiles()[%d] = %q, want %q", i, varFiles[i], want)
		}
	}

	vars := resolved.Vars()
	if vars["lock_timeout"] != "30s" {
		t.Errorf("Vars()[lock_timeout] = %q, want %q", vars["lock_timeout"], "30s")
	}
}

// --- Resolve: no matching workspace fallback ---

func TestResolve_WhenNoWorkspaceMatch_ShouldFallbackToChildTopLevel(t *testing.T) {
	root := &RootConfig{
		Terraform: RootTerraformConfig{Bin: "terraform"},
		Defaults: DefaultsConfig{
			VarFiles: []string{"common/tags.tfvars"},
		},
	}

	child := &ChildConfig{
		VarFiles: []string{"child/base.tfvars"},
		Workspaces: []WorkspaceConfig{
			{
				Name:     "production",
				VarFiles: []string{"child/prod.tfvars"},
			},
		},
	}

	resolved := Resolve(root, child, "development")

	varFiles := resolved.VarFiles()
	expected := []string{"common/tags.tfvars", "child/base.tfvars"}
	if len(varFiles) != len(expected) {
		t.Fatalf("VarFiles() length = %d, want %d; got %v", len(varFiles), len(expected), varFiles)
	}
	for i, want := range expected {
		if varFiles[i] != want {
			t.Errorf("VarFiles()[%d] = %q, want %q", i, varFiles[i], want)
		}
	}
}

// --- Var-file concatenation order ---

func TestResolve_WhenVarFilesAcrossLevels_ShouldConcatenateInOrder(t *testing.T) {
	root := &RootConfig{
		Terraform: RootTerraformConfig{Bin: "terraform"},
		Defaults: DefaultsConfig{
			VarFiles: []string{"root/a.tfvars", "root/b.tfvars"},
		},
	}

	child := &ChildConfig{
		VarFiles: []string{"child/c.tfvars"},
		Workspaces: []WorkspaceConfig{
			{
				Name:     "staging",
				VarFiles: []string{"ws/d.tfvars", "ws/e.tfvars"},
			},
		},
	}

	resolved := Resolve(root, child, "staging")

	varFiles := resolved.VarFiles()
	expected := []string{"root/a.tfvars", "root/b.tfvars", "child/c.tfvars", "ws/d.tfvars", "ws/e.tfvars"}
	if len(varFiles) != len(expected) {
		t.Fatalf("VarFiles() length = %d, want %d; got %v", len(varFiles), len(expected), varFiles)
	}
	for i, want := range expected {
		if varFiles[i] != want {
			t.Errorf("VarFiles()[%d] = %q, want %q", i, varFiles[i], want)
		}
	}
}

// --- Var merge: later level wins ---

func TestResolve_WhenVarsDuplicated_ShouldLaterLevelWin(t *testing.T) {
	root := &RootConfig{
		Terraform: RootTerraformConfig{Bin: "terraform"},
		Defaults: DefaultsConfig{
			Vars: map[string]string{
				"region":  "us-west-2",
				"env":     "dev",
				"project": "base",
			},
		},
	}

	child := &ChildConfig{
		Vars: map[string]string{
			"env":     "staging",
			"service": "api",
		},
		Workspaces: []WorkspaceConfig{
			{
				Name: "production",
				Vars: map[string]string{
					"env":     "prod",
					"service": "api-prod",
				},
			},
		},
	}

	resolved := Resolve(root, child, "production")

	vars := resolved.Vars()
	if vars["region"] != "us-west-2" {
		t.Errorf("Vars()[region] = %q, want %q (from root)", vars["region"], "us-west-2")
	}
	if vars["env"] != "prod" {
		t.Errorf("Vars()[env] = %q, want %q (workspace wins)", vars["env"], "prod")
	}
	if vars["service"] != "api-prod" {
		t.Errorf("Vars()[service] = %q, want %q (workspace wins)", vars["service"], "api-prod")
	}
	if vars["project"] != "base" {
		t.Errorf("Vars()[project] = %q, want %q (from root)", vars["project"], "base")
	}
}

// --- Workspace glob matching ---

func TestResolve_WhenWorkspaceGlob_ShouldMatchPatterns(t *testing.T) {
	tests := []struct {
		name      string
		workspace string
		wantMatch string
	}{
		{"ShouldMatchExact", "production", "production"},
		{"ShouldMatchGlobDevUsEast", "dev-us-east-1", "dev-*"},
		{"ShouldMatchGlobDevStaging", "dev-staging", "dev-*"},
		{"ShouldFallbackWhenNoMatch", "unknown", ""},
	}

	root := &RootConfig{
		Terraform: RootTerraformConfig{Bin: "terraform"},
	}

	child := &ChildConfig{
		Workspaces: []WorkspaceConfig{
			{
				Name:     "production",
				VarFiles: []string{"prod.tfvars"},
			},
			{
				Name:     "dev-*",
				VarFiles: []string{"dev.tfvars"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolved := Resolve(root, child, tt.workspace)
			varFiles := resolved.VarFiles()

			switch tt.wantMatch {
			case "":
				if len(varFiles) != 0 {
					t.Errorf("no match expected but got VarFiles = %v", varFiles)
				}
			case "production":
				if len(varFiles) != 1 || varFiles[0] != "prod.tfvars" {
					t.Errorf("expected prod.tfvars, got %v", varFiles)
				}
			case "dev-*":
				if len(varFiles) != 1 || varFiles[0] != "dev.tfvars" {
					t.Errorf("expected dev.tfvars, got %v", varFiles)
				}
			}
		})
	}
}

func TestResolve_WhenWorkspaceExactAndGlobBothMatch_ShouldPreferExact(t *testing.T) {
	root := &RootConfig{
		Terraform: RootTerraformConfig{Bin: "terraform"},
	}

	child := &ChildConfig{
		Workspaces: []WorkspaceConfig{
			{
				Name:     "dev-*",
				VarFiles: []string{"glob.tfvars"},
			},
			{
				Name:     "dev-special",
				VarFiles: []string{"exact.tfvars"},
			},
		},
	}

	resolved := Resolve(root, child, "dev-special")

	varFiles := resolved.VarFiles()
	if len(varFiles) != 1 {
		t.Fatalf("VarFiles() length = %d, want 1", len(varFiles))
	}
	if varFiles[0] != "exact.tfvars" {
		t.Errorf("VarFiles()[0] = %q, want %q (exact match beats glob)", varFiles[0], "exact.tfvars")
	}
}

func TestResolve_WhenWorkspaceGlobNoHyphen_ShouldNotMatch(t *testing.T) {
	root := &RootConfig{
		Terraform: RootTerraformConfig{Bin: "terraform"},
	}

	child := &ChildConfig{
		Workspaces: []WorkspaceConfig{
			{
				Name:     "dev-*",
				VarFiles: []string{"dev.tfvars"},
			},
		},
	}

	resolved := Resolve(root, child, "dev")

	varFiles := resolved.VarFiles()
	if len(varFiles) != 0 {
		t.Errorf("glob dev-* should not match 'dev', got VarFiles = %v", varFiles)
	}
}

// --- Plugin merge ---

func TestResolve_WhenPluginConfig_ShouldMergeAcrossLevels(t *testing.T) {
	root := &RootConfig{
		Terraform: RootTerraformConfig{Bin: "terraform"},
		Defaults: DefaultsConfig{
			Plugins: map[string]PluginSettings{
				"risk": {Enabled: true, Options: map[string]interface{}{"level": "medium"}},
			},
		},
	}

	child := &ChildConfig{
		Plugins: map[string]PluginSettings{
			"risk": {Enabled: true, Options: map[string]interface{}{"level": "high"}},
		},
	}

	resolved := Resolve(root, child, "default")

	pc := resolved.PluginConfig("risk")
	if pc.Options["level"] != "high" {
		t.Errorf("plugin risk level = %v, want %q (child overrides root)", pc.Options["level"], "high")
	}
}

func TestResolve_WhenPluginConfigWorkspace_ShouldOverrideChild(t *testing.T) {
	root := &RootConfig{
		Terraform: RootTerraformConfig{Bin: "terraform"},
		Defaults: DefaultsConfig{
			Plugins: map[string]PluginSettings{
				"risk": {Enabled: true, Options: map[string]interface{}{"level": "low"}},
			},
		},
	}

	child := &ChildConfig{
		Plugins: map[string]PluginSettings{
			"risk": {Enabled: true, Options: map[string]interface{}{"level": "medium"}},
		},
		Workspaces: []WorkspaceConfig{
			{
				Name: "production",
				Plugins: map[string]PluginSettings{
					"risk": {Enabled: true, Options: map[string]interface{}{"level": "critical"}},
				},
			},
		},
	}

	resolved := Resolve(root, child, "production")

	pc := resolved.PluginConfig("risk")
	if pc.Options["level"] != "critical" {
		t.Errorf("plugin risk level = %v, want %q (workspace overrides child)", pc.Options["level"], "critical")
	}
}

func TestResolve_WhenPluginNotDeclaredAnywhere_ShouldReturnEmptyConfig(t *testing.T) {
	root := &RootConfig{
		Terraform: RootTerraformConfig{Bin: "terraform"},
	}

	resolved := Resolve(root, nil, "default")

	pc := resolved.PluginConfig("undeclared")
	if len(pc.Options) > 0 {
		t.Errorf("undeclared plugin should have empty options, got %v", pc.Options)
	}
}

// --- Resolve: LockTimeout ---

func TestResolve_WhenLockTimeout_ShouldInheritFromWorkspace(t *testing.T) {
	root := &RootConfig{
		Terraform: RootTerraformConfig{Bin: "terraform"},
	}

	child := &ChildConfig{
		Workspaces: []WorkspaceConfig{
			{
				Name:        "production",
				LockTimeout: "30s",
			},
		},
	}

	resolved := Resolve(root, child, "production")

	if resolved.LockTimeout() != "30s" {
		t.Errorf("LockTimeout() = %q, want %q", resolved.LockTimeout(), "30s")
	}
}

func TestResolve_WhenLockTimeoutNotSet_ShouldReturnEmpty(t *testing.T) {
	root := &RootConfig{
		Terraform: RootTerraformConfig{Bin: "terraform"},
	}

	resolved := Resolve(root, nil, "default")

	if resolved.LockTimeout() != "" {
		t.Errorf("LockTimeout() = %q, want empty", resolved.LockTimeout())
	}
}

// --- Optional config: no file returns specific error type ---

func TestConfigNotFoundError_ShouldImplementErrorInterface(t *testing.T) {
	err := &ConfigNotFoundError{Dir: "/some/path"}
	msg := err.Error()
	if !strings.Contains(msg, "/some/path") {
		t.Errorf("error message should contain dir path, got: %q", msg)
	}
}

// --- Edge cases ---

func TestLoadRoot_WhenEmptyDefaultsBlock_ShouldSucceed(t *testing.T) {
	dir := t.TempDir()
	writeHCL(t, dir, `
terraform {
  bin = "terraform"
}

defaults {}
`)

	cfg, err := LoadRoot(dir)
	if err != nil {
		t.Fatalf("LoadRoot() error: %v", err)
	}

	if cfg.Defaults.Parallelism != 0 {
		t.Errorf("Defaults.Parallelism = %d, want 0", cfg.Defaults.Parallelism)
	}
}

func TestLoadChild_WhenEmptyFile_ShouldSucceed(t *testing.T) {
	dir := t.TempDir()
	writeHCL(t, dir, ``)

	cfg, err := LoadChild(dir)
	if err != nil {
		t.Fatalf("LoadChild() error: %v", err)
	}

	if len(cfg.VarFiles) != 0 {
		t.Errorf("VarFiles should be empty, got %v", cfg.VarFiles)
	}
	if len(cfg.Workspaces) != 0 {
		t.Errorf("Workspaces should be empty, got %v", cfg.Workspaces)
	}
}

func TestLoadChild_WhenMultipleWorkspaces_ShouldParseAll(t *testing.T) {
	dir := t.TempDir()
	writeHCL(t, dir, `
workspace "default" {
  var_file "dev.tfvars" {}
}

workspace "staging" {
  var_file "staging.tfvars" {}
}

workspace "production" {
  var_file "prod.tfvars" {}
}
`)

	cfg, err := LoadChild(dir)
	if err != nil {
		t.Fatalf("LoadChild() error: %v", err)
	}

	if len(cfg.Workspaces) != 3 {
		t.Fatalf("Workspaces length = %d, want 3", len(cfg.Workspaces))
	}

	names := make(map[string]bool)
	for _, ws := range cfg.Workspaces {
		names[ws.Name] = true
	}
	for _, want := range []string{"default", "staging", "production"} {
		if !names[want] {
			t.Errorf("workspace %q not found", want)
		}
	}
}

func TestResolve_WhenNilChild_ShouldNotPanic(t *testing.T) {
	root := &RootConfig{
		Terraform: RootTerraformConfig{Bin: "terraform"},
		Defaults: DefaultsConfig{
			Parallelism: 5,
			VarFiles:    []string{"root.tfvars"},
		},
	}

	resolved := Resolve(root, nil, "default")

	if resolved.Parallelism() != 5 {
		t.Errorf("Parallelism() = %d, want 5", resolved.Parallelism())
	}
	if len(resolved.VarFiles()) != 1 {
		t.Errorf("VarFiles() length = %d, want 1", len(resolved.VarFiles()))
	}
}

func TestResolve_WhenEmptyWorkspaceName_ShouldSkipWorkspaceMatching(t *testing.T) {
	root := &RootConfig{
		Terraform: RootTerraformConfig{Bin: "terraform"},
	}

	child := &ChildConfig{
		VarFiles: []string{"base.tfvars"},
		Workspaces: []WorkspaceConfig{
			{
				Name:     "production",
				VarFiles: []string{"prod.tfvars"},
			},
		},
	}

	resolved := Resolve(root, child, "")

	varFiles := resolved.VarFiles()
	if len(varFiles) != 1 {
		t.Fatalf("VarFiles() length = %d, want 1 (child only, no workspace)", len(varFiles))
	}
	if varFiles[0] != "base.tfvars" {
		t.Errorf("VarFiles()[0] = %q, want %q", varFiles[0], "base.tfvars")
	}
}
