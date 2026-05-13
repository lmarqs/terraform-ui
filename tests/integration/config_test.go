//go:build integration

package integration

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/lmarqs/terraform-ui/internal/config"
)

func configFixtureDir(name string) string {
	return filepath.Join(findProjectRoot(), "tests", "fixtures", "config", name)
}

func TestConfig_NoConfigDir_ShouldReturnNotFoundError(t *testing.T) {
	_, err := config.LoadRoot(configFixtureDir("no-config"))
	if err == nil {
		t.Fatal("expected error for directory with no tfui.hcl")
	}
	var notFound *config.ConfigNotFoundError
	if !errors.As(err, &notFound) {
		t.Errorf("expected ConfigNotFoundError, got %T: %v", err, err)
	}
}

func TestConfig_EmptyConfig_ShouldReturnEmptyRootConfig(t *testing.T) {
	cfg, err := config.LoadRoot(configFixtureDir("empty-config"))
	if err != nil {
		t.Fatalf("LoadRoot error: %v", err)
	}
	if cfg.Terraform.Bin != "" {
		t.Errorf("Terraform.Bin = %q, want empty", cfg.Terraform.Bin)
	}
	if len(cfg.Chdir.Members) != 0 {
		t.Errorf("Chdir.Members = %v, want empty", cfg.Chdir.Members)
	}
}

func TestConfig_SingleModule_ShouldParseBinary(t *testing.T) {
	cfg, err := config.LoadRoot(configFixtureDir("single-module"))
	if err != nil {
		t.Fatalf("LoadRoot error: %v", err)
	}
	if cfg.Terraform.Bin != "terraform" {
		t.Errorf("Terraform.Bin = %q, want %q", cfg.Terraform.Bin, "terraform")
	}
	if len(cfg.Chdir.Members) != 0 {
		t.Errorf("Chdir.Members = %v, want empty", cfg.Chdir.Members)
	}
}

func TestConfig_Monorepo_ShouldParseFullConfig(t *testing.T) {
	dir := configFixtureDir("monorepo")
	cfg, err := config.LoadRoot(dir)
	if err != nil {
		t.Fatalf("LoadRoot error: %v", err)
	}

	if cfg.Terraform.Bin != "terraform" {
		t.Errorf("Terraform.Bin = %q, want %q", cfg.Terraform.Bin, "terraform")
	}
	if len(cfg.Chdir.Members) != 2 {
		t.Fatalf("Chdir.Members length = %d, want 2", len(cfg.Chdir.Members))
	}
	if cfg.Chdir.Members[0] != "modules/vpc" {
		t.Errorf("Chdir.Members[0] = %q, want %q", cfg.Chdir.Members[0], "modules/vpc")
	}
	if cfg.Cache.StalenessThreshold != "5m" {
		t.Errorf("Cache.StalenessThreshold = %q, want %q", cfg.Cache.StalenessThreshold, "5m")
	}
	if cfg.Defaults.Parallelism != 10 {
		t.Errorf("Defaults.Parallelism = %d, want 10", cfg.Defaults.Parallelism)
	}
}

func TestConfig_Monorepo_ChildConfig_ShouldLoad(t *testing.T) {
	dir := filepath.Join(configFixtureDir("monorepo"), "modules", "vpc")
	child, err := config.LoadChild(dir)
	if err != nil {
		t.Fatalf("LoadChild error: %v", err)
	}

	if len(child.VarFiles) != 1 || child.VarFiles[0] != "base.tfvars" {
		t.Errorf("VarFiles = %v, want [base.tfvars]", child.VarFiles)
	}
	if len(child.Workspaces) != 3 {
		t.Errorf("Workspaces count = %d, want 3", len(child.Workspaces))
	}
}

func TestConfig_Monorepo_ResolveProduction_ShouldConcatenateVarFiles(t *testing.T) {
	dir := configFixtureDir("monorepo")
	root, err := config.LoadRoot(dir)
	if err != nil {
		t.Fatalf("LoadRoot error: %v", err)
	}

	childDir := filepath.Join(dir, "modules", "vpc")
	child, err := config.LoadChild(childDir)
	if err != nil {
		t.Fatalf("LoadChild error: %v", err)
	}

	resolved := config.Resolve(root, child, "production")

	varFiles := resolved.VarFiles()
	expected := []string{"common/tags.tfvars", "base.tfvars", "prod.tfvars"}
	if len(varFiles) != len(expected) {
		t.Fatalf("VarFiles() = %v, want %v", varFiles, expected)
	}
	for i, want := range expected {
		if varFiles[i] != want {
			t.Errorf("VarFiles()[%d] = %q, want %q", i, varFiles[i], want)
		}
	}

	vars := resolved.Vars()
	if vars["environment"] != "prod" {
		t.Errorf("Vars()[environment] = %q, want %q", vars["environment"], "prod")
	}
}

func TestConfig_Monorepo_ResolveDevGlob_ShouldMatchWorkspace(t *testing.T) {
	dir := configFixtureDir("monorepo")
	root, err := config.LoadRoot(dir)
	if err != nil {
		t.Fatalf("LoadRoot error: %v", err)
	}

	childDir := filepath.Join(dir, "modules", "vpc")
	child, err := config.LoadChild(childDir)
	if err != nil {
		t.Fatalf("LoadChild error: %v", err)
	}

	resolved := config.Resolve(root, child, "dev-us-east-1")

	pc := resolved.PluginConfig("risk")
	if pc.Options["level"] != "low" {
		t.Errorf("risk level = %v, want %q (from dev-* workspace)", pc.Options["level"], "low")
	}
}

func TestConfig_LockedFieldViolation_ShouldRejectChildConfig(t *testing.T) {
	dir := filepath.Join(configFixtureDir("locked-violation"), "modules", "bad")
	_, err := config.LoadChild(dir)
	if err == nil {
		t.Fatal("expected error for child with terraform block")
	}
}

func TestConfig_Monorepo_NoChildConfig_ShouldReturnNotFound(t *testing.T) {
	dir := filepath.Join(configFixtureDir("monorepo"), "modules", "ecs")
	_, err := config.LoadChild(dir)
	if err == nil {
		t.Fatal("expected ConfigNotFoundError for ecs (no child tfui.hcl)")
	}
	var notFound *config.ConfigNotFoundError
	if !errors.As(err, &notFound) {
		t.Errorf("expected ConfigNotFoundError, got %T: %v", err, err)
	}
}

func TestConfig_Monorepo_ResolveStagingWorkspace_ShouldSwitchVarFiles(t *testing.T) {
	dir := configFixtureDir("monorepo")
	root, _ := config.LoadRoot(dir)
	child, _ := config.LoadChild(filepath.Join(dir, "modules", "vpc"))

	prodResolved := config.Resolve(root, child, "production")
	stagingResolved := config.Resolve(root, child, "staging")

	prodFiles := prodResolved.VarFiles()
	stagingFiles := stagingResolved.VarFiles()

	if prodFiles[2] != "prod.tfvars" {
		t.Errorf("production VarFiles[2] = %q, want prod.tfvars", prodFiles[2])
	}
	if stagingFiles[2] != "staging.tfvars" {
		t.Errorf("staging VarFiles[2] = %q, want staging.tfvars", stagingFiles[2])
	}

	if prodResolved.Vars()["environment"] != "prod" {
		t.Errorf("production env = %q", prodResolved.Vars()["environment"])
	}
	if stagingResolved.Vars()["environment"] != "staging" {
		t.Errorf("staging env = %q", stagingResolved.Vars()["environment"])
	}
}
