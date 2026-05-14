//go:build integration

package integration

import (
	"os"
	"sort"
	"testing"
)

func assertStateContains(t *testing.T, dir, address string) {
	t.Helper()
	resources := stateList(t, dir)
	for _, r := range resources {
		if r == address {
			return
		}
	}
	t.Errorf("expected state to contain %q, got: %v", address, resources)
}

func assertStateNotContains(t *testing.T, dir, address string) {
	t.Helper()
	resources := stateList(t, dir)
	for _, r := range resources {
		if r == address {
			t.Errorf("expected state NOT to contain %q, but it was present", address)
			return
		}
	}
}

func assertStateEquals(t *testing.T, dir string, expected []string) {
	t.Helper()
	got := stateList(t, dir)
	sort.Strings(got)
	exp := make([]string, len(expected))
	copy(exp, expected)
	sort.Strings(exp)
	if len(got) != len(exp) {
		t.Fatalf("state mismatch: expected %v, got %v", exp, got)
	}
	for i := range got {
		if got[i] != exp[i] {
			t.Fatalf("state mismatch at [%d]: expected %q, got %q\nfull expected: %v\nfull got: %v",
				i, exp[i], got[i], exp, got)
		}
	}
}

func assertStatesEqual(t *testing.T, dirA, dirB string) {
	t.Helper()
	stateA := stateList(t, dirA)
	stateB := stateList(t, dirB)
	sort.Strings(stateA)
	sort.Strings(stateB)
	if len(stateA) != len(stateB) {
		t.Fatalf("states differ in length: dirA has %d resources, dirB has %d\ndirA: %v\ndirB: %v",
			len(stateA), len(stateB), stateA, stateB)
	}
	for i := range stateA {
		if stateA[i] != stateB[i] {
			t.Fatalf("states differ at [%d]: dirA=%q, dirB=%q", i, stateA[i], stateB[i])
		}
	}
}

func assertFileExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected file %s to exist, got error: %v", path, err)
	}
}

func assertFileNotExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err == nil {
		t.Errorf("expected file %s to NOT exist, but it does", path)
	}
}
