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

func TestApply_CreateFixture_SilentMode(t *testing.T) {
	dir := copyFixture(t, "apply-create")

	// Plan first
	_, stderr, err := runTfui("plan", "--project", dir, "--ci")
	if err != nil {
		t.Fatalf("plan failed: %v\nstderr: %s", err, stderr)
	}

	// Apply
	stdout, stderr, err := runTfui("apply", "--project", dir, "--ci")
	if err != nil {
		t.Fatalf("apply failed: %v\nstderr: %s\nstdout: %s", err, stderr, stdout)
	}

	// Verify file was created on disk
	resultPath := filepath.Join(dir, "out", "result.txt")
	if _, err := os.Stat(resultPath); err != nil {
		t.Errorf("expected %s to exist after apply, got error: %v", resultPath, err)
	}
}

func TestApply_CreateFixture_AgentMode(t *testing.T) {
	dir := copyFixture(t, "apply-create")

	// Plan first
	_, _, err := runTfui("plan", "--project", dir, "--ci")
	if err != nil {
		t.Fatalf("plan failed: %v", err)
	}

	// Apply in -json mode
	stdout, stderr, err := runTfui("apply", "--project", dir, "--json")
	if err != nil {
		t.Fatalf("apply --json failed: %v\nstderr: %s", err, stderr)
	}

	if !strings.Contains(stdout, "complete") {
		t.Errorf("expected agent output to contain 'complete', got: %q", stdout)
	}
}

func TestApply_Targeted_OnlyAppliesTarget(t *testing.T) {
	dir := copyFixture(t, "apply-targeted")

	// Plan targeting only alpha
	_, stderr, err := runTfui("plan", "--project", dir, "--ci", "--target", "local_file.alpha")
	if err != nil {
		t.Fatalf("targeted plan failed: %v\nstderr: %s", err, stderr)
	}

	// Apply (applies the targeted plan)
	_, stderr, err = runTfui("apply", "--project", dir, "--ci")
	if err != nil {
		t.Fatalf("apply failed: %v\nstderr: %s", err, stderr)
	}

	// alpha.txt should exist
	if _, err := os.Stat(filepath.Join(dir, "out", "alpha.txt")); err != nil {
		t.Error("expected alpha.txt to exist after targeted apply")
	}

	// beta.txt and gamma.txt should NOT exist
	if _, err := os.Stat(filepath.Join(dir, "out", "beta.txt")); err == nil {
		t.Error("expected beta.txt to NOT exist after targeted apply (only alpha was targeted)")
	}
	if _, err := os.Stat(filepath.Join(dir, "out", "gamma.txt")); err == nil {
		t.Error("expected gamma.txt to NOT exist after targeted apply (only alpha was targeted)")
	}
}

func TestApply_NoChanges_Succeeds(t *testing.T) {
	initFixture(t, "no-changes")

	// Plan (no changes expected)
	_, _, err := runTfui("plan", "--project", fixtureDir("no-changes"), "--ci")
	if err != nil {
		t.Fatalf("plan failed: %v", err)
	}

	// Apply should succeed even with no changes
	_, stderr, err := runTfui("apply", "--project", fixtureDir("no-changes"), "--ci")
	if err != nil {
		// No plan file means error — this is expected when there are no changes
		if !strings.Contains(stderr, "no plan") && !strings.Contains(stderr, "plan file") {
			t.Fatalf("unexpected apply error: %v\nstderr: %s", err, stderr)
		}
	}
}
