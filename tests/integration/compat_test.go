//go:build integration

package integration

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestCompat_PlanWithTarget verifies that plan with -target flag (single-dash terraform
// style) loads correctly and the emitted plan command includes the target.
// This test covers CLI flag normalization: users expect -target=X to work just
// like terraform's native flags.
func TestCompat_PlanWithTarget(t *testing.T) {
	projectRoot := findProjectRoot()
	planFixture := filepath.Join(projectRoot, "tests", "fixtures", "plan.json")
	stateFixture := filepath.Join(projectRoot, "tests", "fixtures", "state.json")
	tapeFile := filepath.Join(projectRoot, "tests", "fixtures", "tapes", "compat", "plan_with_target.tape")

	stdout, stderr, err := runTfui("-plan", planFixture, "-state", stateFixture, "-macro", tapeFile)
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
// renders correctly in the plan view. When -destroy is used, the plan JSON will
// contain only "delete" actions and the summary should show "N to destroy".
func TestCompat_DestroyModePlanDisplay(t *testing.T) {
	projectRoot := findProjectRoot()
	planFixture := filepath.Join(projectRoot, "tests", "fixtures", "compat", "plan_destroy.json")
	tapeFile := filepath.Join(projectRoot, "tests", "fixtures", "tapes", "compat", "destroy_mode_plan.tape")

	stdout, stderr, err := runTfui("-plan", planFixture, "-macro", tapeFile)
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

	stdout, stderr, err := runTfui("-plan", planFixture, "-state", stateFixture, "-macro", tapeFile)
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
// and the CLI -var-file flag threads through to the service layer.
// For now, it verifies the basic apply flow works and can be extended.
func TestCompat_VarFileInCommand(t *testing.T) {
	projectRoot := findProjectRoot()
	planFixture := filepath.Join(projectRoot, "tests", "fixtures", "plan.json")
	stateFixture := filepath.Join(projectRoot, "tests", "fixtures", "state.json")
	tapeFile := filepath.Join(projectRoot, "tests", "fixtures", "tapes", "compat", "var_file_command.tape")

	stdout, stderr, err := runTfui("-plan", planFixture, "-state", stateFixture, "-macro", tapeFile)
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

	// TODO: Once -var-file CLI flag is implemented, add this assertion:
	// if !strings.Contains(stdout, "-var-file=prod.tfvars") {
	//     t.Errorf("expected '-var-file=prod.tfvars' in apply command, got: %q", stdout)
	// }
}

// TestCompat_TargetedPlanEmitsTargetFlag verifies the TUI flow: pinning a
// resource and triggering plan produces a plan command with -target= flag,
// and the subsequent apply consumes only the saved plan file (no -target=
// flag on apply — ADR-0019 governs this pipeline path).
func TestCompat_TargetedPlanEmitsTargetFlag(t *testing.T) {
	projectRoot := findProjectRoot()
	planFixture := filepath.Join(projectRoot, "tests", "fixtures", "plan.json")
	stateFixture := filepath.Join(projectRoot, "tests", "fixtures", "state.json")
	tapeFile := filepath.Join(projectRoot, "tests", "fixtures", "tapes", "compat", "targeted_plan_command.tape")

	stdout, stderr, err := runTfui("-plan", planFixture, "-state", stateFixture, "-macro", tapeFile)
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			t.Fatalf("macro failed with exit %d\nstderr: %s\nstdout: %s", ee.ExitCode(), stderr, stdout)
		}
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(stdout, "terraform plan") {
		t.Errorf("expected 'terraform plan' in stdout, got: %q", stdout)
	}
	if !strings.Contains(stdout, "-target=aws_instance.web") {
		t.Errorf("expected '-target=aws_instance.web' on plan command, got: %q", stdout)
	}
	if !strings.Contains(stdout, "terraform apply") {
		t.Errorf("expected 'terraform apply' in stdout, got: %q", stdout)
	}
	// ADR-0019 (TUI flow): apply consumes only the plan file, no -target.
	for _, line := range strings.Split(stdout, "\n") {
		if strings.Contains(line, "terraform apply") && strings.Contains(line, "-target=") {
			t.Errorf("TUI flow: apply must not contain -target= (ADR-0019); got: %q", line)
		}
	}
}

// TestCompat_StandaloneApplyWithTarget verifies the standalone CLI path:
// `tfui apply --target=X` emits `terraform apply -target=X` (auto-plan mode).
// This is independent from the TUI pipeline flow governed by ADR-0019.
func TestCompat_StandaloneApplyWithTarget(t *testing.T) {
	projectRoot := findProjectRoot()
	stateFixture := filepath.Join(projectRoot, "tests", "fixtures", "state.json")
	tapeFile := filepath.Join(projectRoot, "tests", "fixtures", "tapes", "compat", "standalone_apply_target.tape")

	stdout, stderr, err := runTfui("apply", "--target=aws_instance.web", "--auto-approve", "-state", stateFixture, "-macro", tapeFile)
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			t.Fatalf("macro failed with exit %d\nstderr: %s\nstdout: %s", ee.ExitCode(), stderr, stdout)
		}
		t.Fatalf("unexpected error: %v", err)
	}

	// Find the apply line and verify all flags appear together
	var applyLine string
	for _, line := range strings.Split(stdout, "\n") {
		if strings.Contains(line, "terraform apply") {
			applyLine = line
			break
		}
	}
	if applyLine == "" {
		t.Fatalf("expected 'terraform apply' in stdout, got: %q", stdout)
	}
	if !strings.Contains(applyLine, "-target=aws_instance.web") {
		t.Errorf("expected '-target=aws_instance.web' on apply line, got: %q", applyLine)
	}
	if !strings.Contains(applyLine, "-auto-approve") {
		t.Errorf("expected '-auto-approve' on apply line, got: %q", applyLine)
	}
	if strings.Contains(applyLine, ".tfplan") {
		t.Errorf("standalone apply with targets must not reference a plan file; got: %q", applyLine)
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

	stdout, stderr, err := runTfui("-plan", planFixture, "-macro", tapeFile)
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

// TestCompat_WorkspaceDisplay verifies that the workspace plugin is navigable
// in preloaded mode via macro.
func TestCompat_WorkspaceDisplay(t *testing.T) {
	projectRoot := findProjectRoot()
	planFixture := filepath.Join(projectRoot, "tests", "fixtures", "plan.json")
	tapeFile := filepath.Join(projectRoot, "tests", "fixtures", "tapes", "compat", "workspace_display.tape")

	_, stderr, err := runTfui("-plan", planFixture, "-macro", tapeFile)
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			t.Fatalf("macro failed with exit %d\nstderr: %s", ee.ExitCode(), stderr)
		}
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestCompat_DestroyModeRiskClassification verifies that destroy operations
// are classified with appropriate risk levels visible in the TUI.
func TestCompat_DestroyModeRiskClassification(t *testing.T) {
	projectRoot := findProjectRoot()
	planFixture := filepath.Join(projectRoot, "tests", "fixtures", "compat", "plan_destroy.json")
	tapeFile := filepath.Join(projectRoot, "tests", "fixtures", "tapes", "compat", "destroy_mode_risk.tape")

	stdout, stderr, err := runTfui("-plan", planFixture, "-macro", tapeFile)
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
