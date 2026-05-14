package sdk

import "testing"

func TestBuildPlanOptions_WhenNilResolved_ShouldReturnTargetsOnly(t *testing.T) {
	opts := BuildPlanOptions(nil, []string{"aws_instance.web"})
	if len(opts.Targets) != 1 || opts.Targets[0] != "aws_instance.web" {
		t.Errorf("Targets = %v, want [aws_instance.web]", opts.Targets)
	}
	if len(opts.VarFiles) != 0 {
		t.Errorf("VarFiles = %v, want empty", opts.VarFiles)
	}
}

func TestBuildPlanOptions_WhenResolvedHasVarFiles_ShouldInclude(t *testing.T) {
	resolved := &ResolvedOptions{
		VarFiles:  []string{"prod.tfvars", "common.tfvars"},
		Vars:      map[string]string{"env": "prod"},
		ExtraArgs: []string{"-no-color"},
	}

	opts := BuildPlanOptions(resolved, []string{"aws_instance.web"})

	if len(opts.VarFiles) != 2 {
		t.Fatalf("VarFiles length = %d, want 2", len(opts.VarFiles))
	}
	if opts.VarFiles[0] != "prod.tfvars" {
		t.Errorf("VarFiles[0] = %q, want prod.tfvars", opts.VarFiles[0])
	}
	if opts.Vars["env"] != "prod" {
		t.Errorf("Vars[env] = %q, want prod", opts.Vars["env"])
	}
	if len(opts.ExtraArgs) != 1 || opts.ExtraArgs[0] != "-no-color" {
		t.Errorf("ExtraArgs = %v, want [-no-color]", opts.ExtraArgs)
	}
}

func TestBuildApplyOptions_WhenResolvedHasVarFiles_ShouldInclude(t *testing.T) {
	resolved := &ResolvedOptions{
		VarFiles: []string{"staging.tfvars"},
		Vars:     map[string]string{"region": "us-west-2"},
	}

	opts := BuildApplyOptions(resolved, nil)

	if len(opts.VarFiles) != 1 || opts.VarFiles[0] != "staging.tfvars" {
		t.Errorf("VarFiles = %v, want [staging.tfvars]", opts.VarFiles)
	}
	if opts.Vars["region"] != "us-west-2" {
		t.Errorf("Vars[region] = %q, want us-west-2", opts.Vars["region"])
	}
}

func TestBuildPlanOptions_WhenEmptyResolved_ShouldReturnEmpty(t *testing.T) {
	resolved := &ResolvedOptions{}
	opts := BuildPlanOptions(resolved, nil)

	if len(opts.VarFiles) != 0 {
		t.Errorf("VarFiles = %v, want empty", opts.VarFiles)
	}
	if len(opts.Vars) != 0 {
		t.Errorf("Vars = %v, want empty", opts.Vars)
	}
	if len(opts.ExtraArgs) != 0 {
		t.Errorf("ExtraArgs = %v, want empty", opts.ExtraArgs)
	}
}

func TestBuildApplyOptions_WhenNilResolved_ShouldReturnTargetsOnly(t *testing.T) {
	opts := BuildApplyOptions(nil, []string{"aws_instance.web"})
	if len(opts.Targets) != 1 || opts.Targets[0] != "aws_instance.web" {
		t.Errorf("Targets = %v, want [aws_instance.web]", opts.Targets)
	}
	if len(opts.VarFiles) != 0 {
		t.Errorf("VarFiles = %v, want empty", opts.VarFiles)
	}
}

func TestBuildApplyOptions_WhenEmptyResolved_ShouldReturnEmpty(t *testing.T) {
	resolved := &ResolvedOptions{}
	opts := BuildApplyOptions(resolved, nil)

	if len(opts.VarFiles) != 0 {
		t.Errorf("VarFiles = %v, want empty", opts.VarFiles)
	}
	if len(opts.Vars) != 0 {
		t.Errorf("Vars = %v, want empty", opts.Vars)
	}
	if len(opts.ExtraArgs) != 0 {
		t.Errorf("ExtraArgs = %v, want empty", opts.ExtraArgs)
	}
}

func TestBuildApplyOptions_WhenResolvedHasExtraArgs_ShouldInclude(t *testing.T) {
	resolved := &ResolvedOptions{
		VarFiles:  []string{"prod.tfvars"},
		Vars:      map[string]string{"env": "prod"},
		ExtraArgs: []string{"-no-color", "-compact-warnings"},
	}

	opts := BuildApplyOptions(resolved, []string{"aws_instance.web"})

	if len(opts.ExtraArgs) != 2 {
		t.Fatalf("ExtraArgs length = %d, want 2", len(opts.ExtraArgs))
	}
	if opts.ExtraArgs[0] != "-no-color" {
		t.Errorf("ExtraArgs[0] = %q, want %q", opts.ExtraArgs[0], "-no-color")
	}
}
