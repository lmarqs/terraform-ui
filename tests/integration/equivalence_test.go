//go:build integration

package integration

import (
	"path/filepath"
	"testing"
)

func TestEquivalence_Apply_CLI(t *testing.T) {
	dir := copyFixture(t, "apply-create")

	_, stderr, err := runTfui("plan", "--project", dir, "--ci")
	if err != nil {
		t.Fatalf("plan failed: %v\nstderr: %s", err, stderr)
	}

	_, stderr, err = runTfui("apply", "--project", dir, "--ci")
	if err != nil {
		t.Fatalf("apply failed: %v\nstderr: %s", err, stderr)
	}

	assertStateContains(t, dir, "local_file.result")
	assertStateEquals(t, dir, []string{"local_file.result"})
	assertFileExists(t, filepath.Join(dir, "out", "result.txt"))
}

func TestEquivalence_Apply_Targeted(t *testing.T) {
	dir := copyFixture(t, "apply-targeted")

	_, stderr, err := runTfui("plan", "--project", dir, "--ci", "--target", "local_file.alpha")
	if err != nil {
		t.Fatalf("targeted plan failed: %v\nstderr: %s", err, stderr)
	}

	_, stderr, err = runTfui("apply", "--project", dir, "--ci")
	if err != nil {
		t.Fatalf("apply failed: %v\nstderr: %s", err, stderr)
	}

	assertStateContains(t, dir, "local_file.alpha")
	assertStateNotContains(t, dir, "local_file.beta")
	assertStateNotContains(t, dir, "local_file.gamma")
	assertStateEquals(t, dir, []string{"local_file.alpha"})
	assertFileExists(t, filepath.Join(dir, "out", "alpha.txt"))
	assertFileNotExists(t, filepath.Join(dir, "out", "beta.txt"))
	assertFileNotExists(t, filepath.Join(dir, "out", "gamma.txt"))
}

func TestEquivalence_StateRm_CLI(t *testing.T) {
	dir := copyFixture(t, "state-ops")

	assertStateContains(t, dir, "local_file.one")
	assertStateContains(t, dir, "local_file.two")

	_, stderr, err := runTfui("state", "rm", "local_file.one", "--project", dir)
	if err != nil {
		t.Fatalf("state rm failed: %v\nstderr: %s", err, stderr)
	}

	assertStateNotContains(t, dir, "local_file.one")
	assertStateContains(t, dir, "local_file.two")
	assertStateEquals(t, dir, []string{"local_file.two"})
}

func TestEquivalence_Apply_CIvsJSON(t *testing.T) {
	// Path A: --ci (tree output, no spinner)
	dirA := copyFixture(t, "apply-create")
	_, stderr, err := runTfui("plan", "--project", dirA, "--ci")
	if err != nil {
		t.Fatalf("plan (ci) failed: %v\nstderr: %s", err, stderr)
	}
	_, stderr, err = runTfui("apply", "--project", dirA, "--ci")
	if err != nil {
		t.Fatalf("apply (ci) failed: %v\nstderr: %s", err, stderr)
	}

	// Path B: -json (NDJSON, terraform-compatible)
	dirB := copyFixture(t, "apply-create")
	_, stderr, err = runTfui("plan", "--project", dirB, "--json")
	if err != nil {
		t.Fatalf("plan (json) failed: %v\nstderr: %s", err, stderr)
	}
	_, stderr, err = runTfui("apply", "--project", dirB, "--json")
	if err != nil {
		t.Fatalf("apply (json) failed: %v\nstderr: %s", err, stderr)
	}

	assertStatesEqual(t, dirA, dirB)
	assertFileExists(t, filepath.Join(dirA, "out", "result.txt"))
	assertFileExists(t, filepath.Join(dirB, "out", "result.txt"))
}

func TestEquivalence_Apply_Targeted_CIvsJSON(t *testing.T) {
	// Path A: --ci with target
	dirA := copyFixture(t, "apply-targeted")
	_, stderr, err := runTfui("plan", "--project", dirA, "--ci", "--target", "local_file.alpha")
	if err != nil {
		t.Fatalf("plan (ci) failed: %v\nstderr: %s", err, stderr)
	}
	_, stderr, err = runTfui("apply", "--project", dirA, "--ci")
	if err != nil {
		t.Fatalf("apply (ci) failed: %v\nstderr: %s", err, stderr)
	}

	// Path B: -json with target
	dirB := copyFixture(t, "apply-targeted")
	_, stderr, err = runTfui("plan", "--project", dirB, "--json", "--target", "local_file.alpha")
	if err != nil {
		t.Fatalf("plan (json) failed: %v\nstderr: %s", err, stderr)
	}
	_, stderr, err = runTfui("apply", "--project", dirB, "--json")
	if err != nil {
		t.Fatalf("apply (json) failed: %v\nstderr: %s", err, stderr)
	}

	assertStatesEqual(t, dirA, dirB)
	assertFileExists(t, filepath.Join(dirA, "out", "alpha.txt"))
	assertFileExists(t, filepath.Join(dirB, "out", "alpha.txt"))
	assertFileNotExists(t, filepath.Join(dirA, "out", "beta.txt"))
	assertFileNotExists(t, filepath.Join(dirB, "out", "beta.txt"))
}
