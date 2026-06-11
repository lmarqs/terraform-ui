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

member "modules/vpc" {}
member "modules/ecs" {}

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

	if len(cfg.Members) != 2 {
		t.Fatalf("Members length = %d, want 2", len(cfg.Members))
	}
	if cfg.Members[0].Path != "modules/vpc" {
		t.Errorf("Members[0].Path = %q, want %q", cfg.Members[0].Path, "modules/vpc")
	}
	if cfg.Members[1].Path != "modules/ecs" {
		t.Errorf("Members[1].Path = %q, want %q", cfg.Members[1].Path, "modules/ecs")
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
	if len(cfg.Members) != 1 || cfg.Members[0].Path != "." {
		t.Errorf("Members = %v, want single %q member", cfg.Members, ".")
	}
}

// --- LoadRoot: empty file is valid ---

func TestLoadRoot_WhenEmptyFile_ShouldDefaultToCurrentDirMember(t *testing.T) {
	dir := t.TempDir()
	writeHCL(t, dir, ``)

	cfg, err := LoadRoot(dir)
	if err != nil {
		t.Fatalf("LoadRoot() error: %v", err)
	}

	if cfg.Terraform.Bin != "" {
		t.Errorf("Terraform.Bin = %q, want empty", cfg.Terraform.Bin)
	}
	if len(cfg.Members) != 1 || cfg.Members[0].Path != "." {
		t.Errorf("Members = %v, want single %q member", cfg.Members, ".")
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

// --- LoadRoot: member blocks ---

func TestLoadRoot_WhenMemberBlocks_ShouldPreserveOrder(t *testing.T) {
	dir := t.TempDir()
	writeHCL(t, dir, `
terraform {
  bin = "terraform"
}

member "modules/ecs" {}
member "modules/vpc" {}
member "modules/rds" {}
`)

	cfg, err := LoadRoot(dir)
	if err != nil {
		t.Fatalf("LoadRoot() error: %v", err)
	}

	expected := []string{"modules/ecs", "modules/vpc", "modules/rds"}
	if len(cfg.Members) != len(expected) {
		t.Fatalf("Members length = %d, want %d", len(cfg.Members), len(expected))
	}
	for i, want := range expected {
		if cfg.Members[i].Path != want {
			t.Errorf("Members[%d].Path = %q, want %q", i, cfg.Members[i].Path, want)
		}
	}
}

func TestLoadRoot_WhenNoMemberBlocks_ShouldDefaultToCurrentDirMember(t *testing.T) {
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

	if len(cfg.Members) != 1 || cfg.Members[0].Path != "." {
		t.Errorf("Members = %v, want single %q member", cfg.Members, ".")
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
			name: "ShouldRejectMemberBlock",
			hcl: `
member "modules/vpc" {}
`,
			keyword: "member",
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

func TestResolve_WhenNilRoot_ShouldReturnEmptyResolved(t *testing.T) {
	resolved := Resolve(nil, nil, "default")

	if resolved.Parallelism() != 0 {
		t.Errorf("Parallelism() = %d, want 0", resolved.Parallelism())
	}
	if len(resolved.VarFiles()) != 0 {
		t.Errorf("VarFiles() should be empty, got %v", resolved.VarFiles())
	}
	if resolved.Vars() != nil {
		t.Errorf("Vars() should be nil, got %v", resolved.Vars())
	}
	if resolved.Lock() != nil {
		t.Errorf("Lock() should be nil")
	}
	if resolved.LockTimeout() != "" {
		t.Errorf("LockTimeout() should be empty")
	}
}

func TestLoadChild_WhenInvalidHCLSyntax_ShouldReturnError(t *testing.T) {
	dir := t.TempDir()
	writeHCL(t, dir, `this is {{{{ not valid HCL`)

	_, err := LoadChild(dir)
	if err == nil {
		t.Error("LoadChild() should return error for invalid HCL syntax")
	}
}

func TestLoadChild_WhenPermissionDenied_ShouldReturnError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping permission test when running as root")
	}

	dir := t.TempDir()
	configPath := filepath.Join(dir, HCLConfigFileName)
	err := os.WriteFile(configPath, []byte(`plugin "risk" { level = "high" }`), 0000)
	if err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	_, err = LoadChild(dir)
	if err == nil {
		t.Error("LoadChild() should return error for unreadable file")
	}
	if errors.Is(err, &ConfigNotFoundError{}) {
		t.Error("should not be ConfigNotFoundError for permission denied")
	}
}

func TestLoadChild_WhenTopLevelVarBlock_ShouldParseVars(t *testing.T) {
	dir := t.TempDir()
	writeHCL(t, dir, `
var "region" { value = "us-west-2" }
var "env"    { value = "staging" }
`)

	cfg, err := LoadChild(dir)
	if err != nil {
		t.Fatalf("LoadChild() error: %v", err)
	}

	if len(cfg.Vars) != 2 {
		t.Fatalf("Vars length = %d, want 2", len(cfg.Vars))
	}
	if cfg.Vars["region"] != "us-west-2" {
		t.Errorf("Vars[region] = %q, want %q", cfg.Vars["region"], "us-west-2")
	}
	if cfg.Vars["env"] != "staging" {
		t.Errorf("Vars[env] = %q, want %q", cfg.Vars["env"], "staging")
	}
}

func TestConvertWorkspaceBlock_WhenLockTimeout_ShouldParseAttribute(t *testing.T) {
	dir := t.TempDir()
	writeHCL(t, dir, `
workspace "prod" {
  lock_timeout = "30s"
}
`)

	cfg, err := LoadChild(dir)
	if err != nil {
		t.Fatalf("LoadChild() error: %v", err)
	}

	if len(cfg.Workspaces) != 1 {
		t.Fatalf("Workspaces length = %d, want 1", len(cfg.Workspaces))
	}
	if cfg.Workspaces[0].LockTimeout != "30s" {
		t.Errorf("LockTimeout = %q, want %q", cfg.Workspaces[0].LockTimeout, "30s")
	}
}

func TestConvertWorkspaceBlock_WhenPluginBlock_ShouldParsePlugins(t *testing.T) {
	dir := t.TempDir()
	writeHCL(t, dir, `
workspace "prod" {
  plugin "risk" {
    enabled = true
    level   = "critical"
  }
  plugin "phantom" {
    enabled = false
  }
}
`)

	cfg, err := LoadChild(dir)
	if err != nil {
		t.Fatalf("LoadChild() error: %v", err)
	}

	ws := cfg.Workspaces[0]
	if len(ws.Plugins) != 2 {
		t.Fatalf("workspace plugins length = %d, want 2", len(ws.Plugins))
	}
	risk := ws.Plugins["risk"]
	if !risk.Enabled {
		t.Error("workspace plugin risk should be enabled")
	}
	if risk.Options["level"] != "critical" {
		t.Errorf("workspace plugin risk level = %v, want %q", risk.Options["level"], "critical")
	}
	phantom := ws.Plugins["phantom"]
	if phantom.Enabled {
		t.Error("workspace plugin phantom should be disabled")
	}
}

func TestConvertWorkspaceBlock_WhenVarFileBlocks_ShouldParseAll(t *testing.T) {
	dir := t.TempDir()
	writeHCL(t, dir, `
workspace "staging" {
  var_file "base.tfvars" {}
  var_file "staging.tfvars" {}
}
`)

	cfg, err := LoadChild(dir)
	if err != nil {
		t.Fatalf("LoadChild() error: %v", err)
	}

	ws := cfg.Workspaces[0]
	if len(ws.VarFiles) != 2 {
		t.Fatalf("workspace VarFiles length = %d, want 2", len(ws.VarFiles))
	}
	if ws.VarFiles[0] != "base.tfvars" {
		t.Errorf("VarFiles[0] = %q, want %q", ws.VarFiles[0], "base.tfvars")
	}
	if ws.VarFiles[1] != "staging.tfvars" {
		t.Errorf("VarFiles[1] = %q, want %q", ws.VarFiles[1], "staging.tfvars")
	}
}

func TestExtractChildPlugin_WhenBoolAndNumberOptions_ShouldParse(t *testing.T) {
	dir := t.TempDir()
	writeHCL(t, dir, `
plugin "risk" {
  enabled   = true
  verbose   = true
  threshold = 0.8
  label     = "high"
}
`)

	cfg, err := LoadChild(dir)
	if err != nil {
		t.Fatalf("LoadChild() error: %v", err)
	}

	risk := cfg.Plugins["risk"]
	if !risk.Enabled {
		t.Error("plugin risk should be enabled")
	}
	if risk.Options["verbose"] != true {
		t.Errorf("plugin risk verbose = %v, want true", risk.Options["verbose"])
	}
	if risk.Options["threshold"] != 0.8 {
		t.Errorf("plugin risk threshold = %v, want 0.8", risk.Options["threshold"])
	}
	if risk.Options["label"] != "high" {
		t.Errorf("plugin risk label = %v, want %q", risk.Options["label"], "high")
	}
}

func TestExtractChildPlugin_WhenEnabledFalse_ShouldSetEnabled(t *testing.T) {
	dir := t.TempDir()
	writeHCL(t, dir, `
plugin "phantom" {
  enabled = false
}
`)

	cfg, err := LoadChild(dir)
	if err != nil {
		t.Fatalf("LoadChild() error: %v", err)
	}

	phantom := cfg.Plugins["phantom"]
	if phantom.Enabled {
		t.Error("plugin phantom should be disabled")
	}
}

func TestExtractVarValue_WhenNoValueAttribute_ShouldReturnEmpty(t *testing.T) {
	dir := t.TempDir()
	writeHCL(t, dir, `
workspace "test" {
  var "empty" {}
}
`)

	cfg, err := LoadChild(dir)
	if err != nil {
		t.Fatalf("LoadChild() error: %v", err)
	}

	ws := cfg.Workspaces[0]
	if ws.Vars["empty"] != "" {
		t.Errorf("var without value attribute should be empty, got %q", ws.Vars["empty"])
	}
}

func TestExtractVarValue_WhenNonStringValue_ShouldReturnEmpty(t *testing.T) {
	dir := t.TempDir()
	writeHCL(t, dir, `
workspace "test" {
  var "num" { value = 42 }
}
`)

	cfg, err := LoadChild(dir)
	if err != nil {
		t.Fatalf("LoadChild() error: %v", err)
	}

	ws := cfg.Workspaces[0]
	if ws.Vars["num"] != "" {
		t.Errorf("non-string var value should be empty, got %q", ws.Vars["num"])
	}
}

func TestMergePlugins_WhenNewPluginAdded_ShouldAddToExisting(t *testing.T) {
	root := &RootConfig{
		Defaults: DefaultsConfig{
			Plugins: map[string]PluginSettings{
				"risk": {Enabled: true, Options: map[string]interface{}{"level": "low"}},
			},
		},
	}

	child := &ChildConfig{
		Plugins: map[string]PluginSettings{
			"phantom": {Enabled: true, Options: map[string]interface{}{"mode": "full"}},
		},
	}

	resolved := Resolve(root, child, "")

	risk := resolved.PluginConfig("risk")
	if risk.Options["level"] != "low" {
		t.Errorf("risk level should be preserved as %q, got %v", "low", risk.Options["level"])
	}
	phantom := resolved.PluginConfig("phantom")
	if phantom.Options["mode"] != "full" {
		t.Errorf("phantom mode = %v, want %q", phantom.Options["mode"], "full")
	}
}

func TestMergePlugins_WhenExistingPluginUpdated_ShouldMergeOptions(t *testing.T) {
	root := &RootConfig{
		Defaults: DefaultsConfig{
			Plugins: map[string]PluginSettings{
				"risk": {Enabled: true, Options: map[string]interface{}{"level": "low", "verbose": true}},
			},
		},
	}

	child := &ChildConfig{
		Plugins: map[string]PluginSettings{
			"risk": {Enabled: false, Options: map[string]interface{}{"level": "high", "extra": "new"}},
		},
	}

	resolved := Resolve(root, child, "")

	risk := resolved.PluginConfig("risk")
	if risk.Enabled {
		t.Error("risk should be disabled (child overrides)")
	}
	if risk.Options["level"] != "high" {
		t.Errorf("risk level = %v, want %q (child overrides)", risk.Options["level"], "high")
	}
	if risk.Options["verbose"] != true {
		t.Errorf("risk verbose = %v, want true (preserved from root)", risk.Options["verbose"])
	}
	if risk.Options["extra"] != "new" {
		t.Errorf("risk extra = %v, want %q (added by child)", risk.Options["extra"], "new")
	}
}

func TestMergePlugins_WhenExistingPluginHasNilOptions_ShouldCreateOptions(t *testing.T) {
	root := &RootConfig{
		Defaults: DefaultsConfig{
			Plugins: map[string]PluginSettings{
				"risk": {Enabled: true, Options: nil},
			},
		},
	}

	child := &ChildConfig{
		Plugins: map[string]PluginSettings{
			"risk": {Enabled: true, Options: map[string]interface{}{"level": "high"}},
		},
	}

	resolved := Resolve(root, child, "")

	risk := resolved.PluginConfig("risk")
	if risk.Options["level"] != "high" {
		t.Errorf("risk level = %v, want %q", risk.Options["level"], "high")
	}
}

func TestCopyOptions_WhenNilInput_ShouldReturnNil(t *testing.T) {
	result := copyOptions(nil)
	if result != nil {
		t.Errorf("copyOptions(nil) = %v, want nil", result)
	}
}

func TestCopyOptions_WhenEmptyMap_ShouldReturnNil(t *testing.T) {
	result := copyOptions(map[string]interface{}{})
	if result != nil {
		t.Errorf("copyOptions(empty) = %v, want nil", result)
	}
}

func TestCopyOptions_WhenPopulated_ShouldDeepCopy(t *testing.T) {
	src := map[string]interface{}{"key": "value", "num": 42.0}
	result := copyOptions(src)

	if result["key"] != "value" {
		t.Errorf("result[key] = %v, want %q", result["key"], "value")
	}
	if result["num"] != 42.0 {
		t.Errorf("result[num] = %v, want 42.0", result["num"])
	}

	result["key"] = "mutated"
	if src["key"] != "value" {
		t.Error("mutating copy should not affect original")
	}
}

func TestDefaultsConfig_PluginConfig_WhenNilPlugins_ShouldReturnEmpty(t *testing.T) {
	d := &DefaultsConfig{Plugins: nil}
	ps := d.PluginConfig("anything")
	if ps.Enabled {
		t.Error("nil plugins map should return zero-value PluginSettings (Enabled=false)")
	}
}

func TestDefaultsConfig_PluginConfig_WhenPluginNotFound_ShouldReturnEmpty(t *testing.T) {
	d := &DefaultsConfig{Plugins: map[string]PluginSettings{
		"risk": {Enabled: true},
	}}
	ps := d.PluginConfig("nonexistent")
	if ps.Enabled {
		t.Error("nonexistent plugin should return zero-value PluginSettings")
	}
}

func TestDefaultsConfig_PluginConfig_WhenPluginFound_ShouldReturnIt(t *testing.T) {
	d := &DefaultsConfig{Plugins: map[string]PluginSettings{
		"risk": {Enabled: true, Options: map[string]interface{}{"level": "high"}},
	}}
	ps := d.PluginConfig("risk")
	if !ps.Enabled {
		t.Error("found plugin should be enabled")
	}
	if ps.Options["level"] != "high" {
		t.Errorf("plugin options level = %v, want %q", ps.Options["level"], "high")
	}
}

func TestChildConfig_PluginConfig_WhenNilPlugins_ShouldReturnEmpty(t *testing.T) {
	c := &ChildConfig{Plugins: nil}
	ps := c.PluginConfig("anything")
	if ps.Enabled {
		t.Error("nil plugins map should return zero-value PluginSettings")
	}
}

func TestChildConfig_PluginConfig_WhenPluginNotFound_ShouldReturnEmpty(t *testing.T) {
	c := &ChildConfig{Plugins: map[string]PluginSettings{
		"risk": {Enabled: true},
	}}
	ps := c.PluginConfig("nonexistent")
	if ps.Enabled {
		t.Error("nonexistent plugin should return zero-value PluginSettings")
	}
}

func TestResolvedConfig_PluginConfig_WhenNilPlugins_ShouldReturnEmpty(t *testing.T) {
	r := &ResolvedConfig{plugins: nil}
	ps := r.PluginConfig("anything")
	if ps.Enabled {
		t.Error("nil plugins map should return zero-value PluginSettings")
	}
}

func TestResolvedConfig_PluginConfig_WhenPluginNotFound_ShouldReturnEmpty(t *testing.T) {
	r := &ResolvedConfig{plugins: map[string]PluginSettings{
		"risk": {Enabled: true},
	}}
	ps := r.PluginConfig("nonexistent")
	if ps.Enabled {
		t.Error("nonexistent plugin should return zero-value PluginSettings")
	}
}

func TestLoadRoot_WhenPluginWithBoolAndNumberOptions_ShouldParse(t *testing.T) {
	dir := t.TempDir()
	writeHCL(t, dir, `
defaults {
  plugin "risk" {
    enabled   = true
    verbose   = true
    threshold = 0.9
    level     = "high"
  }
}
`)

	cfg, err := LoadRoot(dir)
	if err != nil {
		t.Fatalf("LoadRoot() error: %v", err)
	}

	risk := cfg.Defaults.PluginConfig("risk")
	if !risk.Enabled {
		t.Error("risk should be enabled")
	}
	if risk.Options["verbose"] != true {
		t.Errorf("risk verbose = %v, want true", risk.Options["verbose"])
	}
	if risk.Options["threshold"] != 0.9 {
		t.Errorf("risk threshold = %v, want 0.9", risk.Options["threshold"])
	}
	if risk.Options["level"] != "high" {
		t.Errorf("risk level = %v, want %q", risk.Options["level"], "high")
	}
}

func TestLoadRoot_WhenPluginWithNoRemainBody_ShouldHaveNilOptions(t *testing.T) {
	dir := t.TempDir()
	writeHCL(t, dir, `
defaults {
  plugin "risk" {
    enabled = true
  }
}
`)

	cfg, err := LoadRoot(dir)
	if err != nil {
		t.Fatalf("LoadRoot() error: %v", err)
	}

	risk := cfg.Defaults.PluginConfig("risk")
	if !risk.Enabled {
		t.Error("risk should be enabled")
	}
}

func TestResolve_WhenWorkspaceLockTimeout_ShouldApply(t *testing.T) {
	root := &RootConfig{}
	child := &ChildConfig{
		Workspaces: []WorkspaceConfig{
			{
				Name:        "prod",
				LockTimeout: "60s",
			},
		},
	}

	resolved := Resolve(root, child, "prod")
	if resolved.LockTimeout() != "60s" {
		t.Errorf("LockTimeout() = %q, want %q", resolved.LockTimeout(), "60s")
	}
}

func TestResolve_WhenWorkspaceHasEmptyLockTimeout_ShouldNotOverride(t *testing.T) {
	root := &RootConfig{}
	child := &ChildConfig{
		Workspaces: []WorkspaceConfig{
			{
				Name:        "prod",
				LockTimeout: "",
			},
		},
	}

	resolved := Resolve(root, child, "prod")
	if resolved.LockTimeout() != "" {
		t.Errorf("LockTimeout() = %q, want empty", resolved.LockTimeout())
	}
}

func TestMergePlugins_WhenRootHasNoPluginsButChildDoes_ShouldCreatePluginsMap(t *testing.T) {
	root := &RootConfig{
		Defaults: DefaultsConfig{
			Plugins: nil,
		},
	}

	child := &ChildConfig{
		Plugins: map[string]PluginSettings{
			"risk": {Enabled: true, Options: map[string]interface{}{"level": "high"}},
		},
	}

	resolved := Resolve(root, child, "")

	risk := resolved.PluginConfig("risk")
	if !risk.Enabled {
		t.Error("risk should be enabled")
	}
	if risk.Options["level"] != "high" {
		t.Errorf("risk level = %v, want %q", risk.Options["level"], "high")
	}
}

func TestMergePlugins_WhenChildPluginsEmpty_ShouldNotModifyResolved(t *testing.T) {
	root := &RootConfig{
		Defaults: DefaultsConfig{
			Plugins: map[string]PluginSettings{
				"risk": {Enabled: true, Options: map[string]interface{}{"level": "low"}},
			},
		},
	}

	child := &ChildConfig{
		Plugins: nil,
	}

	resolved := Resolve(root, child, "")

	risk := resolved.PluginConfig("risk")
	if risk.Options["level"] != "low" {
		t.Errorf("risk level = %v, want %q (root preserved)", risk.Options["level"], "low")
	}
}

func TestLoadChild_WhenVarValueHasExpressionError_ShouldReturnEmpty(t *testing.T) {
	dir := t.TempDir()
	writeHCL(t, dir, `
var "broken" { value = unknown_reference }
`)

	cfg, err := LoadChild(dir)
	if err != nil {
		t.Fatalf("LoadChild() error: %v", err)
	}

	if cfg.Vars["broken"] != "" {
		t.Errorf("broken var should be empty, got %q", cfg.Vars["broken"])
	}
}

func TestLoadRoot_WhenPluginOptionHasExpressionError_ShouldSkipIt(t *testing.T) {
	dir := t.TempDir()
	writeHCL(t, dir, `
defaults {
  plugin "risk" {
    level = "high"
  }
}
`)

	cfg, err := LoadRoot(dir)
	if err != nil {
		t.Fatalf("LoadRoot() error: %v", err)
	}

	risk := cfg.Defaults.PluginConfig("risk")
	if risk.Options["level"] != "high" {
		t.Errorf("risk level = %v, want %q", risk.Options["level"], "high")
	}
}

func TestExtractChildPlugin_WhenOptionHasExpressionError_ShouldSkipIt(t *testing.T) {
	dir := t.TempDir()
	writeHCL(t, dir, `
plugin "risk" {
  level = unknown_var
  label = "valid"
}
`)

	cfg, err := LoadChild(dir)
	if err != nil {
		t.Fatalf("LoadChild() error: %v", err)
	}

	risk := cfg.Plugins["risk"]
	if risk.Options["label"] != "valid" {
		t.Errorf("risk label = %v, want %q", risk.Options["label"], "valid")
	}
	if _, exists := risk.Options["level"]; exists {
		t.Error("risk level should be skipped due to expression error")
	}
}

func TestExtractChildPlugin_WhenBodyHasNestedBlock_ShouldReturnDefaultEnabled(t *testing.T) {
	dir := t.TempDir()
	writeHCL(t, dir, `
plugin "risk" {
  nested_block {
    key = "value"
  }
}
`)

	cfg, err := LoadChild(dir)
	if err != nil {
		t.Fatalf("LoadChild() error: %v", err)
	}

	risk := cfg.Plugins["risk"]
	if !risk.Enabled {
		t.Error("plugin with nested block should default to enabled=true")
	}
}

func TestExtractVarValue_WhenBodyHasExtraAttributes_ShouldReturnEmpty(t *testing.T) {
	dir := t.TempDir()
	writeHCL(t, dir, `
var "test" {
  value = "hello"
  extra = "unexpected"
}
`)

	cfg, err := LoadChild(dir)
	if err != nil {
		t.Fatalf("LoadChild() error: %v", err)
	}

	if cfg.Vars["test"] != "" {
		t.Errorf("var with extra attributes should return empty due to Content error, got %q", cfg.Vars["test"])
	}
}

func TestConvertWorkspaceBlock_WhenUnexpectedAttributes_ShouldReturnPartial(t *testing.T) {
	dir := t.TempDir()
	writeHCL(t, dir, `
workspace "prod" {
  lock_timeout = "30s"
  unexpected_attr = "bad"
}
`)

	cfg, err := LoadChild(dir)
	if err != nil {
		t.Fatalf("LoadChild() error: %v", err)
	}

	ws := cfg.Workspaces[0]
	if ws.Name != "prod" {
		t.Errorf("workspace name = %q, want %q", ws.Name, "prod")
	}
	if ws.LockTimeout != "" {
		t.Errorf("workspace with content error should have empty LockTimeout, got %q", ws.LockTimeout)
	}
}

func TestLoadChild_WhenUnknownBlockType_ShouldReturnSchemaError(t *testing.T) {
	dir := t.TempDir()
	writeHCL(t, dir, `
unknown_block_type {
  key = "value"
}
`)

	_, err := LoadChild(dir)
	if err == nil {
		t.Error("LoadChild() should return error for unknown block type not in child schema")
	}
}

func TestLoadRoot_WhenPluginBlockWithOnlyEnabled_ShouldHaveNilOptions(t *testing.T) {
	dir := t.TempDir()
	writeHCL(t, dir, `
defaults {
  plugin "minimal" {
  }
}
`)

	cfg, err := LoadRoot(dir)
	if err != nil {
		t.Fatalf("LoadRoot() error: %v", err)
	}

	minimal := cfg.Defaults.PluginConfig("minimal")
	if !minimal.Enabled {
		t.Error("plugin with empty body should default to enabled=true")
	}
	if minimal.Options != nil {
		t.Errorf("plugin with no remain attributes should have nil options, got %v", minimal.Options)
	}
}

func TestLoadRoot_WhenPluginOptionExpressionFails_ShouldSkipOption(t *testing.T) {
	dir := t.TempDir()
	writeHCL(t, dir, `
defaults {
  plugin "risk" {
    level   = "high"
    broken  = undefined_var
  }
}
`)

	cfg, err := LoadRoot(dir)
	if err != nil {
		t.Fatalf("LoadRoot() error: %v", err)
	}

	risk := cfg.Defaults.PluginConfig("risk")
	if risk.Options["level"] != "high" {
		t.Errorf("risk level = %v, want %q", risk.Options["level"], "high")
	}
	if _, exists := risk.Options["broken"]; exists {
		t.Error("broken option should be skipped due to expression error")
	}
}

func TestLoadRoot_WhenPluginHasNestedBlock_ShouldReturnNilOptions(t *testing.T) {
	dir := t.TempDir()
	writeHCL(t, dir, `
defaults {
  plugin "risk" {
    nested {
      key = "value"
    }
  }
}
`)

	cfg, err := LoadRoot(dir)
	if err != nil {
		t.Fatalf("LoadRoot() error: %v", err)
	}

	risk := cfg.Defaults.PluginConfig("risk")
	if risk.Options != nil {
		t.Errorf("plugin with nested block should have nil options (JustAttributes fails), got %v", risk.Options)
	}
}

func TestExtractPluginOptions_WhenNilBody_ShouldReturnNil(t *testing.T) {
	result := extractPluginOptions(nil)
	if result != nil {
		t.Errorf("extractPluginOptions(nil) = %v, want nil", result)
	}
}

func TestExtractPluginOptions_WhenUnsupportedType_ShouldSkipOption(t *testing.T) {
	dir := t.TempDir()
	writeHCL(t, dir, `
defaults {
  plugin "risk" {
    tags  = ["a", "b"]
    level = "high"
  }
}
`)

	cfg, err := LoadRoot(dir)
	if err != nil {
		t.Fatalf("LoadRoot() error: %v", err)
	}

	risk := cfg.Defaults.PluginConfig("risk")
	if risk.Options["level"] != "high" {
		t.Errorf("risk level = %v, want %q", risk.Options["level"], "high")
	}
	if _, exists := risk.Options["tags"]; exists {
		t.Error("list option should be skipped (unsupported type)")
	}
}

func TestExtractPluginOptions_WhenRemainBodyHasBlocks_ShouldReturnNil(t *testing.T) {
	dir := t.TempDir()
	writeHCL(t, dir, `
defaults {
  plugin "risk" {
    enabled = true
    nested "label" {
      key = "value"
    }
  }
}
`)

	cfg, err := LoadRoot(dir)
	if err != nil {
		t.Fatalf("LoadRoot() error: %v", err)
	}

	risk := cfg.Defaults.PluginConfig("risk")
	if risk.Options != nil {
		t.Errorf("plugin with nested labeled block should have nil options, got %v", risk.Options)
	}
}
