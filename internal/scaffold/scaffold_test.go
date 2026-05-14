package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectBinary_WhenCalled_ShouldReturnNonEmptyString(t *testing.T) {
	result := DetectBinary()
	if result == "" {
		t.Error("DetectBinary() returned empty string, want non-empty")
	}
}

func TestDetectBinary_WhenCalled_ShouldReturnValidBinary(t *testing.T) {
	result := DetectBinary()
	valid := map[string]bool{"terraform": true, "tofu": true, "terragrunt": true}
	if !valid[result] {
		t.Errorf("DetectBinary() = %q, want one of terraform/tofu/terragrunt", result)
	}
}

func TestDetectBinary_WhenNoBinaryOnPath_ShouldReturnTerraformDefault(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	result := DetectBinary()
	if result != "terraform" {
		t.Errorf("DetectBinary() = %q, want %q when no binary found", result, "terraform")
	}
}

func TestDetectBinary_WhenOnlyTofuOnPath_ShouldReturnTofu(t *testing.T) {
	binDir := t.TempDir()
	tofuPath := filepath.Join(binDir, "tofu")
	os.WriteFile(tofuPath, []byte("#!/bin/sh\n"), 0755)
	t.Setenv("PATH", binDir)

	result := DetectBinary()
	if result != "tofu" {
		t.Errorf("DetectBinary() = %q, want %q", result, "tofu")
	}
}

func TestDetectBinary_WhenOnlyTerragruntOnPath_ShouldReturnTerragrunt(t *testing.T) {
	binDir := t.TempDir()
	tgPath := filepath.Join(binDir, "terragrunt")
	os.WriteFile(tgPath, []byte("#!/bin/sh\n"), 0755)
	t.Setenv("PATH", binDir)

	result := DetectBinary()
	if result != "terragrunt" {
		t.Errorf("DetectBinary() = %q, want %q", result, "terragrunt")
	}
}

func TestDetectBinary_WhenTerraformAndTofuOnPath_ShouldPreferTerraform(t *testing.T) {
	binDir := t.TempDir()
	os.WriteFile(filepath.Join(binDir, "terraform"), []byte("#!/bin/sh\n"), 0755)
	os.WriteFile(filepath.Join(binDir, "tofu"), []byte("#!/bin/sh\n"), 0755)
	t.Setenv("PATH", binDir)

	result := DetectBinary()
	if result != "terraform" {
		t.Errorf("DetectBinary() = %q, want %q (terraform preferred over tofu)", result, "terraform")
	}
}

func TestBuildHCL_WhenNoMembers_ShouldReturnOnlyTerraformBlock(t *testing.T) {
	result := BuildHCL("terraform", nil)
	expected := "terraform {\n  bin = \"terraform\"\n}\n"
	if result != expected {
		t.Errorf("BuildHCL(terraform, nil) =\n%q\nwant\n%q", result, expected)
	}
}

func TestBuildHCL_WhenEmptyMembers_ShouldReturnOnlyTerraformBlock(t *testing.T) {
	result := BuildHCL("tofu", []string{})
	expected := "terraform {\n  bin = \"tofu\"\n}\n"
	if result != expected {
		t.Errorf("BuildHCL(tofu, []) =\n%q\nwant\n%q", result, expected)
	}
}

func TestBuildHCL_WhenSingleMember_ShouldIncludeMemberBlock(t *testing.T) {
	result := BuildHCL("terraform", []string{"modules/vpc"})
	if !strings.Contains(result, "member \"modules/vpc\" {}") {
		t.Errorf("BuildHCL should contain member block, got:\n%s", result)
	}
}

func TestBuildHCL_WhenMultipleMembers_ShouldSortAlphabetically(t *testing.T) {
	members := []string{"modules/ecs", "modules/vpc", "envs/dev", "."}
	result := BuildHCL("terraform", members)

	lines := strings.Split(result, "\n")
	var memberLines []string
	for _, line := range lines {
		if strings.HasPrefix(line, "member ") {
			memberLines = append(memberLines, line)
		}
	}

	if len(memberLines) != 4 {
		t.Fatalf("expected 4 member lines, got %d", len(memberLines))
	}

	expected := []string{
		`member "." {}`,
		`member "envs/dev" {}`,
		`member "modules/ecs" {}`,
		`member "modules/vpc" {}`,
	}
	for i, want := range expected {
		if memberLines[i] != want {
			t.Errorf("member line %d = %q, want %q", i, memberLines[i], want)
		}
	}
}

func TestBuildHCL_WhenCustomBinary_ShouldUseBinaryName(t *testing.T) {
	result := BuildHCL("terragrunt", []string{"."})
	if !strings.Contains(result, `bin = "terragrunt"`) {
		t.Errorf("BuildHCL should contain bin = \"terragrunt\", got:\n%s", result)
	}
}

func TestBuildHCL_WhenMembersUnsorted_ShouldSortOutput(t *testing.T) {
	members := []string{"z-module", "a-module", "m-module"}
	result := BuildHCL("terraform", members)

	zIdx := strings.Index(result, "z-module")
	aIdx := strings.Index(result, "a-module")
	mIdx := strings.Index(result, "m-module")

	if aIdx > mIdx || mIdx > zIdx {
		t.Errorf("members not sorted: a=%d, m=%d, z=%d", aIdx, mIdx, zIdx)
	}
}

func TestDetectMembers_WhenEmptyDir_ShouldReturnNil(t *testing.T) {
	dir := t.TempDir()
	members := DetectMembers(dir)
	if members != nil {
		t.Errorf("DetectMembers(empty) = %v, want nil", members)
	}
}

func TestDetectMembers_WhenAbsPathFails_ShouldReturnNil(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "vanish")
	os.MkdirAll(subDir, 0755)
	t.Chdir(subDir)
	os.RemoveAll(subDir)

	members := DetectMembers(".")
	if members != nil {
		t.Errorf("DetectMembers(deleted cwd) = %v, want nil", members)
	}
}

func TestDetectMembers_WhenDirNameCausesGlobError_ShouldSkipPattern(t *testing.T) {
	tmpDir := t.TempDir()
	// A directory name containing "[" causes filepath.Glob to return ErrBadPattern
	badDir := filepath.Join(tmpDir, "bad[dir")
	os.MkdirAll(badDir, 0755)
	// Put a .tf file in the root so we can verify the function still works past the error
	os.WriteFile(filepath.Join(badDir, "main.tf"), []byte(""), 0644)

	members := DetectMembers(badDir)
	// Despite glob errors on patterns, the root .tf check still works
	if len(members) != 1 {
		t.Fatalf("len(members) = %d, want 1", len(members))
	}
	if members[0].Path != "." {
		t.Errorf("members[0].Path = %q, want %q", members[0].Path, ".")
	}
}

func TestDetectMembers_WhenRootHasTfFiles_ShouldIncludeDot(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "main.tf"), []byte(""), 0644)

	members := DetectMembers(dir)
	if len(members) != 1 {
		t.Fatalf("len(members) = %d, want 1", len(members))
	}
	if members[0].Path != "." {
		t.Errorf("members[0].Path = %q, want %q", members[0].Path, ".")
	}
	if !members[0].Enabled {
		t.Error("members[0].Enabled = false, want true")
	}
}

func TestDetectMembers_WhenModulesExist_ShouldDetectThem(t *testing.T) {
	dir := t.TempDir()
	vpcDir := filepath.Join(dir, "modules", "vpc")
	os.MkdirAll(vpcDir, 0755)
	os.WriteFile(filepath.Join(vpcDir, "main.tf"), []byte(""), 0644)

	ecsDir := filepath.Join(dir, "modules", "ecs")
	os.MkdirAll(ecsDir, 0755)
	os.WriteFile(filepath.Join(ecsDir, "main.tf"), []byte(""), 0644)

	members := DetectMembers(dir)
	if len(members) != 2 {
		t.Fatalf("len(members) = %d, want 2", len(members))
	}

	paths := map[string]bool{}
	for _, m := range members {
		paths[m.Path] = true
	}
	if !paths["modules/vpc"] {
		t.Error("expected modules/vpc in members")
	}
	if !paths["modules/ecs"] {
		t.Error("expected modules/ecs in members")
	}
}

func TestDetectMembers_WhenEnvsExist_ShouldDetectThem(t *testing.T) {
	dir := t.TempDir()
	devDir := filepath.Join(dir, "envs", "dev")
	os.MkdirAll(devDir, 0755)
	os.WriteFile(filepath.Join(devDir, "main.tf"), []byte(""), 0644)

	members := DetectMembers(dir)
	if len(members) != 1 {
		t.Fatalf("len(members) = %d, want 1", len(members))
	}
	if members[0].Path != "envs/dev" {
		t.Errorf("members[0].Path = %q, want %q", members[0].Path, "envs/dev")
	}
}

func TestDetectMembers_WhenInfraExist_ShouldDetectThem(t *testing.T) {
	dir := t.TempDir()
	infraDir := filepath.Join(dir, "infra", "network")
	os.MkdirAll(infraDir, 0755)
	os.WriteFile(filepath.Join(infraDir, "vpc.tf"), []byte(""), 0644)

	members := DetectMembers(dir)
	if len(members) != 1 {
		t.Fatalf("len(members) = %d, want 1", len(members))
	}
	if members[0].Path != "infra/network" {
		t.Errorf("members[0].Path = %q, want %q", members[0].Path, "infra/network")
	}
}

func TestDetectMembers_WhenServicesExist_ShouldDetectThem(t *testing.T) {
	dir := t.TempDir()
	svcDir := filepath.Join(dir, "services", "api", "terraform")
	os.MkdirAll(svcDir, 0755)
	os.WriteFile(filepath.Join(svcDir, "main.tf"), []byte(""), 0644)

	members := DetectMembers(dir)
	if len(members) != 1 {
		t.Fatalf("len(members) = %d, want 1", len(members))
	}
	if members[0].Path != filepath.Join("services", "api", "terraform") {
		t.Errorf("members[0].Path = %q, want %q", members[0].Path, "services/api/terraform")
	}
}

func TestDetectMembers_WhenDirWithoutTfFiles_ShouldSkipIt(t *testing.T) {
	dir := t.TempDir()
	emptyModule := filepath.Join(dir, "modules", "empty")
	os.MkdirAll(emptyModule, 0755)
	os.WriteFile(filepath.Join(emptyModule, "readme.md"), []byte(""), 0644)

	members := DetectMembers(dir)
	if members != nil {
		t.Errorf("DetectMembers should skip dirs without .tf files, got %v", members)
	}
}

func TestDetectMembers_WhenMixedDirs_ShouldOnlyIncludeTfDirs(t *testing.T) {
	dir := t.TempDir()

	vpcDir := filepath.Join(dir, "modules", "vpc")
	os.MkdirAll(vpcDir, 0755)
	os.WriteFile(filepath.Join(vpcDir, "main.tf"), []byte(""), 0644)

	emptyDir := filepath.Join(dir, "modules", "empty")
	os.MkdirAll(emptyDir, 0755)
	os.WriteFile(filepath.Join(emptyDir, "readme.md"), []byte(""), 0644)

	members := DetectMembers(dir)
	if len(members) != 1 {
		t.Fatalf("len(members) = %d, want 1", len(members))
	}
	if members[0].Path != "modules/vpc" {
		t.Errorf("members[0].Path = %q, want %q", members[0].Path, "modules/vpc")
	}
}

func TestDetectMembers_WhenMultiplePatterns_ShouldDetectAll(t *testing.T) {
	dir := t.TempDir()

	// modules/vpc
	vpcDir := filepath.Join(dir, "modules", "vpc")
	os.MkdirAll(vpcDir, 0755)
	os.WriteFile(filepath.Join(vpcDir, "main.tf"), []byte(""), 0644)

	// envs/prod
	prodDir := filepath.Join(dir, "envs", "prod")
	os.MkdirAll(prodDir, 0755)
	os.WriteFile(filepath.Join(prodDir, "main.tf"), []byte(""), 0644)

	// infra/base
	baseDir := filepath.Join(dir, "infra", "base")
	os.MkdirAll(baseDir, 0755)
	os.WriteFile(filepath.Join(baseDir, "main.tf"), []byte(""), 0644)

	// services/web/terraform
	webDir := filepath.Join(dir, "services", "web", "terraform")
	os.MkdirAll(webDir, 0755)
	os.WriteFile(filepath.Join(webDir, "main.tf"), []byte(""), 0644)

	// root
	os.WriteFile(filepath.Join(dir, "root.tf"), []byte(""), 0644)

	members := DetectMembers(dir)
	if len(members) != 5 {
		t.Fatalf("len(members) = %d, want 5", len(members))
	}

	paths := map[string]bool{}
	for _, m := range members {
		paths[m.Path] = true
		if !m.Enabled {
			t.Errorf("member %q Enabled = false, want true", m.Path)
		}
	}
	if !paths["modules/vpc"] {
		t.Error("missing modules/vpc")
	}
	if !paths["envs/prod"] {
		t.Error("missing envs/prod")
	}
	if !paths["infra/base"] {
		t.Error("missing infra/base")
	}
	if !paths[filepath.Join("services", "web", "terraform")] {
		t.Error("missing services/web/terraform")
	}
	if !paths["."] {
		t.Error("missing root (.)")
	}
}

func TestDetectMembers_WhenTofuFiles_ShouldDetectThem(t *testing.T) {
	dir := t.TempDir()
	modDir := filepath.Join(dir, "modules", "network")
	os.MkdirAll(modDir, 0755)
	os.WriteFile(filepath.Join(modDir, "main.tofu"), []byte(""), 0644)

	members := DetectMembers(dir)
	if len(members) != 1 {
		t.Fatalf("len(members) = %d, want 1", len(members))
	}
	if members[0].Path != "modules/network" {
		t.Errorf("members[0].Path = %q, want %q", members[0].Path, "modules/network")
	}
}

func TestDetectMembers_WhenRootOnlyHasNonTfFiles_ShouldNotIncludeDot(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "readme.md"), []byte(""), 0644)
	os.WriteFile(filepath.Join(dir, "main.go"), []byte(""), 0644)

	members := DetectMembers(dir)
	if members != nil {
		t.Errorf("DetectMembers should not include root without .tf files, got %v", members)
	}
}

func TestDetectMembers_WhenNestedDirsNotMatchingPattern_ShouldSkip(t *testing.T) {
	dir := t.TempDir()
	// A terraform dir that doesn't match any known pattern
	otherDir := filepath.Join(dir, "other", "stuff")
	os.MkdirAll(otherDir, 0755)
	os.WriteFile(filepath.Join(otherDir, "main.tf"), []byte(""), 0644)

	members := DetectMembers(dir)
	if members != nil {
		t.Errorf("DetectMembers should skip non-matching patterns, got %v", members)
	}
}

func TestDetectMembers_WhenAllEnabled_ShouldHaveEnabledTrue(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "main.tf"), []byte(""), 0644)

	members := DetectMembers(dir)
	for _, m := range members {
		if !m.Enabled {
			t.Errorf("member %q Enabled = false, want true", m.Path)
		}
	}
}

func TestGenerateConfig_WhenRootHasTfFiles_ShouldReturnValidHCL(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "main.tf"), []byte(""), 0644)

	result, err := GenerateConfig(dir)
	if err != nil {
		t.Fatalf("GenerateConfig() error = %v, want nil", err)
	}
	if result == "" {
		t.Error("GenerateConfig() returned empty string")
	}
	if !strings.Contains(result, "terraform {") {
		t.Error("result should contain terraform block")
	}
	if !strings.Contains(result, `member "." {}`) {
		t.Error("result should contain root member block")
	}
}

func TestGenerateConfig_WhenNoTfFiles_ShouldReturnOnlyTerraformBlock(t *testing.T) {
	dir := t.TempDir()

	result, err := GenerateConfig(dir)
	if err != nil {
		t.Fatalf("GenerateConfig() error = %v, want nil", err)
	}
	if !strings.Contains(result, "terraform {") {
		t.Error("result should contain terraform block")
	}
	if strings.Contains(result, "member") {
		t.Error("result should not contain member blocks when no tf files found")
	}
}

func TestGenerateConfig_WhenMultipleMembers_ShouldSortThem(t *testing.T) {
	dir := t.TempDir()

	// Create modules in reverse alphabetical order
	for _, name := range []string{"z-mod", "a-mod", "m-mod"} {
		modDir := filepath.Join(dir, "modules", name)
		os.MkdirAll(modDir, 0755)
		os.WriteFile(filepath.Join(modDir, "main.tf"), []byte(""), 0644)
	}

	result, err := GenerateConfig(dir)
	if err != nil {
		t.Fatalf("GenerateConfig() error = %v, want nil", err)
	}

	aIdx := strings.Index(result, "a-mod")
	mIdx := strings.Index(result, "m-mod")
	zIdx := strings.Index(result, "z-mod")

	if aIdx > mIdx || mIdx > zIdx {
		t.Errorf("members not sorted alphabetically in output: a=%d, m=%d, z=%d", aIdx, mIdx, zIdx)
	}
}
