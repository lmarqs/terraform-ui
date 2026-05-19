//go:build integration

package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// copyFixture copies a fixture to a temp directory for destructive tests.
func copyFixture(t *testing.T, name string) string {
	t.Helper()
	src := fixtureDir(name)
	dst := t.TempDir()
	copyDir(t, src, dst)
	initInDir(t, dst)
	return dst
}

// copyDir recursively copies src to dst, skipping .terraform directories.
func copyDir(t *testing.T, src, dst string) {
	t.Helper()
	entries, err := os.ReadDir(src)
	if err != nil {
		t.Fatalf("reading dir %q: %v", src, err)
	}

	for _, e := range entries {
		if e.Name() == ".terraform" || e.Name() == ".terraform.lock.hcl" {
			continue
		}
		srcPath := filepath.Join(src, e.Name())
		dstPath := filepath.Join(dst, e.Name())

		if e.IsDir() {
			if err := os.MkdirAll(dstPath, 0755); err != nil {
				t.Fatalf("mkdir %s: %v", dstPath, err)
			}
			copyDir(t, srcPath, dstPath)
		} else {
			data, err := os.ReadFile(srcPath)
			if err != nil {
				t.Fatalf("reading %s: %v", srcPath, err)
			}
			if err := os.WriteFile(dstPath, data, 0644); err != nil {
				t.Fatalf("writing %s: %v", dstPath, err)
			}
		}
	}
}

// initInDir runs terraform init in the given directory.
func initInDir(t *testing.T, dir string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(dir, ".terraform")); err == nil {
		return
	}
	cmd := exec.Command("terraform", "init", "-input=false")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("terraform init in %s: %v\n%s", dir, err, out)
	}
}

// applyInDir runs terraform apply directly to set up state.
func applyInDir(t *testing.T, dir string) {
	t.Helper()
	cmd := exec.Command("terraform", "apply", "-auto-approve", "-input=false")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("terraform apply in %s: %v\n%s", dir, err, out)
	}
}

func TestApply_CreateFixture_SilentMode(t *testing.T) {
	dir := copyFixture(t, "apply-create")

	_, stderr, err := runTfui("plan", "-project", dir, "-ci")
	if err != nil && !isExitCode(err, 2) {
		t.Fatalf("plan failed: %v\nstderr: %s", err, stderr)
	}

	stdout, stderr, err := runTfui("apply", "-project", dir, "-ci")
	if err != nil {
		t.Fatalf("apply failed: %v\nstderr: %s\nstdout: %s", err, stderr, stdout)
	}

	resultPath := filepath.Join(dir, "out", "result.txt")
	if _, err := os.Stat(resultPath); err != nil {
		t.Errorf("expected %s to exist after apply, got error: %v", resultPath, err)
	}
}

func TestApply_CreateFixture_AgentMode(t *testing.T) {
	dir := copyFixture(t, "apply-create")

	_, _, err := runTfui("plan", "-project", dir, "-ci")
	if err != nil && !isExitCode(err, 2) {
		t.Fatalf("plan failed: %v", err)
	}

	stdout, stderr, err := runTfui("apply", "-project", dir, "-json")
	if err != nil {
		t.Fatalf("apply -json failed: %v\nstderr: %s", err, stderr)
	}

	if !strings.Contains(stdout, "complete") {
		t.Errorf("expected agent output to contain 'complete', got: %q", stdout)
	}
}

func TestApply_Targeted_OnlyAppliesTarget(t *testing.T) {
	dir := copyFixture(t, "apply-targeted")

	_, stderr, err := runTfui("plan", "-project", dir, "-ci", "-target", "local_file.alpha")
	if err != nil && !isExitCode(err, 2) {
		t.Fatalf("targeted plan failed: %v\nstderr: %s", err, stderr)
	}

	_, stderr, err = runTfui("apply", "-project", dir, "-ci")
	if err != nil {
		t.Fatalf("apply failed: %v\nstderr: %s", err, stderr)
	}

	if _, err := os.Stat(filepath.Join(dir, "out", "alpha.txt")); err != nil {
		t.Error("expected alpha.txt to exist after targeted apply")
	}

	if _, err := os.Stat(filepath.Join(dir, "out", "beta.txt")); err == nil {
		t.Error("expected beta.txt to NOT exist after targeted apply (only alpha was targeted)")
	}
	if _, err := os.Stat(filepath.Join(dir, "out", "gamma.txt")); err == nil {
		t.Error("expected gamma.txt to NOT exist after targeted apply (only alpha was targeted)")
	}
}

func TestApply_NoChanges_Succeeds(t *testing.T) {
	dir := copyFixture(t, "no-changes")
	applyInDir(t, dir)

	stdout, _, err := runTfui("plan", "-project", dir, "-ci")
	if err != nil {
		t.Fatalf("plan failed: %v", err)
	}

	if !strings.Contains(stdout, "0 to add") {
		t.Fatalf("expected no changes in plan, got: %q", stdout)
	}

	_, stderr, err := runTfui("apply", "-project", dir, "-ci")
	if err != nil {
		if !strings.Contains(stderr, "no plan") && !strings.Contains(stderr, "plan file") {
			t.Fatalf("unexpected apply error: %v\nstderr: %s", err, stderr)
		}
	}
}
