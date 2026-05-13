package sdk

import "testing"

func TestBuildPlanOptions_WhenNilSession_ShouldReturnTargetsOnly(t *testing.T) {
	opts := BuildPlanOptions(nil, []string{"aws_instance.web"})
	if len(opts.Targets) != 1 || opts.Targets[0] != "aws_instance.web" {
		t.Errorf("Targets = %v, want [aws_instance.web]", opts.Targets)
	}
	if len(opts.VarFiles) != 0 {
		t.Errorf("VarFiles = %v, want empty", opts.VarFiles)
	}
}

func TestBuildPlanOptions_WhenSessionHasVarFiles_ShouldInclude(t *testing.T) {
	session := NewSession()
	session.Set(SessionKeyVarFiles, []string{"prod.tfvars", "common.tfvars"})
	session.Set(SessionKeyVars, map[string]string{"env": "prod"})
	session.Set(SessionKeyExtraArgs, []string{"-no-color"})

	opts := BuildPlanOptions(session, []string{"aws_instance.web"})

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

func TestBuildApplyOptions_WhenSessionHasVarFiles_ShouldInclude(t *testing.T) {
	session := NewSession()
	session.Set(SessionKeyVarFiles, []string{"staging.tfvars"})
	session.Set(SessionKeyVars, map[string]string{"region": "us-west-2"})

	opts := BuildApplyOptions(session, nil)

	if len(opts.VarFiles) != 1 || opts.VarFiles[0] != "staging.tfvars" {
		t.Errorf("VarFiles = %v, want [staging.tfvars]", opts.VarFiles)
	}
	if opts.Vars["region"] != "us-west-2" {
		t.Errorf("Vars[region] = %q, want us-west-2", opts.Vars["region"])
	}
}

func TestBuildPlanOptions_WhenEmptySession_ShouldReturnEmpty(t *testing.T) {
	session := NewSession()
	opts := BuildPlanOptions(session, nil)

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
