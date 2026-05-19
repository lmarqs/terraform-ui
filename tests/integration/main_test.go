//go:build integration

package integration

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

var binaryPath string

func TestMain(m *testing.M) {
	if _, err := exec.LookPath("terraform"); err != nil {
		fmt.Println("skipping integration tests: terraform not found")
		os.Exit(0)
	}

	projectRoot := findProjectRoot()
	binaryPath = filepath.Join(projectRoot, "dist", "tfui-integration-test")
	if runtime.GOOS == "windows" {
		binaryPath += ".exe"
	}

	buildCmd := exec.Command("go", "build", "-ldflags", "-X main.version=0.0.0-test", "-o", binaryPath, "./cmd/tfui")
	buildCmd.Dir = projectRoot
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr
	if err := buildCmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to build binary: %v\n", err)
		os.Exit(1)
	}

	code := m.Run()

	os.Remove(binaryPath)
	os.Exit(code)
}

func findProjectRoot() string {
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			panic("could not find project root (go.mod)")
		}
		dir = parent
	}
}

func fixtureDir(name string) string {
	return filepath.Join(findProjectRoot(), "tests", "fixtures", name)
}

func runTfui(args ...string) (string, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, binaryPath, args...)
	out, err := cmd.Output()
	var stderr []byte
	if ee, ok := err.(*exec.ExitError); ok {
		stderr = ee.Stderr
	}
	return string(out), string(stderr), err
}

func initFixture(t *testing.T, fixtureName string) string {
	t.Helper()
	dir := fixtureDir(fixtureName)

	cmd := exec.Command("terraform", "init")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("terraform init failed for fixture %q: %v\n%s", fixtureName, err, out)
	}

	return dir
}

func isExitCode(err error, code int) bool {
	if ee, ok := err.(*exec.ExitError); ok {
		return ee.ExitCode() == code
	}
	return false
}
