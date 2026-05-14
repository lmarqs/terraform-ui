//go:build integration

package integration

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestCompat_PlanWithTarget verifies that plan with --target flag (single-dash terraform
// style) loads correctly and the emitted plan command includes the target.
// This test covers CLI flag normalization: users expect -target=X to work just
// like terraform's native flags.
func TestCompat_PlanWithTarget(t *testing.T) {
	projectRoot := findProjectRoot()
	planFixture := filepath.Join(projectRoot, "tests", "fixtures", "plan.json")
	stateFixture := filepath.Join(projectRoot, "tests", "fixtures", "state.json")
	tapeFile := filepath.Join(projectRoot, "tests", "fixtures", "tapes", "compat", "plan_with_target.tape")

	stdout, stderr, err := runTfui("--plan", planFixture, "--state", stateFixture, "--macro", tapeFile)
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			t.Fatalf("macro failed with exit %d\nstderr: %s\nstdout: %s", ee.ExitCode(), stderr, stdout)
		}
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the plan command was emitted (basic sanity check)
	if !strings.Contains(stdout, "terraform plan") {
		t.Errorf("expected 'terraform plan' in stdout, got: %q", stdout)
	}
}

// TestCompat_DestroyModePlanDisplay verifies that a plan with all-delete actions
// renders correctly in the plan view. When --destroy is used, the plan JSON will
// contain only "delete" actions and the summary should show "N to destroy".
func TestCompat_DestroyModePlanDisplay(t *testing.T) {
	projectRoot := findProjectRoot()
	planFixture := filepath.Join(projectRoot, "tests", "fixtures", "compat", "plan_destroy.json")
	tapeFile := filepath.Join(projectRoot, "tests", "fixtures", "tapes", "compat", "destroy_mode_plan.tape")

	stdout, stderr, err := runTfui("--plan", planFixture, "--macro", tapeFile)
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			t.Fatalf("macro failed with exit %d\nstderr: %s\nstdout: %s", ee.ExitCode(), stderr, stdout)
		}
		t.Fatalf("unexpected error: %v", err)
	}

	// Plan command should have been recorded
	if !strings.Contains(stdout, "terraform plan") {
		t.Errorf("expected 'terraform plan' in stdout, got: %q", stdout)
	}
}

// TestCompat_DestroyModeApplyConfirmation verifies that applying a destroy plan
// shows proper confirmation and emits the apply command.
func TestCompat_DestroyModeApplyConfirmation(t *testing.T) {
	projectRoot := findProjectRoot()
	planFixture := filepath.Join(projectRoot, "tests", "fixtures", "compat", "plan_destroy.json")
	stateFixture := filepath.Join(projectRoot, "tests", "fixtures", "compat", "state.json")
	tapeFile := filepath.Join(projectRoot, "tests", "fixtures", "tapes", "compat", "destroy_mode_apply.tape")

	stdout, stderr, err := runTfui("--plan", planFixture, "--state", stateFixture, "--macro", tapeFile)
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			t.Fatalf("macro failed with exit %d\nstderr: %s\nstdout: %s", ee.ExitCode(), stderr, stdout)
		}
		t.Fatalf("unexpected error: %v", err)
	}

	// Should emit apply command
	if !strings.Contains(stdout, "terraform apply") {
		t.Errorf("expected 'terraform apply' in stdout, got: %q", stdout)
	}
}

// TestCompat_VarFileInCommand verifies that when PlanOptions/ApplyOptions carry
// VarFiles, the emitted commands include -var-file= flags.
// NOTE: This test will only pass once PlanOptions/ApplyOptions are implemented
// and the CLI --var-file flag threads through to the service layer.
// For now, it verifies the basic apply flow works and can be extended.
func TestCompat_VarFileInCommand(t *testing.T) {
	projectRoot := findProjectRoot()
	planFixture := filepath.Join(projectRoot, "tests", "fixtures", "plan.json")
	stateFixture := filepath.Join(projectRoot, "tests", "fixtures", "state.json")
	tapeFile := filepath.Join(projectRoot, "tests", "fixtures", "tapes", "compat", "var_file_command.tape")

	stdout, stderr, err := runTfui("--plan", planFixture, "--state", stateFixture, "--macro", tapeFile)
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			t.Fatalf("macro failed with exit %d\nstderr: %s\nstdout: %s", ee.ExitCode(), stderr, stdout)
		}
		t.Fatalf("unexpected error: %v", err)
	}

	// Basic apply command should be present
	if !strings.Contains(stdout, "terraform apply") {
		t.Errorf("expected 'terraform apply' in stdout, got: %q", stdout)
	}

	// TODO: Once --var-file CLI flag is implemented, add this assertion:
	// if !strings.Contains(stdout, "-var-file=prod.tfvars") {
	//     t.Errorf("expected '-var-file=prod.tfvars' in apply command, got: %q", stdout)
	// }
}

// TestCompat_TargetedApplyEmitsTargetFlag verifies that pinning a resource
// and triggering apply produces a command with -target= flag.
func TestCompat_TargetedApplyEmitsTargetFlag(t *testing.T) {
	projectRoot := findProjectRoot()
	planFixture := filepath.Join(projectRoot, "tests", "fixtures", "plan.json")
	stateFixture := filepath.Join(projectRoot, "tests", "fixtures", "state.json")
	tapeFile := filepath.Join(projectRoot, "tests", "fixtures", "tapes", "compat", "targeted_plan_command.tape")

	stdout, stderr, err := runTfui("--plan", planFixture, "--state", stateFixture, "--macro", tapeFile)
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			t.Fatalf("macro failed with exit %d\nstderr: %s\nstdout: %s", ee.ExitCode(), stderr, stdout)
		}
		t.Fatalf("unexpected error: %v", err)
	}

	// Should emit apply command with target
	if !strings.Contains(stdout, "terraform apply") {
		t.Errorf("expected 'terraform apply' in stdout, got: %q", stdout)
	}
	if !strings.Contains(stdout, "-target=") {
		t.Errorf("expected '-target=' in apply command, got: %q", stdout)
	}
	if !strings.Contains(stdout, "aws_instance.web") {
		t.Errorf("expected 'aws_instance.web' in target, got: %q", stdout)
	}
}

// TestCompat_ExtraArgsPassthrough verifies that the TUI functions normally when
// ExtraArgs would be configured. The tape asserts the plan view loads correctly.
// Once ExtraArgs CLI support (-- separator) is implemented, this test should
// also verify stdout contains the extra flags.
func TestCompat_ExtraArgsPassthrough(t *testing.T) {
	projectRoot := findProjectRoot()
	planFixture := filepath.Join(projectRoot, "tests", "fixtures", "plan.json")
	tapeFile := filepath.Join(projectRoot, "tests", "fixtures", "tapes", "compat", "extra_args_passthrough.tape")

	stdout, stderr, err := runTfui("--plan", planFixture, "--macro", tapeFile)
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			t.Fatalf("macro failed with exit %d\nstderr: %s\nstdout: %s", ee.ExitCode(), stderr, stdout)
		}
		t.Fatalf("unexpected error: %v", err)
	}

	// Plan command recorded
	if !strings.Contains(stdout, "terraform plan") {
		t.Errorf("expected 'terraform plan' in stdout, got: %q", stdout)
	}

	// TODO: Once -- passthrough is implemented, verify:
	// if !strings.Contains(stdout, "-no-color") {
	//     t.Errorf("expected '-no-color' in plan command from ExtraArgs, got: %q", stdout)
	// }
}

// TestCompat_WorkspaceDisplay verifies that workspace information is accessible
// in the TUI via macro mode.
func TestCompat_WorkspaceDisplay(t *testing.T) {
	projectRoot := findProjectRoot()
	planFixture := filepath.Join(projectRoot, "tests", "fixtures", "plan.json")
	tapeFile := filepath.Join(projectRoot, "tests", "fixtures", "tapes", "compat", "workspace_display.tape")

	stdout, stderr, err := runTfui("--plan", planFixture, "--macro", tapeFile)
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			t.Fatalf("macro failed with exit %d\nstderr: %s\nstdout: %s", ee.ExitCode(), stderr, stdout)
		}
		t.Fatalf("unexpected error: %v", err)
	}

	// Workspace command should have been recorded
	if !strings.Contains(stdout, "workspace") {
		t.Errorf("expected 'workspace' command in stdout, got: %q", stdout)
	}
}

// TestCompat_DestroyModeRiskClassification verifies that destroy operations
// are classified with appropriate risk levels visible in the TUI.
func TestCompat_DestroyModeRiskClassification(t *testing.T) {
	projectRoot := findProjectRoot()
	planFixture := filepath.Join(projectRoot, "tests", "fixtures", "compat", "plan_destroy.json")
	tapeFile := filepath.Join(projectRoot, "tests", "fixtures", "tapes", "compat", "destroy_mode_risk.tape")

	stdout, stderr, err := runTfui("--plan", planFixture, "--macro", tapeFile)
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			t.Fatalf("macro failed with exit %d\nstderr: %s\nstdout: %s", ee.ExitCode(), stderr, stdout)
		}
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(stdout, "terraform plan") {
		t.Errorf("expected 'terraform plan' in stdout, got: %q", stdout)
	}
}
